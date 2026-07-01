package audit

import (
	"context"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
)

func TestRecordAndList(t *testing.T) {
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// An admin row is needed for the FK on audit_logs.admin_id.
	res, err := database.Exec(`INSERT INTO admins(username, password_hash) VALUES ('admin','x')`)
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	adminID, _ := res.LastInsertId()

	rec := New(database)
	ctx := context.Background()
	if err := rec.Record(ctx, adminID, "create", "server", 7, map[string]any{"name": "vps1"}, "1.2.3.4", "curl"); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := rec.Record(ctx, adminID, "delete", "server", 7, nil, "1.2.3.4", "curl"); err != nil {
		t.Fatalf("record 2: %v", err)
	}

	entries, err := rec.List(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Newest first.
	if entries[0].Action != "delete" {
		t.Fatalf("expected newest first, got %q", entries[0].Action)
	}
	if string(entries[1].Detail) == "" {
		t.Fatal("detail should be JSON, not empty")
	}
}
