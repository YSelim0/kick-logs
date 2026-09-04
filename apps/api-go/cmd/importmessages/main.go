// Command importmessages backfills chat_messages from a JSON message export
// (the "items" shape returned by the app's own message export/search paths).
// It is append-only: existing rows are looked up by kick_message_id and are
// never updated, and nothing is written unless -dry-run=false is combined
// with the exact -confirm phrase.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
)

// requiredConfirmPhrase guards the write path the same way admin cleanup
// requires exact confirmation text: -dry-run=false alone is not enough.
const requiredConfirmPhrase = "IMPORT-CHAT-MESSAGES"

const existsCheckChunkSize = 500
const insertChunkSize = 500

type exportFile struct {
	Items     []exportItem `json:"items"`
	Count     int          `json:"count"`
	MaxRows   int          `json:"max_rows"`
	Truncated bool         `json:"truncated"`
}

type exportItem struct {
	ID                     int64           `json:"id"`
	KickMessageID          string          `json:"kick_message_id"`
	ChatroomID             int64           `json:"chatroom_id"`
	Content                string          `json:"content"`
	MessageType            string          `json:"message_type"`
	SenderUsernameSnapshot string          `json:"sender_username_snapshot"`
	SenderSlugSnapshot     string          `json:"sender_slug_snapshot"`
	SenderColorSnapshot    string          `json:"sender_color_snapshot"`
	SenderBadges           json.RawMessage `json:"sender_badges"`
	Emotes                 []exportEmote   `json:"emotes"`
	ReplyMetadata          json.RawMessage `json:"reply_metadata"`
	ThreadParentID         *string         `json:"thread_parent_id"`
	MessageCreatedAt       string          `json:"message_created_at"`
	IngestedAt             string          `json:"ingested_at"`
	Sender                 exportSender    `json:"sender"`
	Channel                exportChannel   `json:"channel"`
}

type exportEmote struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Token    string `json:"token"`
	ImageURL string `json:"image_url"`
}

type exportSender struct {
	ID              int64   `json:"id"`
	KickUserID      int64   `json:"kick_user_id"`
	Username        string  `json:"username"`
	Slug            string  `json:"slug"`
	ProfileImageURL *string `json:"profile_image_url"`
}

type exportChannel struct {
	ID              int64   `json:"id"`
	Slug            string  `json:"slug"`
	DisplayName     string  `json:"display_name"`
	ProfileImageURL *string `json:"profile_image_url"`
	BannerImageURL  *string `json:"banner_image_url"`
}

type replyMetadata struct {
	OriginalMessage struct {
		Content string `json:"content"`
		ID      string `json:"id"`
	} `json:"original_message"`
	OriginalSender struct {
		Username string `json:"username"`
	} `json:"original_sender"`
}

type invalidRecord struct {
	Index         int
	KickMessageID string
	Reason        string
}

type report struct {
	TotalRead        int
	ToInsert         []domain.ChatMessage
	AlreadyExists    int
	AlreadyExistsIDs []string
	DuplicateInFile  int
	DuplicateIDs     []string
	Invalid          []invalidRecord
}

func main() {
	input := flag.String("input", "", "path to the JSON export file (required)")
	dryRun := flag.Bool("dry-run", true, "analyze and report only; no ClickHouse writes (default true)")
	limit := flag.Int("limit", 0, "only process the first N records from the export (0 = all)")
	confirm := flag.String("confirm", "", "must equal "+requiredConfirmPhrase+" to allow a real write when -dry-run=false")
	verifyCSV := flag.String("verify-csv", "", "optional: path to a CSV export of the same dataset, cross-checked by kick_message_id only")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if strings.TrimSpace(*input) == "" {
		logger.Error("missing required -input flag")
		os.Exit(1)
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		logger.Error("failed to read input file", "path", *input, "error", err)
		os.Exit(1)
	}

	var file exportFile
	if err := json.Unmarshal(data, &file); err != nil {
		logger.Error("failed to parse input JSON", "path", *input, "error", err)
		os.Exit(1)
	}
	logger.Info("loaded export file", "path", *input, "declared_count", file.Count, "items_in_file", len(file.Items), "truncated", file.Truncated)

	if *verifyCSV != "" {
		if err := verifyAgainstCSV(*verifyCSV, file.Items, logger); err != nil {
			logger.Error("csv verification failed", "error", err)
			os.Exit(1)
		}
	}

	items := file.Items
	if *limit > 0 && *limit < len(items) {
		items = items[:*limit]
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	conn, err := clickhouseinfra.Open(ctx, cfg)
	if err != nil {
		logger.Error("failed to open clickhouse connection", "error", err)
		os.Exit(1)
	}
	repo := clickhouseinfra.NewMessageRepository(conn)

	rep := buildReport(ctx, repo, items)
	printReport(logger, rep, len(items))

	if *dryRun {
		logger.Info("dry-run finished; no rows were written")
		printBackupReminder(logger)
		return
	}

	if *confirm != requiredConfirmPhrase {
		logger.Error(
			"refusing to write: pass -dry-run=false together with the exact -confirm phrase",
			"required_confirm", requiredConfirmPhrase,
		)
		os.Exit(1)
	}

	written, err := insertBatches(ctx, repo, rep.ToInsert, insertChunkSize)
	if err != nil {
		logger.Error("import failed partway through", "written_before_failure", written, "error", err)
		os.Exit(1)
	}
	logger.Info(
		"import complete",
		"written", written,
		"skipped_already_existed", rep.AlreadyExists,
		"skipped_duplicate_in_file", rep.DuplicateInFile,
		"skipped_invalid", len(rep.Invalid),
	)
}

func buildReport(ctx context.Context, repo *clickhouseinfra.MessageRepository, items []exportItem) report {
	rep := report{TotalRead: len(items)}
	seen := make(map[string]int, len(items))
	candidates := make([]domain.ChatMessage, 0, len(items))

	for index, item := range items {
		message, err := buildMessage(item)
		if err != nil {
			rep.Invalid = append(rep.Invalid, invalidRecord{
				Index:         index,
				KickMessageID: item.KickMessageID,
				Reason:        err.Error(),
			})
			continue
		}
		if firstIndex, dup := seen[message.KickMessageID]; dup {
			rep.DuplicateInFile++
			if len(rep.DuplicateIDs) < 10 {
				rep.DuplicateIDs = append(rep.DuplicateIDs, fmt.Sprintf("%s (rows %d and %d)", message.KickMessageID, firstIndex, index))
			}
			continue
		}
		seen[message.KickMessageID] = index
		candidates = append(candidates, message)
	}

	existing := checkExisting(ctx, repo, candidates)
	for _, message := range candidates {
		if existing[message.KickMessageID] {
			rep.AlreadyExists++
			if len(rep.AlreadyExistsIDs) < 10 {
				rep.AlreadyExistsIDs = append(rep.AlreadyExistsIDs, message.KickMessageID)
			}
			continue
		}
		rep.ToInsert = append(rep.ToInsert, message)
	}

	return rep
}

func checkExisting(ctx context.Context, repo *clickhouseinfra.MessageRepository, candidates []domain.ChatMessage) map[string]bool {
	result := make(map[string]bool, len(candidates))
	ids := make([]string, len(candidates))
	for i, message := range candidates {
		ids[i] = message.KickMessageID
	}
	for start := 0; start < len(ids); start += existsCheckChunkSize {
		end := start + existsCheckChunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk, err := repo.ExistingKickMessageIDs(ctx, ids[start:end])
		if err != nil {
			slog.Default().Error("failed to check existing kick_message_ids against ClickHouse; treating this chunk as unknown/not-existing", "error", err)
			continue
		}
		for id := range chunk {
			result[id] = true
		}
	}
	return result
}

func insertBatches(ctx context.Context, repo *clickhouseinfra.MessageRepository, messages []domain.ChatMessage, chunkSize int) (int, error) {
	written := 0
	for start := 0; start < len(messages); start += chunkSize {
		end := start + chunkSize
		if end > len(messages) {
			end = len(messages)
		}
		if err := repo.InsertMessagesBatch(ctx, messages[start:end]); err != nil {
			return written, fmt.Errorf("insert batch [%d:%d]: %w", start, end, err)
		}
		written += end - start
	}
	return written, nil
}

func buildMessage(item exportItem) (domain.ChatMessage, error) {
	kickMessageID := strings.TrimSpace(item.KickMessageID)
	if kickMessageID == "" {
		return domain.ChatMessage{}, fmt.Errorf("missing kick_message_id")
	}

	createdAt, err := parseExportTime(item.MessageCreatedAt)
	if err != nil {
		return domain.ChatMessage{}, fmt.Errorf("invalid message_created_at %q: %w", item.MessageCreatedAt, err)
	}

	channelSlug := strings.TrimSpace(item.Channel.Slug)
	if channelSlug == "" {
		return domain.ChatMessage{}, fmt.Errorf("missing channel.slug")
	}

	senderUsername := strings.TrimSpace(firstNonEmpty(item.Sender.Username, item.SenderUsernameSnapshot))
	senderSlug := normalizeKickProfileSlug(firstNonEmpty(item.Sender.Slug, item.SenderSlugSnapshot))
	if senderUsername == "" && senderSlug == "" {
		return domain.ChatMessage{}, fmt.Errorf("missing sender username and slug")
	}
	if senderSlug == "" {
		senderSlug = normalizeKickProfileSlug(senderUsername)
	}
	if senderUsername == "" {
		senderUsername = senderSlug
	}

	ingestedAt := createdAt
	if parsed, err := parseExportTime(item.IngestedAt); err == nil {
		ingestedAt = parsed
	}

	messageType := strings.TrimSpace(item.MessageType)
	if messageType == "" {
		messageType = "message"
	}

	var thread string
	if item.ThreadParentID != nil {
		thread = strings.TrimSpace(*item.ThreadParentID)
	}

	replySender, replyContent, replyMessageID := extractReplyMetadata(item.ReplyMetadata)

	emotes := make([]domain.ChatEmote, 0, len(item.Emotes))
	for _, e := range item.Emotes {
		emotes = append(emotes, domain.ChatEmote{ID: e.ID, Name: e.Name, Token: e.Token, ImageURL: e.ImageURL})
	}

	message := domain.ChatMessage{
		ID:                     deterministicMessageID(kickMessageID),
		KickMessageID:          kickMessageID,
		ChannelID:              item.Channel.ID,
		ChannelChatroomID:      item.ChatroomID,
		ChannelSlug:            channelSlug,
		ChannelDisplayName:     item.Channel.DisplayName,
		ChannelProfileImageURL: derefString(item.Channel.ProfileImageURL),
		ChannelBannerImageURL:  derefString(item.Channel.BannerImageURL),
		ChannelPublicURL:       kickPublicURL(channelSlug),
		SenderID:               item.Sender.ID,
		SenderKickID:           item.Sender.KickUserID,
		SenderUsername:         senderUsername,
		SenderSlug:             senderSlug,
		SenderDisplayColor:     item.SenderColorSnapshot,
		SenderProfileImageURL:  derefString(item.Sender.ProfileImageURL),
		SenderPublicURL:        kickPublicURL(senderSlug),
		SenderBadgesJSON:       rawOrDefault(item.SenderBadges, "[]"),
		MessageType:            messageType,
		Content:                item.Content,
		Emotes:                 emotes,
		ReplyToSender:          replySender,
		ReplyToContent:         replyContent,
		ReplyToMessageID:       replyMessageID,
		ThreadParentID:         thread,
		ReplyMetadataJSON:      rawOrDefault(item.ReplyMetadata, "{}"),
		RawPayloadJSON:         "{}",
		MessageCreatedAt:       createdAt,
		IngestedAt:             ingestedAt,
	}
	return message, nil
}

func extractReplyMetadata(raw json.RawMessage) (sender, content, messageID string) {
	if len(raw) == 0 {
		return "", "", ""
	}
	var meta replyMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return "", "", ""
	}
	return meta.OriginalSender.Username, meta.OriginalMessage.Content, meta.OriginalMessage.ID
}

func rawOrDefault(raw json.RawMessage, fallback string) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return fallback
	}
	return trimmed
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func parseExportTime(value string) (time.Time, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	if parsed, err := time.Parse(time.RFC3339, text); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

// deterministicMessageID mirrors internal/usecase/listener/normalizer.go's
// deterministicMessageID exactly (fnv64a over kick_message_id, masked into
// int63) so backfilled rows get the same id the live pipeline would have
// produced for the same kick_message_id. Duplicated here instead of
// importing the listener package, which pulls in unrelated live-ingestion
// dependencies for a one-off import tool.
func deterministicMessageID(kickMessageID string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(kickMessageID))
	id := int64(hash.Sum64() & uint64(math.MaxInt64))
	if id == 0 {
		return time.Now().UTC().UnixNano()
	}
	return id
}

func normalizeKickProfileSlug(value string) string {
	cleaned := strings.ToLower(strings.TrimSpace(value))
	return strings.ReplaceAll(cleaned, "_", "-")
}

func kickPublicURL(slug string) string {
	if strings.TrimSpace(slug) == "" {
		return ""
	}
	return "https://kick.com/" + strings.TrimSpace(slug)
}

func verifyAgainstCSV(path string, jsonItems []exportItem, logger *slog.Logger) error {
	handle, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}
	defer handle.Close()

	reader := csv.NewReader(handle)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read csv header: %w", err)
	}
	col := -1
	for i, name := range header {
		if strings.TrimSpace(name) == "kick_message_id" {
			col = i
			break
		}
	}
	if col == -1 {
		return fmt.Errorf("csv header has no kick_message_id column")
	}

	csvIDs := make(map[string]bool)
	for {
		row, err := reader.Read()
		if err != nil {
			break
		}
		if col < len(row) {
			id := strings.TrimSpace(row[col])
			if id != "" {
				csvIDs[id] = true
			}
		}
	}

	jsonIDs := make(map[string]bool, len(jsonItems))
	for _, item := range jsonItems {
		if id := strings.TrimSpace(item.KickMessageID); id != "" {
			jsonIDs[id] = true
		}
	}

	onlyInJSON := make([]string, 0)
	for id := range jsonIDs {
		if !csvIDs[id] {
			onlyInJSON = append(onlyInJSON, id)
		}
	}
	onlyInCSV := make([]string, 0)
	for id := range csvIDs {
		if !jsonIDs[id] {
			onlyInCSV = append(onlyInCSV, id)
		}
	}

	logger.Info(
		"csv cross-check (kick_message_id sets, informational only, does not affect import)",
		"csv_path", path,
		"json_ids", len(jsonIDs),
		"csv_ids", len(csvIDs),
		"only_in_json", len(onlyInJSON),
		"only_in_csv", len(onlyInCSV),
	)
	if len(onlyInJSON) > 0 || len(onlyInCSV) > 0 {
		sample := func(list []string) []string {
			if len(list) > 5 {
				return list[:5]
			}
			return list
		}
		logger.Warn("json/csv kick_message_id sets differ", "sample_only_in_json", sample(onlyInJSON), "sample_only_in_csv", sample(onlyInCSV))
	}
	return nil
}

func printReport(logger *slog.Logger, rep report, processedCount int) {
	logger.Info(
		"import plan summary",
		"records_read", processedCount,
		"would_insert", len(rep.ToInsert),
		"already_in_clickhouse", rep.AlreadyExists,
		"duplicate_within_file", rep.DuplicateInFile,
		"invalid_records", len(rep.Invalid),
	)
	if len(rep.AlreadyExistsIDs) > 0 {
		logger.Info("sample already-existing kick_message_ids (skipped, never modified)", "sample", rep.AlreadyExistsIDs)
	}
	if len(rep.DuplicateIDs) > 0 {
		logger.Info("sample duplicate-within-file kick_message_ids (only first occurrence considered)", "sample", rep.DuplicateIDs)
	}
	if len(rep.Invalid) > 0 {
		byReason := make(map[string]int)
		samples := make(map[string]string)
		for _, inv := range rep.Invalid {
			byReason[inv.Reason]++
			if _, ok := samples[inv.Reason]; !ok {
				samples[inv.Reason] = fmt.Sprintf("row %d (kick_message_id=%q)", inv.Index, inv.KickMessageID)
			}
		}
		for reason, count := range byReason {
			logger.Warn("invalid record reason", "reason", reason, "count", count, "example", samples[reason])
		}
	}
}

func printBackupReminder(logger *slog.Logger) {
	logger.Info(
		"before running with -dry-run=false, back up the clickhouse_data volume (see docs/operations/backup_restore.md); this import only INSERTs rows for kick_message_ids that do not already exist, so a backup lets you fully undo the run if needed",
	)
}
