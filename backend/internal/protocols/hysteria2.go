package protocols

import (
	"encoding/json"
	"errors"
)

// ErrNotImplemented is returned by protocols that are registered as future
// adapters but not usable in the MVP.
var ErrNotImplemented = errors.New("protocol not implemented in MVP")

// hysteria2 is a placeholder. Hysteria2 is a QUIC protocol whose server is NOT
// implemented in xray-core, so it cannot be an inbound in an xray config.json.
// It is registered so the shape is visible, but building anything fails
// explicitly. A real adapter would drive a separate hysteria/sing-box daemon on
// the node — see docs/QUESTIONS.md and docs/SPEC.md §5.
type hysteria2 struct{}

func (hysteria2) Name() string { return "hysteria2" }

func (hysteria2) BuildInbound(Inbound, []Client) (json.RawMessage, error) {
	return nil, ErrNotImplemented
}

func (hysteria2) BuildLink(Server, Inbound, Client) (string, error) {
	return "", ErrNotImplemented
}

// Note: intentionally NOT registered — an unimplemented protocol must not be
// selectable when creating inbounds. It exists to document the extension point.
