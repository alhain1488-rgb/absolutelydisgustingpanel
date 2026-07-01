package xrayconfig

import (
	"encoding/json"
	"testing"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/protocols"
)

func rawJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// sampleItems builds VLESS Reality + VMess + Trojan inbounds with a client each.
func sampleItems(t *testing.T) []InboundWithClients {
	t.Helper()
	kp, err := protocols.GenerateRealityKeypair()
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	sid, _ := protocols.GenerateShortID(8)

	reality := protocols.Inbound{
		Tag: "reality-in", Protocol: "vless", Port: 443,
		Settings: rawJSON(t, protocols.Settings{Flow: "xtls-rprx-vision"}),
		Stream: rawJSON(t, protocols.Stream{
			Network: "tcp", Security: "reality",
			Reality: &protocols.Reality{
				PrivateKey: kp.PrivateKey, PublicKey: kp.PublicKey,
				ShortIDs: []string{sid}, Dest: "www.microsoft.com:443",
				ServerNames: []string{"www.microsoft.com"}, Fingerprint: "chrome",
			},
		}),
	}
	vmess := protocols.Inbound{
		Tag: "vmess-in", Protocol: "vmess", Port: 10001,
		Stream: rawJSON(t, protocols.Stream{Network: "ws", Security: "none", WS: &protocols.WSParams{Path: "/ray"}}),
	}
	trojan := protocols.Inbound{
		Tag: "trojan-in", Protocol: "trojan", Port: 10002,
		Stream: rawJSON(t, protocols.Stream{Network: "tcp", Security: "none"}),
	}

	return []InboundWithClients{
		{Inbound: reality, Clients: []protocols.Client{{Name: "a", UUID: "11111111-1111-1111-1111-111111111111"}}},
		{Inbound: vmess, Clients: []protocols.Client{{Name: "b", UUID: "22222222-2222-2222-2222-222222222222"}}},
		{Inbound: trojan, Clients: []protocols.Client{{Name: "c", Password: "trojanpass"}}},
	}
}

func TestBuildConfig_Structure(t *testing.T) {
	raw, err := BuildConfig(sampleItems(t))
	if err != nil {
		t.Fatalf("build config: %v", err)
	}
	var cfg struct {
		Inbounds  []map[string]any `json:"inbounds"`
		Outbounds []map[string]any `json:"outbounds"`
		Log       map[string]any   `json:"log"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(cfg.Inbounds) != 3 {
		t.Fatalf("expected 3 inbounds, got %d", len(cfg.Inbounds))
	}
	if len(cfg.Outbounds) == 0 {
		t.Fatal("expected outbounds")
	}
	protos := map[string]bool{}
	for _, in := range cfg.Inbounds {
		protos[in["protocol"].(string)] = true
	}
	for _, want := range []string{"vless", "vmess", "trojan"} {
		if !protos[want] {
			t.Errorf("config missing protocol %q", want)
		}
	}
}

func TestBuildConfig_UnknownProtocol(t *testing.T) {
	_, err := BuildConfig([]InboundWithClients{
		{Inbound: protocols.Inbound{Tag: "x", Protocol: "nope", Port: 1}},
	})
	if err == nil {
		t.Fatal("expected error for unknown protocol")
	}
}
