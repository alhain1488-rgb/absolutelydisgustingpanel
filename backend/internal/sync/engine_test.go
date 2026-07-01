package sync

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/clients"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/crypto"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/inbounds"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/servers"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh/sshtest"
)

func setup(t *testing.T) (*Engine, *servers.Service, *sshtest.FakeRunner, int64) {
	t.Helper()
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	cipher, _ := crypto.NewCipher(make([]byte, 32))
	serversSvc := servers.NewService(servers.NewRepo(database), cipher, nil)
	fake := sshtest.NewFakeRunner()
	// xray -test passes and restart succeeds by default.
	serversSvc.SetRunnerFactory(func(ssh.Config) ssh.Runner { return fake })

	ctx := context.Background()
	srv, err := serversSvc.Create(ctx, servers.CreateInput{
		Name: "vps1", Host: "vpn.example.com", SSHUser: "root", SSHAuthMethod: "password", SSHSecret: "pw",
	})
	if err != nil {
		t.Fatalf("server: %v", err)
	}

	// One vless inbound + one client granted.
	inSvc := inbounds.NewService(inbounds.NewRepo(database))
	in, err := inSvc.Create(ctx, srv.ID, inbounds.Input{
		Tag: "reality", Protocol: "vless", Port: 443,
		Stream: json.RawMessage(`{"network":"tcp","security":"reality","reality":{"dest":"a:443","server_names":["a"]}}`),
	})
	if err != nil {
		t.Fatalf("inbound: %v", err)
	}
	cSvc := clients.NewService(clients.NewRepo(database))
	c, _ := cSvc.Create(ctx, "alice", "")
	if _, err := cSvc.SetInbounds(ctx, c.ID, []int64{in.ID}); err != nil {
		t.Fatalf("grant: %v", err)
	}

	return NewEngine(database, serversSvc), serversSvc, fake, srv.ID
}

func TestSyncServer_PushesAndIsIdempotent(t *testing.T) {
	engine, serversSvc, fake, serverID := setup(t)
	ctx := context.Background()

	res, err := engine.SyncServer(ctx, serverID)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !res.Changed || res.Skipped {
		t.Fatalf("expected changed push, got %+v", res)
	}
	// The config must have been uploaded and validated.
	if len(fake.Uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(fake.Uploads))
	}
	var sawTest, sawRestart bool
	for _, cmd := range fake.Commands {
		if strings.Contains(cmd, "xray -test") {
			sawTest = true
		}
		if strings.Contains(cmd, "systemctl restart") {
			sawRestart = true
		}
	}
	if !sawTest || !sawRestart {
		t.Fatalf("expected xray -test and restart, commands=%v", fake.Commands)
	}
	// last_sync_at recorded, no error.
	srv, _ := serversSvc.Repo().GetByID(ctx, serverID)
	if srv.LastSyncAt == "" || srv.LastSyncError != "" {
		t.Fatalf("expected clean last_sync, got at=%q err=%q", srv.LastSyncAt, srv.LastSyncError)
	}

	// Second sync with no changes → skipped, no new upload.
	res2, err := engine.SyncServer(ctx, serverID)
	if err != nil {
		t.Fatalf("sync2: %v", err)
	}
	if !res2.Skipped || res2.Changed {
		t.Fatalf("expected skipped second sync, got %+v", res2)
	}
	if len(fake.Uploads) != 1 {
		t.Fatalf("no new upload expected, got %d", len(fake.Uploads))
	}
}

func TestSyncServer_XrayTestFailsRestoresBackup(t *testing.T) {
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	cipher, _ := crypto.NewCipher(make([]byte, 32))
	serversSvc := servers.NewService(servers.NewRepo(database), cipher, nil)

	fake := sshtest.NewFakeRunner()
	fake.On("xray -test", sshtest.Response{Err: &ssh.ExitError{Code: 1, Stderr: "bad config"}, Stderr: "bad config"})
	serversSvc.SetRunnerFactory(func(ssh.Config) ssh.Runner { return fake })

	ctx := context.Background()
	srv, _ := serversSvc.Create(ctx, servers.CreateInput{Name: "s", Host: "h", SSHUser: "root", SSHAuthMethod: "password", SSHSecret: "pw"})
	inSvc := inbounds.NewService(inbounds.NewRepo(database))
	_, _ = inSvc.Create(ctx, srv.ID, inbounds.Input{Tag: "t", Protocol: "vmess", Port: 80})

	engine := NewEngine(database, serversSvc)
	res, err := engine.SyncServer(ctx, srv.ID)
	if err != nil {
		t.Fatalf("sync returned error: %v", err)
	}
	if res.Error == "" || res.Changed {
		t.Fatalf("expected failed sync, got %+v", res)
	}
	// A restore command should have run.
	var restored bool
	for _, cmd := range fake.Commands {
		if strings.Contains(cmd, ".bak") && strings.Contains(cmd, "cp") {
			restored = true
		}
	}
	if !restored {
		t.Fatalf("expected backup restore command, got %v", fake.Commands)
	}
	// Error recorded on the server.
	got, _ := serversSvc.Repo().GetByID(ctx, srv.ID)
	if got.LastSyncError == "" {
		t.Fatal("expected last_sync_error recorded")
	}
}

func TestBuildConfig_ContainsInbound(t *testing.T) {
	engine, _, _, serverID := setup(t)
	cfg, err := engine.BuildConfig(context.Background(), serverID)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(string(cfg), "\"protocol\": \"vless\"") {
		t.Fatalf("config missing vless inbound: %s", cfg)
	}
}
