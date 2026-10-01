package ports

import "github.com/YSelim0/kick-logs/apps/api-go/internal/domain"

type HomepageSnapshotStore interface {
	Load() (domain.HomepageSnapshot, error)
	Save(snapshot domain.HomepageSnapshot) error
}
