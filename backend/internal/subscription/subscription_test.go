package subscription

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/clients"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/inbounds"
)

// e2eSetup seeds two servers each with one inbound, a client granted both.
func e2eSetup(t *testing.T) (*sql.DB, *clients.Service, *clients.Client) {
	t.Helper()
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	s1 := seedServer(t, database, "vpn1.example.com")
	s2 := seedServer(t, database, "vpn2.example.com")

	inSvc := inbounds.NewService(inbounds.NewRepo(database))
	ctx := context.Background()
	reality, err := inSvc.Create(ctx, s1, inbounds.Input{
		Tag: "reality", Protocol: "vless", Port: 443,
		Settings: json.RawMessage(`{"flow":"xtls-rprx-vision"}`),
		Stream:   json.RawMessage(`{"network":"tcp","security":"reality","reality":{"dest":"a:443","server_names":["a"]}}`),
	})
	if err != nil {
		t.Fatalf("inbound1: %v", err)
	}
	trojan, err := inSvc.Create(ctx, s2, inbounds.Input{
		Tag: "trojan", Protocol: "trojan", Port: 8443,
		Stream: json.RawMessage(`{"network":"tcp","security":"tls","tls":{"server_name":"vpn2.example.com"}}`),
	})
	if err != nil {
		t.Fatalf("inbound2: %v", err)
	}

	cSvc := clients.NewService(clients.NewRepo(database))
	client, err := cSvc.Create(ctx, "alice", "")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if _, err := cSvc.SetInbounds(ctx, client.ID, []int64{reality.ID, trojan.ID}); err != nil {
		t.Fatalf("grant: %v", err)
	}
	return database, cSvc, client
}

func seedServer(t *testing.T, database *sql.DB, host string) int64 {
	t.Helper()
	res, err := database.Exec(`INSERT INTO servers(name, host, ssh_user, ssh_auth_method) VALUES (?, ?, 'root', 'password')`, host, host)
	if err != nil {
		t.Fatalf("seed server: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func decodeSub(t *testing.T, encoded string) []string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode sub: %v", err)
	}
	if len(raw) == 0 {
		return nil
	}
	return strings.Split(string(raw), "\n")
}

func TestSubscription_E2E(t *testing.T) {
	database, cSvc, client := e2eSetup(t)
	b := New(database)
	ctx := context.Background()

	// Both inbounds present.
	sub, err := b.BuildByToken(ctx, client.SubscriptionToken)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	links := decodeSub(t, sub)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d: %v", len(links), links)
	}
	var haveVless, haveTrojan bool
	for _, l := range links {
		if strings.HasPrefix(l, "vless://") && strings.Contains(l, "vpn1.example.com:443") {
			haveVless = true
		}
		if strings.HasPrefix(l, "trojan://") && strings.Contains(l, "vpn2.example.com:8443") {
			haveTrojan = true
		}
	}
	if !haveVless || !haveTrojan {
		t.Fatalf("expected both vless and trojan links, got %v", links)
	}

	// Disabling the client empties the subscription.
	if err := cSvc.SetEnabled(ctx, client.ID, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	sub, _ = b.BuildByToken(ctx, client.SubscriptionToken)
	if links := decodeSub(t, sub); len(links) != 0 {
		t.Fatalf("disabled client should have empty sub, got %v", links)
	}

	// Re-enable and revoke one grant → one link.
	_ = cSvc.SetEnabled(ctx, client.ID, true)
	full, _ := cSvc.Repo().GetByID(ctx, client.ID)
	_, _ = cSvc.SetInbounds(ctx, client.ID, full.InboundIDs[:1])
	sub, _ = b.BuildByToken(ctx, client.SubscriptionToken)
	if links := decodeSub(t, sub); len(links) != 1 {
		t.Fatalf("expected 1 link after revoke, got %v", links)
	}
}

func TestSubscription_UnknownToken(t *testing.T) {
	database, _, _ := e2eSetup(t)
	if _, err := New(database).BuildByToken(context.Background(), "nope"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLinksByClientID(t *testing.T) {
	database, _, client := e2eSetup(t)
	links, err := New(database).LinksByClientID(context.Background(), client.ID)
	if err != nil {
		t.Fatalf("links: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
}

func TestQRPNG(t *testing.T) {
	png, err := QRPNG("https://panel.example.com/sub/abc", 256)
	if err != nil {
		t.Fatalf("qr: %v", err)
	}
	if len(png) == 0 || string(png[1:4]) != "PNG" {
		t.Fatalf("expected PNG output, got %d bytes", len(png))
	}
}
