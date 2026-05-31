package ports

type RateLimitResult struct {
	Limited    bool
	RetryAfter int
}

type RateLimiter interface {
	RateLimit(key string, perPeriod int, periodSeconds int, maxBurst int) (RateLimitResult, error)
}
