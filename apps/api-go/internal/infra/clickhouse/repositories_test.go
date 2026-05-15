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

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	messageRepo := clickhouseinfra.NewMessageRepository(conn)
	message := domain.ChatMessage{
		ID:                 time.Now().UnixNano(),
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
		ReplyToSender:    "other_user",
		ReplyToContent:   "older message",
		ReplyToMessageID: "parent-" + suffix,
		ThreadParentID:   "parent-" + suffix,
		RawPayloadJSON:   `{"type":"ChatMessageEvent"}`,
		MessageCreatedAt: time.Now().UTC(),
	}
	if err := messageRepo.Insert(ctx, message); err != nil {
		t.Fatalf("messageRepo.Insert() error = %v", err)
	}

	messages, err := messageRepo.SearchBasic(ctx, domain.MessageSearchFilter{
		Sender: message.SenderUsername,
		Query:  "phase3 needle " + suffix,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("messageRepo.SearchBasic() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages len = %d", len(messages))
	}
	if !strings.Contains(messages[0].Content, suffix) || len(messages[0].Emotes) != 1 {
		t.Fatalf("message mismatch = %#v", messages[0])
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
