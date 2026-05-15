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
}

func TestLoadParsesOverrides(t *testing.T) {
	t.Setenv("APP_NAME", "Custom Logs")
	t.Setenv("API_HOST", "127.0.0.1")
	t.Setenv("API_PORT", "18080")
	t.Setenv("BACKEND_CORS_ORIGINS", "http://localhost:3000, http://127.0.0.1:3000")
	t.Setenv("MESSAGE_EXPORT_MAX_ROWS", "25")

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
}

func TestLoadRejectsInvalidInteger(t *testing.T) {
	t.Setenv("API_PORT", "nope")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil")
	}
}
