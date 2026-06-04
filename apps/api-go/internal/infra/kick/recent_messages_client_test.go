package kick

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

func TestRecentMessagesClientBuildsRawChatEnvelopes(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.RequestURI()
		if request.Header.Get("origin") != "https://kick.com" {
			t.Fatalf("origin header = %q", request.Header.Get("origin"))
		}
		response.Header().Set("content-type", "application/json")
		_, _ = response.Write([]byte(`{
			"status": {"error": false, "code": 200, "message": "SUCCESS"},
			"data": {
				"messages": [
					{
						"id": "message-1",
						"chat_id": 250,
						"user_id": 456,
						"content": "hello [emote:37226:KEKW]",
						"type": "message",
						"metadata": "{\"message_ref\":\"reply-1\"}",
						"created_at": "2026-06-02T08:14:13Z",
						"sender": {
							"id": 456,
							"slug": "yavuz_user",
							"username": "Yavuz_User",
							"identity": {
								"color": "#75FD46",
								"badges": [{"type": "moderator"}]
							}
						}
					}
				],
				"cursor": "1780387960537204",
				"pinned_message": null
			}
		}`))
	}))
	defer server.Close()

	client := newRecentMessagesClient(server.URL, server.Client())
	envelopes, err := client.FetchRecentMessages(context.Background(), domain.FollowedChannel{
		ID:             7,
		KickChannelID:  250,
		KickChatroomID: 123,
		Slug:           "gokhanoner",
	})
	if err != nil {
		t.Fatalf("FetchRecentMessages() error = %v", err)
	}
	if gotPath != "/api/v2/channels/250/messages?sort=desc" {
		t.Fatalf("request path = %q", gotPath)
	}
	if len(envelopes) != 1 {
		t.Fatalf("envelopes = %#v", envelopes)
	}

	envelope := envelopes[0]
	if envelope.RawEventID != "kick:message-1" ||
		envelope.KickMessageID != "message-1" ||
		envelope.PusherChannel != "kick-api:channels.250.messages" ||
		envelope.FollowedChannelID != 7 ||
		envelope.KickChatroomID != 123 ||
		envelope.ChannelSlug != "gokhanoner" {
		t.Fatalf("envelope = %#v", envelope)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(envelope.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["chatroom_id"].(float64) != 123 {
		t.Fatalf("chatroom_id = %#v", payload["chatroom_id"])
	}
	metadata, ok := payload["metadata"].(map[string]any)
	if !ok || metadata["message_ref"] != "reply-1" {
		t.Fatalf("metadata = %#v", payload["metadata"])
	}
}

func TestRecentMessagesClientReportsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newRecentMessagesClient(server.URL, server.Client())
	if _, err := client.FetchRecentMessages(context.Background(), domain.FollowedChannel{
		KickChannelID: 250,
	}); err == nil {
		t.Fatal("FetchRecentMessages() error = nil")
	}
}
