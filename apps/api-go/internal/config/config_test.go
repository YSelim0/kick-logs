package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("APP_NAME", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("API_HOST", "")
	t.Setenv("API_PORT", "")
	t.Setenv("BACKEND_CORS_ORIGINS", "")
	t.Setenv("MESSAGE_EXPORT_MAX_ROWS", "")
	t.Setenv("JWT_SECRET_KEY", "")
	t.Setenv("JWT_ALGORITHM", "")
	t.Setenv("JWT_EXPIRES_MINUTES", "")
	t.Setenv("JWT_COOKIE_NAME", "")
	t.Setenv("JWT_COOKIE_SECURE", "")
	t.Setenv("JWT_COOKIE_SAMESITE", "")
	t.Setenv("SEED_SUPER_ADMIN_ON_STARTUP", "")
	t.Setenv("LISTENER_HEARTBEAT_STALE_AFTER_SECONDS", "")
	t.Setenv("LISTENER_RECENT_MESSAGE_POLL_ENABLED", "")
	t.Setenv("LISTENER_RECENT_MESSAGE_POLL_INTERVAL_SECONDS", "")
	t.Setenv("LISTENER_RECENT_MESSAGE_POLL_CONCURRENCY", "")
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("CLICKHOUSE_ADDR", "")
	t.Setenv("CLICKHOUSE_DATABASE", "")
	t.Setenv("CLICKHOUSE_USERNAME", "")
	t.Setenv("CLICKHOUSE_PASSWORD", "")
	t.Setenv("CLICKHOUSE_DEBUG", "")
	t.Setenv("DEFAULT_SUPER_ADMIN_EMAIL", "")
	t.Setenv("DEFAULT_SUPER_ADMIN_PASSWORD", "")
	t.Setenv("RATE_LIMIT_ENABLED", "")
	t.Setenv("RATE_LIMIT_STORE_MAX_KEYS", "")
	t.Setenv("RATE_LIMIT_TRUST_PROXY", "")
	t.Setenv("RATE_LIMIT_CLIENT_IP_HEADER", "")
	t.Setenv("KICK_CLIENT_ID", "")
	t.Setenv("KICK_CLIENT_SECRET", "")
	t.Setenv("KICK_API_BASE_URL", "")
	t.Setenv("KICK_OAUTH_TOKEN_URL", "")
	t.Setenv("KICK_WEBHOOK_PUBLIC_KEY", "")
	t.Setenv("KICK_WEBHOOK_SYNC_ENABLED", "")
	t.Setenv("KICK_WEBHOOK_EVENTS", "")
	t.Setenv("KICK_WEBHOOK_PROCESS_BATCH_SIZE", "")
	t.Setenv("KICK_WEBHOOK_PROCESS_MAX_ATTEMPTS", "")
	t.Setenv("NATS_URL", "")
	t.Setenv("NATS_RAW_EVENT_STREAM", "")
	t.Setenv("NATS_RAW_EVENT_SUBJECT", "")
	t.Setenv("NATS_RAW_EVENT_CONSUMER", "")
	t.Setenv("NATS_RAW_EVENT_ACK_WAIT_SECONDS", "")
	t.Setenv("NATS_RAW_EVENT_FETCH_BATCH_SIZE", "")
	t.Setenv("NATS_RAW_EVENT_FETCH_TIMEOUT_SECONDS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.RateLimitEnabled {
		t.Fatal("RateLimitEnabled = false")
	}
	if cfg.RateLimitStoreMaxKeys != 65536 {
		t.Fatalf("RateLimitStoreMaxKeys = %d", cfg.RateLimitStoreMaxKeys)
	}
	if !cfg.RateLimitTrustProxy {
		t.Fatal("RateLimitTrustProxy = false")
	}
	if cfg.RateLimitClientIPHeader != "CF-Connecting-IP" {
		t.Fatalf("RateLimitClientIPHeader = %q", cfg.RateLimitClientIPHeader)
	}

	if cfg.AppName != "Kick Logs" {
		t.Fatalf("AppName = %q", cfg.AppName)
	}
	if cfg.APIAddress() != "0.0.0.0:8000" {
		t.Fatalf("APIAddress() = %q", cfg.APIAddress())
	}
	if len(cfg.BackendCORSOrigins) != 1 || cfg.BackendCORSOrigins[0] != "http://localhost:3000" {
		t.Fatalf("BackendCORSOrigins = %#v", cfg.BackendCORSOrigins)
	}
	if cfg.MessageExportMaxRows != 1000 {
		t.Fatalf("MessageExportMaxRows = %d", cfg.MessageExportMaxRows)
	}
	if cfg.JWTSecretKey != "change-me-for-local-development-secret-key" {
		t.Fatalf("JWTSecretKey = %q", cfg.JWTSecretKey)
	}
	if cfg.JWTExpiresMinutes != 10080 {
		t.Fatalf("JWTExpiresMinutes = %d", cfg.JWTExpiresMinutes)
	}
	if cfg.JWTCookieName != "kick_logs_session" {
		t.Fatalf("JWTCookieName = %q", cfg.JWTCookieName)
	}
	if cfg.JWTCookieSecure {
		t.Fatal("JWTCookieSecure = true")
	}
	if !cfg.SeedSuperAdmin {
		t.Fatal("SeedSuperAdmin = false")
	}
	if cfg.ListenerStaleAfter != 45 {
		t.Fatalf("ListenerStaleAfter = %d", cfg.ListenerStaleAfter)
	}
	if !cfg.ListenerRecentMessagePollEnabled {
		t.Fatal("ListenerRecentMessagePollEnabled = false")
	}
	if cfg.ListenerRecentMessagePollInterval != 10 {
		t.Fatalf("ListenerRecentMessagePollInterval = %f", cfg.ListenerRecentMessagePollInterval)
	}
	if cfg.ListenerRecentMessagePollConcurrency != 8 {
		t.Fatalf("ListenerRecentMessagePollConcurrency = %d", cfg.ListenerRecentMessagePollConcurrency)
	}
	if cfg.SQLitePath != "var/kick-logs-go.sqlite3" {
		t.Fatalf("SQLitePath = %q", cfg.SQLitePath)
	}
	if cfg.ClickHouseAddr != "127.0.0.1:9000" {
		t.Fatalf("ClickHouseAddr = %q", cfg.ClickHouseAddr)
	}
	if cfg.ClickHouseDatabase != "kick_logs" {
		t.Fatalf("ClickHouseDatabase = %q", cfg.ClickHouseDatabase)
	}
	if cfg.DefaultAdminEmail != "admin@kicklogs.local" {
		t.Fatalf("DefaultAdminEmail = %q", cfg.DefaultAdminEmail)
	}

	if cfg.KickAPIBaseURL != "https://api.kick.com" {
		t.Fatalf("KickAPIBaseURL = %q", cfg.KickAPIBaseURL)
	}
	if cfg.KickOAuthTokenURL != "https://id.kick.com/oauth/token" {
		t.Fatalf("KickOAuthTokenURL = %q", cfg.KickOAuthTokenURL)
	}
	if !cfg.KickWebhookSyncEnabled {
		t.Fatal("KickWebhookSyncEnabled = false")
	}
	if len(cfg.KickWebhookEvents) != 3 {
		t.Fatalf("KickWebhookEvents = %v", cfg.KickWebhookEvents)
	}
	if cfg.KickWebhookProcessBatchSize != 50 {
		t.Fatalf("KickWebhookProcessBatchSize = %d", cfg.KickWebhookProcessBatchSize)
	}
	if cfg.KickWebhookProcessMaxAttempts != 5 {
		t.Fatalf("KickWebhookProcessMaxAttempts = %d", cfg.KickWebhookProcessMaxAttempts)
	}
	if cfg.NATSURL != "nats://127.0.0.1:4222" {
		t.Fatalf("NATSURL = %q", cfg.NATSURL)
	}
	if cfg.NATSRawEventStream != "KICK_RAW_EVENTS" {
		t.Fatalf("NATSRawEventStream = %q", cfg.NATSRawEventStream)
	}
	if cfg.NATSRawEventSubject != "kick.raw.chat" {
		t.Fatalf("NATSRawEventSubject = %q", cfg.NATSRawEventSubject)
	}
	if cfg.NATSRawEventConsumer != "kick-raw-event-processor" {
		t.Fatalf("NATSRawEventConsumer = %q", cfg.NATSRawEventConsumer)
	}
	if cfg.NATSRawEventAckWaitSeconds != 60 {
		t.Fatalf("NATSRawEventAckWaitSeconds = %d", cfg.NATSRawEventAckWaitSeconds)
	}
	if cfg.NATSRawEventFetchBatchSize != 500 {
		t.Fatalf("NATSRawEventFetchBatchSize = %d", cfg.NATSRawEventFetchBatchSize)
	}
	if cfg.NATSRawEventFetchTimeoutSeconds != 2 {
		t.Fatalf("NATSRawEventFetchTimeoutSeconds = %d", cfg.NATSRawEventFetchTimeoutSeconds)
	}
}

func TestLoadParsesOverrides(t *testing.T) {
	t.Setenv("APP_NAME", "Custom Logs")
	t.Setenv("API_HOST", "127.0.0.1")
	t.Setenv("API_PORT", "18080")
	t.Setenv("BACKEND_CORS_ORIGINS", "http://localhost:3000, http://127.0.0.1:3000")
	t.Setenv("MESSAGE_EXPORT_MAX_ROWS", "25")
	t.Setenv("JWT_SECRET_KEY", "custom-secret")
	t.Setenv("JWT_EXPIRES_MINUTES", "15")
	t.Setenv("JWT_COOKIE_SECURE", "true")
	t.Setenv("SEED_SUPER_ADMIN_ON_STARTUP", "false")
	t.Setenv("LISTENER_HEARTBEAT_STALE_AFTER_SECONDS", "99")
	t.Setenv("LISTENER_RECENT_MESSAGE_POLL_ENABLED", "false")
	t.Setenv("LISTENER_RECENT_MESSAGE_POLL_INTERVAL_SECONDS", "3.5")
	t.Setenv("LISTENER_RECENT_MESSAGE_POLL_CONCURRENCY", "3")
	t.Setenv("SQLITE_PATH", "tmp/app.sqlite3")
	t.Setenv("CLICKHOUSE_ADDR", "clickhouse:9000")
	t.Setenv("CLICKHOUSE_DATABASE", "custom_logs")
	t.Setenv("CLICKHOUSE_USERNAME", "custom_user")
	t.Setenv("CLICKHOUSE_PASSWORD", "secret")
	t.Setenv("CLICKHOUSE_DEBUG", "true")
	t.Setenv("DEFAULT_SUPER_ADMIN_EMAIL", "root@example.test")
	t.Setenv("DEFAULT_SUPER_ADMIN_PASSWORD", "local-password")
	t.Setenv("RATE_LIMIT_ENABLED", "false")
	t.Setenv("RATE_LIMIT_STORE_MAX_KEYS", "1234")
	t.Setenv("RATE_LIMIT_TRUST_PROXY", "false")
	t.Setenv("RATE_LIMIT_CLIENT_IP_HEADER", "X-Real-IP")
	t.Setenv("NATS_URL", "nats://nats:4222")
	t.Setenv("NATS_RAW_EVENT_STREAM", "CUSTOM_RAW_EVENTS")
	t.Setenv("NATS_RAW_EVENT_SUBJECT", "custom.raw.chat")
	t.Setenv("NATS_RAW_EVENT_CONSUMER", "custom-processor")
	t.Setenv("NATS_RAW_EVENT_ACK_WAIT_SECONDS", "90")
	t.Setenv("NATS_RAW_EVENT_FETCH_BATCH_SIZE", "250")
	t.Setenv("NATS_RAW_EVENT_FETCH_TIMEOUT_SECONDS", "5")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RateLimitEnabled {
		t.Fatal("RateLimitEnabled = true")
	}
	if cfg.RateLimitStoreMaxKeys != 1234 {
		t.Fatalf("RateLimitStoreMaxKeys = %d", cfg.RateLimitStoreMaxKeys)
	}
	if cfg.RateLimitTrustProxy {
		t.Fatal("RateLimitTrustProxy = true")
	}
	if cfg.RateLimitClientIPHeader != "X-Real-IP" {
		t.Fatalf("RateLimitClientIPHeader = %q", cfg.RateLimitClientIPHeader)
	}

	if cfg.AppName != "Custom Logs" {
		t.Fatalf("AppName = %q", cfg.AppName)
	}
	if cfg.APIAddress() != "127.0.0.1:18080" {
		t.Fatalf("APIAddress() = %q", cfg.APIAddress())
	}
	if len(cfg.BackendCORSOrigins) != 2 {
		t.Fatalf("BackendCORSOrigins = %#v", cfg.BackendCORSOrigins)
	}
	if cfg.MessageExportMaxRows != 25 {
		t.Fatalf("MessageExportMaxRows = %d", cfg.MessageExportMaxRows)
	}
	if cfg.JWTSecretKey != "custom-secret" {
		t.Fatalf("JWTSecretKey = %q", cfg.JWTSecretKey)
	}
	if cfg.JWTExpiresMinutes != 15 {
		t.Fatalf("JWTExpiresMinutes = %d", cfg.JWTExpiresMinutes)
	}
	if !cfg.JWTCookieSecure {
		t.Fatal("JWTCookieSecure = false")
	}
	if cfg.SeedSuperAdmin {
		t.Fatal("SeedSuperAdmin = true")
	}
	if cfg.ListenerStaleAfter != 99 {
		t.Fatalf("ListenerStaleAfter = %d", cfg.ListenerStaleAfter)
	}
	if cfg.ListenerRecentMessagePollEnabled {
		t.Fatal("ListenerRecentMessagePollEnabled = true")
	}
	if cfg.ListenerRecentMessagePollInterval != 3.5 {
		t.Fatalf("ListenerRecentMessagePollInterval = %f", cfg.ListenerRecentMessagePollInterval)
	}
	if cfg.ListenerRecentMessagePollConcurrency != 3 {
		t.Fatalf("ListenerRecentMessagePollConcurrency = %d", cfg.ListenerRecentMessagePollConcurrency)
	}
	if cfg.SQLitePath != "tmp/app.sqlite3" {
		t.Fatalf("SQLitePath = %q", cfg.SQLitePath)
	}
	if cfg.ClickHouseAddr != "clickhouse:9000" {
		t.Fatalf("ClickHouseAddr = %q", cfg.ClickHouseAddr)
	}
	if cfg.ClickHouseDatabase != "custom_logs" {
		t.Fatalf("ClickHouseDatabase = %q", cfg.ClickHouseDatabase)
	}
	if cfg.ClickHouseUsername != "custom_user" {
		t.Fatalf("ClickHouseUsername = %q", cfg.ClickHouseUsername)
	}
	if cfg.ClickHousePassword != "secret" {
		t.Fatalf("ClickHousePassword = %q", cfg.ClickHousePassword)
	}
	if !cfg.ClickHouseDebug {
		t.Fatal("ClickHouseDebug = false")
	}
	if cfg.DefaultAdminEmail != "root@example.test" {
		t.Fatalf("DefaultAdminEmail = %q", cfg.DefaultAdminEmail)
	}
	if cfg.NATSURL != "nats://nats:4222" {
		t.Fatalf("NATSURL = %q", cfg.NATSURL)
	}
	if cfg.NATSRawEventStream != "CUSTOM_RAW_EVENTS" {
		t.Fatalf("NATSRawEventStream = %q", cfg.NATSRawEventStream)
	}
	if cfg.NATSRawEventSubject != "custom.raw.chat" {
		t.Fatalf("NATSRawEventSubject = %q", cfg.NATSRawEventSubject)
	}
	if cfg.NATSRawEventConsumer != "custom-processor" {
		t.Fatalf("NATSRawEventConsumer = %q", cfg.NATSRawEventConsumer)
	}
	if cfg.NATSRawEventAckWaitSeconds != 90 {
		t.Fatalf("NATSRawEventAckWaitSeconds = %d", cfg.NATSRawEventAckWaitSeconds)
	}
	if cfg.NATSRawEventFetchBatchSize != 250 {
		t.Fatalf("NATSRawEventFetchBatchSize = %d", cfg.NATSRawEventFetchBatchSize)
	}
	if cfg.NATSRawEventFetchTimeoutSeconds != 5 {
		t.Fatalf("NATSRawEventFetchTimeoutSeconds = %d", cfg.NATSRawEventFetchTimeoutSeconds)
	}
}

func TestLoadRejectsInvalidInteger(t *testing.T) {
	t.Setenv("API_PORT", "nope")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil")
	}
}

func TestLoadRejectsInvalidBoolean(t *testing.T) {
	t.Setenv("JWT_COOKIE_SECURE", "maybe")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil")
	}
}
