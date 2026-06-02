package kick

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

func TestSubscriptionMessagesIncludeChatroomAndChannelStreams(t *testing.T) {
	messages := subscriptionMessages(domain.ListenerChannel{
		KickChannelID:  100,
		KickChatroomID: 200,
	})

	encoded := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		var value map[string]any
		if err := json.Unmarshal([]byte(marshalSubscription(message)), &value); err != nil {
			t.Fatalf("decode subscription: %v", err)
		}
		encoded = append(encoded, value)
	}

	if encoded[0]["event"] != "pusher:subscribe" ||
		encoded[0]["data"].(map[string]any)["channel"] != "chatrooms.200.v2" ||
		encoded[1]["data"].(map[string]any)["channel"] != "chatrooms.200" ||
		encoded[2]["data"].(map[string]any)["channel"] != "channel.100" {
		t.Fatalf("encoded subscriptions = %#v", encoded)
	}
}

func TestPusherClientRespondsToProtocolPing(t *testing.T) {
	upgrader := websocket.Upgrader{}
	gotPong := make(chan struct{}, 1)
	gotChat := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(response, request, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		for range 3 {
			if _, _, err := conn.ReadMessage(); err != nil {
				t.Errorf("read subscription: %v", err)
				return
			}
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"event":"pusher:ping","data":{}}`)); err != nil {
			t.Errorf("write ping: %v", err)
			return
		}
		_, payload, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read pong: %v", err)
			return
		}
		if !strings.Contains(string(payload), `"event":"pusher:pong"`) {
			t.Errorf("pong payload = %s", string(payload))
			return
		}
		gotPong <- struct{}{}

		if err := conn.WriteMessage(
			websocket.TextMessage,
			[]byte(`{"event":"App\\Events\\ChatMessageEvent","channel":"chatrooms.200.v2","data":"{\"id\":\"message-1\"}"}`),
		); err != nil {
			t.Errorf("write chat: %v", err)
		}
		<-gotChat
	}))
	defer server.Close()

	client := NewPusherClient("ws" + strings.TrimPrefix(server.URL, "http"))
	ctx, cancel := context.WithCancel(context.Background())
	err := client.Listen(ctx, []domain.ListenerChannel{{
		KickChannelID:  100,
		KickChatroomID: 200,
	}}, func(raw string) error {
		if !strings.Contains(raw, `ChatMessageEvent`) {
			t.Fatalf("raw chat = %s", raw)
		}
		gotChat <- struct{}{}
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Listen() error = %v", err)
	}

	select {
	case <-gotPong:
	default:
		t.Fatal("pusher pong was not sent")
	}
}
