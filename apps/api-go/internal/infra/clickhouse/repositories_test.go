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
		ChannelSlug:        "hype-" + suffix,
		ChannelDisplayName: "Hype " + suffix,
		ChannelPublicURL:   "https://kick.com/hype-" + suffix,
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
	otherChannelMessage.ChannelID = 124
	otherChannelMessage.ChannelKickID = 124
	otherChannelMessage.ChannelChatroomID = 457
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

	analyticsRepo := clickhouseinfra.NewAnalyticsRepository(conn)
	analyticsFilter := domain.AnalyticsFilter{Sender: message.SenderUsername}
	overview, err := analyticsRepo.Overview(ctx, analyticsFilter)
	if err != nil {
		t.Fatalf("analyticsRepo.Overview() error = %v", err)
	}
	if overview.TotalMessages != 2 || overview.TotalSenders != 1 || overview.TotalChannels != 2 || overview.TotalEmoteUsages != 1 {
		t.Fatalf("overview = %#v", overview)
	}
	volume, err := analyticsRepo.MessageVolume(ctx, analyticsFilter, domain.AnalyticsBucketDay)
	if err != nil {
		t.Fatalf("analyticsRepo.MessageVolume() error = %v", err)
	}
	if len(volume) != 1 || volume[0].MessageCount != 2 {
		t.Fatalf("volume = %#v", volume)
	}
	topSenders, err := analyticsRepo.TopSenders(ctx, analyticsFilter, 5)
	if err != nil {
		t.Fatalf("analyticsRepo.TopSenders() error = %v", err)
	}
	if len(topSenders) != 1 || topSenders[0].MessageCount != 2 || topSenders[0].Username != message.SenderUsername {
		t.Fatalf("top senders = %#v", topSenders)
	}
	topChannels, err := analyticsRepo.TopChannels(ctx, analyticsFilter, 5)
	if err != nil {
		t.Fatalf("analyticsRepo.TopChannels() error = %v", err)
	}
	if len(topChannels) != 2 || topChannels[0].Slug != message.ChannelSlug {
		t.Fatalf("top channels = %#v", topChannels)
	}
	topEmotes, err := analyticsRepo.TopEmotes(ctx, domain.AnalyticsFilter{Channel: message.ChannelSlug}, 5)
	if err != nil {
		t.Fatalf("analyticsRepo.TopEmotes() error = %v", err)
	}
	if len(topEmotes) != 1 || topEmotes[0].ID != "1" || topEmotes[0].UsageCount != 1 {
		t.Fatalf("top emotes = %#v", topEmotes)
	}
	latestMessages, err := analyticsRepo.LatestMessages(ctx, analyticsFilter, 10)
	if err != nil {
		t.Fatalf("analyticsRepo.LatestMessages() error = %v", err)
	}
	if len(latestMessages) != 2 || latestMessages[0].ID != message.ID || latestMessages[1].ID != otherChannelMessage.ID {
		t.Fatalf("latest analytics messages = %#v", latestMessages)
	}

	duplicateMessage := message
	duplicateMessage.Content = "updated duplicate clickhouse phase3 needle " + suffix
	duplicateMessage.IngestedAt = baseTime.Add(10 * time.Minute)
	if err := messageRepo.Insert(ctx, duplicateMessage); err != nil {
		t.Fatalf("messageRepo.Insert(duplicate) error = %v", err)
	}
	dedupedMessages, err := messageRepo.Search(ctx, domain.MessageSearchFilter{
		Query: "phase3 needle " + suffix,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("messageRepo.Search(deduped) error = %v", err)
	}
	if len(dedupedMessages) != 3 {
		t.Fatalf("deduped messages len = %d messages=%#v", len(dedupedMessages), dedupedMessages)
	}
	seenKickMessageIDs := make(map[string]bool)
	for _, foundMessage := range dedupedMessages {
		if seenKickMessageIDs[foundMessage.KickMessageID] {
			t.Fatalf("duplicate kick message returned = %#v", dedupedMessages)
		}
		seenKickMessageIDs[foundMessage.KickMessageID] = true
	}
	if dedupedMessages[0].KickMessageID != message.KickMessageID ||
		dedupedMessages[0].Content != duplicateMessage.Content {
		t.Fatalf("latest duplicate was not selected = %#v", dedupedMessages[0])
	}
	dedupedOverview, err := analyticsRepo.Overview(ctx, analyticsFilter)
	if err != nil {
		t.Fatalf("analyticsRepo.Overview(deduped) error = %v", err)
	}
	if dedupedOverview.TotalMessages != 2 || dedupedOverview.TotalSenders != 1 || dedupedOverview.TotalChannels != 2 {
		t.Fatalf("deduped overview = %#v", dedupedOverview)
	}
	dedupedVolume, err := analyticsRepo.MessageVolume(ctx, analyticsFilter, domain.AnalyticsBucketDay)
	if err != nil {
		t.Fatalf("analyticsRepo.MessageVolume(deduped) error = %v", err)
	}
	if len(dedupedVolume) != 1 || dedupedVolume[0].MessageCount != 2 {
		t.Fatalf("deduped volume = %#v", dedupedVolume)
	}
	dedupedTopSenders, err := analyticsRepo.TopSenders(ctx, analyticsFilter, 5)
	if err != nil {
		t.Fatalf("analyticsRepo.TopSenders(deduped) error = %v", err)
	}
	if len(dedupedTopSenders) != 1 || dedupedTopSenders[0].MessageCount != 2 {
		t.Fatalf("deduped top senders = %#v", dedupedTopSenders)
	}
	dedupedLatestMessages, err := analyticsRepo.LatestMessages(ctx, analyticsFilter, 10)
	if err != nil {
		t.Fatalf("analyticsRepo.LatestMessages(deduped) error = %v", err)
	}
	if len(dedupedLatestMessages) != 2 ||
		dedupedLatestMessages[0].KickMessageID != duplicateMessage.KickMessageID ||
		dedupedLatestMessages[0].Content != duplicateMessage.Content {
		t.Fatalf("deduped latest analytics messages = %#v", dedupedLatestMessages)
	}

	rawRepo := clickhouseinfra.NewRawEventRepository(conn)
	rawEvent := domain.RawKickEvent{
		ID:            "raw-" + suffix,
		ChannelSlug:   "hype",
		EventType:     "pusher",
		EventName:     "App\\Events\\ChatMessageEvent",
		KickMessageID: "raw-message-" + suffix,
		ChatroomID:    456,
		ChannelID:     123,
		PayloadJSON:   `{"event":"App\\Events\\ChatMessageEvent"}`,
		Status:        "pending",
		ReceivedAt:    time.Now().UTC(),
	}
	if err := rawRepo.InsertEvent(ctx, rawEvent); err != nil {
		t.Fatalf("rawRepo.InsertEvent() error = %v", err)
	}
	unprocessed, err := rawRepo.ListUnprocessed(ctx, 10, 5)
	if err != nil {
		t.Fatalf("rawRepo.ListUnprocessed() error = %v", err)
	}
	foundRaw := false
	for _, event := range unprocessed {
		if event.ID == rawEvent.ID {
			foundRaw = true
			if event.KickMessageID != rawEvent.KickMessageID || event.ChatroomID != 456 || event.ChannelID != 123 {
				t.Fatalf("raw event metadata = %#v", event)
			}
		}
	}
	if !foundRaw {
		t.Fatalf("raw event %s not found in unprocessed = %#v", rawEvent.ID, unprocessed)
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
	attemptCount, err := rawRepo.AttemptCount(ctx, rawEvent.ID)
	if err != nil {
		t.Fatalf("rawRepo.AttemptCount() error = %v", err)
	}
	if attemptCount != 1 {
		t.Fatalf("attemptCount = %d", attemptCount)
	}
	pendingCount, err := rawRepo.CountUnprocessed(ctx, 5)
	if err != nil {
		t.Fatalf("rawRepo.CountUnprocessed() error = %v", err)
	}
	if pendingCount < 0 {
		t.Fatalf("pendingCount = %d", pendingCount)
	}

	var rawCount uint64
	if err := conn.QueryRow(ctx, "SELECT count() FROM raw_kick_events WHERE id = ?", rawEvent.ID).Scan(&rawCount); err != nil {
		t.Fatalf("count raw events error = %v", err)
	}
	if rawCount != 1 {
		t.Fatalf("rawCount = %d", rawCount)
	}

	migrationRepo := clickhouseinfra.NewDataMigrationRepository(conn)
	migrationCounts, err := migrationRepo.DataCounts(ctx)
	if err != nil {
		t.Fatalf("migrationRepo.DataCounts() error = %v", err)
	}
	if migrationCounts.ChatMessages < 3 || migrationCounts.RawEvents < 1 || migrationCounts.RawEventAttempts < 1 {
		t.Fatalf("migration counts = %#v", migrationCounts)
	}
	migratedRaw, err := migrationRepo.FindRawEvent(ctx, rawEvent.ID)
	if err != nil {
		t.Fatalf("migrationRepo.FindRawEvent() error = %v", err)
	}
	if migratedRaw.ID != rawEvent.ID || migratedRaw.PayloadJSON == "" {
		t.Fatalf("migratedRaw = %#v", migratedRaw)
	}

	stats := clickhouseinfra.NewStatsRepository(conn)
	sizes, err := stats.TableSizes(ctx)
	if err != nil {
		t.Fatalf("stats.TableSizes() error = %v", err)
	}
	if len(sizes) == 0 {
		t.Fatal("stats.TableSizes() returned no rows")
	}

	if err := rawRepo.InsertEventsBatch(ctx, nil); err != nil {
		t.Fatalf("rawRepo.InsertEventsBatch(empty) error = %v", err)
	}
	if err := messageRepo.InsertMessagesBatch(ctx, nil); err != nil {
		t.Fatalf("messageRepo.InsertMessagesBatch(empty) error = %v", err)
	}
	if err := rawRepo.InsertAttemptsBatch(ctx, nil); err != nil {
		t.Fatalf("rawRepo.InsertAttemptsBatch(empty) error = %v", err)
	}

	batchEvents := []domain.RawKickEvent{
		{
			ID:            "raw-batch-1-" + suffix,
			ChannelSlug:   "hype",
			EventType:     "pusher",
			EventName:     "App\\Events\\ChatMessageEvent",
			KickMessageID: "raw-batch-1-" + suffix,
			ChatroomID:    456,
			ChannelID:     123,
			PayloadJSON:   `{"event":"App\\Events\\ChatMessageEvent","seq":1}`,
			Status:        "pending",
			ReceivedAt:    time.Now().UTC(),
		},
		{
			ID:            "raw-batch-2-" + suffix,
			ChannelSlug:   "hype",
			EventType:     "pusher",
			EventName:     "App\\Events\\ChatMessageEvent",
			KickMessageID: "raw-batch-2-" + suffix,
			ChatroomID:    456,
			ChannelID:     123,
			PayloadJSON:   `{"event":"App\\Events\\ChatMessageEvent","seq":2}`,
			Status:        "pending",
			ReceivedAt:    time.Now().UTC(),
		},
		{
			ID:            "raw-batch-3-" + suffix,
			ChannelSlug:   "hype",
			EventType:     "pusher",
			EventName:     "App\\Events\\ChatMessageEvent",
			KickMessageID: "raw-batch-3-" + suffix,
			ChatroomID:    456,
			ChannelID:     123,
			PayloadJSON:   `{"event":"App\\Events\\ChatMessageEvent","seq":3}`,
			Status:        "pending",
			ReceivedAt:    time.Now().UTC(),
		},
	}
	if err := rawRepo.InsertEventsBatch(ctx, batchEvents); err != nil {
		t.Fatalf("rawRepo.InsertEventsBatch() error = %v", err)
	}
	for _, event := range batchEvents {
		fetched, err := rawRepo.GetByID(ctx, event.ID)
		if err != nil {
			t.Fatalf("rawRepo.GetByID(%s) error = %v", event.ID, err)
		}
		if fetched.ID != event.ID || fetched.KickMessageID != event.KickMessageID {
			t.Fatalf("batch event mismatch = %#v", fetched)
		}
	}

	batchAttempts := []domain.RawEventAttempt{
		{ID: "attempt-batch-1-" + suffix, RawEventID: batchEvents[0].ID, Attempt: 1, Status: "processed", StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()},
		{ID: "attempt-batch-2-" + suffix, RawEventID: batchEvents[1].ID, Attempt: 1, Status: "processed", StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()},
		{ID: "attempt-batch-3-" + suffix, RawEventID: batchEvents[2].ID, Attempt: 1, Status: "failed", StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()},
	}
	if err := rawRepo.InsertAttemptsBatch(ctx, batchAttempts); err != nil {
		t.Fatalf("rawRepo.InsertAttemptsBatch() error = %v", err)
	}
	for _, attempt := range batchAttempts {
		count, err := rawRepo.AttemptCount(ctx, attempt.RawEventID)
		if err != nil {
			t.Fatalf("rawRepo.AttemptCount(%s) error = %v", attempt.RawEventID, err)
		}
		if count != 1 {
			t.Fatalf("batch attempt count = %d for %s", count, attempt.RawEventID)
		}
	}

	batchMessages := []domain.ChatMessage{
		{
			ID:                 baseID + 100,
			KickMessageID:      "kick-batch-1-" + suffix,
			ChannelKickID:      123,
			ChannelChatroomID:  456,
			ChannelSlug:        "hype-batch-" + suffix,
			ChannelDisplayName: "Hype Batch " + suffix,
			ChannelPublicURL:   "https://kick.com/hype-batch-" + suffix,
			SenderKickID:       321,
			SenderUsername:     "batch_user_" + suffix,
			SenderSlug:         "batch-user-" + suffix,
			SenderPublicURL:    "https://kick.com/batch-user-" + suffix,
			MessageType:        "message",
			Content:            "batch needle 1 " + suffix,
			RawPayloadJSON:     `{"type":"ChatMessageEvent"}`,
			MessageCreatedAt:   baseTime.Add(5 * time.Minute),
		},
		{
			ID:                 baseID + 101,
			KickMessageID:      "kick-batch-2-" + suffix,
			ChannelKickID:      123,
			ChannelChatroomID:  456,
			ChannelSlug:        "hype-batch-" + suffix,
			ChannelDisplayName: "Hype Batch " + suffix,
			ChannelPublicURL:   "https://kick.com/hype-batch-" + suffix,
			SenderKickID:       321,
			SenderUsername:     "batch_user_" + suffix,
			SenderSlug:         "batch-user-" + suffix,
			SenderPublicURL:    "https://kick.com/batch-user-" + suffix,
			MessageType:        "message",
			Content:            "batch needle 2 " + suffix,
			RawPayloadJSON:     `{"type":"ChatMessageEvent"}`,
			MessageCreatedAt:   baseTime.Add(6 * time.Minute),
		},
	}
	if err := messageRepo.InsertMessagesBatch(ctx, batchMessages); err != nil {
		t.Fatalf("messageRepo.InsertMessagesBatch() error = %v", err)
	}
	batchFound, err := messageRepo.Search(ctx, domain.MessageSearchFilter{
		Sender: batchMessages[0].SenderUsername,
		Query:  "batch needle",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("messageRepo.Search(batch) error = %v", err)
	}
	if len(batchFound) != 2 {
		t.Fatalf("batch messages len = %d", len(batchFound))
	}

	subPeriodRepo := clickhouseinfra.NewSubscriptionPeriodRepository(conn)

	now := time.Now().UTC().Truncate(time.Millisecond)
	periods := []domain.ChannelSubscriptionPeriod{
		{
			ID:                   "period-new-" + suffix,
			EventMessageID:       "msg-new-" + suffix,
			EventType:            "channel.subscription.new",
			FollowedChannelID:    1,
			BroadcasterUserID:    1001,
			ChannelSlug:          "hype-" + suffix,
			ChannelDisplayName:   "Hype",
			SubscriberKickUserID: 5001,
			SubscriberUsername:   "subscriber_" + suffix,
			SubscriberSlug:       "subscriber-" + suffix,
			IsGift:               false,
			StartedAt:            now.Add(-24 * time.Hour),
			ExpiresAt:            now.Add(6 * 24 * time.Hour),
			RawPayloadJSON:       `{}`,
			IngestedAt:           now,
		},
		{
			ID:                   "period-gift-" + suffix,
			EventMessageID:       "msg-gift-" + suffix,
			EventType:            "channel.subscription.gifts",
			FollowedChannelID:    1,
			BroadcasterUserID:    1001,
			ChannelSlug:          "hype-" + suffix,
			ChannelDisplayName:   "Hype",
			SubscriberKickUserID: 5002,
			SubscriberUsername:   "giftee_" + suffix,
			SubscriberSlug:       "giftee-" + suffix,
			GifterKickUserID:     5003,
			GifterUsername:       "gifter_" + suffix,
			GifterSlug:           "gifter-" + suffix,
			IsGift:               true,
			StartedAt:            now.Add(-24 * time.Hour),
			ExpiresAt:            now.Add(6 * 24 * time.Hour),
			RawPayloadJSON:       `{}`,
			IngestedAt:           now,
		},
		{
			ID:                   "period-expired-" + suffix,
			EventMessageID:       "msg-expired-" + suffix,
			EventType:            "channel.subscription.renewal",
			FollowedChannelID:    1,
			BroadcasterUserID:    1001,
			ChannelSlug:          "hype-" + suffix,
			ChannelDisplayName:   "Hype",
			SubscriberKickUserID: 5004,
			SubscriberUsername:   "expired_" + suffix,
			SubscriberSlug:       "expired-" + suffix,
			IsGift:               false,
			StartedAt:            now.Add(-60 * 24 * time.Hour),
			ExpiresAt:            now.Add(-30 * 24 * time.Hour),
			RawPayloadJSON:       `{}`,
			IngestedAt:           now,
		},
	}

	if err := subPeriodRepo.InsertBatch(ctx, periods); err != nil {
		t.Fatalf("subPeriodRepo.InsertBatch() error = %v", err)
	}

	if err := subPeriodRepo.InsertBatch(ctx, nil); err != nil {
		t.Fatalf("subPeriodRepo.InsertBatch(empty) error = %v", err)
	}

	if err := subPeriodRepo.InsertBatch(ctx, periods); err != nil {
		t.Fatalf("subPeriodRepo.InsertBatch(duplicate) error = %v", err)
	}

	summary, err := subPeriodRepo.ActiveSummary(ctx, 1)
	if err != nil {
		t.Fatalf("subPeriodRepo.ActiveSummary() error = %v", err)
	}
	if summary.ActiveCount != 2 {
		t.Fatalf("ActiveCount = %d, want 2 (expired excluded)", summary.ActiveCount)
	}
	if summary.ActiveGiftedCount != 1 {
		t.Fatalf("ActiveGiftedCount = %d, want 1", summary.ActiveGiftedCount)
	}
}
