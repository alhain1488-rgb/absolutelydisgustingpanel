package protocols

import "encoding/json"

// Settings is our parameter schema for protocol-level (non-transport) options,
// persisted in inbounds.settings_json.
type Settings struct {
	Flow     string `json:"flow,omitempty"`     // vless flow, e.g. xtls-rprx-vision
	Method   string `json:"method,omitempty"`   // shadowsocks cipher
	Password string `json:"password,omitempty"` // shadowsocks inbound password
}

func parseSettings(raw json.RawMessage) Settings {
	var s Settings
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

// ParseSettings is the exported form used by other packages to read the
// settings JSON blob into the shared schema.
func ParseSettings(raw json.RawMessage) Settings { return parseSettings(raw) }
