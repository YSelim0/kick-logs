package clickhouse_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/config"
	"github.com/YSelim0/kick-logs/apps/api-go/internal/domain"
	clickhouseinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/clickhouse"
)

// Explicitly opt in against existing data. This benchmark never migrates or inserts.
func BenchmarkProfileSummaries(b *testing.B) {
	channel := os.Getenv("KICK_LOGS_PROFILE_BENCHMARK_CHANNEL")
	sender := os.Getenv("KICK_LOGS_PROFILE_BENCHMARK_SENDER")
	if channel == "" && sender == "" {
		b.Skip("set KICK_LOGS_PROFILE_BENCHMARK_CHANNEL or KICK_LOGS_PROFILE_BENCHMARK_SENDER")
	}
	end, err := time.Parse(time.RFC3339, os.Getenv("KICK_LOGS_PROFILE_BENCHMARK_END"))
	if err != nil {
		b.Fatal("set KICK_LOGS_PROFILE_BENCHMARK_END to a fixed historical RFC3339 instant")
	}
	cfg, err := config.Load()
	if err != nil {
		b.Fatal(err)
	}
	conn, err := clickhouseinfra.Open(context.Background(), cfg)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = conn.Close() })
	repo := clickhouseinfra.NewAnalyticsRepository(conn)
	for _, target := range []struct{ kind, slug string }{{"channel", channel}, {"user", sender}} {
		if target.slug == "" {
			continue
		}
		b.Run(target.kind, func(b *testing.B) {
			filter := domain.AnalyticsFilter{End: end}
			if target.kind == "channel" {
				filter.Channel = target.slug
			} else {
				filter.Sender = target.slug
			}
			fetch := func(ctx context.Context, combined bool) (profileBenchmarkResult, error) {
				var result profileBenchmarkResult
				var err error
				if combined {
					if target.kind == "channel" {
						result.overview, result.senders, err = repo.OverviewAndTopSenders(ctx, filter, 5)
					} else {
						result.overview, result.channels, err = repo.OverviewAndTopChannels(ctx, filter, 5)
					}
					return result, err
				}
				result.overview, err = repo.Overview(ctx, filter)
				if err != nil {
					return result, err
				}
				if target.kind == "channel" {
					result.senders, err = repo.TopSenders(ctx, filter, 5)
				} else {
					result.channels, err = repo.TopChannels(ctx, filter, 5)
				}
				return result, err
			}
			var rows, bytes atomic.Uint64
			ctx := ch.Context(context.Background(), ch.WithSettings(ch.Settings{
				"max_threads": 1, "max_memory_usage": 256 * 1024 * 1024,
				"max_execution_time": 10, "use_query_cache": 0,
			}), ch.WithProgress(func(progress *ch.Progress) {
				rows.Add(progress.Rows)
				bytes.Add(progress.Bytes)
			}))
			before, err := fetch(ctx, false)
			if err != nil {
				b.Fatal(err)
			}
			after, err := fetch(ctx, true)
			if err != nil || !reflect.DeepEqual(before, after) {
				b.Fatalf("results differ (or source data changed during comparison): err=%v", err)
			}
			if before.overview.TotalMessages == 0 {
				b.Fatal("selected profile has no matching messages; choose a representative profile")
			}
			b.Logf("equal results: messages=%d", before.overview.TotalMessages)
			for _, combined := range []bool{false, true} {
				b.Run(fmt.Sprintf("combined=%t", combined), func(b *testing.B) {
					rows.Store(0)
					bytes.Store(0)
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						if _, err := fetch(ctx, combined); err != nil {
							b.Fatal(err)
						}
					}
					b.StopTimer()
					b.ReportMetric(float64(rows.Load())/float64(b.N), "read-rows/op")
					b.ReportMetric(float64(bytes.Load())/float64(b.N), "read-bytes/op")
				})
			}
		})
	}
}

type profileBenchmarkResult struct {
	overview domain.AnalyticsOverview
	senders  []domain.TopSenderAnalytics
	channels []domain.TopChannelAnalytics
}
