package kick

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

const recentChatMessageEventName = `App\Events\ChatMessageEvent`

type RecentMessagesClient struct {
	client  *http.Client
	baseURL string
}

func NewRecentMessagesClient() *RecentMessagesClient {
	return newRecentMessagesClient("https://kick.com", &http.Client{Timeout: 15 * time.Second})
}

func newRecentMessagesClient(baseURL string, client *http.Client) *RecentMessagesClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &RecentMessagesClient{
		client:  client,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (client *RecentMessagesClient) FetchRecentMessages(
	ctx context.Context,
	channel domain.FollowedChannel,
) ([]domain.RawChatEventEnvelope, error) {
	if channel.KickChannelID <= 0 {
		return nil, fmt.Errorf("kick channel id is required")
	}

	url := fmt.Sprintf("%s/api/v2/channels/%d/messages?sort=desc", client.baseURL, channel.KickChannelID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build Kick recent messages request: %w", err)
	}
	request.Header.Set("accept", "application/json")
	request.Header.Set("origin", "https://kick.com")
	request.Header.Set("referer", "https://kick.com/"+strings.TrimSpace(channel.Slug))
	request.Header.Set(
		"user-agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
	)

	response, err := client.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch Kick recent messages: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("Kick recent messages returned status %d", response.StatusCode)
	}

	rawBody, err := io.ReadAll(io.LimitReader(response.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read Kick recent messages response: %w", err)
	}

	var payload recentMessagesResponse
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, fmt.Errorf("decode Kick recent messages response: %w", err)
	}

	envelopes := make([]domain.RawChatEventEnvelope, 0, len(payload.Data.Messages))
	receivedAt := time.Now().UTC()
	for _, rawMessage := range payload.Data.Messages {
		envelope, ok := recentMessageEnvelope(rawMessage, channel, receivedAt)
		if !ok {
			continue
		}
		envelopes = append(envelopes, envelope)
	}
	return envelopes, nil
}

type recentMessagesResponse struct {
	Data struct {
		Messages []json.RawMessage `json:"messages"`
	} `json:"data"`
}

func recentMessageEnvelope(
	rawMessage json.RawMessage,
	channel domain.FollowedChannel,
	receivedAt time.Time,
) (domain.RawChatEventEnvelope, bool) {
	var message map[string]any
	if err := json.Unmarshal(rawMessage, &message); err != nil {
		return domain.RawChatEventEnvelope{}, false
	}

	messageID := cleanRecentText(message["id"])
	if messageID == "" {
		return domain.RawChatEventEnvelope{}, false
	}
	message["chatroom_id"] = channel.KickChatroomID
	message["metadata"] = normalizeRecentMetadata(message["metadata"])
	if cleanRecentText(message["type"]) == "" {
		message["type"] = "message"
	}
	if message["content"] == nil {
		message["content"] = ""
	}

	payloadJSON := marshalRecentJSON(message)
	return domain.RawChatEventEnvelope{
		RawEventID:        "kick:" + messageID,
		KickMessageID:     messageID,
		EventName:         recentChatMessageEventName,
		PusherChannel:     fmt.Sprintf("kick-api:channels.%d.messages", channel.KickChannelID),
		FollowedChannelID: channel.ID,
		ChannelSlug:       channel.Slug,
		KickChannelID:     channel.KickChannelID,
		KickChatroomID:    channel.KickChatroomID,
		ReceivedAt:        receivedAt.UTC(),
		PayloadJSON:       payloadJSON,
		RawPusherJSON:     string(rawMessage),
	}, true
}

func normalizeRecentMetadata(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if object, ok := value.(map[string]any); ok {
		return object
	}
	text := cleanRecentText(value)
	if text == "" || text == "null" {
		return map[string]any{}
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(text), &object); err != nil {
		return map[string]any{}
	}
	return object
}

func marshalRecentJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func cleanRecentText(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
