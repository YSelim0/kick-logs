package routes

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
	messagesusecase "github.com/YSelim0/kick-logs/apps/api-go/internal/usecase/messages"
)

func adminUserResponse(user domain.AdminUser) schemas.AdminUserResponse {
	return schemas.AdminUserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Role:     string(user.Role),
		IsActive: user.IsActive,
	}
}

func channelResponse(channel domain.FollowedChannel) schemas.ChannelResponse {
	return schemas.ChannelResponse{
		ID:              channel.ID,
		KickChannelID:   nullableInt64(channel.KickChannelID),
		KickChatroomID:  nullableInt64(channel.KickChatroomID),
		Slug:            channel.Slug,
		DisplayName:     channel.DisplayName,
		ProfileImageURL: nullableString(channel.ProfileImageURL),
		BannerImageURL:  nullableString(channel.BannerImageURL),
		IsEnabled:       channel.IsEnabled,
	}
}

func operationsSummaryResponse(summary domain.OperationsSummary) schemas.OperationsSummaryResponse {
	tables := make([]schemas.OperationsStorageTableResponse, 0, len(summary.StorageTables))
	for _, table := range summary.StorageTables {
		tables = append(tables, schemas.OperationsStorageTableResponse{
			TableName:  table.Name,
			TotalBytes: table.BytesOnDisk,
		})
	}

	return schemas.OperationsSummaryResponse{
		Counts: schemas.OperationsCountsResponse{
			Channels:        summary.Counts.Channels,
			EnabledChannels: summary.Counts.EnabledChannels,
			Senders:         summary.Counts.Senders,
			Messages:        summary.Counts.Messages,
			RawEvents:       summary.Counts.RawEvents,
		},
		RawEventStatusCounts: summary.RawEventStatusCounts,
		Storage: schemas.OperationsStorageResponse{
			DatabaseBytes: summary.StorageDatabaseBytes,
			Tables:        tables,
		},
		Timestamps: schemas.OperationsTimestampsResponse{
			LatestMessageAt:                 nullableTime(summary.Timestamps.LatestMessageAt),
			LatestRawEventReceivedAt:        nullableTime(summary.Timestamps.LatestRawEventReceivedAt),
			LatestRawEventProcessedAt:       nullableTime(summary.Timestamps.LatestRawEventProcessedAt),
			OldestPendingRawEventReceivedAt: nullableTime(summary.Timestamps.OldestPendingRawEventReceivedAt),
		},
		Listener: schemas.ListenerHeartbeatResponse{
			ServiceName:          summary.Listener.ServiceName,
			LastSeenAt:           nullableTime(summary.Listener.LastSeenAt),
			IsFresh:              summary.Listener.IsFresh,
			StaleAfterSeconds:    summary.Listener.StaleAfterSeconds,
			SecondsSinceLastSeen: nullableListenerSeconds(summary.Listener),
		},
	}
}

func dataManagementSummaryResponse(summary domain.DataManagementSummary) schemas.DataManagementSummaryResponse {
	tables := make([]schemas.DataManagementTableResponse, 0, len(summary.Tables))
	for _, table := range summary.Tables {
		tables = append(tables, schemas.DataManagementTableResponse{
			TableName:  table.Name,
			TotalBytes: table.BytesOnDisk,
			RowCount:   table.Rows,
		})
	}
	return schemas.DataManagementSummaryResponse{
		Counts: schemas.DataManagementCountsResponse{
			Channels:  summary.Counts.Channels,
			Senders:   summary.Counts.Senders,
			Messages:  summary.Counts.Messages,
			RawEvents: summary.Counts.RawEvents,
		},
		DatabaseBytes:     summary.DatabaseBytes,
		Tables:            tables,
		RetentionSettings: retentionSettingsResponse(summary.RetentionSettings),
	}
}

func retentionSettingsResponse(settings domain.RetentionSettings) schemas.RetentionSettingsResponse {
	return schemas.RetentionSettingsResponse{
		MessageRetentionDays:  settings.MessageRetentionDays,
		RawEventRetentionDays: settings.RawEventRetentionDays,
		UpdatedAt:             nullableTime(settings.UpdatedAt),
	}
}

func dataCleanupPreviewResponse(preview domain.DataCleanupPreview) schemas.DataCleanupPreviewResponse {
	return schemas.DataCleanupPreviewResponse{
		Target:           string(preview.Target),
		Affected:         dataCleanupCountsResponse(preview.Affected),
		ConfirmationText: preview.ConfirmationText,
		CanExecute:       preview.CanExecute,
		CutoffAt:         nullableTime(preview.CutoffAt),
		ChannelSlug:      nullableString(preview.ChannelSlug),
		Sender:           nullableString(preview.Sender),
		RetentionDays:    preview.RetentionDays,
		Reason:           nullableString(preview.Reason),
	}
}

func dataCleanupResultResponse(result domain.DataCleanupResult) schemas.DataCleanupResultResponse {
	return schemas.DataCleanupResultResponse{
		Target:           string(result.Target),
		Deleted:          dataCleanupCountsResponse(result.Deleted),
		ConfirmationText: result.ConfirmationText,
		CutoffAt:         nullableTime(result.CutoffAt),
		ChannelSlug:      nullableString(result.ChannelSlug),
		Sender:           nullableString(result.Sender),
		RetentionDays:    result.RetentionDays,
	}
}

func dataCleanupCountsResponse(counts domain.DataCleanupCounts) schemas.DataCleanupCountsResponse {
	return schemas.DataCleanupCountsResponse{
		Messages:  counts.Messages,
		RawEvents: counts.RawEvents,
		Total:     counts.Total(),
	}
}

func messageSearchResponse(page messagesusecase.SearchPage) schemas.MessageSearchResponse {
	items := make([]schemas.MessageResponse, 0, len(page.Items))
	for _, message := range page.Items {
		items = append(items, messageResponse(message))
	}

	var nextCursor *string
	if page.NextCursor != nil {
		formatted := fmt.Sprintf(
			"%s|%d",
			formatCursorTime(page.NextCursor.MessageCreatedAt),
			page.NextCursor.MessageID,
		)
		nextCursor = &formatted
	}

	return schemas.MessageSearchResponse{Items: items, NextCursor: nextCursor}
}

func messageExportResponse(export messagesusecase.MessageExport) schemas.MessageExportResponse {
	items := make([]schemas.MessageResponse, 0, len(export.Items))
	for _, message := range export.Items {
		items = append(items, messageResponse(message))
	}
	return schemas.MessageExportResponse{
		Items:     items,
		Count:     export.Count,
		MaxRows:   export.MaxRows,
		Truncated: export.Truncated,
	}
}

func analyticsOverviewResponse(overview domain.AnalyticsOverview) schemas.AnalyticsOverviewResponse {
	return schemas.AnalyticsOverviewResponse{
		TotalMessages:    overview.TotalMessages,
		TotalSenders:     overview.TotalSenders,
		TotalChannels:    overview.TotalChannels,
		TotalEmoteUsages: overview.TotalEmoteUsages,
		FirstMessageAt:   nullableTime(overview.FirstMessageAt),
		LatestMessageAt:  nullableTime(overview.LatestMessageAt),
	}
}

func messageVolumeResponse(points []domain.MessageVolumePoint) schemas.MessageVolumeResponse {
	items := make([]schemas.MessageVolumePointResponse, 0, len(points))
	for _, point := range points {
		items = append(items, schemas.MessageVolumePointResponse{
			BucketStart:  point.BucketStart.UTC().Format(time.RFC3339),
			MessageCount: point.MessageCount,
		})
	}
	return schemas.MessageVolumeResponse{Items: items}
}

func topSendersResponse(senders []domain.TopSenderAnalytics) schemas.TopSendersResponse {
	items := make([]schemas.TopSenderResponse, 0, len(senders))
	for _, sender := range senders {
		items = append(items, topSenderResponse(sender))
	}
	return schemas.TopSendersResponse{Items: items}
}

func topSenderResponse(sender domain.TopSenderAnalytics) schemas.TopSenderResponse {
	return schemas.TopSenderResponse{
		SenderID:        sender.SenderID,
		KickUserID:      sender.KickUserID,
		Username:        sender.Username,
		Slug:            sender.Slug,
		ProfileImageURL: nullableString(sender.ProfileImageURL),
		MessageCount:    sender.MessageCount,
		FirstMessageAt:  sender.FirstMessageAt.UTC().Format(time.RFC3339),
		LatestMessageAt: sender.LatestMessageAt.UTC().Format(time.RFC3339),
	}
}

func topChannelsResponse(channels []domain.TopChannelAnalytics) schemas.TopChannelsResponse {
	items := make([]schemas.TopChannelResponse, 0, len(channels))
	for _, channel := range channels {
		items = append(items, topChannelResponse(channel))
	}
	return schemas.TopChannelsResponse{Items: items}
}

func topChannelResponse(channel domain.TopChannelAnalytics) schemas.TopChannelResponse {
	return schemas.TopChannelResponse{
		ChannelID:       channel.ChannelID,
		Slug:            channel.Slug,
		DisplayName:     channel.DisplayName,
		ProfileImageURL: nullableString(channel.ProfileImageURL),
		BannerImageURL:  nullableString(channel.BannerImageURL),
		MessageCount:    channel.MessageCount,
		FirstMessageAt:  channel.FirstMessageAt.UTC().Format(time.RFC3339),
		LatestMessageAt: channel.LatestMessageAt.UTC().Format(time.RFC3339),
	}
}

func topEmotesResponse(emotes []domain.TopEmoteAnalytics) schemas.TopEmotesResponse {
	items := make([]schemas.TopEmoteResponse, 0, len(emotes))
	for _, emote := range emotes {
		items = append(items, schemas.TopEmoteResponse{
			ID:           emote.ID,
			Name:         emote.Name,
			Token:        emote.Token,
			ImageURL:     emote.ImageURL,
			UsageCount:   emote.UsageCount,
			MessageCount: emote.MessageCount,
		})
	}
	return schemas.TopEmotesResponse{Items: items}
}

func userProfileResponse(profile domain.UserProfile) schemas.UserProfileResponse {
	return schemas.UserProfileResponse{
		Sender: schemas.UserProfileSenderResponse{
			ID:              profile.Sender.ID,
			KickUserID:      profile.Sender.KickUserID,
			Username:        profile.Sender.Username,
			Slug:            profile.Sender.Slug,
			ProfileImageURL: nullableString(profile.Sender.ProfileImageURL),
		},
		Overview:       analyticsOverviewResponse(profile.Overview),
		MessageVolume:  messageVolumeResponse(profile.MessageVolume).Items,
		TopChannels:    topChannelsResponse(profile.TopChannels).Items,
		TopEmotes:      topEmotesResponse(profile.TopEmotes).Items,
		LatestMessages: latestMessageResponses(profile.LatestMessages),
	}
}

func channelProfileResponse(profile domain.ChannelProfile) schemas.ChannelProfileResponse {
	return schemas.ChannelProfileResponse{
		Channel: schemas.ChannelProfileChannelResponse{
			ID:              profile.Channel.ID,
			KickChannelID:   nullableInt64(profile.Channel.KickChannelID),
			KickChatroomID:  nullableInt64(profile.Channel.KickChatroomID),
			Slug:            profile.Channel.Slug,
			DisplayName:     profile.Channel.DisplayName,
			ProfileImageURL: nullableString(profile.Channel.ProfileImageURL),
			BannerImageURL:  nullableString(profile.Channel.BannerImageURL),
			IsEnabled:       profile.Channel.IsEnabled,
		},
		Overview:       analyticsOverviewResponse(profile.Overview),
		MessageVolume:  messageVolumeResponse(profile.MessageVolume).Items,
		TopSenders:     topSendersResponse(profile.TopSenders).Items,
		TopEmotes:      topEmotesResponse(profile.TopEmotes).Items,
		LatestMessages: latestMessageResponses(profile.LatestMessages),
	}
}

func latestMessageResponses(messages []domain.ChatMessage) []schemas.MessageResponse {
	items := make([]schemas.MessageResponse, 0, len(messages))
	for _, message := range messages {
		items = append(items, messageResponse(message))
	}
	return items
}

func messageResponse(message domain.ChatMessage) schemas.MessageResponse {
	emotes := make([]schemas.MessageEmoteResponse, 0, len(message.Emotes))
	for _, emote := range message.Emotes {
		emotes = append(emotes, schemas.MessageEmoteResponse{
			ID:       emote.ID,
			Name:     emote.Name,
			Token:    emote.Token,
			ImageURL: emote.ImageURL,
		})
	}

	return schemas.MessageResponse{
		ID:                     message.ID,
		KickMessageID:          message.KickMessageID,
		ChatroomID:             message.ChannelChatroomID,
		Content:                message.Content,
		MessageType:            message.MessageType,
		SenderUsernameSnapshot: message.SenderUsername,
		SenderSlugSnapshot:     message.SenderSlug,
		SenderColorSnapshot:    nullableString(message.SenderDisplayColor),
		SenderBadges:           parseJSONList(message.SenderBadgesJSON),
		Emotes:                 emotes,
		ReplyMetadata:          replyMetadata(message),
		ThreadParentID:         nullableString(message.ThreadParentID),
		MessageCreatedAt:       message.MessageCreatedAt.UTC().Format(time.RFC3339),
		IngestedAt:             message.IngestedAt.UTC().Format(time.RFC3339),
		Sender: schemas.MessageSenderResponse{
			ID:              message.SenderID,
			KickUserID:      message.SenderKickID,
			Username:        message.SenderUsername,
			Slug:            message.SenderSlug,
			ProfileImageURL: nullableString(message.SenderProfileImageURL),
		},
		Channel: schemas.MessageChannelResponse{
			ID:              message.ChannelID,
			Slug:            message.ChannelSlug,
			DisplayName:     message.ChannelDisplayName,
			ProfileImageURL: nullableString(message.ChannelProfileImageURL),
			BannerImageURL:  nullableString(message.ChannelBannerImageURL),
		},
	}
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullableInt64(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func nullableTime(value time.Time) *string {
	if value.IsZero() {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func parseJSONList(raw string) []map[string]any {
	if raw == "" {
		return []map[string]any{}
	}
	var values []map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []map[string]any{}
	}
	return values
}

func parseJSONObject(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return map[string]any{}
	}
	return value
}

func replyMetadata(message domain.ChatMessage) map[string]any {
	value := parseJSONObject(message.ReplyMetadataJSON)
	if len(value) > 0 {
		return value
	}
	if message.ReplyToSender == "" && message.ReplyToContent == "" {
		return map[string]any{}
	}
	return map[string]any{
		"original_sender":  map[string]any{"username": message.ReplyToSender},
		"original_message": map[string]any{"content": message.ReplyToContent},
	}
}

func formatCursorTime(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05.999999999+00:00")
}

func nullableListenerSeconds(heartbeat domain.ListenerHeartbeat) *int64 {
	if heartbeat.LastSeenAt.IsZero() {
		return nil
	}
	seconds := heartbeat.SecondsSinceLastSeen
	return &seconds
}
