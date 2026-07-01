package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
)

func TestHealthz(t *testing.T) {
	r := NewRouter(Deps{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz: got %d want 200", rec.Code)
	}
}

func TestReadyz_WithDB(t *testing.T) {
	database, err := db.InMemory("ready")
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer database.Close()

	r := NewRouter(Deps{DB: database})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz: got %d want 200", rec.Code)
	}
}
