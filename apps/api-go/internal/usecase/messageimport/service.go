package messageimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
)

var (
	ErrValidation    = errors.New("validation failed")
	ErrCannotExecute = errors.New("import cannot be executed")
	ErrConfirmation  = errors.New("import confirmation mismatch")
)

// ConfirmationText is the exact phrase an operator must type before an import
// writes anything, mirroring the data cleanup confirmation contract.
const ConfirmationText = "IMPORT MESSAGES"

const (
	existsCheckChunkSize = 500
	insertChunkSize      = 500
	maxSampleSize        = 10
)

// Service imports chat messages from a JSON export into the message store.
// It is append-only: rows are matched by kick_message_id and an existing row
// is never updated, only skipped.
type Service struct {
	messages ports.MessageRepository
	maxRows  int
}

func NewService(messages ports.MessageRepository, maxRows int) *Service {
	return &Service{messages: messages, maxRows: maxRows}
}

// Preview reports what an import of the given export payload would do without
// writing anything.
func (service *Service) Preview(
	ctx context.Context,
	payload []byte,
	limit int,
) (domain.MessageImportPreview, error) {
	analysis, err := service.analyze(ctx, payload, limit)
	if err != nil {
		return domain.MessageImportPreview{}, err
	}
	return analysis.preview, nil
}

// Confirm re-analyzes the payload and inserts only the rows whose
// kick_message_id is not already present.
func (service *Service) Confirm(
	ctx context.Context,
	payload []byte,
	limit int,
	confirmationText string,
) (domain.MessageImportResult, error) {
	analysis, err := service.analyze(ctx, payload, limit)
	if err != nil {
		return domain.MessageImportResult{}, err
	}
	if !analysis.preview.CanExecute {
		return domain.MessageImportResult{}, fmt.Errorf("%w: %s", ErrCannotExecute, analysis.preview.Reason)
	}
	if strings.TrimSpace(confirmationText) != ConfirmationText {
		return domain.MessageImportResult{}, ErrConfirmation
	}

	written, err := service.insertBatches(ctx, analysis.toInsert)
	if err != nil {
		return domain.MessageImportResult{}, err
	}
	return domain.MessageImportResult{
		Written:          written,
		AlreadyExists:    analysis.preview.AlreadyExists,
		DuplicateInFile:  analysis.preview.DuplicateInFile,
		Invalid:          analysis.preview.Invalid,
		ConfirmationText: ConfirmationText,
	}, nil
}

type analysis struct {
	preview  domain.MessageImportPreview
	toInsert []domain.ChatMessage
}

func (service *Service) analyze(ctx context.Context, payload []byte, limit int) (analysis, error) {
	file, err := ParseExport(payload)
	if err != nil {
		return analysis{}, err
	}

	items := file.Items
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	if service.maxRows > 0 && len(items) > service.maxRows {
		return analysis{}, fmt.Errorf(
			"%w: export has %d rows, which exceeds the %d row limit; use a limit or split the file",
			ErrValidation, len(items), service.maxRows,
		)
	}

	preview := domain.MessageImportPreview{
		TotalInFile:      len(file.Items),
		RecordsRead:      len(items),
		Limit:            limit,
		ConfirmationText: ConfirmationText,
	}

	seen := make(map[string]bool, len(items))
	candidates := make([]domain.ChatMessage, 0, len(items))
	invalidByReason := make(map[string]*domain.MessageImportInvalidReason)
	invalidOrder := make([]string, 0)

	for index, item := range items {
		message, err := BuildMessage(item)
		if err != nil {
			preview.Invalid++
			reason := err.Error()
			existing, ok := invalidByReason[reason]
			if !ok {
				invalidByReason[reason] = &domain.MessageImportInvalidReason{
					Reason:  reason,
					Count:   1,
					Example: fmt.Sprintf("row %d (kick_message_id=%q)", index, item.KickMessageID),
				}
				invalidOrder = append(invalidOrder, reason)
				continue
			}
			existing.Count++
			continue
		}
		if seen[message.KickMessageID] {
			preview.DuplicateInFile++
			continue
		}
		seen[message.KickMessageID] = true
		candidates = append(candidates, message)
	}

	for _, reason := range invalidOrder {
		preview.InvalidReasons = append(preview.InvalidReasons, *invalidByReason[reason])
	}

	existing, err := service.existingIDs(ctx, candidates)
	if err != nil {
		return analysis{}, err
	}

	toInsert := make([]domain.ChatMessage, 0, len(candidates))
	for _, message := range candidates {
		if existing[message.KickMessageID] {
			preview.AlreadyExists++
			continue
		}
		toInsert = append(toInsert, message)
		if len(preview.SampleToInsertIDs) < maxSampleSize {
			preview.SampleToInsertIDs = append(preview.SampleToInsertIDs, message.KickMessageID)
		}
	}
	preview.ToInsert = len(toInsert)

	if preview.ToInsert == 0 {
		preview.CanExecute = false
		preview.Reason = "No new messages to import."
	} else {
		preview.CanExecute = true
	}

	return analysis{preview: preview, toInsert: toInsert}, nil
}

func (service *Service) existingIDs(
	ctx context.Context,
	candidates []domain.ChatMessage,
) (map[string]bool, error) {
	result := make(map[string]bool, len(candidates))
	ids := make([]string, len(candidates))
	for index, message := range candidates {
		ids[index] = message.KickMessageID
	}
	for start := 0; start < len(ids); start += existsCheckChunkSize {
		end := start + existsCheckChunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk, err := service.messages.ExistingKickMessageIDs(ctx, ids[start:end])
		if err != nil {
			return nil, fmt.Errorf("check existing kick message ids: %w", err)
		}
		for id := range chunk {
			result[id] = true
		}
	}
	return result, nil
}

func (service *Service) insertBatches(ctx context.Context, messages []domain.ChatMessage) (int, error) {
	written := 0
	for start := 0; start < len(messages); start += insertChunkSize {
		end := start + insertChunkSize
		if end > len(messages) {
			end = len(messages)
		}
		if err := service.messages.InsertMessagesBatch(ctx, messages[start:end]); err != nil {
			return written, fmt.Errorf("insert message batch [%d:%d]: %w", start, end, err)
		}
		written += end - start
	}
	return written, nil
}

// ParseExport decodes the JSON export shape produced by the app's own message
// export/search paths.
func ParseExport(payload []byte) (ExportFile, error) {
	var file ExportFile
	if err := json.Unmarshal(payload, &file); err != nil {
		return ExportFile{}, fmt.Errorf("%w: input is not a valid JSON export: %s", ErrValidation, err)
	}
	if file.Items == nil {
		return ExportFile{}, fmt.Errorf("%w: input JSON has no \"items\" array", ErrValidation)
	}
	return file, nil
}

// BuildMessage maps one export row onto the stored chat message shape. Fields
// the export does not carry are left unset rather than invented.
func BuildMessage(item ExportItem) (domain.ChatMessage, error) {
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

	var threadParentID string
	if item.ThreadParentID != nil {
		threadParentID = strings.TrimSpace(*item.ThreadParentID)
	}

	replySender, replyContent, replyMessageID := extractReplyMetadata(item.ReplyMetadata)

	emotes := make([]domain.ChatEmote, 0, len(item.Emotes))
	for _, emote := range item.Emotes {
		emotes = append(emotes, domain.ChatEmote{
			ID:       emote.ID,
			Name:     emote.Name,
			Token:    emote.Token,
			ImageURL: emote.ImageURL,
		})
	}

	return domain.ChatMessage{
		ID:                     DeterministicMessageID(kickMessageID),
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
		ThreadParentID:         threadParentID,
		ReplyMetadataJSON:      rawOrDefault(item.ReplyMetadata, "{}"),
		RawPayloadJSON:         "{}",
		MessageCreatedAt:       createdAt,
		IngestedAt:             ingestedAt,
	}, nil
}

// DeterministicMessageID mirrors the live listener normalizer's message id
// derivation so a backfilled row gets the same id live ingestion would have
// produced for the same kick_message_id.
func DeterministicMessageID(kickMessageID string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(kickMessageID))
	id := int64(hash.Sum64() & uint64(math.MaxInt64))
	if id == 0 {
		return time.Now().UTC().UnixNano()
	}
	return id
}

func extractReplyMetadata(raw json.RawMessage) (sender string, content string, messageID string) {
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
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
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

func normalizeKickProfileSlug(value string) string {
	cleaned := strings.ToLower(strings.TrimSpace(value))
	return strings.ReplaceAll(cleaned, "_", "-")
}

func kickPublicURL(slug string) string {
	trimmed := strings.TrimSpace(slug)
	if trimmed == "" {
		return ""
	}
	return "https://kick.com/" + trimmed
}
