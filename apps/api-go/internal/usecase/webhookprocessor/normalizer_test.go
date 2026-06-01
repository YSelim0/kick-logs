package webhookprocessor_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/webhookprocessor"
)

var testChannel = domain.FollowedChannel{
	ID:                1,
	BroadcasterUserID: 9000,
	Slug:              "test-channel",
	DisplayName:       "Test Channel",
}

func TestNormalizeNewSubscription(t *testing.T) {
	payload := map[string]any{
		"broadcaster": map[string]any{"user_id": 9000, "username": "test-channel", "channel_slug": "test-channel"},
		"subscriber":  map[string]any{"user_id": 1001, "username": "subscriber1", "channel_slug": "subscriber1", "profile_picture": "https://pic.example.com/1"},
		"created_at":  "2026-06-01T10:00:00Z",
		"expires_at":  "2026-07-01T10:00:00Z",
	}
	event := makeEvent("msg-new-001", webhookprocessor.EventTypeNew, payload)

	periods, err := webhookprocessor.NormalizeEvent(event, testChannel)
	if err != nil {
		t.Fatalf("NormalizeEvent: %v", err)
	}
	if len(periods) != 1 {
		t.Fatalf("periods len = %d, want 1", len(periods))
	}
	p := periods[0]
	if p.ID != "msg-new-001" {
		t.Errorf("ID = %q", p.ID)
	}
	if p.SubscriberKickUserID != 1001 {
		t.Errorf("SubscriberKickUserID = %d", p.SubscriberKickUserID)
	}
	if p.IsGift {
		t.Error("IsGift = true on new sub")
	}
	if p.ExpiresAt.IsZero() {
		t.Error("ExpiresAt is zero")
	}
	if p.FollowedChannelID != 1 {
		t.Errorf("FollowedChannelID = %d", p.FollowedChannelID)
	}
}

func TestNormalizeRenewal(t *testing.T) {
	payload := map[string]any{
		"broadcaster": map[string]any{"user_id": 9000},
		"subscriber":  map[string]any{"user_id": 2002, "username": "renewer"},
		"created_at":  "2026-06-01T10:00:00Z",
		"expires_at":  "2026-07-01T10:00:00Z",
	}
	event := makeEvent("msg-renewal-001", webhookprocessor.EventTypeRenewal, payload)

	periods, err := webhookprocessor.NormalizeEvent(event, testChannel)
	if err != nil {
		t.Fatalf("NormalizeEvent: %v", err)
	}
	if len(periods) != 1 || periods[0].SubscriberKickUserID != 2002 {
		t.Fatalf("periods = %+v", periods)
	}
}

func TestNormalizeMissingExpiresAtFallsBackTo30Days(t *testing.T) {
	payload := map[string]any{
		"broadcaster": map[string]any{"user_id": 9000},
		"subscriber":  map[string]any{"user_id": 3003},
		"created_at":  "2026-06-01T00:00:00Z",
	}
	event := makeEvent("msg-noexpiry", webhookprocessor.EventTypeNew, payload)

	periods, err := webhookprocessor.NormalizeEvent(event, testChannel)
	if err != nil {
		t.Fatalf("NormalizeEvent: %v", err)
	}
	p := periods[0]
	wantExpiry := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if !p.ExpiresAt.Equal(wantExpiry) {
		t.Errorf("ExpiresAt = %v, want %v", p.ExpiresAt, wantExpiry)
	}
}

func TestNormalizeGiftCreatesOnePeriodPerGiftee(t *testing.T) {
	payload := map[string]any{
		"broadcaster": map[string]any{"user_id": 9000, "username": "test-channel"},
		"gifter":      map[string]any{"user_id": 5000, "username": "gifter_user", "is_anonymous": false},
		"recipients": []map[string]any{
			{"user_id": 6001, "username": "giftee1"},
			{"user_id": 6002, "username": "giftee2"},
			{"user_id": 6003, "username": "giftee3"},
		},
		"created_at": "2026-06-01T10:00:00Z",
		"expires_at": "2026-07-01T10:00:00Z",
	}
	event := makeEvent("msg-gift-001", webhookprocessor.EventTypeGifts, payload)

	periods, err := webhookprocessor.NormalizeEvent(event, testChannel)
	if err != nil {
		t.Fatalf("NormalizeEvent: %v", err)
	}
	if len(periods) != 3 {
		t.Fatalf("periods len = %d, want 3", len(periods))
	}
	ids := map[string]bool{}
	for _, p := range periods {
		if !p.IsGift {
			t.Errorf("period IsGift = false")
		}
		if p.GifterKickUserID != 5000 {
			t.Errorf("GifterKickUserID = %d", p.GifterKickUserID)
		}
		ids[p.ID] = true
	}
	if len(ids) != 3 {
		t.Errorf("duplicate IDs in gift periods: %v", ids)
	}
	if periods[0].ID != "msg-gift-001_6001" {
		t.Errorf("first period ID = %q", periods[0].ID)
	}
}

func TestNormalizeGiftAnonymousGifterNotStored(t *testing.T) {
	payload := map[string]any{
		"broadcaster": map[string]any{"user_id": 9000},
		"gifter":      map[string]any{"user_id": 5000, "username": "anon", "is_anonymous": true},
		"recipients":  []map[string]any{{"user_id": 7001, "username": "giftee"}},
		"created_at":  "2026-06-01T10:00:00Z",
		"expires_at":  "2026-07-01T10:00:00Z",
	}
	event := makeEvent("msg-anon-gift", webhookprocessor.EventTypeGifts, payload)

	periods, err := webhookprocessor.NormalizeEvent(event, testChannel)
	if err != nil {
		t.Fatalf("NormalizeEvent: %v", err)
	}
	if periods[0].GifterKickUserID != 0 {
		t.Errorf("anonymous gifter ID should be 0, got %d", periods[0].GifterKickUserID)
	}
}

func TestNormalizeGiftUsesGifteesFieldFallback(t *testing.T) {
	payload := map[string]any{
		"broadcaster": map[string]any{"user_id": 9000},
		"giftees":     []map[string]any{{"user_id": 8001, "username": "giftee_alt"}},
		"created_at":  "2026-06-01T10:00:00Z",
	}
	event := makeEvent("msg-giftees-field", webhookprocessor.EventTypeGifts, payload)

	periods, err := webhookprocessor.NormalizeEvent(event, testChannel)
	if err != nil {
		t.Fatalf("NormalizeEvent: %v", err)
	}
	if len(periods) != 1 || periods[0].SubscriberKickUserID != 8001 {
		t.Fatalf("periods = %+v", periods)
	}
}

func TestNormalizeUnsupportedEventTypeReturnsIgnored(t *testing.T) {
	event := makeEvent("msg-unsupported", "channel.follow", map[string]any{})
	_, err := webhookprocessor.NormalizeEvent(event, testChannel)
	if err == nil {
		t.Fatal("expected error for unsupported event type")
	}
	var ignored *webhookprocessor.ErrIgnored
	if !errors.As(err, &ignored) {
		t.Errorf("expected ErrIgnored, got %T: %v", err, err)
	}
}

func TestExtractBroadcasterUserID(t *testing.T) {
	payload := `{"broadcaster":{"user_id":9999},"subscriber":{"user_id":1}}`
	id, err := webhookprocessor.ExtractBroadcasterUserID(payload)
	if err != nil {
		t.Fatalf("ExtractBroadcasterUserID: %v", err)
	}
	if id != 9999 {
		t.Errorf("id = %d, want 9999", id)
	}

	_, err = webhookprocessor.ExtractBroadcasterUserID(`{"broadcaster":{"user_id":0}}`)
	if err == nil {
		t.Fatal("expected error for zero user_id")
	}

	_, err = webhookprocessor.ExtractBroadcasterUserID(`not-json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func makeEvent(messageID, eventType string, payload any) domain.KickWebhookEvent {
	raw, _ := json.Marshal(payload)
	return domain.KickWebhookEvent{
		MessageID:      messageID,
		EventType:      eventType,
		EventVersion:   "v1",
		RawPayloadJSON: string(raw),
		ReceivedAt:     time.Now().UTC(),
	}
}
