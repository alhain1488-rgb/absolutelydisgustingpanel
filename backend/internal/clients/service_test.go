package clients

import (
	"context"
	"database/sql"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
)

func newService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return NewService(NewRepo(database)), database
}

func TestCreate_GeneratesIdentity(t *testing.T) {
	svc, _ := newService(t)
	c, err := svc.Create(context.Background(), "alice", "vip")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.UUID == "" || c.Password == "" || c.SubscriptionToken == "" {
		t.Fatalf("identity not generated: %+v", c)
	}
	if !c.Enabled {
		t.Error("new client should be enabled")
	}
}

func TestCreate_RequiresName(t *testing.T) {
	svc, _ := newService(t)
	if _, err := svc.Create(context.Background(), "  ", ""); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestRotateToken_Changes(t *testing.T) {
	svc, _ := newService(t)
	c, _ := svc.Create(context.Background(), "alice", "")
	old := c.SubscriptionToken
	rotated, err := svc.RotateToken(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if rotated.SubscriptionToken == old {
		t.Fatal("token should change on rotate")
	}
}

func TestSetInbounds_And_Grants(t *testing.T) {
	svc, h := newService(t)
	ctx := context.Background()
	// Seed a server and two inbounds.
	res, _ := h.Exec(`INSERT INTO servers(name, host, ssh_user, ssh_auth_method) VALUES ('s','h','root','password')`)
	serverID, _ := res.LastInsertId()
	r1, _ := h.Exec(`INSERT INTO inbounds(server_id, tag, protocol, port) VALUES (?, 'a', 'vless', 443)`, serverID)
	r2, _ := h.Exec(`INSERT INTO inbounds(server_id, tag, protocol, port) VALUES (?, 'b', 'vmess', 80)`, serverID)
	in1, _ := r1.LastInsertId()
	in2, _ := r2.LastInsertId()

	c, _ := svc.Create(ctx, "alice", "")
	updated, err := svc.SetInbounds(ctx, c.ID, []int64{in1, in2})
	if err != nil {
		t.Fatalf("set inbounds: %v", err)
	}
	if len(updated.InboundIDs) != 2 {
		t.Fatalf("expected 2 grants, got %v", updated.InboundIDs)
	}

	// Replace with just one.
	updated, _ = svc.SetInbounds(ctx, c.ID, []int64{in2})
	if len(updated.InboundIDs) != 1 || updated.InboundIDs[0] != in2 {
		t.Fatalf("expected only in2, got %v", updated.InboundIDs)
	}
}

func TestEnableDisable(t *testing.T) {
	svc, _ := newService(t)
	c, _ := svc.Create(context.Background(), "alice", "")
	if err := svc.SetEnabled(context.Background(), c.ID, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	got, _ := svc.Repo().GetByID(context.Background(), c.ID)
	if got.Enabled {
		t.Fatal("client should be disabled")
	}
}
