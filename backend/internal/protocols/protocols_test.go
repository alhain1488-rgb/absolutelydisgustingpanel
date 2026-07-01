package protocols

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestRegistry_HasMVPProtocols(t *testing.T) {
	for _, name := range []string{"vless", "vmess", "trojan", "shadowsocks"} {
		if _, ok := Get(name); !ok {
			t.Errorf("protocol %q not registered", name)
		}
	}
	// hysteria2 must NOT be registered (not selectable in MVP).
	if _, ok := Get("hysteria2"); ok {
		t.Error("hysteria2 should not be registered in MVP")
	}
}

func TestVLESS_RealityInboundAndLink(t *testing.T) {
	kp, err := GenerateRealityKeypair()
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	sid, _ := GenerateShortID(8)
	in := Inbound{
		Tag:      "reality-in",
		Protocol: "vless",
		Port:     443,
		Settings: mustJSON(t, Settings{Flow: "xtls-rprx-vision"}),
		Stream: mustJSON(t, Stream{
			Network:  "tcp",
			Security: "reality",
			Reality: &Reality{
				PrivateKey:  kp.PrivateKey,
				PublicKey:   kp.PublicKey,
				ShortIDs:    []string{sid},
				Dest:        "www.microsoft.com:443",
				ServerNames: []string{"www.microsoft.com"},
				Fingerprint: "chrome",
			},
		}),
	}
	client := Client{Name: "alice", UUID: "11111111-1111-1111-1111-111111111111"}

	p, _ := Get("vless")
	raw, err := p.BuildInbound(in, []Client{client})
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("unmarshal inbound: %v", err)
	}
	if obj["protocol"] != "vless" {
		t.Errorf("protocol: %v", obj["protocol"])
	}
	stream := obj["streamSettings"].(map[string]any)
	if stream["security"] != "reality" {
		t.Errorf("expected reality security, got %v", stream["security"])
	}
	rs := stream["realitySettings"].(map[string]any)
	if rs["privateKey"] != kp.PrivateKey {
		t.Error("private key not embedded")
	}
	// publicKey must NOT leak into the xray config.
	if _, bad := rs["publicKey"]; bad {
		t.Error("publicKey must not be in realitySettings")
	}

	link, err := p.BuildLink(Server{Address: "vpn.example.com"}, in, client)
	if err != nil {
		t.Fatalf("build link: %v", err)
	}
	if !strings.HasPrefix(link, "vless://11111111-1111-1111-1111-111111111111@vpn.example.com:443") {
		t.Errorf("unexpected link: %s", link)
	}
	for _, want := range []string{"security=reality", "pbk=" + kp.PublicKey, "sid=" + sid, "flow=xtls-rprx-vision", "sni=www.microsoft.com"} {
		if !strings.Contains(link, want) {
			t.Errorf("link missing %q: %s", want, link)
		}
	}
}

func TestTrojan_PerClientPassword(t *testing.T) {
	in := Inbound{
		Tag: "trojan-in", Protocol: "trojan", Port: 443,
		Stream: mustJSON(t, Stream{Network: "tcp", Security: "tls", TLS: &TLSParams{ServerName: "t.example.com"}}),
	}
	c := Client{Name: "bob", Password: "s3cr3t"}
	p, _ := Get("trojan")

	raw, err := p.BuildInbound(in, []Client{c})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var obj map[string]any
	_ = json.Unmarshal(raw, &obj)
	clients := obj["settings"].(map[string]any)["clients"].([]any)
	if len(clients) != 1 || clients[0].(map[string]any)["password"] != "s3cr3t" {
		t.Fatalf("client password not embedded: %v", clients)
	}

	link, err := p.BuildLink(Server{Address: "t.example.com"}, in, c)
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	if !strings.HasPrefix(link, "trojan://s3cr3t@t.example.com:443") || !strings.Contains(link, "sni=t.example.com") {
		t.Errorf("unexpected trojan link: %s", link)
	}
}

func TestVMess_Base64JSONLink(t *testing.T) {
	in := Inbound{Tag: "vmess-in", Protocol: "vmess", Port: 80, Stream: mustJSON(t, Stream{Network: "ws", Security: "none", WS: &WSParams{Path: "/ray"}})}
	c := Client{Name: "carol", UUID: "22222222-2222-2222-2222-222222222222"}
	p, _ := Get("vmess")

	link, err := p.BuildLink(Server{Address: "v.example.com"}, in, c)
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	if !strings.HasPrefix(link, "vmess://") {
		t.Fatalf("expected vmess:// link, got %s", link)
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(link, "vmess://"))
	if err != nil {
		t.Fatalf("decode vmess payload: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(payload, &m); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if m["add"] != "v.example.com" || m["id"] != c.UUID || m["net"] != "ws" || m["path"] != "/ray" {
		t.Errorf("unexpected vmess payload: %v", m)
	}
}

func TestShadowsocks_SharedPassword(t *testing.T) {
	pw, _ := GenerateShadowsocksPassword()
	in := Inbound{
		Tag: "ss-in", Protocol: "shadowsocks", Port: 8388,
		Settings: mustJSON(t, Settings{Method: "aes-256-gcm", Password: pw}),
	}
	p, _ := Get("shadowsocks")
	raw, err := p.BuildInbound(in, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var obj map[string]any
	_ = json.Unmarshal(raw, &obj)
	settings := obj["settings"].(map[string]any)
	if settings["method"] != "aes-256-gcm" || settings["password"] != pw {
		t.Errorf("ss settings wrong: %v", settings)
	}

	link, err := p.BuildLink(Server{Address: "s.example.com"}, in, Client{})
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	if !strings.HasPrefix(link, "ss://") || !strings.Contains(link, "@s.example.com:8388") {
		t.Errorf("unexpected ss link: %s", link)
	}
}

func TestReality_KeypairValid(t *testing.T) {
	kp, err := GenerateRealityKeypair()
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	priv, err := base64.RawURLEncoding.DecodeString(kp.PrivateKey)
	if err != nil || len(priv) != 32 {
		t.Fatalf("private key invalid: len=%d err=%v", len(priv), err)
	}
	pub, err := base64.RawURLEncoding.DecodeString(kp.PublicKey)
	if err != nil || len(pub) != 32 {
		t.Fatalf("public key invalid: len=%d err=%v", len(pub), err)
	}
}

func TestHysteria2_NotImplemented(t *testing.T) {
	h := hysteria2{}
	if _, err := h.BuildInbound(Inbound{}, nil); err != ErrNotImplemented {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}
