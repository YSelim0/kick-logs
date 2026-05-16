package routes

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	messagesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messages"
)

var csvFields = []string{
	"message_created_at",
	"kick_message_id",
	"channel_slug",
	"channel_display_name",
	"sender_username",
	"sender_slug",
	"message_type",
	"content",
	"emotes",
	"reply_to_sender",
	"reply_to_content",
	"thread_parent_id",
}

func RegisterMessageRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /messages", func(response http.ResponseWriter, request *http.Request) {
		searchMessages(response, request, deps)
	})
	mux.HandleFunc("GET /messages/export", func(response http.ResponseWriter, request *http.Request) {
		exportMessages(response, request, deps)
	})
}

func searchMessages(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Messages == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	filter, err := parseMessageFilter(request, 50, 100)
	if err != nil {
		writeMessageFilterError(response, err)
		return
	}

	page, err := deps.Messages.Search(request.Context(), filter)
	if err != nil {
		writeMessageUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, messageSearchResponse(page))
}

func exportMessages(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if deps.Messages == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	format := strings.TrimSpace(request.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		writeError(response, http.StatusUnprocessableEntity, "Invalid export format.")
		return
	}

	maxRows := uint64(deps.Config.MessageExportMaxRows)
	if maxRows == 0 {
		maxRows = 1000
	}
	limit := maxRows
	if value := strings.TrimSpace(request.URL.Query().Get("limit")); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil || parsed < 1 {
			writeError(response, http.StatusUnprocessableEntity, "Invalid limit.")
			return
		}
		if parsed < limit {
			limit = parsed
		}
	}

	filter, err := parseMessageFilter(request, int(limit), int(limit))
	if err != nil {
		writeMessageFilterError(response, err)
		return
	}
	filter.Cursor = nil

	export, err := deps.Messages.Export(request.Context(), filter, limit)
	if err != nil {
		writeMessageUseCaseError(response, err)
		return
	}

	if format == "csv" {
		response.Header().Set("Content-Type", "text/csv; charset=utf-8")
		response.Header().Set("Content-Disposition", `attachment; filename="kick-logs-export.csv"`)
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte(messagesToCSV(export.Items)))
		return
	}

	writeJSON(response, http.StatusOK, messageExportResponse(export))
}

func parseMessageFilter(request *http.Request, defaultLimit int, maxLimit int) (domain.MessageSearchFilter, error) {
	query := request.URL.Query()
	filter := domain.MessageSearchFilter{}

	var err error
	if filter.Sender, err = optionalText(query.Get("sender"), 160); err != nil {
		return domain.MessageSearchFilter{}, err
	}
	if filter.Channel, err = optionalText(query.Get("channel"), 160); err != nil {
		return domain.MessageSearchFilter{}, err
	}
	if filter.Query, err = optionalText(query.Get("q"), 500); err != nil {
		return domain.MessageSearchFilter{}, err
	}
	if filter.Start, err = optionalTime(query.Get("start")); err != nil {
		return domain.MessageSearchFilter{}, err
	}
	if filter.End, err = optionalTime(query.Get("end")); err != nil {
		return domain.MessageSearchFilter{}, err
	}
	if !filter.Start.IsZero() && !filter.End.IsZero() && filter.Start.After(filter.End) {
		return domain.MessageSearchFilter{}, messagesusecase.ErrInvalidRange
	}
	if filter.ReplyOnly, err = optionalBool(query.Get("reply_only")); err != nil {
		return domain.MessageSearchFilter{}, err
	}
	if filter.EmoteOnly, err = optionalBool(query.Get("emote_only")); err != nil {
		return domain.MessageSearchFilter{}, err
	}
	if filter.Cursor, err = optionalCursor(query.Get("cursor")); err != nil {
		return domain.MessageSearchFilter{}, messagesusecase.ErrInvalidCursor
	}

	limit := defaultLimit
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 {
			return domain.MessageSearchFilter{}, errors.New("invalid limit")
		}
		limit = parsed
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	filter.Limit = uint64(limit)
	return filter, nil
}

func optionalText(value string, maxLength int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > maxLength {
		return "", errors.New("text filter too long")
	}
	return trimmed, nil
}

func optionalBool(value string) (bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, err
	}
	return parsed, nil
}

func optionalTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}
	normalized := strings.Replace(trimmed, "Z", "+00:00", 1)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	} {
		parsed, err := time.Parse(layout, normalized)
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, errors.New("invalid datetime")
}

func optionalCursor(value string) (*domain.MessageCursor, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	lastSeparator := strings.LastIndex(trimmed, "|")
	if lastSeparator < 1 || lastSeparator == len(trimmed)-1 {
		return nil, messagesusecase.ErrInvalidCursor
	}
	timestampText := trimmed[:lastSeparator]
	messageIDText := trimmed[lastSeparator+1:]

	createdAt, err := optionalTime(timestampText)
	if err != nil || createdAt.IsZero() {
		return nil, messagesusecase.ErrInvalidCursor
	}
	messageID, err := strconv.ParseInt(messageIDText, 10, 64)
	if err != nil || messageID < 1 {
		return nil, messagesusecase.ErrInvalidCursor
	}
	return &domain.MessageCursor{MessageCreatedAt: createdAt, MessageID: messageID}, nil
}

func messagesToCSV(messages []domain.ChatMessage) string {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	_ = writer.Write(csvFields)
	for _, message := range messages {
		_ = writer.Write(csvRow(message))
	}
	writer.Flush()
	return builder.String()
}

func csvRow(message domain.ChatMessage) []string {
	emotes, _ := json.Marshal(message.Emotes)
	metadata := replyMetadata(message)
	return []string{
		safeCSVValue(message.MessageCreatedAt.UTC().Format(time.RFC3339)),
		safeCSVValue(message.KickMessageID),
		safeCSVValue(message.ChannelSlug),
		safeCSVValue(message.ChannelDisplayName),
		safeCSVValue(message.SenderUsername),
		safeCSVValue(message.SenderSlug),
		safeCSVValue(message.MessageType),
		safeCSVValue(message.Content),
		safeCSVValue(string(emotes)),
		safeCSVValue(replyMetadataString(metadata, "original_sender", "username")),
		safeCSVValue(replyMetadataString(metadata, "original_message", "content")),
		safeCSVValue(message.ThreadParentID),
	}
}

func replyMetadataString(metadata map[string]any, objectKey string, valueKey string) string {
	if objectValue, ok := metadata[objectKey].(map[string]any); ok {
		if value, ok := objectValue[valueKey].(string); ok {
			return value
		}
	}
	return ""
}

func safeCSVValue(value string) string {
	if strings.HasPrefix(value, "=") ||
		strings.HasPrefix(value, "+") ||
		strings.HasPrefix(value, "-") ||
		strings.HasPrefix(value, "@") {
		return "'" + value
	}
	return value
}

func writeMessageFilterError(response http.ResponseWriter, err error) {
	if errors.Is(err, messagesusecase.ErrInvalidCursor) {
		writeError(response, http.StatusUnprocessableEntity, "Invalid cursor.")
		return
	}
	if errors.Is(err, messagesusecase.ErrInvalidRange) {
		writeError(response, http.StatusUnprocessableEntity, "Search start datetime must be before end datetime.")
		return
	}
	writeError(response, http.StatusUnprocessableEntity, "Invalid query parameters.")
}

func writeMessageUseCaseError(response http.ResponseWriter, err error) {
	if errors.Is(err, messagesusecase.ErrInvalidRange) {
		writeError(response, http.StatusUnprocessableEntity, "Search start datetime must be before end datetime.")
		return
	}
	writeError(response, http.StatusInternalServerError, "Internal server error.")
}
