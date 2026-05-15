package clickhouse_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/infra/migrations"
)

func TestClickHouseMigrationsAndRepositories(t *testing.T) {
	if os.Getenv("KICK_LOGS_RUN_CLICKHOUSE_TESTS") != "1" {
		t.Skip("set KICK_LOGS_RUN_CLICKHOUSE_TESTS=1 with a local ClickHouse service to run this test")
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	conn, err := clickhouseinfra.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if err := migrations.ApplyClickHouse(ctx, conn); err != nil {
		t.Fatalf("ApplyClickHouse() error = %v", err)
	}
	if err := migrations.ApplyClickHouse(ctx, conn); err != nil {
		t.Fatalf("ApplyClickHouse() second run error = %v", err)
	}

	baseID := time.Now().UnixNano()
	baseTime := time.Now().UTC().Truncate(time.Millisecond)
	suffix := fmt.Sprintf("%d", baseID)
	messageRepo := clickhouseinfra.NewMessageRepository(conn)
	message := domain.ChatMessage{
		ID:                 baseID + 3,
		KickMessageID:      "kick-" + suffix,
		ChannelKickID:      123,
		ChannelChatroomID:  456,
		ChannelSlug:        "hype",
		ChannelDisplayName: "Hype",
		ChannelPublicURL:   "https://kick.com/hype",
		SenderKickID:       789,
		SenderUsername:     "phase3_user_" + suffix,
		SenderSlug:         "phase3-user-" + suffix,
		SenderPublicURL:    "https://kick.com/phase3-user-" + suffix,
		MessageType:        "reply",
		Content:            "hello clickhouse phase3 needle " + suffix,
		Emotes: []domain.ChatEmote{
			{ID: "1", Name: "wave", Token: "[emote:1:wave]", ImageURL: "https://files.kick.com/emotes/1/fullsize"},
		},
		ReplyToSender:     "other_user",
		ReplyToContent:    "older message",
		ReplyToMessageID:  "parent-" + suffix,
		ThreadParentID:    "parent-" + suffix,
		ReplyMetadataJSON: `{"original_sender":{"username":"other_user"},"original_message":{"content":"older message"}}`,
		RawPayloadJSON:    `{"type":"ChatMessageEvent"}`,
		MessageCreatedAt:  baseTime.Add(3 * time.Minute),
	}
	if err := messageRepo.Insert(ctx, message); err != nil {
		t.Fatalf("messageRepo.Insert() error = %v", err)
	}
	partialSenderMessage := message
	partialSenderMessage.ID = baseID + 2
	partialSenderMessage.KickMessageID = "kick-partial-" + suffix
	partialSenderMessage.SenderUsername = message.SenderUsername + "_extra"
	partialSenderMessage.SenderSlug = message.SenderSlug + "-extra"
	partialSenderMessage.MessageType = "message"
	partialSenderMessage.Emotes = nil
	partialSenderMessage.MessageCreatedAt = baseTime.Add(2 * time.Minute)
	if err := messageRepo.Insert(ctx, partialSenderMessage); err != nil {
		t.Fatalf("messageRepo.Insert(partial) error = %v", err)
	}
	otherChannelMessage := message
	otherChannelMessage.ID = baseID + 1
	otherChannelMessage.KickMessageID = "kick-other-" + suffix
	otherChannelMessage.ChannelSlug = "other-" + suffix
	otherChannelMessage.ChannelDisplayName = "Other " + suffix
	otherChannelMessage.MessageType = "message"
	otherChannelMessage.Emotes = nil
	otherChannelMessage.MessageCreatedAt = baseTime.Add(time.Minute)
	if err := messageRepo.Insert(ctx, otherChannelMessage); err != nil {
		t.Fatalf("messageRepo.Insert(other channel) error = %v", err)
	}

	messages, err := messageRepo.Search(ctx, domain.MessageSearchFilter{
		Sender: message.SenderUsername,
		Query:  "phase3 needle " + suffix,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("messageRepo.Search() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("messages len = %d", len(messages))
	}
	for _, foundMessage := range messages {
		if foundMessage.SenderUsername != message.SenderUsername {
			t.Fatalf("partial sender matched = %#v", foundMessage)
		}
	}
	if !strings.Contains(messages[0].Content, suffix) || len(messages[0].Emotes) != 1 {
		t.Fatalf("message mismatch = %#v", messages[0])
	}
	if messages[0].ReplyMetadataJSON == "" || messages[0].SenderID == 0 || messages[0].ChannelID == 0 {
		t.Fatalf("message response columns not populated = %#v", messages[0])
	}

	channelMessages, err := messageRepo.Search(ctx, domain.MessageSearchFilter{
		Channel: "hyp",
		Query:   "phase3 needle " + suffix,
		Start:   baseTime.Add(time.Minute + time.Second),
		End:     baseTime.Add(4 * time.Minute),
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("messageRepo.Search(channel/date) error = %v", err)
	}
	if len(channelMessages) != 2 || channelMessages[0].ID != message.ID || channelMessages[1].ID != partialSenderMessage.ID {
		t.Fatalf("channel/date messages = %#v", channelMessages)
	}

	replyMessages, err := messageRepo.Search(ctx, domain.MessageSearchFilter{
		Query:     "phase3 needle " + suffix,
		ReplyOnly: true,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("messageRepo.Search(reply) error = %v", err)
	}
	if len(replyMessages) != 1 || replyMessages[0].ID != message.ID {
		t.Fatalf("reply messages = %#v", replyMessages)
	}

	emoteMessages, err := messageRepo.Search(ctx, domain.MessageSearchFilter{
		Query:     "phase3 needle " + suffix,
		EmoteOnly: true,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("messageRepo.Search(emote) error = %v", err)
	}
	if len(emoteMessages) != 1 || emoteMessages[0].ID != message.ID {
		t.Fatalf("emote messages = %#v", emoteMessages)
	}

	firstPage, err := messageRepo.Search(ctx, domain.MessageSearchFilter{
		Query: "phase3 needle " + suffix,
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("messageRepo.Search(first page) error = %v", err)
	}
	if len(firstPage) != 1 {
		t.Fatalf("first page = %#v", firstPage)
	}
	secondPage, err := messageRepo.Search(ctx, domain.MessageSearchFilter{
		Query: "phase3 needle " + suffix,
		Cursor: &domain.MessageCursor{
			MessageCreatedAt: firstPage[0].MessageCreatedAt,
			MessageID:        firstPage[0].ID,
		},
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("messageRepo.Search(second page) error = %v", err)
	}
	if len(secondPage) != 1 || firstPage[0].ID == secondPage[0].ID {
		t.Fatalf("cursor pages first=%#v second=%#v", firstPage, secondPage)
	}

	rawRepo := clickhouseinfra.NewRawEventRepository(conn)
	rawEvent := domain.RawKickEvent{
		ID:          "raw-" + suffix,
		ChannelSlug: "hype",
		EventType:   "pusher",
		EventName:   "App\\Events\\ChatMessageEvent",
		PayloadJSON: `{"event":"App\\Events\\ChatMessageEvent"}`,
		Status:      "processed",
		ReceivedAt:  time.Now().UTC(),
	}
	if err := rawRepo.InsertEvent(ctx, rawEvent); err != nil {
		t.Fatalf("rawRepo.InsertEvent() error = %v", err)
	}
	if err := rawRepo.InsertAttempt(ctx, domain.RawEventAttempt{
		ID:         "attempt-" + suffix,
		RawEventID: rawEvent.ID,
		Attempt:    1,
		Status:     "processed",
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("rawRepo.InsertAttempt() error = %v", err)
	}

	var rawCount uint64
	if err := conn.QueryRow(ctx, "SELECT count() FROM raw_kick_events WHERE id = ?", rawEvent.ID).Scan(&rawCount); err != nil {
		t.Fatalf("count raw events error = %v", err)
	}
	if rawCount != 1 {
		t.Fatalf("rawCount = %d", rawCount)
	}

	stats := clickhouseinfra.NewStatsRepository(conn)
	sizes, err := stats.TableSizes(ctx)
	if err != nil {
		t.Fatalf("stats.TableSizes() error = %v", err)
	}
	if len(sizes) == 0 {
		t.Fatal("stats.TableSizes() returned no rows")
	}
}
