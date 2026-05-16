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
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("CLICKHOUSE_ADDR", "")
	t.Setenv("CLICKHOUSE_DATABASE", "")
	t.Setenv("CLICKHOUSE_USERNAME", "")
	t.Setenv("CLICKHOUSE_PASSWORD", "")
	t.Setenv("CLICKHOUSE_DEBUG", "")
	t.Setenv("DEFAULT_SUPER_ADMIN_EMAIL", "")
	t.Setenv("DEFAULT_SUPER_ADMIN_PASSWORD", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
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
	t.Setenv("SQLITE_PATH", "tmp/app.sqlite3")
	t.Setenv("CLICKHOUSE_ADDR", "clickhouse:9000")
	t.Setenv("CLICKHOUSE_DATABASE", "custom_logs")
	t.Setenv("CLICKHOUSE_USERNAME", "custom_user")
	t.Setenv("CLICKHOUSE_PASSWORD", "secret")
	t.Setenv("CLICKHOUSE_DEBUG", "true")
	t.Setenv("DEFAULT_SUPER_ADMIN_EMAIL", "root@example.test")
	t.Setenv("DEFAULT_SUPER_ADMIN_PASSWORD", "local-password")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
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
