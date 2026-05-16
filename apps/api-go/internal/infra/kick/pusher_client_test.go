package kick

import (
	"encoding/json"
	"testing"

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
		encoded[1]["data"].(map[string]any)["channel"] != "channel.100" {
		t.Fatalf("encoded subscriptions = %#v", encoded)
	}
}
