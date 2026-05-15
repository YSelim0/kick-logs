package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
)

type StatsRepository struct {
	conn driver.Conn
}

func NewStatsRepository(conn driver.Conn) *StatsRepository {
	return &StatsRepository{conn: conn}
}

func (repo *StatsRepository) TableSizes(ctx context.Context) ([]domain.TableSize, error) {
	rows, err := repo.conn.Query(
		ctx,
		`SELECT table, sum(rows), sum(bytes_on_disk)
		 FROM system.parts
		 WHERE active AND database = currentDatabase()
		 GROUP BY table
		 ORDER BY table ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query clickhouse table sizes: %w", err)
	}
	defer rows.Close()

	sizes := make([]domain.TableSize, 0)
	for rows.Next() {
		var size domain.TableSize
		var rowCount uint64
		var bytesOnDisk uint64
		if err := rows.Scan(&size.Name, &rowCount, &bytesOnDisk); err != nil {
			return nil, fmt.Errorf("scan clickhouse table size: %w", err)
		}
		size.Rows = int64(rowCount)
		size.BytesOnDisk = int64(bytesOnDisk)
		sizes = append(sizes, size)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clickhouse table sizes: %w", err)
	}
	return sizes, nil
}
