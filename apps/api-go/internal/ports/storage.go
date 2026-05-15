package ports

import (
	"context"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type AdminUserRepository interface {
	Upsert(ctx context.Context, user domain.AdminUser) (domain.AdminUser, error)
	GetByID(ctx context.Context, id int64) (domain.AdminUser, error)
	GetByEmail(ctx context.Context, email string) (domain.AdminUser, error)
	ListActive(ctx context.Context) ([]domain.AdminUser, error)
}

type FollowedChannelRepository interface {
	Upsert(ctx context.Context, channel domain.FollowedChannel) (domain.FollowedChannel, error)
	GetByID(ctx context.Context, id int64) (domain.FollowedChannel, error)
	GetBySlug(ctx context.Context, slug string) (domain.FollowedChannel, error)
	List(ctx context.Context) ([]domain.FollowedChannel, error)
	ListEnabled(ctx context.Context) ([]domain.FollowedChannel, error)
	Disable(ctx context.Context, id int64) (domain.FollowedChannel, error)
}

type MessageRepository interface {
	Insert(ctx context.Context, message domain.ChatMessage) error
	SearchBasic(ctx context.Context, filter domain.MessageSearchFilter) ([]domain.ChatMessage, error)
}

type RawEventRepository interface {
	InsertEvent(ctx context.Context, event domain.RawKickEvent) error
	InsertAttempt(ctx context.Context, attempt domain.RawEventAttempt) error
}

type StorageStatsRepository interface {
	TableSizes(ctx context.Context) ([]domain.TableSize, error)
}

type OperationsRepository interface {
	Summary(ctx context.Context) (domain.OperationsSummary, error)
}
