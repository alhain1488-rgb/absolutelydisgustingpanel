package inbounds

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/protocols"
)

func newService(t *testing.T) (*Service, int64) {
	t.Helper()
	database, err := db.InMemory(t.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	res, err := database.Exec(`INSERT INTO servers(name, host, ssh_user, ssh_auth_method) VALUES ('s','h','root','password')`)
	if err != nil {
		t.Fatalf("seed server: %v", err)
	}
	serverID, _ := res.LastInsertId()
	return NewService(NewRepo(database)), serverID
}

func TestCreate_RealityKeysGenerated(t *testing.T) {
	svc, serverID := newService(t)

	in := Input{
		Tag:      "reality",
		Protocol: "vless",
		Port:     443,
		Settings: json.RawMessage(`{"flow":"xtls-rprx-vision"}`),
		Stream:   json.RawMessage(`{"network":"tcp","security":"reality","reality":{"dest":"www.microsoft.com:443","server_names":["www.microsoft.com"]}}`),
	}
	got, err := svc.Create(context.Background(), serverID, in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	st := protocols.ParseStreamOrDefault(got.Stream)
	if st.Reality == nil || st.Reality.PrivateKey == "" || st.Reality.PublicKey == "" {
		t.Fatalf("reality keys not generated: %+v", st.Reality)
	}
	if len(st.Reality.ShortIDs) == 0 {
		t.Fatal("shortIds not generated")
	}
	if st.Reality.Fingerprint != "chrome" {
		t.Errorf("expected default fingerprint chrome, got %q", st.Reality.Fingerprint)
	}
}

func TestCreate_ShadowsocksPasswordGenerated(t *testing.T) {
	svc, serverID := newService(t)
	got, err := svc.Create(context.Background(), serverID, Input{
		Tag: "ss", Protocol: "shadowsocks", Port: 8388,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	set := protocols.ParseSettings(got.Settings)
	if set.Method != "aes-256-gcm" || set.Password == "" {
		t.Fatalf("ss method/password not filled: %+v", set)
	}
}

func TestCreate_UnknownProtocol(t *testing.T) {
	svc, serverID := newService(t)
	if _, err := svc.Create(context.Background(), serverID, Input{Tag: "x", Protocol: "hysteria2", Port: 443}); err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
	if _, err := svc.Create(context.Background(), serverID, Input{Tag: "x", Protocol: "vless", Port: 0}); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestUpdate_KeepsGeneratedKeys(t *testing.T) {
	svc, serverID := newService(t)
	created, err := svc.Create(context.Background(), serverID, Input{
		Tag: "reality", Protocol: "vless", Port: 443,
		Stream: json.RawMessage(`{"network":"tcp","security":"reality","reality":{"dest":"a:443","server_names":["a"]}}`),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := protocols.ParseStreamOrDefault(created.Stream)

	// Update, resending the existing reality block (keys included) → unchanged.
	updated, err := svc.Update(context.Background(), created.ID, Input{
		Tag: "reality2", Protocol: "vless", Port: 443, Stream: created.Stream,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	after := protocols.ParseStreamOrDefault(updated.Stream)
	if after.Reality.PrivateKey != before.Reality.PrivateKey {
		t.Error("reality private key should be preserved on update")
	}
	if updated.Tag != "reality2" {
		t.Error("tag not updated")
	}
}
