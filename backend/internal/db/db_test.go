package db

import (
	"path/filepath"
	"testing"
)

func TestOpen_AppliesMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	// All core tables should exist.
	for _, table := range []string{"admins", "servers", "inbounds", "clients", "client_inbounds", "audit_logs", "settings", "schema_migrations"} {
		var name string
		err := database.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %q to exist: %v", table, err)
		}
	}
}

func TestOpen_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "panel.db")
	d1, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	d1.Close()

	// Re-opening applies no migrations again and still works.
	d2, err := Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer d2.Close()

	var count int
	if err := d2.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 applied migration, got %d", count)
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	// Inserting an inbound referencing a missing server must fail.
	_, err = database.Exec(`INSERT INTO inbounds(server_id, tag, protocol, port) VALUES (999, 'x', 'vless', 443)`)
	if err == nil {
		t.Fatal("expected foreign key violation")
	}
}
