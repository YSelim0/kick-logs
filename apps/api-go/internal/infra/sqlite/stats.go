package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type StatsRepository struct {
	db   *sql.DB
	path string
}

func NewStatsRepository(db *sql.DB, path string) *StatsRepository {
	return &StatsRepository{db: db, path: path}
}

func (repo *StatsRepository) TableSizes(ctx context.Context) ([]domain.TableSize, error) {
	tables := []string{
		"admin_users",
		"followed_channels",
		"sender_profiles",
		"retention_settings",
		"worker_heartbeats",
		"data_migrations",
		"schema_migrations",
	}

	fileSize := int64(0)
	if stat, err := os.Stat(repo.path); err == nil {
		fileSize = stat.Size()
	}

	sizes := make([]domain.TableSize, 0, len(tables)+1)
	sizes = append(sizes, domain.TableSize{Name: "_database_file", Rows: 0, BytesOnDisk: fileSize})
	for _, table := range tables {
		rows, err := repo.countRows(ctx, table)
		if err != nil {
			return nil, err
		}
		sizes = append(sizes, domain.TableSize{Name: table, Rows: rows, BytesOnDisk: 0})
	}
	return sizes, nil
}

func (repo *StatsRepository) countRows(ctx context.Context, table string) (int64, error) {
	var rows int64
	if err := repo.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&rows); err != nil {
		return 0, fmt.Errorf("count sqlite rows for %s: %w", table, err)
	}
	return rows, nil
}
