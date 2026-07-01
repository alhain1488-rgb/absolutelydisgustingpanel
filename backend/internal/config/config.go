// Package config reads and validates the panel configuration from the
// environment. Required variables that are missing cause a fatal, descriptive
// error at startup — the panel never invents secrets or falls back to defaults
// for anything security-sensitive.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// Config holds all runtime configuration for the backend.
type Config struct {
	// Secrets / identity.
	EncryptionKey []byte // 32 raw bytes, decoded from PANEL_ENCRYPTION_KEY (base64).
	JWTSecret     []byte // signing secret for JWTs.
	AdminUsername string
	AdminPassword string // plaintext bootstrap password; hashed on first start, never stored.
	Domain        string

	// Runtime.
	DBPath   string
	HTTPAddr string
	LogLevel string
}

// Load reads configuration from the environment and validates it.
func Load() (*Config, error) {
	var missing []string
	req := func(key string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	encKeyRaw := req("PANEL_ENCRYPTION_KEY")
	jwtSecret := req("PANEL_JWT_SECRET")
	adminUser := req("PANEL_ADMIN_USERNAME")
	adminPass := req("PANEL_ADMIN_PASSWORD")
	domain := req("PANEL_DOMAIN")
	dbPath := req("DB_PATH")

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	encKey, err := base64.StdEncoding.DecodeString(encKeyRaw)
	if err != nil {
		return nil, fmt.Errorf("PANEL_ENCRYPTION_KEY must be valid base64: %w", err)
	}
	if len(encKey) != 32 {
		return nil, fmt.Errorf("PANEL_ENCRYPTION_KEY must decode to 32 bytes (got %d); generate with: openssl rand -base64 32", len(encKey))
	}

	cfg := &Config{
		EncryptionKey: encKey,
		JWTSecret:     []byte(jwtSecret),
		AdminUsername: adminUser,
		AdminPassword: adminPass,
		Domain:        domain,
		DBPath:        dbPath,
		HTTPAddr:      envOr("PANEL_HTTP_ADDR", ":8080"),
		LogLevel:      envOr("LOG_LEVEL", "info"),
	}
	return cfg, nil
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
