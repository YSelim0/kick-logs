package clickhouse

import (
	"context"

	clickhousedriver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Limit only the periodic homepage work; existing API/profile contracts keep
// their current behavior. Sequential refreshes leave capacity for ingestion.
func NewHomepageAnalyticsRepository(conn driver.Conn) *AnalyticsRepository {
	return NewAnalyticsRepository(homepageConnection{conn})
}

type homepageConnection struct{ driver.Conn }

func (conn homepageConnection) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return conn.Conn.Query(homepageContext(ctx), query, args...)
}

func (conn homepageConnection) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	return conn.Conn.QueryRow(homepageContext(ctx), query, args...)
}

func homepageContext(ctx context.Context) context.Context {
	return clickhousedriver.Context(ctx, clickhousedriver.WithSettings(clickhousedriver.Settings{
		"max_threads":        2,
		"max_memory_usage":   384 * 1024 * 1024,
		"max_execution_time": 15,
	}))
}
