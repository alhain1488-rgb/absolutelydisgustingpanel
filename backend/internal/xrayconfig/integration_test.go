//go:build integration

// This test validates that a generated config is accepted by a real Xray
// binary. It is excluded from the default suite (build tag "integration")
// because it needs the xray binary. Run it with:
//
//	XRAY_BIN=/usr/local/bin/xray go test -tags integration ./internal/xrayconfig/...
//
// or via the dockerized node in test/xray-node (see its README).
package xrayconfig

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGeneratedConfig_AcceptedByXray(t *testing.T) {
	bin := os.Getenv("XRAY_BIN")
	if bin == "" {
		bin = "xray"
	}
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("xray binary %q not found; skipping (set XRAY_BIN)", bin)
	}

	raw, err := BuildConfig(sampleItems(t))
	if err != nil {
		t.Fatalf("build config: %v", err)
	}

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	out, err := exec.Command(bin, "-test", "-config", path).CombinedOutput()
	if err != nil {
		t.Fatalf("xray -test rejected the config: %v\n%s\n--- config ---\n%s", err, out, raw)
	}
	t.Logf("xray -test accepted config:\n%s", out)
}
