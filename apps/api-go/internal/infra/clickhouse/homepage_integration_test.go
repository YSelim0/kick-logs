package clickhouse_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
)

// Called by the existing opt-in CI integration suite against its test database.
func testHomepageAnalytics(t *testing.T, conn driver.Conn) {
	t.Helper()
	ctx := context.Background()
	asOf := time.Now().UTC().Truncate(time.Second)
	start := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -13)
	id := time.Now().UnixNano()
	slug := fmt.Sprintf("homepage-%d", id)
	messages := make([]domain.ChatMessage, 0, 5)
	for i, created := range []time.Time{start.Add(-time.Millisecond), start, asOf, asOf.Add(time.Millisecond)} {
		messages = append(messages, domain.ChatMessage{
			ID: id + int64(i), KickMessageID: fmt.Sprintf("%s-%d", slug, i),
			ChannelID: id, ChannelKickID: id, ChannelSlug: slug, ChannelDisplayName: slug,
			SenderID: id, SenderKickID: id, SenderSlug: slug, SenderUsername: slug,
			MessageType: "message", Content: "bounded homepage fixture",
			MessageCreatedAt: created, IngestedAt: asOf,
			Emotes: []domain.ChatEmote{{ID: slug, Name: slug, Token: "[emote]", ImageURL: "/emote.png"}},
		})
	}
	duplicate := messages[1]
	duplicate.IngestedAt = asOf.Add(time.Second)
	messages = append(messages, duplicate)
	if err := clickhouseinfra.NewMessageRepository(conn).InsertMessagesBatch(ctx, messages); err != nil {
		t.Fatal(err)
	}
	repo := clickhouseinfra.NewHomepageAnalyticsRepository(conn)
	filter := domain.AnalyticsFilter{Start: start, End: asOf, Channel: slug}
	overview, err := repo.Overview(ctx, filter)
	if err != nil || overview.TotalMessages != 2 || overview.TotalSenders != 1 || overview.TotalChannels != 1 || overview.TotalEmoteUsages != 2 {
		t.Fatalf("overview=%+v err=%v", overview, err)
	}
	volume, err := repo.MessageVolume(ctx, filter, domain.AnalyticsBucketDay)
	if err != nil || len(volume) != 2 || volume[0].MessageCount != 1 || volume[1].MessageCount != 1 {
		t.Fatalf("volume=%+v err=%v", volume, err)
	}
	channels, err := repo.TopChannels(ctx, filter, 5)
	if err != nil || len(channels) != 1 || channels[0].MessageCount != 2 {
		t.Fatalf("channels=%+v err=%v", channels, err)
	}
	senders, err := repo.TopSenders(ctx, filter, 5)
	if err != nil || len(senders) != 1 || senders[0].MessageCount != 2 {
		t.Fatalf("senders=%+v err=%v", senders, err)
	}
	emotes, err := repo.TopEmotes(ctx, filter, 5)
	if err != nil || len(emotes) != 1 || emotes[0].UsageCount != 2 {
		t.Fatalf("emotes=%+v err=%v", emotes, err)
	}
}
