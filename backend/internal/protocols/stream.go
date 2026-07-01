package protocols

import (
	"encoding/json"
	"net/url"
	"strings"
)

// Stream is our parameter schema for transport + security, persisted in
// inbounds.stream_settings_json. The adapter converts it into a valid xray
// streamSettings object and into share-link query parameters.
type Stream struct {
	Network  string      `json:"network"`  // tcp | ws | grpc | http
	Security string      `json:"security"` // none | tls | reality
	Reality  *Reality    `json:"reality,omitempty"`
	TLS      *TLSParams  `json:"tls,omitempty"`
	WS       *WSParams   `json:"ws,omitempty"`
	GRPC     *GRPCParams `json:"grpc,omitempty"`
}

// Reality holds REALITY parameters. PrivateKey lives on the server config;
// PublicKey is kept for building client links (xray does not read it on the
// inbound, so it is stripped from the emitted config).
type Reality struct {
	PrivateKey  string   `json:"private_key"`
	PublicKey   string   `json:"public_key"`
	ShortIDs    []string `json:"short_ids"`
	Dest        string   `json:"dest"`         // e.g. "example.com:443"
	ServerNames []string `json:"server_names"` // SNI allow-list
	Fingerprint string   `json:"fingerprint"`  // uTLS fingerprint, e.g. "chrome"
}

// TLSParams holds standard TLS parameters.
type TLSParams struct {
	ServerName  string   `json:"server_name"`
	ALPN        []string `json:"alpn,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"`
	// CertificateFile/KeyFile point to files already present on the node.
	CertificateFile string `json:"certificate_file,omitempty"`
	KeyFile         string `json:"key_file,omitempty"`
}

// WSParams holds WebSocket transport parameters.
type WSParams struct {
	Path string `json:"path"`
	Host string `json:"host,omitempty"`
}

// GRPCParams holds gRPC transport parameters.
type GRPCParams struct {
	ServiceName string `json:"service_name"`
}

func parseStream(raw json.RawMessage) (*Stream, error) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return nil, nil
	}
	var s Stream
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	if s.Network == "" {
		s.Network = "tcp"
	}
	if s.Security == "" {
		s.Security = "none"
	}
	return &s, nil
}

// ParseStreamOrDefault parses the stream JSON, returning a tcp/none default
// when empty. Exported for other packages that fill in generated key material.
func ParseStreamOrDefault(raw json.RawMessage) *Stream {
	s, err := parseStream(raw)
	if err != nil || s == nil {
		return &Stream{Network: "tcp", Security: "none"}
	}
	return s
}

// xrayStreamSettings renders the xray streamSettings object.
func (s *Stream) xrayStreamSettings() map[string]any {
	out := map[string]any{
		"network":  s.Network,
		"security": s.Security,
	}

	switch s.Security {
	case "reality":
		if s.Reality != nil {
			rs := map[string]any{
				"show":        false,
				"dest":        s.Reality.Dest,
				"serverNames": s.Reality.ServerNames,
				"privateKey":  s.Reality.PrivateKey,
				"shortIds":    s.Reality.ShortIDs,
			}
			out["realitySettings"] = rs
		}
	case "tls":
		if s.TLS != nil {
			ts := map[string]any{}
			if s.TLS.ServerName != "" {
				ts["serverName"] = s.TLS.ServerName
			}
			if len(s.TLS.ALPN) > 0 {
				ts["alpn"] = s.TLS.ALPN
			}
			if s.TLS.CertificateFile != "" {
				ts["certificates"] = []map[string]any{{
					"certificateFile": s.TLS.CertificateFile,
					"keyFile":         s.TLS.KeyFile,
				}}
			}
			out["tlsSettings"] = ts
		}
	}

	switch s.Network {
	case "ws":
		if s.WS != nil {
			ws := map[string]any{"path": s.WS.Path}
			if s.WS.Host != "" {
				ws["headers"] = map[string]any{"Host": s.WS.Host}
			}
			out["wsSettings"] = ws
		}
	case "grpc":
		if s.GRPC != nil {
			out["grpcSettings"] = map[string]any{"serviceName": s.GRPC.ServiceName}
		}
	}
	return out
}

// linkQuery builds the query parameters shared by VLESS/Trojan share links.
func (s *Stream) linkQuery() url.Values {
	q := url.Values{}
	network := s.Network
	if network == "" {
		network = "tcp"
	}
	q.Set("type", network)
	q.Set("security", s.Security)

	switch s.Security {
	case "reality":
		if s.Reality != nil {
			q.Set("pbk", s.Reality.PublicKey)
			if len(s.Reality.ShortIDs) > 0 {
				q.Set("sid", s.Reality.ShortIDs[0])
			}
			if len(s.Reality.ServerNames) > 0 {
				q.Set("sni", s.Reality.ServerNames[0])
			}
			if s.Reality.Fingerprint != "" {
				q.Set("fp", s.Reality.Fingerprint)
			}
		}
	case "tls":
		if s.TLS != nil {
			if s.TLS.ServerName != "" {
				q.Set("sni", s.TLS.ServerName)
			}
			if len(s.TLS.ALPN) > 0 {
				q.Set("alpn", strings.Join(s.TLS.ALPN, ","))
			}
			if s.TLS.Fingerprint != "" {
				q.Set("fp", s.TLS.Fingerprint)
			}
		}
	}

	switch network {
	case "ws":
		if s.WS != nil {
			if s.WS.Path != "" {
				q.Set("path", s.WS.Path)
			}
			if s.WS.Host != "" {
				q.Set("host", s.WS.Host)
			}
		}
	case "grpc":
		if s.GRPC != nil {
			q.Set("serviceName", s.GRPC.ServiceName)
		}
	}
	return q
}
