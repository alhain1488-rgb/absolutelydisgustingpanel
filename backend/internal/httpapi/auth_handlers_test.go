package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/audit"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/auth"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/crypto"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	cipher, _ := crypto.NewCipher(make([]byte, 32))
	repo := auth.NewRepo(database)
	svc := auth.NewService(repo, auth.NewTokenManager([]byte("secret")), cipher, "Test")
	if err := svc.Bootstrap(context.Background(), "admin", "pw"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	return NewRouter(Deps{
		DB:           database,
		Auth:         svc,
		Audit:        audit.New(database),
		LoginLimiter: auth.NewRateLimiter(100, time.Minute),
	})
}

func doJSON(t *testing.T, r http.Handler, method, path, bearer string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestLoginFlow_NoTOTP(t *testing.T) {
	r := newTestRouter(t)

	// Bad password -> 401.
	rec := doJSON(t, r, http.MethodPost, "/api/auth/login", "", map[string]string{"username": "admin", "password": "bad"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	// Good password -> token.
	rec = doJSON(t, r, http.MethodPost, "/api/auth/login", "", map[string]string{"username": "admin", "password": "pw"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var res struct {
		Token   string `json:"token"`
		Need2FA bool   `json:"need_2fa"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Token == "" || res.Need2FA {
		t.Fatalf("expected token without 2fa, got %+v", res)
	}

	// /api/auth/me with the token -> 200.
	rec = doJSON(t, r, http.MethodGet, "/api/auth/me", res.Token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("me: expected 200, got %d", rec.Code)
	}

	// /api/auth/me without token -> 401.
	rec = doJSON(t, r, http.MethodGet, "/api/auth/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me without token: expected 401, got %d", rec.Code)
	}
}

func TestSetup2FA_RequiresAuth(t *testing.T) {
	r := newTestRouter(t)
	rec := doJSON(t, r, http.MethodPost, "/api/auth/2fa/setup", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}
