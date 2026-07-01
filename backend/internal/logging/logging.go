// Package logging provides a small wrapper around log/slog for structured,
// leveled logging across the backend.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON structured logger at the given level
// ("debug"|"info"|"warn"|"error"; unknown values default to info).
func New(level string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(h)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
