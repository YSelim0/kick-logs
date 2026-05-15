package httpapi

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/routes"
	messagesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messages"
)

func TestMessageRoutesSearchFiltersAndPagination(t *testing.T) {
	handler := newMessageTestRouter()

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/messages?limit=1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}

	var firstPage messageSearchPayload
	decodeResponse(t, response, &firstPage)
	if len(firstPage.Items) != 1 || firstPage.Items[0].ID != 101 {
		t.Fatalf("first page = %#v", firstPage.Items)
	}
	if firstPage.NextCursor == nil {
		t.Fatal("next cursor is nil")
	}

	nextResponse := httptest.NewRecorder()
	handler.ServeHTTP(
		nextResponse,
		httptest.NewRequest(http.MethodGet, "/messages?limit=1&cursor="+url.QueryEscape(*firstPage.NextCursor), nil),
	)
	if nextResponse.Code != http.StatusOK {
		t.Fatalf("next status = %d body = %s", nextResponse.Code, nextResponse.Body.String())
	}

	var nextPage messageSearchPayload
	decodeResponse(t, nextResponse, &nextPage)
	if len(nextPage.Items) != 1 || nextPage.Items[0].ID == firstPage.Items[0].ID {
		t.Fatalf("cursor duplicated items: first=%#v next=%#v", firstPage.Items, nextPage.Items)
	}

	senderResponse := httptest.NewRecorder()
	handler.ServeHTTP(senderResponse, httptest.NewRequest(http.MethodGet, "/messages?sender=YAVUZ&limit=10", nil))
	if senderResponse.Code != http.StatusOK {
		t.Fatalf("sender status = %d body = %s", senderResponse.Code, senderResponse.Body.String())
	}

	var senderPage messageSearchPayload
	decodeResponse(t, senderResponse, &senderPage)
	if len(senderPage.Items) != 1 || senderPage.Items[0].Sender.Username != "Yavuz" {
		t.Fatalf("sender exact page = %#v", senderPage.Items)
	}
}

func TestMessageRoutesSearchCombinationFilters(t *testing.T) {
	handler := newMessageTestRouter()
	request := httptest.NewRequest(
		http.MethodGet,
		"/messages?channel=hyp&q=combo&start=2035-01-01T12:00:00Z&end=2035-01-01T12:03:00Z&limit=10",
		nil,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}

	var page messageSearchPayload
	decodeResponse(t, response, &page)
	if len(page.Items) != 2 {
		t.Fatalf("combined filters item count = %d items = %#v", len(page.Items), page.Items)
	}
	if page.Items[0].ID != 101 || page.Items[1].ID != 100 {
		t.Fatalf("combined filters order = %#v", page.Items)
	}
}

func TestMessageRoutesReplyAndEmoteFilters(t *testing.T) {
	handler := newMessageTestRouter()

	replyResponse := httptest.NewRecorder()
	handler.ServeHTTP(replyResponse, httptest.NewRequest(http.MethodGet, "/messages?reply_only=true&limit=10", nil))
	if replyResponse.Code != http.StatusOK {
		t.Fatalf("reply status = %d body = %s", replyResponse.Code, replyResponse.Body.String())
	}
	var replyPage messageSearchPayload
	decodeResponse(t, replyResponse, &replyPage)
	if len(replyPage.Items) != 1 || replyPage.Items[0].MessageType != "reply" {
		t.Fatalf("reply page = %#v", replyPage.Items)
	}

	emoteResponse := httptest.NewRecorder()
	handler.ServeHTTP(emoteResponse, httptest.NewRequest(http.MethodGet, "/messages?emote_only=true&limit=10", nil))
	if emoteResponse.Code != http.StatusOK {
		t.Fatalf("emote status = %d body = %s", emoteResponse.Code, emoteResponse.Body.String())
	}
	var emotePage messageSearchPayload
	decodeResponse(t, emoteResponse, &emotePage)
	if len(emotePage.Items) != 1 || len(emotePage.Items[0].Emotes) != 1 {
		t.Fatalf("emote page = %#v", emotePage.Items)
	}
}

func TestMessageRoutesExportJSONAndCSV(t *testing.T) {
	handler := newMessageTestRouter()

	jsonResponse := httptest.NewRecorder()
	handler.ServeHTTP(jsonResponse, httptest.NewRequest(http.MethodGet, "/messages/export?format=json&limit=1", nil))
	if jsonResponse.Code != http.StatusOK {
		t.Fatalf("json export status = %d body = %s", jsonResponse.Code, jsonResponse.Body.String())
	}
	var export messageExportPayload
	decodeResponse(t, jsonResponse, &export)
	if export.Count != 1 || export.MaxRows != 1 || !export.Truncated || len(export.Items) != 1 {
		t.Fatalf("json export = %#v", export)
	}
	if export.Items[0].SenderBadges[0]["text"] != "founder" {
		t.Fatalf("sender badges = %#v", export.Items[0].SenderBadges)
	}

	csvResponse := httptest.NewRecorder()
	handler.ServeHTTP(csvResponse, httptest.NewRequest(http.MethodGet, "/messages/export?format=csv&q=command&limit=10", nil))
	if csvResponse.Code != http.StatusOK {
		t.Fatalf("csv export status = %d body = %s", csvResponse.Code, csvResponse.Body.String())
	}
	if contentType := csvResponse.Result().Header.Get("Content-Type"); contentType != "text/csv; charset=utf-8" {
		t.Fatalf("content type = %q", contentType)
	}

	rows, err := csv.NewReader(strings.NewReader(csvResponse.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	expectedHeader := []string{
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
	if len(rows) != 2 || strings.Join(rows[0], "|") != strings.Join(expectedHeader, "|") {
		t.Fatalf("csv rows = %#v", rows)
	}
	if rows[1][7] != "'=command reply" {
		t.Fatalf("csv formula-safe content = %q", rows[1][7])
	}
	if rows[1][9] != "Yavuz" || rows[1][10] != "original content" {
		t.Fatalf("csv reply fields = %#v", rows[1])
	}
}

func TestMessageRoutesRejectInvalidCursor(t *testing.T) {
	handler := newMessageTestRouter()

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/messages?cursor=not-a-cursor", nil))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Invalid cursor.") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func newMessageTestRouter() http.Handler {
	cfg := config.Config{
		BackendCORSOrigins:   []string{"http://localhost:3000"},
		JWTCookieName:        "kick_logs_session",
		MessageExportMaxRows: 2,
	}
	repository := &fakeMessageRepository{messages: messageFixtures()}
	return NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{
		Config:   cfg,
		Messages: messagesusecase.NewService(repository),
	})
}

func messageFixtures() []domain.ChatMessage {
	base := time.Date(2035, 1, 1, 12, 0, 0, 0, time.UTC)
	return []domain.ChatMessage{
		{
			ID:                     101,
			KickMessageID:          "kick-101",
			ChannelID:              1,
			ChannelKickID:          1001,
			ChannelChatroomID:      2001,
			ChannelSlug:            "hype",
			ChannelDisplayName:     "Hype",
			ChannelProfileImageURL: "https://files.kick.com/channel/hype.png",
			ChannelBannerImageURL:  "https://files.kick.com/channel/hype-banner.png",
			ChannelPublicURL:       "https://kick.com/hype",
			SenderID:               10,
			SenderKickID:           110,
			SenderUsername:         "Yavuz",
			SenderSlug:             "yavuz",
			SenderDisplayColor:     "#FFF600",
			SenderProfileImageURL:  "https://files.kick.com/user/yavuz.png",
			SenderPublicURL:        "https://kick.com/yavuz",
			SenderBadgesJSON:       `[{"text":"founder"}]`,
			MessageType:            "message",
			Content:                "hello combo search",
			Emotes: []domain.ChatEmote{
				{ID: "1", Name: "wave", Token: "[emote:1:wave]", ImageURL: "https://files.kick.com/emotes/1/fullsize"},
			},
			ReplyMetadataJSON: "{}",
			RawPayloadJSON:    "{}",
			MessageCreatedAt:  base.Add(3 * time.Minute),
			IngestedAt:        base.Add(4 * time.Minute),
		},
		{
			ID:                 100,
			KickMessageID:      "kick-100",
			ChannelID:          1,
			ChannelKickID:      1001,
			ChannelChatroomID:  2001,
			ChannelSlug:        "hype",
			ChannelDisplayName: "Hype",
			ChannelPublicURL:   "https://kick.com/hype",
			SenderID:           11,
			SenderKickID:       111,
			SenderUsername:     "yavuz_extra",
			SenderSlug:         "yavuz-extra",
			SenderBadgesJSON:   "[]",
			MessageType:        "message",
			Content:            "hello combo partial",
			ReplyMetadataJSON:  "{}",
			RawPayloadJSON:     "{}",
			MessageCreatedAt:   base.Add(2 * time.Minute),
			IngestedAt:         base.Add(3 * time.Minute),
		},
		{
			ID:                 99,
			KickMessageID:      "kick-99",
			ChannelID:          2,
			ChannelKickID:      1002,
			ChannelChatroomID:  2002,
			ChannelSlug:        "other",
			ChannelDisplayName: "Other",
			ChannelPublicURL:   "https://kick.com/other",
			SenderID:           12,
			SenderKickID:       112,
			SenderUsername:     "Alice",
			SenderSlug:         "alice",
			SenderBadgesJSON:   "[]",
			MessageType:        "reply",
			Content:            "=command reply",
			ReplyMetadataJSON:  `{"original_sender":{"username":"Yavuz"},"original_message":{"content":"original content"}}`,
			ThreadParentID:     "kick-101",
			RawPayloadJSON:     "{}",
			MessageCreatedAt:   base.Add(time.Minute),
			IngestedAt:         base.Add(2 * time.Minute),
		},
		{
			ID:                 98,
			KickMessageID:      "kick-98",
			ChannelID:          2,
			ChannelKickID:      1002,
			ChannelChatroomID:  2002,
			ChannelSlug:        "other",
			ChannelDisplayName: "Other",
			ChannelPublicURL:   "https://kick.com/other",
			SenderID:           13,
			SenderKickID:       113,
			SenderUsername:     "Bob",
			SenderSlug:         "bob",
			SenderBadgesJSON:   "[]",
			MessageType:        "message",
			Content:            "older message",
			ReplyMetadataJSON:  "{}",
			RawPayloadJSON:     "{}",
			MessageCreatedAt:   base,
			IngestedAt:         base.Add(time.Minute),
		},
	}
}

type fakeMessageRepository struct {
	messages []domain.ChatMessage
}

func (repository *fakeMessageRepository) Insert(_ context.Context, message domain.ChatMessage) error {
	repository.messages = append(repository.messages, message)
	return nil
}

func (repository *fakeMessageRepository) Search(
	_ context.Context,
	filter domain.MessageSearchFilter,
) ([]domain.ChatMessage, error) {
	messages := append([]domain.ChatMessage(nil), repository.messages...)
	sort.Slice(messages, func(left, right int) bool {
		if messages[left].MessageCreatedAt.Equal(messages[right].MessageCreatedAt) {
			return messages[left].ID > messages[right].ID
		}
		return messages[left].MessageCreatedAt.After(messages[right].MessageCreatedAt)
	})

	limit := filter.Limit
	if limit == 0 {
		limit = 50
	}
	results := make([]domain.ChatMessage, 0, len(messages))
	for _, message := range messages {
		if !messageMatchesFilter(message, filter) {
			continue
		}
		results = append(results, message)
		if uint64(len(results)) >= limit {
			break
		}
	}
	return results, nil
}

func messageMatchesFilter(message domain.ChatMessage, filter domain.MessageSearchFilter) bool {
	if filter.Sender != "" {
		sender := strings.ToLower(filter.Sender)
		if strings.ToLower(message.SenderUsername) != sender && strings.ToLower(message.SenderSlug) != sender {
			return false
		}
	}
	if filter.Channel != "" {
		channel := strings.ToLower(filter.Channel)
		if !strings.Contains(strings.ToLower(message.ChannelSlug), channel) &&
			!strings.Contains(strings.ToLower(message.ChannelDisplayName), channel) {
			return false
		}
	}
	if filter.Query != "" && !strings.Contains(strings.ToLower(message.Content), strings.ToLower(filter.Query)) {
		return false
	}
	if !filter.Start.IsZero() && message.MessageCreatedAt.Before(filter.Start) {
		return false
	}
	if !filter.End.IsZero() && message.MessageCreatedAt.After(filter.End) {
		return false
	}
	if filter.ReplyOnly && message.MessageType != "reply" {
		return false
	}
	if filter.EmoteOnly && len(message.Emotes) == 0 {
		return false
	}
	if filter.Cursor != nil &&
		!message.MessageCreatedAt.Before(filter.Cursor.MessageCreatedAt) &&
		!(message.MessageCreatedAt.Equal(filter.Cursor.MessageCreatedAt) && message.ID < filter.Cursor.MessageID) {
		return false
	}
	return true
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v body = %s", err, response.Body.String())
	}
}

type messageSearchPayload struct {
	Items      []messagePayload `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

type messageExportPayload struct {
	Items     []messagePayload `json:"items"`
	Count     int              `json:"count"`
	MaxRows   int              `json:"max_rows"`
	Truncated bool             `json:"truncated"`
}

type messagePayload struct {
	ID           int64            `json:"id"`
	MessageType  string           `json:"message_type"`
	SenderBadges []map[string]any `json:"sender_badges"`
	Emotes       []struct {
		ID string `json:"id"`
	} `json:"emotes"`
	Sender struct {
		Username string `json:"username"`
	} `json:"sender"`
}
