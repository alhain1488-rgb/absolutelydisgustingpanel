package ssh

import (
	"context"
	"errors"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh/sshtest"
)

func TestDetectXray(t *testing.T) {
	ctx := context.Background()

	active := sshtest.NewFakeRunner().On("is-active", sshtest.Response{Stdout: "active\n"})
	if st, _ := DetectXray(ctx, active, "xray"); st != XrayActive {
		t.Fatalf("expected active, got %q", st)
	}

	inactive := sshtest.NewFakeRunner().On("is-active", sshtest.Response{Stdout: "inactive\n"})
	if st, _ := DetectXray(ctx, inactive, "xray"); st != XrayInactive {
		t.Fatalf("expected inactive, got %q", st)
	}
}

func TestRestartXray_Error(t *testing.T) {
	ctx := context.Background()
	failing := sshtest.NewFakeRunner().On("systemctl restart", sshtest.Response{Err: errors.New("boom"), Stderr: "no perms"})
	if err := RestartXray(ctx, failing, "xray"); err == nil {
		t.Fatal("expected restart error")
	}

	ok := sshtest.NewFakeRunner().On("systemctl restart", sshtest.Response{})
	if err := RestartXray(ctx, ok, "xray"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Command should have been recorded with the service name.
	if len(ok.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(ok.Commands))
	}
}

func TestPing(t *testing.T) {
	ctx := context.Background()
	if err := Ping(ctx, sshtest.NewFakeRunner()); err != nil {
		t.Fatalf("ping ok: %v", err)
	}
	failing := &sshtest.FakeRunner{Default: sshtest.Response{Err: errors.New("unreachable")}}
	if err := Ping(ctx, failing); err == nil {
		t.Fatal("expected ping error")
	}
}
