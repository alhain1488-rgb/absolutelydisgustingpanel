package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func validEnv(t *testing.T) {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("PANEL_ENCRYPTION_KEY", key)
	t.Setenv("PANEL_JWT_SECRET", "jwt-secret")
	t.Setenv("PANEL_ADMIN_USERNAME", "admin")
	t.Setenv("PANEL_ADMIN_PASSWORD", "s3cret")
	t.Setenv("PANEL_DOMAIN", "panel.example.com")
	t.Setenv("DB_PATH", "/data/panel.db")
}

func TestLoad_OK(t *testing.T) {
	validEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.EncryptionKey) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(cfg.EncryptionKey))
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("expected default HTTPAddr, got %q", cfg.HTTPAddr)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default LogLevel, got %q", cfg.LogLevel)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	// No env set at all.
	t.Setenv("PANEL_ENCRYPTION_KEY", "")
	t.Setenv("PANEL_JWT_SECRET", "")
	t.Setenv("PANEL_ADMIN_USERNAME", "")
	t.Setenv("PANEL_ADMIN_PASSWORD", "")
	t.Setenv("PANEL_DOMAIN", "")
	t.Setenv("DB_PATH", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing env")
	}
	if !strings.Contains(err.Error(), "PANEL_ENCRYPTION_KEY") {
		t.Fatalf("error should list missing keys, got: %v", err)
	}
}

func TestLoad_BadKeyLength(t *testing.T) {
	validEnv(t)
	t.Setenv("PANEL_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 16)))
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "32 bytes") {
		t.Fatalf("expected 32-byte length error, got: %v", err)
	}
}

func TestLoad_BadKeyBase64(t *testing.T) {
	validEnv(t)
	t.Setenv("PANEL_ENCRYPTION_KEY", "not!base64!")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "base64") {
		t.Fatalf("expected base64 error, got: %v", err)
	}
}

func TestLoad_Overrides(t *testing.T) {
	validEnv(t)
	t.Setenv("PANEL_HTTP_ADDR", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPAddr != ":9090" || cfg.LogLevel != "debug" {
		t.Fatalf("overrides not applied: %+v", cfg)
	}
}
