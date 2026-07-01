// Package protocols is the single place where knowledge about a proxy protocol
// lives. Each protocol implements the Protocol interface (BuildInbound +
// BuildLink) and registers itself. Adding a protocol never requires a DB schema
// change: protocol-specific parameters travel in the inbound's JSON columns.
package protocols

import (
	"encoding/json"
	"fmt"
)

// Inbound is the protocol-agnostic view of an inbound as the registry needs it.
// Settings/Stream are our own parameter schema (see settings.go / stream.go),
// not raw xray JSON — the adapter translates them into a valid xray object.
type Inbound struct {
	Tag      string
	Protocol string
	Listen   string
	Port     int
	Remark   string
	Settings json.RawMessage // maps to inbounds.settings_json
	Stream   json.RawMessage // maps to inbounds.stream_settings_json
	Sniffing json.RawMessage // maps to inbounds.sniffing_json
}

// Client is a single user's credentials.
type Client struct {
	Name     string
	UUID     string
	Password string
}

// Server carries the connection address used when building share links.
type Server struct {
	Address string // public host or IP clients connect to
}

// Protocol is implemented by every supported proxy protocol.
type Protocol interface {
	Name() string
	// BuildInbound returns the full xray inbound object for config.json, with
	// the granted clients injected.
	BuildInbound(in Inbound, clients []Client) (json.RawMessage, error)
	// BuildLink returns a subscription URI (vless://…, trojan://…, …) for one
	// client on this inbound.
	BuildLink(srv Server, in Inbound, c Client) (string, error)
}

// registry holds the known protocols by name.
var registry = map[string]Protocol{}

// register adds a protocol to the registry (called from init()).
func register(p Protocol) { registry[p.Name()] = p }

// Get returns the protocol adapter by name.
func Get(name string) (Protocol, bool) {
	p, ok := registry[name]
	return p, ok
}

// Names lists the registered protocol names.
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	return out
}

// baseInbound assembles the common inbound skeleton shared by all protocols.
func baseInbound(in Inbound, protocol string, settings map[string]any) map[string]any {
	listen := in.Listen
	if listen == "" {
		listen = "0.0.0.0"
	}
	obj := map[string]any{
		"tag":      in.Tag,
		"listen":   listen,
		"port":     in.Port,
		"protocol": protocol,
		"settings": settings,
	}
	if stream, err := parseStream(in.Stream); err == nil && stream != nil {
		obj["streamSettings"] = stream.xrayStreamSettings()
	}
	if len(in.Sniffing) > 0 && string(in.Sniffing) != "null" && string(in.Sniffing) != "{}" {
		var sn any
		if err := json.Unmarshal(in.Sniffing, &sn); err == nil {
			obj["sniffing"] = sn
		}
	} else {
		// Sensible default: sniff tls/http for routing.
		obj["sniffing"] = map[string]any{"enabled": true, "destOverride": []string{"http", "tls"}}
	}
	return obj
}

func marshal(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal inbound: %w", err)
	}
	return b, nil
}
