package kick

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEventSubscriptionClientCreateUsesKickEventsContract(t *testing.T) {
	var received struct {
		BroadcasterUserID int64 `json:"broadcaster_user_id"`
		Events            []struct {
			Name    string `json:"name"`
			Version int    `json:"version"`
		} `json:"events"`
		Method string `json:"method"`
	}

	server := newKickAPITestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/public/v1/events/subscriptions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		writeJSON(t, w, map[string]any{
			"data": []map[string]any{
				{"subscription_id": "sub-new", "name": "channel.subscription.new", "method": "webhook"},
				{"subscription_id": "sub-gifts", "name": "channel.subscription.gifts", "method": "webhook"},
			},
		})
	})
	defer server.Close()

	client := NewEventSubscriptionClient(server.URL, server.URL+"/oauth/token", "client", "secret")
	subs, err := client.CreateEventSubscriptions(context.Background(), 12345, []string{"channel.subscription.new", "channel.subscription.gifts"})
	if err != nil {
		t.Fatalf("CreateEventSubscriptions: %v", err)
	}

	if received.BroadcasterUserID != 12345 || received.Method != "webhook" {
		t.Fatalf("request body = %+v", received)
	}
	if len(received.Events) != 2 {
		t.Fatalf("events len = %d, want 2", len(received.Events))
	}
	if received.Events[0].Name != "channel.subscription.new" || received.Events[0].Version != 1 {
		t.Fatalf("first event = %+v", received.Events[0])
	}
	if len(subs) != 2 {
		t.Fatalf("subs len = %d, want 2", len(subs))
	}
	if subs[0].BroadcasterUserID != 12345 || subs[0].SubscriptionID != "sub-new" {
		t.Fatalf("first sub = %+v", subs[0])
	}
}

func TestEventSubscriptionClientFetchesWebhookPublicKey(t *testing.T) {
	const publicKey = "-----BEGIN PUBLIC KEY-----\ntest\n-----END PUBLIC KEY-----"
	server := newKickAPITestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/public/v1/public-key" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		writeJSON(t, w, map[string]any{
			"data": map[string]any{"public_key": publicKey},
		})
	})
	defer server.Close()

	client := NewEventSubscriptionClient(server.URL, server.URL+"/oauth/token", "client", "secret")
	got, err := client.FetchWebhookPublicKey(context.Background())
	if err != nil {
		t.Fatalf("FetchWebhookPublicKey: %v", err)
	}
	if got != publicKey {
		t.Fatalf("public key = %q", got)
	}
}

func TestEventSubscriptionClientDeleteUsesQueryID(t *testing.T) {
	server := newKickAPITestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/public/v1/events/subscriptions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if got := r.URL.Query().Get("id"); got != "sub-123" {
			t.Fatalf("id query = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := NewEventSubscriptionClient(server.URL, server.URL+"/oauth/token", "client", "secret")
	if err := client.DeleteEventSubscription(context.Background(), "sub-123"); err != nil {
		t.Fatalf("DeleteEventSubscription: %v", err)
	}
}

func newKickAPITestServer(t *testing.T, apiHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			if r.Method != http.MethodPost {
				t.Fatalf("token method = %s", r.Method)
			}
			writeJSON(t, w, map[string]any{"access_token": "test-token", "expires_in": 3600})
			return
		}
		apiHandler(w, r)
	}))
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}
