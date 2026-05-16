package listener

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

var emotePattern = regexp.MustCompile(`\[emote:([^:\]]+):([^\]]+)\]`)

func rawPayloadJSON(payload map[string]any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func normalizeMessagePayload(
	payload map[string]any,
	channel domain.FollowedChannel,
	sender domain.SenderProfile,
) (domain.ChatMessage, error) {
	kickMessageID := cleanText(payload["id"])
	if kickMessageID == "" {
		return domain.ChatMessage{}, fmt.Errorf("message payload missing id")
	}
	chatroomID := asInt64(payload["chatroom_id"])
	if chatroomID < 1 {
		return domain.ChatMessage{}, fmt.Errorf("message payload has invalid chatroom_id")
	}

	senderPayload, ok := payload["sender"].(map[string]any)
	if !ok {
		return domain.ChatMessage{}, fmt.Errorf("message payload missing sender")
	}
	senderUsername := cleanText(senderPayload["username"])
	if senderUsername == "" {
		return domain.ChatMessage{}, fmt.Errorf("message payload missing sender username")
	}
	senderSlug := normalizeKickProfileSlug(cleanText(senderPayload["slug"]))
	if senderSlug == "" {
		senderSlug = normalizeKickProfileSlug(senderUsername)
	}
	if senderSlug == "" {
		senderSlug = strings.ToLower(senderUsername)
	}

	identity, _ := senderPayload["identity"].(map[string]any)
	metadata, _ := payload["metadata"].(map[string]any)
	content := fmt.Sprint(payload["content"])
	messageType := cleanText(payload["type"])
	if messageType == "" {
		messageType = "message"
	}
	if sender.ID == 0 {
		sender.ID = sender.KickUserID
	}
	if sender.ProfileImageURL == "" {
		sender.ProfileImageURL = firstCleanText(
			senderPayload["profile_image_url"],
			senderPayload["profile_pic"],
			senderPayload["profilepic"],
		)
	}

	message := domain.ChatMessage{
		ID:                     deterministicMessageID(kickMessageID),
		KickMessageID:          kickMessageID,
		ChannelID:              channel.ID,
		ChannelKickID:          channel.KickChannelID,
		ChannelChatroomID:      chatroomID,
		ChannelSlug:            channel.Slug,
		ChannelDisplayName:     channel.DisplayName,
		ChannelProfileImageURL: channel.ProfileImageURL,
		ChannelBannerImageURL:  channel.BannerImageURL,
		ChannelPublicURL:       kickPublicURL(channel.Slug),
		SenderID:               sender.ID,
		SenderKickID:           asInt64(senderPayload["id"]),
		SenderUsername:         senderUsername,
		SenderSlug:             senderSlug,
		SenderDisplayColor:     cleanText(identity["color"]),
		SenderProfileImageURL:  sender.ProfileImageURL,
		SenderPublicURL:        kickPublicURL(senderSlug),
		SenderBadgesJSON:       marshalJSONList(identity["badges"]),
		MessageType:            messageType,
		Content:                content,
		Emotes:                 parseEmotes(content),
		ReplyToSender:          nestedText(metadata, "original_sender", "username"),
		ReplyToContent:         nestedText(metadata, "original_message", "content"),
		ReplyToMessageID:       firstCleanText(metadata["message_ref"], metadata["message_id"]),
		ThreadParentID:         firstCleanText(payload["thread_parent_id"], metadata["thread_parent_id"]),
		ReplyMetadataJSON:      marshalJSONObject(metadata),
		RawPayloadJSON:         rawPayloadJSON(payload),
		MessageCreatedAt:       parseMessageTime(payload["created_at"]),
		IngestedAt:             time.Now().UTC(),
	}
	return message, nil
}

func senderProfileFromPayload(payload map[string]any) (domain.SenderProfile, error) {
	senderPayload, ok := payload["sender"].(map[string]any)
	if !ok {
		return domain.SenderProfile{}, fmt.Errorf("message payload missing sender")
	}
	username := cleanText(senderPayload["username"])
	if username == "" {
		return domain.SenderProfile{}, fmt.Errorf("message payload missing sender username")
	}
	kickUserID := asInt64(senderPayload["id"])
	if kickUserID < 1 {
		return domain.SenderProfile{}, fmt.Errorf("message payload has invalid sender id")
	}
	identity, _ := senderPayload["identity"].(map[string]any)
	slug := normalizeKickProfileSlug(cleanText(senderPayload["slug"]))
	if slug == "" {
		slug = normalizeKickProfileSlug(username)
	}
	return domain.SenderProfile{
		KickUserID:            kickUserID,
		Username:              username,
		Slug:                  slug,
		ProfileImageURL:       firstCleanText(senderPayload["profile_image_url"], senderPayload["profile_pic"], senderPayload["profilepic"]),
		LastSeenColor:         cleanText(identity["color"]),
		RawProfilePayloadJSON: rawPayloadJSON(senderPayload),
		LastSeenAt:            time.Now().UTC(),
	}, nil
}

func parseEmotes(content string) []domain.ChatEmote {
	matches := emotePattern.FindAllStringSubmatch(content, -1)
	emotes := make([]domain.ChatEmote, 0, len(matches))
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		id := strings.TrimSpace(match[1])
		name := strings.TrimSpace(match[2])
		if id == "" || name == "" {
			continue
		}
		emotes = append(emotes, domain.ChatEmote{
			ID:       id,
			Name:     name,
			Token:    match[0],
			ImageURL: "https://files.kick.com/emotes/" + id + "/fullsize",
		})
	}
	return emotes
}

func parseMessageTime(value any) time.Time {
	text := cleanText(value)
	if text == "" {
		return time.Now().UTC()
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.Replace(text, "Z", "+00:00", 1))
	if err != nil {
		return time.Now().UTC()
	}
	return parsed.UTC()
}

func deterministicMessageID(kickMessageID string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(kickMessageID))
	id := int64(hash.Sum64() & uint64(math.MaxInt64))
	if id == 0 {
		return time.Now().UTC().UnixNano()
	}
	return id
}

func normalizeKickProfileSlug(value string) string {
	cleaned := strings.ToLower(strings.TrimSpace(value))
	return strings.ReplaceAll(cleaned, "_", "-")
}

func kickPublicURL(slug string) string {
	if strings.TrimSpace(slug) == "" {
		return ""
	}
	return "https://kick.com/" + strings.TrimSpace(slug)
}

func marshalJSONObject(value map[string]any) string {
	if len(value) == 0 {
		return "{}"
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func marshalJSONList(value any) string {
	items, ok := value.([]any)
	if !ok {
		return "[]"
	}
	filtered := make([]any, 0, len(items))
	for _, item := range items {
		if _, ok := item.(map[string]any); ok {
			filtered = append(filtered, item)
		}
	}
	encoded, err := json.Marshal(filtered)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func nestedText(value map[string]any, objectKey string, valueKey string) string {
	object, ok := value[objectKey].(map[string]any)
	if !ok {
		return ""
	}
	return cleanText(object[valueKey])
}

func firstCleanText(values ...any) string {
	for _, value := range values {
		if text := cleanText(value); text != "" {
			return text
		}
	}
	return ""
}

func cleanText(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func asInt64(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed
	case string:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed
	default:
		return 0
	}
}
