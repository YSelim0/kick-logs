package ratelimit

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"
	throttled "github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

type policyKey struct {
	perPeriod     int
	periodSeconds int
	maxBurst      int
}

type GCRARateLimiter struct {
	store    throttled.GCRAStoreCtx
	limiters sync.Map
}

func NewGCRA(maxKeys int) (*GCRARateLimiter, error) {
	store, err := memstore.NewCtx(maxKeys)
	if err != nil {
		return nil, fmt.Errorf("create memstore: %w", err)
	}
	return &GCRARateLimiter{store: store}, nil
}

func (g *GCRARateLimiter) RateLimit(key string, perPeriod int, periodSeconds int, maxBurst int) (ports.RateLimitResult, error) {
	pk := policyKey{perPeriod, periodSeconds, maxBurst}

	var limiter *throttled.GCRARateLimiterCtx
	if v, ok := g.limiters.Load(pk); ok {
		limiter = v.(*throttled.GCRARateLimiterCtx)
	} else {
		period := time.Duration(periodSeconds) * time.Second
		quota := throttled.RateQuota{
			MaxRate:  throttled.PerDuration(perPeriod, period),
			MaxBurst: maxBurst,
		}
		l, err := throttled.NewGCRARateLimiterCtx(g.store, quota)
		if err != nil {
			return ports.RateLimitResult{}, fmt.Errorf("create gcra limiter: %w", err)
		}
		actual, _ := g.limiters.LoadOrStore(pk, l)
		limiter = actual.(*throttled.GCRARateLimiterCtx)
	}

	storeKey := fmt.Sprintf("%d:%d:%d:%s", perPeriod, periodSeconds, maxBurst, key)
	limited, result, err := limiter.RateLimitCtx(context.Background(), storeKey, 1)
	if err != nil {
		return ports.RateLimitResult{}, fmt.Errorf("rate limit check: %w", err)
	}

	retryAfter := 0
	if limited && result.RetryAfter >= 0 {
		retryAfter = int(math.Ceil(result.RetryAfter.Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}
	}

	return ports.RateLimitResult{
		Limited:    limited,
		RetryAfter: retryAfter,
	}, nil
}
