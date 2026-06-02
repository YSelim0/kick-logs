package listener

import (
	"encoding/json"
	"strconv"
	"strings"
)

const chatMessageEventName = `App\Events\ChatMessageEvent`

type ChatMessageEvent struct {
	EventName     string
	PusherChannel string
	Payload       map[string]any
	RawJSON       string
}

type EventParser struct{}

func NewEventParser() EventParser {
	return EventParser{}
}

func (EventParser) Parse(raw string) (ChatMessageEvent, bool) {
	envelope, ok := readJSONObject(raw)
	if !ok {
		return ChatMessageEvent{}, false
	}
	eventName, ok := envelope["event"].(string)
	if !ok || eventName != chatMessageEventName {
		return ChatMessageEvent{}, false
	}
	payload, ok := readJSONValueObject(envelope["data"])
	if !ok {
		return ChatMessageEvent{}, false
	}

	channel, _ := envelope["channel"].(string)
	return ChatMessageEvent{
		EventName:     eventName,
		PusherChannel: strings.TrimSpace(channel),
		Payload:       payload,
		RawJSON:       raw,
	}, true
}

func readJSONObject(raw string) (map[string]any, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, false
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, false
	}
	return value, true
}

func readJSONValueObject(value any) (map[string]any, bool) {
	if object, ok := value.(map[string]any); ok {
		return object, true
	}
	text, ok := value.(string)
	if !ok {
		return nil, false
	}
	return readJSONObject(text)
}

func hasRequiredMessageFields(payload map[string]any) bool {
	sender, ok := payload["sender"].(map[string]any)
	if !ok {
		return false
	}
	return cleanText(payload["id"]) != "" &&
		asInt64(payload["chatroom_id"]) > 0 &&
		payloadHasContent(payload) &&
		asInt64(sender["id"]) > 0 &&
		cleanText(sender["username"]) != ""
}

func payloadHasContent(payload map[string]any) bool {
	_, ok := payload["content"]
	return ok
}

func chatroomIDFromPusherChannel(channel string) int64 {
	parts := strings.Split(strings.TrimSpace(channel), ".")
	if len(parts) < 2 || parts[0] != "chatrooms" {
		return 0
	}
	value, _ := strconv.ParseInt(parts[1], 10, 64)
	return value
}
