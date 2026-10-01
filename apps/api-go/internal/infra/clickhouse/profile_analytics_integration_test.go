package clickhouse_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
)

type profileSummaryReader interface {
	OverviewAndTopSenders(context.Context, domain.AnalyticsFilter, uint64) (domain.AnalyticsOverview, []domain.TopSenderAnalytics, error)
	OverviewAndTopChannels(context.Context, domain.AnalyticsFilter, uint64) (domain.AnalyticsOverview, []domain.TopChannelAnalytics, error)
}

func testProfileCombinedAnalytics(t *testing.T, conn driver.Conn) {
	ctx := context.Background()
	counted := &profileCountingConnection{Conn: conn}
	optimized, ok := any(clickhouseinfra.NewAnalyticsRepository(counted)).(profileSummaryReader)
	if !ok {
		t.Fatal("ClickHouse repository must provide single-query profile summaries")
	}
	id := time.Now().UnixNano()
	prefix := fmt.Sprintf("profile-summary-%d", id)
	created := time.Date(2020, 1, 2, 3, 0, 0, 0, time.UTC)
	ingested := time.Now().UTC().Truncate(time.Millisecond)
	messages := make([]domain.ChatMessage, 0, 34)
	for mode := range 2 {
		for i := range 8 {
			for j := range 2 {
				message := domain.ChatMessage{
					ID: id + int64(len(messages)), KickMessageID: fmt.Sprintf("%s-%d", prefix, len(messages)),
					ChannelID: id + int64(i), ChannelSlug: fmt.Sprintf("%s-channel-%d", prefix, i),
					ChannelDisplayName: fmt.Sprintf("Channel %d", i), ChannelProfileImageURL: "/channel.png",
					ChannelBannerImageURL: "/banner.png", SenderID: id + int64(i) + 100,
					SenderKickID: id + int64(i) + 200, SenderSlug: fmt.Sprintf("%s-sender-%d", prefix, i),
					SenderUsername: fmt.Sprintf("Sender %d", i), SenderProfileImageURL: "/sender.png",
					MessageCreatedAt: created.Add(time.Duration(j) * time.Hour), IngestedAt: ingested,
					Content: "historical profile fixture",
					Emotes:  []domain.ChatEmote{{ID: prefix, Name: "wave"}, {ID: prefix, Name: "wave"}},
				}
				if mode == 0 {
					message.ChannelID, message.ChannelSlug = id, prefix
					if i >= 5 {
						message.SenderKickID = 0
						message.SenderProfileImageURL = ""
					}
					if i >= 6 {
						message.SenderSlug = ""
					}
					if i == 7 {
						message.SenderUsername = ""
					}
				} else {
					message.SenderKickID, message.SenderSlug = id, strings.ReplaceAll(prefix, "-", "_")+"_user"
					message.SenderUsername = strings.ToUpper(message.SenderSlug)
					if i == 5 || i == 6 {
						message.ChannelID = 0
						message.ChannelProfileImageURL, message.ChannelBannerImageURL = "", ""
					}
				}
				messages = append(messages, message)
			}
		}
	}
	for _, index := range []int{1, 17} {
		duplicate := messages[index]
		duplicate.IngestedAt = ingested.Add(time.Second)
		duplicate.ChannelDisplayName = "Updated channel"
		if index == 1 {
			duplicate.SenderSlug = prefix + "-renamed"
			duplicate.SenderUsername = "Renamed user"
		}
		messages = append(messages, duplicate)
	}
	if err := clickhouseinfra.NewMessageRepository(conn).InsertMessagesBatch(ctx, messages); err != nil {
		t.Fatal(err)
	}
	// Add newer tombstones without mutating shared fixtures or deleting history.
	for _, index := range []int{3, 19} {
		if err := conn.Exec(ctx, `INSERT INTO chat_messages SELECT * REPLACE (
			ingested_at + INTERVAL 2 SECOND AS ingested_at, toUInt8(1) AS is_deleted
		) FROM chat_messages FINAL WHERE id = ?`, messages[index].ID); err != nil {
			t.Fatal(err)
		}
	}
	legacy := clickhouseinfra.NewAnalyticsRepository(conn)
	for _, mode := range []string{"channel", "user", "empty", "empty-user"} {
		for _, limit := range []uint64{1, 5, 0} {
			t.Run(fmt.Sprintf("%s/limit-%d", mode, limit), func(t *testing.T) {
				filter := domain.AnalyticsFilter{Channel: prefix}
				if mode == "user" {
					filter = domain.AnalyticsFilter{Sender: strings.ToUpper(prefix + "-user")}
				} else if mode == "empty" {
					filter.Channel += "-missing"
				} else if mode == "empty-user" {
					filter = domain.AnalyticsFilter{Sender: prefix + "-missing-user"}
				}
				want, err := legacy.Overview(ctx, filter)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(mode, "empty") && (want.TotalMessages != 15 || want.TotalEmoteUsages != 30 || !want.FirstMessageAt.Equal(created)) {
					t.Fatalf("fixture overview=%+v; want 15 historical messages and 30 emote usages", want)
				}
				counted.queries = 0
				var got domain.AnalyticsOverview
				if mode == "user" || mode == "empty-user" {
					var top []domain.TopChannelAnalytics
					got, top, err = optimized.OverviewAndTopChannels(ctx, filter, limit)
					oldTop, oldErr := legacy.TopChannels(ctx, filter, limit)
					if oldErr != nil || !reflect.DeepEqual(top, oldTop) {
						t.Fatalf("channel ranking changed: got=%+v want=%+v err=%v", top, oldTop, oldErr)
					}
				} else {
					var top []domain.TopSenderAnalytics
					got, top, err = optimized.OverviewAndTopSenders(ctx, filter, limit)
					oldTop, oldErr := legacy.TopSenders(ctx, filter, limit)
					if oldErr != nil || !reflect.DeepEqual(top, oldTop) {
						t.Fatalf("sender ranking changed: got=%+v want=%+v err=%v", top, oldTop, oldErr)
					}
				}
				if err != nil || !reflect.DeepEqual(got, want) || counted.queries != 1 {
					t.Fatalf("got=%+v want=%+v queries=%d err=%v", got, want, counted.queries, err)
				}
			})
		}
	}
}

type profileCountingConnection struct {
	driver.Conn
	queries int
}

func (conn *profileCountingConnection) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	conn.queries++
	return conn.Conn.Query(ctx, query, args...)
}

func (conn *profileCountingConnection) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	conn.queries++
	return conn.Conn.QueryRow(ctx, query, args...)
}
