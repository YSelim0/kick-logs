package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppName              string
	AppEnv               string
	LogLevel             string
	APIHost              string
	APIPort              int
	BackendCORSOrigins   []string
	KickPusherURL        string
	MessageExportMaxRows int
	SQLitePath           string
	ClickHouseAddr       string
	ClickHouseDatabase   string
	ClickHouseUsername   string
	ClickHousePassword   string
	ClickHouseDebug      bool
	DefaultAdminEmail    string
	DefaultAdminPassword string
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

	clickHouseDebug, err := envBool("CLICKHOUSE_DEBUG", false)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppName:              envString("APP_NAME", "Kick Logs"),
		AppEnv:               envString("APP_ENV", "local"),
		LogLevel:             envString("LOG_LEVEL", "INFO"),
		APIHost:              envString("API_HOST", "0.0.0.0"),
		APIPort:              apiPort,
		BackendCORSOrigins:   envCSV("BACKEND_CORS_ORIGINS", "http://localhost:3000"),
		KickPusherURL:        envString("KICK_PUSHER_URL", "wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679?protocol=7&client=js&version=8.4.0-rc2&flash=false"),
		MessageExportMaxRows: maxRows,
		SQLitePath:           envString("SQLITE_PATH", "var/kick-logs-go.sqlite3"),
		ClickHouseAddr:       envString("CLICKHOUSE_ADDR", "127.0.0.1:9000"),
		ClickHouseDatabase:   envString("CLICKHOUSE_DATABASE", "kick_logs"),
		ClickHouseUsername:   envString("CLICKHOUSE_USERNAME", "kick_logs"),
		ClickHousePassword:   envString("CLICKHOUSE_PASSWORD", "kick_logs"),
		ClickHouseDebug:      clickHouseDebug,
		DefaultAdminEmail:    envString("DEFAULT_SUPER_ADMIN_EMAIL", "admin@kicklogs.local"),
		DefaultAdminPassword: envString("DEFAULT_SUPER_ADMIN_PASSWORD", "admin123"),
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
