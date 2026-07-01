// Package xrayconfig assembles a complete Xray config.json from a set of
// inbounds and their granted clients, using the protocol registry. It is used
// by the sync engine to push configs and by integration tests that validate the
// result with `xray -test`.
package xrayconfig

import (
	"encoding/json"
	"fmt"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/protocols"
)

// InboundWithClients pairs an inbound with the clients granted access to it.
type InboundWithClients struct {
	Inbound protocols.Inbound
	Clients []protocols.Client
}

// BuildConfig returns a full Xray config.json for a server.
func BuildConfig(items []InboundWithClients) (json.RawMessage, error) {
	inbounds := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		p, ok := protocols.Get(item.Inbound.Protocol)
		if !ok {
			return nil, fmt.Errorf("unknown protocol %q for inbound %q", item.Inbound.Protocol, item.Inbound.Tag)
		}
		raw, err := p.BuildInbound(item.Inbound, item.Clients)
		if err != nil {
			return nil, fmt.Errorf("build inbound %q: %w", item.Inbound.Tag, err)
		}
		inbounds = append(inbounds, raw)
	}

	config := map[string]any{
		"log":      map[string]any{"loglevel": "warning"},
		"inbounds": inbounds,
		"outbounds": []map[string]any{
			{"protocol": "freedom", "tag": "direct"},
			{"protocol": "blackhole", "tag": "blocked"},
		},
	}
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}
	return out, nil
}
