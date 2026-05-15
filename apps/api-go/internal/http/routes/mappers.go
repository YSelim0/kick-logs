package routes

import (
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/http/schemas"
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

func nullableListenerSeconds(heartbeat domain.ListenerHeartbeat) *int64 {
	if heartbeat.LastSeenAt.IsZero() {
		return nil
	}
	seconds := heartbeat.SecondsSinceLastSeen
	return &seconds
}
