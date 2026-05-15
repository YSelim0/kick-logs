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
