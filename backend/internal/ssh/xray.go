package ssh

import (
	"context"
	"fmt"
	"strings"
)

// XrayStatus is the detected state of the Xray service on a node.
type XrayStatus string

const (
	XrayActive   XrayStatus = "active"
	XrayInactive XrayStatus = "inactive"
	XrayUnknown  XrayStatus = "unknown"
)

// DetectXray reports whether the given systemd service is active.
func DetectXray(ctx context.Context, r Runner, serviceName string) (XrayStatus, error) {
	if serviceName == "" {
		serviceName = "xray"
	}
	// `systemctl is-active` exits non-zero when not active; tolerate that.
	out, _, _ := r.Run(ctx, fmt.Sprintf("systemctl is-active %s 2>/dev/null || true", shellQuote(serviceName)))
	switch strings.TrimSpace(out) {
	case "active":
		return XrayActive, nil
	case "inactive", "failed", "deactivating", "unknown", "":
		return XrayInactive, nil
	default:
		return XrayInactive, nil
	}
}

// RestartXray restarts the Xray service and validates the config first.
func RestartXray(ctx context.Context, r Runner, serviceName string) error {
	if serviceName == "" {
		serviceName = "xray"
	}
	_, stderr, err := r.Run(ctx, fmt.Sprintf("systemctl restart %s", shellQuote(serviceName)))
	if err != nil {
		return fmt.Errorf("restart %s: %w (%s)", serviceName, err, stderr)
	}
	return nil
}

// TestConfig runs `xray -test` against a config file path on the node.
func TestConfig(ctx context.Context, r Runner, configPath string) error {
	out, stderr, err := r.Run(ctx, fmt.Sprintf("xray -test -config %s", shellQuote(configPath)))
	if err != nil {
		return fmt.Errorf("xray -test failed: %w (%s%s)", err, out, stderr)
	}
	return nil
}

// Ping runs a trivial command to confirm reachability.
func Ping(ctx context.Context, r Runner) error {
	_, _, err := r.Run(ctx, "echo ok")
	return err
}

// shellQuote single-quotes a string for safe embedding in a shell command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
