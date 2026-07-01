package servers

import (
	"context"
	"errors"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/crypto"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh/sshtest"
)

func newService(t *testing.T) (*Service, *sshtest.FakeRunner) {
	t.Helper()
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	cipher, _ := crypto.NewCipher(make([]byte, 32))
	svc := NewService(NewRepo(database), cipher, nil)

	fake := sshtest.NewFakeRunner()
	svc.SetRunnerFactory(func(ssh.Config) ssh.Runner { return fake })
	svc.SetResolver(func(string) ([]string, error) { return []string{"203.0.113.9"}, nil })
	return svc, fake
}

func seedServer(t *testing.T, svc *Service) *Server {
	t.Helper()
	srv, err := svc.Create(context.Background(), CreateInput{
		Name:          "vps1",
		Host:          "example.com",
		SSHUser:       "root",
		SSHAuthMethod: "password",
		SSHSecret:     "hunter2",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return srv
}

func TestCreate_EncryptsSecret(t *testing.T) {
	svc, _ := newService(t)
	srv := seedServer(t, svc)

	if srv.SSHPort != 22 {
		t.Errorf("expected default port 22, got %d", srv.SSHPort)
	}
	// Stored secret must be encrypted, not plaintext.
	stored, err := svc.Repo().GetByID(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.SSHSecretEnc == "" || stored.SSHSecretEnc == "hunter2" {
		t.Fatalf("secret should be encrypted, got %q", stored.SSHSecretEnc)
	}
	dec, err := svc.decrypt(stored.SSHSecretEnc)
	if err != nil || dec != "hunter2" {
		t.Fatalf("decrypt mismatch: %q err=%v", dec, err)
	}
}

func TestCreate_Validation(t *testing.T) {
	svc, _ := newService(t)
	_, err := svc.Create(context.Background(), CreateInput{Name: "", Host: "h", SSHUser: "u", SSHAuthMethod: "password"})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
	_, err = svc.Create(context.Background(), CreateInput{Name: "n", Host: "h", SSHUser: "u", SSHAuthMethod: "bogus"})
	if err == nil {
		t.Fatal("expected validation error for bad auth method")
	}
}

func TestCheck_OnlineAndGeo(t *testing.T) {
	svc, fake := newService(t)
	srv := seedServer(t, svc)

	fake.On("echo ok", sshtest.Response{Stdout: "ok\n"})
	fake.On("is-active", sshtest.Response{Stdout: "active\n"})
	svc.geo = stubGeo{country: "Neverland", city: "Capital", asn: "AS123"}

	got, err := svc.Check(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got.Status != StatusOnline {
		t.Errorf("status: got %q want online", got.Status)
	}
	if got.IP != "203.0.113.9" {
		t.Errorf("ip: got %q", got.IP)
	}
	if got.GeoCountry != "Neverland" {
		t.Errorf("geo country: got %q", got.GeoCountry)
	}
	if got.LastCheckAt == "" {
		t.Error("last_check_at should be set")
	}
}

func TestCheck_Offline(t *testing.T) {
	svc, fake := newService(t)
	srv := seedServer(t, svc)
	fake.Default = sshtest.Response{Err: errors.New("connection refused")}

	got, err := svc.Check(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got.Status != StatusOffline {
		t.Errorf("status: got %q want offline", got.Status)
	}
}

func TestStatsAndRestart(t *testing.T) {
	svc, fake := newService(t)
	srv := seedServer(t, svc)

	fake.On("/proc/meminfo", sshtest.Response{Stdout: "###MEM###\nMemTotal: 1024000 kB\nMemAvailable: 512000 kB\n###UP###\n10.0 5.0\n"})
	st, err := svc.Stats(context.Background(), srv.ID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if st.MemPercent != 50 {
		t.Errorf("mem percent: got %v", st.MemPercent)
	}

	fake.On("systemctl restart", sshtest.Response{})
	if err := svc.RestartXray(context.Background(), srv.ID); err != nil {
		t.Fatalf("restart: %v", err)
	}
}

func TestUpdate_KeepsSecretWhenEmpty(t *testing.T) {
	svc, _ := newService(t)
	srv := seedServer(t, svc)
	before, _ := svc.Repo().GetByID(context.Background(), srv.ID)

	_, err := svc.Update(context.Background(), srv.ID, CreateInput{
		Name: "renamed", Host: "example.com", SSHUser: "root", SSHAuthMethod: "password",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	after, _ := svc.Repo().GetByID(context.Background(), srv.ID)
	if after.Name != "renamed" {
		t.Errorf("name not updated")
	}
	if after.SSHSecretEnc != before.SSHSecretEnc {
		t.Errorf("secret should be unchanged when not provided")
	}
}

type stubGeo struct{ country, city, asn string }

func (s stubGeo) Lookup(context.Context, string) (Geo, error) {
	return Geo{Country: s.country, City: s.city, ASN: s.asn}, nil
}
