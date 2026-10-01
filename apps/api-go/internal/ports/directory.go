package ports

import (
	"context"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

// DirectoryRepository reads stored identities without consulting message history.
type DirectoryRepository interface {
	Search(context.Context, domain.DirectoryKind, domain.DirectoryQuery) ([]domain.DirectoryIdentity, error)
}
