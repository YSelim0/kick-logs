package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppName                              string
	AppEnv                               string
	LogLevel                             string
	APIHost                              string
	APIPort                              int
	BackendCORSOrigins                   []string
	KickPusherURL                        string
	MessageExportMaxRows                 int
	JWTSecretKey                         string
	JWTAlgorithm                         string
	JWTExpiresMinutes                    int
	JWTCookieName                        string
	JWTCookieSecure                      bool
	JWTCookieSameSite                    string
	SeedSuperAdmin                       bool
	ListenerReconnectInitialDelaySeconds float64
	ListenerReconnectMaxDelaySeconds     float64
	ListenerReconnectMultiplier          float64
	ListenerWorkerCount                  int
	ListenerRawEventBatchSize            int
	ListenerRawEventProcessingTimeout    int
	ListenerRawEventMaxAttempts          int
	ListenerRawEventWorkerIdleDelay      float64
	ListenerChannelResyncInterval        float64
	ListenerHeartbeatInterval            float64
	ListenerStaleAfter                   int
	ListenerRecentMessagePollEnabled     bool
	ListenerRecentMessagePollInterval    float64
	ListenerRecentMessagePollConcurrency int
	ListenerRawEventWriteBatchSize       int
	ListenerRawEventWriteFlushIntervalMS int
	ListenerRawEventWriteQueueSize       int
	ListenerRawEventWriteMaxRetries      int
	ListenerBootstrapRawQueueOnStartup   bool
	ListenerClickHouseBackoffInitialMS   int
	ListenerClickHouseBackoffMaxMS       int
	ListenerClickHouseBackoffMultiplier  float64
	ListenerClickHouseBreakerThreshold   int
	NATSURL                              string
	NATSRawEventStream                   string
	NATSRawEventSubject                  string
	NATSRawEventConsumer                 string
	NATSRawEventAckWaitSeconds           int
	NATSRawEventFetchBatchSize           int
	NATSRawEventFetchTimeoutSeconds      int
	SQLitePath                           string
	ClickHouseAddr                       string
	ClickHouseDatabase                   string
	ClickHouseUsername                   string
	ClickHousePassword                   string
	ClickHouseDebug                      bool
	PostgresSourceDSN                    string
	DefaultAdminEmail                    string
	DefaultAdminPassword                 string
	RateLimitEnabled                     bool
	RateLimitStoreMaxKeys                int
	RateLimitTrustProxy                  bool
	RateLimitClientIPHeader              string
	KickClientID                         string
	KickClientSecret                     string
	KickAPIBaseURL                       string
	KickOAuthTokenURL                    string
	KickWebhookPublicKey                 string
	KickWebhookSyncEnabled               bool
	KickWebhookEvents                    []string
	KickWebhookProcessBatchSize          int
	KickWebhookProcessMaxAttempts        int
	KickWebhookSkipVerification          bool
	NotifyEmailTo                        string
	NotifyEmailCooldownSeconds           int
	WatchlistRefreshIntervalSeconds      int
	SMTPHost                             string
	SMTPPort                             int
	SMTPUsername                         string
	SMTPPassword                         string
	SMTPFrom                             string
}

func Load() (Config, error) {
	apiPort, err := envInt("API_PORT", 8000)
	if err != nil {
		return Config{}, err
	}

	maxRows, err := envInt("MESSAGE_EXPORT_MAX_ROWS", 1000)
	if err != nil {
		return Config{}, err
	}

	jwtExpiresMinutes, err := envInt("JWT_EXPIRES_MINUTES", 10080)
	if err != nil {
		return Config{}, err
	}

	jwtCookieSecure, err := envBool("JWT_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}

	seedSuperAdmin, err := envBool("SEED_SUPER_ADMIN_ON_STARTUP", true)
	if err != nil {
		return Config{}, err
	}

	reconnectInitialDelay, err := envFloat("LISTENER_RECONNECT_INITIAL_DELAY_SECONDS", 1.0)
	if err != nil {
		return Config{}, err
	}

	reconnectMaxDelay, err := envFloat("LISTENER_RECONNECT_MAX_DELAY_SECONDS", 30.0)
	if err != nil {
		return Config{}, err
	}

	reconnectMultiplier, err := envFloat("LISTENER_RECONNECT_MULTIPLIER", 2.0)
	if err != nil {
		return Config{}, err
	}

	listenerWorkerCount, err := envInt("LISTENER_WORKER_COUNT", 4)
	if err != nil {
		return Config{}, err
	}

	rawEventBatchSize, err := envInt("LISTENER_RAW_EVENT_BATCH_SIZE", 100)
	if err != nil {
		return Config{}, err
	}

	rawEventProcessingTimeout, err := envInt("LISTENER_RAW_EVENT_PROCESSING_TIMEOUT_SECONDS", 300)
	if err != nil {
		return Config{}, err
	}

	rawEventMaxAttempts, err := envInt("LISTENER_RAW_EVENT_MAX_ATTEMPTS", 5)
	if err != nil {
		return Config{}, err
	}

	rawEventWorkerIdleDelay, err := envFloat("LISTENER_RAW_EVENT_WORKER_IDLE_DELAY_SECONDS", 0.25)
	if err != nil {
		return Config{}, err
	}

	channelResyncInterval, err := envFloat("LISTENER_CHANNEL_RESYNC_INTERVAL_SECONDS", 60.0)
	if err != nil {
		return Config{}, err
	}

	heartbeatInterval, err := envFloat("LISTENER_HEARTBEAT_INTERVAL_SECONDS", 15.0)
	if err != nil {
		return Config{}, err
	}

	listenerStaleAfter, err := envInt("LISTENER_HEARTBEAT_STALE_AFTER_SECONDS", 45)
	if err != nil {
		return Config{}, err
	}

	recentMessagePollEnabled, err := envBool("LISTENER_RECENT_MESSAGE_POLL_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	recentMessagePollInterval, err := envFloat("LISTENER_RECENT_MESSAGE_POLL_INTERVAL_SECONDS", 10.0)
	if err != nil {
		return Config{}, err
	}

	recentMessagePollConcurrency, err := envInt("LISTENER_RECENT_MESSAGE_POLL_CONCURRENCY", 8)
	if err != nil {
		return Config{}, err
	}

	rawEventWriteBatchSize, err := envInt("LISTENER_RAW_EVENT_WRITE_BATCH_SIZE", 500)
	if err != nil {
		return Config{}, err
	}

	rawEventWriteFlushIntervalMS, err := envInt("LISTENER_RAW_EVENT_WRITE_FLUSH_INTERVAL_MS", 500)
	if err != nil {
		return Config{}, err
	}

	rawEventWriteQueueSize, err := envInt("LISTENER_RAW_EVENT_WRITE_QUEUE_SIZE", 50000)
	if err != nil {
		return Config{}, err
	}

	rawEventWriteMaxRetries, err := envInt("LISTENER_RAW_EVENT_WRITE_MAX_RETRIES", 10)
	if err != nil {
		return Config{}, err
	}

	bootstrapRawQueueOnStartup, err := envBool("LISTENER_BOOTSTRAP_RAW_QUEUE_ON_STARTUP", false)
	if err != nil {
		return Config{}, err
	}

	clickHouseBackoffInitialMS, err := envInt("LISTENER_CLICKHOUSE_BACKOFF_INITIAL_MS", 1000)
	if err != nil {
		return Config{}, err
	}

	clickHouseBackoffMaxMS, err := envInt("LISTENER_CLICKHOUSE_BACKOFF_MAX_MS", 30000)
	if err != nil {
		return Config{}, err
	}

	clickHouseBackoffMultiplier, err := envFloat("LISTENER_CLICKHOUSE_BACKOFF_MULTIPLIER", 2.0)
	if err != nil {
		return Config{}, err
	}

	clickHouseBreakerThreshold, err := envInt("LISTENER_CLICKHOUSE_BREAKER_FAILURE_THRESHOLD", 5)
	if err != nil {
		return Config{}, err
	}

	natsRawEventAckWaitSeconds, err := envInt("NATS_RAW_EVENT_ACK_WAIT_SECONDS", 60)
	if err != nil {
		return Config{}, err
	}

	natsRawEventFetchBatchSize, err := envInt("NATS_RAW_EVENT_FETCH_BATCH_SIZE", 500)
	if err != nil {
		return Config{}, err
	}

	natsRawEventFetchTimeoutSeconds, err := envInt("NATS_RAW_EVENT_FETCH_TIMEOUT_SECONDS", 2)
	if err != nil {
		return Config{}, err
	}

	clickHouseDebug, err := envBool("CLICKHOUSE_DEBUG", false)
	if err != nil {
		return Config{}, err
	}

	rateLimitEnabled, err := envBool("RATE_LIMIT_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	rateLimitStoreMaxKeys, err := envInt("RATE_LIMIT_STORE_MAX_KEYS", 65536)
	if err != nil {
		return Config{}, err
	}

	rateLimitTrustProxy, err := envBool("RATE_LIMIT_TRUST_PROXY", true)
	if err != nil {
		return Config{}, err
	}

	kickWebhookSyncEnabled, err := envBool("KICK_WEBHOOK_SYNC_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	kickWebhookProcessBatchSize, err := envInt("KICK_WEBHOOK_PROCESS_BATCH_SIZE", 50)
	if err != nil {
		return Config{}, err
	}

	kickWebhookProcessMaxAttempts, err := envInt("KICK_WEBHOOK_PROCESS_MAX_ATTEMPTS", 5)
	if err != nil {
		return Config{}, err
	}

	kickWebhookSkipVerification, err := envBool("KICK_WEBHOOK_SKIP_VERIFICATION", false)
	if err != nil {
		return Config{}, err
	}

	notifyEmailCooldownSeconds, err := envInt("NOTIFY_EMAIL_COOLDOWN_SECONDS", 600)
	if err != nil {
		return Config{}, err
	}

	smtpPort, err := envInt("SMTP_PORT", 587)
	if err != nil {
		return Config{}, err
	}

	watchlistRefreshIntervalSeconds, err := envInt("WATCHLIST_REFRESH_INTERVAL_SECONDS", 30)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppName:                              envString("APP_NAME", "Kick Logs"),
		AppEnv:                               envString("APP_ENV", "local"),
		LogLevel:                             envString("LOG_LEVEL", "INFO"),
		APIHost:                              envString("API_HOST", "0.0.0.0"),
		APIPort:                              apiPort,
		BackendCORSOrigins:                   envCSV("BACKEND_CORS_ORIGINS", "http://localhost:3000"),
		KickPusherURL:                        envString("KICK_PUSHER_URL", "wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679?protocol=7&client=js&version=8.4.0-rc2&flash=false"),
		MessageExportMaxRows:                 maxRows,
		JWTSecretKey:                         envString("JWT_SECRET_KEY", "change-me-for-local-development-secret-key"),
		JWTAlgorithm:                         envString("JWT_ALGORITHM", "HS256"),
		JWTExpiresMinutes:                    jwtExpiresMinutes,
		JWTCookieName:                        envString("JWT_COOKIE_NAME", "kick_logs_session"),
		JWTCookieSecure:                      jwtCookieSecure,
		JWTCookieSameSite:                    envString("JWT_COOKIE_SAMESITE", "lax"),
		SeedSuperAdmin:                       seedSuperAdmin,
		ListenerReconnectInitialDelaySeconds: reconnectInitialDelay,
		ListenerReconnectMaxDelaySeconds:     reconnectMaxDelay,
		ListenerReconnectMultiplier:          reconnectMultiplier,
		ListenerWorkerCount:                  listenerWorkerCount,
		ListenerRawEventBatchSize:            rawEventBatchSize,
		ListenerRawEventProcessingTimeout:    rawEventProcessingTimeout,
		ListenerRawEventMaxAttempts:          rawEventMaxAttempts,
		ListenerRawEventWorkerIdleDelay:      rawEventWorkerIdleDelay,
		ListenerChannelResyncInterval:        channelResyncInterval,
		ListenerHeartbeatInterval:            heartbeatInterval,
		ListenerStaleAfter:                   listenerStaleAfter,
		ListenerRecentMessagePollEnabled:     recentMessagePollEnabled,
		ListenerRecentMessagePollInterval:    recentMessagePollInterval,
		ListenerRecentMessagePollConcurrency: recentMessagePollConcurrency,
		ListenerRawEventWriteBatchSize:       rawEventWriteBatchSize,
		ListenerRawEventWriteFlushIntervalMS: rawEventWriteFlushIntervalMS,
		ListenerRawEventWriteQueueSize:       rawEventWriteQueueSize,
		ListenerRawEventWriteMaxRetries:      rawEventWriteMaxRetries,
		ListenerBootstrapRawQueueOnStartup:   bootstrapRawQueueOnStartup,
		ListenerClickHouseBackoffInitialMS:   clickHouseBackoffInitialMS,
		ListenerClickHouseBackoffMaxMS:       clickHouseBackoffMaxMS,
		ListenerClickHouseBackoffMultiplier:  clickHouseBackoffMultiplier,
		ListenerClickHouseBreakerThreshold:   clickHouseBreakerThreshold,
		NATSURL:                              envString("NATS_URL", "nats://127.0.0.1:4222"),
		NATSRawEventStream:                   envString("NATS_RAW_EVENT_STREAM", "KICK_RAW_EVENTS"),
		NATSRawEventSubject:                  envString("NATS_RAW_EVENT_SUBJECT", "kick.raw.chat"),
		NATSRawEventConsumer:                 envString("NATS_RAW_EVENT_CONSUMER", "kick-raw-event-processor"),
		NATSRawEventAckWaitSeconds:           natsRawEventAckWaitSeconds,
		NATSRawEventFetchBatchSize:           natsRawEventFetchBatchSize,
		NATSRawEventFetchTimeoutSeconds:      natsRawEventFetchTimeoutSeconds,
		SQLitePath:                           envString("SQLITE_PATH", "var/kick-logs-go.sqlite3"),
		ClickHouseAddr:                       envString("CLICKHOUSE_ADDR", "127.0.0.1:9000"),
		ClickHouseDatabase:                   envString("CLICKHOUSE_DATABASE", "kick_logs"),
		ClickHouseUsername:                   envString("CLICKHOUSE_USERNAME", "kick_logs"),
		ClickHousePassword:                   envString("CLICKHOUSE_PASSWORD", "kick_logs"),
		ClickHouseDebug:                      clickHouseDebug,
		PostgresSourceDSN:                    envString("POSTGRES_SOURCE_DSN", envString("DATABASE_URL", "")),
		DefaultAdminEmail:                    envString("DEFAULT_SUPER_ADMIN_EMAIL", "admin@kicklogs.local"),
		DefaultAdminPassword:                 envString("DEFAULT_SUPER_ADMIN_PASSWORD", "admin123"),
		RateLimitEnabled:                     rateLimitEnabled,
		RateLimitStoreMaxKeys:                rateLimitStoreMaxKeys,
		RateLimitTrustProxy:                  rateLimitTrustProxy,
		RateLimitClientIPHeader:              envString("RATE_LIMIT_CLIENT_IP_HEADER", "CF-Connecting-IP"),
		KickClientID:                         envString("KICK_CLIENT_ID", ""),
		KickClientSecret:                     envString("KICK_CLIENT_SECRET", ""),
		KickAPIBaseURL:                       envString("KICK_API_BASE_URL", "https://api.kick.com"),
		KickOAuthTokenURL:                    envString("KICK_OAUTH_TOKEN_URL", "https://id.kick.com/oauth/token"),
		KickWebhookPublicKey:                 envString("KICK_WEBHOOK_PUBLIC_KEY", ""),
		KickWebhookSyncEnabled:               kickWebhookSyncEnabled,
		KickWebhookEvents:                    envCSV("KICK_WEBHOOK_EVENTS", "channel.subscription.new,channel.subscription.renewal,channel.subscription.gifts"),
		KickWebhookProcessBatchSize:          kickWebhookProcessBatchSize,
		KickWebhookProcessMaxAttempts:        kickWebhookProcessMaxAttempts,
		KickWebhookSkipVerification:          kickWebhookSkipVerification,
		NotifyEmailTo:                        envString("NOTIFY_EMAIL_TO", ""),
		NotifyEmailCooldownSeconds:           notifyEmailCooldownSeconds,
		WatchlistRefreshIntervalSeconds:      watchlistRefreshIntervalSeconds,
		SMTPHost:                             envString("SMTP_HOST", ""),
		SMTPPort:                             smtpPort,
		SMTPUsername:                         envString("SMTP_USERNAME", ""),
		SMTPPassword:                         envString("SMTP_PASSWORD", ""),
		SMTPFrom:                             envString("SMTP_FROM", ""),
	}, nil
}

func (cfg Config) APIAddress() string {
	return net.JoinHostPort(cfg.APIHost, strconv.Itoa(cfg.APIPort))
}

func envString(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return parsed, nil
}

func envBool(name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", name, err)
	}
	return parsed, nil
}

func envFloat(name string, fallback float64) (float64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number: %w", name, err)
	}
	return parsed, nil
}

func envCSV(name string, fallback string) []string {
	raw := envString(name, fallback)
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
