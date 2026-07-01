package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/audit"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/auth"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/crypto"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/servers"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh/sshtest"
)

func routerWithServers(t *testing.T) (http.Handler, string) {
	t.Helper()
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	cipher, _ := crypto.NewCipher(make([]byte, 32))
	authSvc := auth.NewService(auth.NewRepo(database), auth.NewTokenManager([]byte("secret")), cipher, "Test")
	if err := authSvc.Bootstrap(context.Background(), "admin", "pw"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	serversSvc := servers.NewService(servers.NewRepo(database), cipher, nil)
	fake := sshtest.NewFakeRunner()
	fake.On("echo ok", sshtest.Response{Stdout: "ok\n"})
	serversSvc.SetRunnerFactory(func(ssh.Config) ssh.Runner { return fake })
	serversSvc.SetResolver(func(string) ([]string, error) { return []string{"198.51.100.7"}, nil })

	r := NewRouter(Deps{
		DB:           database,
		Auth:         authSvc,
		Audit:        audit.New(database),
		LoginLimiter: auth.NewRateLimiter(100, time.Minute),
		Servers:      serversSvc,
	})

	// Obtain a session token.
	res := doJSON(t, r, http.MethodPost, "/api/auth/login", "", map[string]string{"username": "admin", "password": "pw"})
	var body struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &body)
	return r, body.Token
}

func TestServersCRUD_RequiresAuth(t *testing.T) {
	r, _ := routerWithServers(t)
	rec := doJSON(t, r, http.MethodGet, "/api/servers", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestServersCRUD_Flow(t *testing.T) {
	r, token := routerWithServers(t)

	// Empty list.
	rec := doJSON(t, r, http.MethodGet, "/api/servers", token, nil)
	if rec.Code != http.StatusOK || rec.Body.String() == "null\n" {
		t.Fatalf("list: code=%d body=%s", rec.Code, rec.Body.String())
	}

	// Create.
	rec = doJSON(t, r, http.MethodPost, "/api/servers", token, map[string]any{
		"name": "vps1", "host": "example.com", "ssh_user": "root",
		"ssh_auth_method": "password", "ssh_secret": "pw",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var created servers.Server
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.ID == 0 {
		t.Fatal("expected created server id")
	}

	// Check (uses fake runner -> online).
	rec = doJSON(t, r, http.MethodPost, "/api/servers/1/check", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("check: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var checked servers.Server
	_ = json.Unmarshal(rec.Body.Bytes(), &checked)
	if checked.Status != servers.StatusOnline {
		t.Fatalf("expected online, got %q", checked.Status)
	}

	// Delete.
	rec = doJSON(t, r, http.MethodDelete, "/api/servers/1", token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: code=%d", rec.Code)
	}

	// Now 404.
	rec = doJSON(t, r, http.MethodGet, "/api/servers/1", token, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rec.Code)
	}
}
