package protocols

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
)

func init() { register(vmess{}) }

// vmess implements VMess. Share links use the base64-encoded JSON format used
// by v2rayN and compatible clients.
type vmess struct{}

func (vmess) Name() string { return "vmess" }

func (vmess) BuildInbound(in Inbound, clients []Client) (json.RawMessage, error) {
	cs := make([]map[string]any, 0, len(clients))
	for _, c := range clients {
		cs = append(cs, map[string]any{
			"id":      c.UUID,
			"email":   clientEmail(in.Tag, c),
			"alterId": 0,
		})
	}
	settings := map[string]any{"clients": cs}
	return marshal(baseInbound(in, "vmess", settings))
}

func (vmess) BuildLink(srv Server, in Inbound, c Client) (string, error) {
	stream, _ := parseStream(in.Stream)
	if stream == nil {
		stream = &Stream{Network: "tcp", Security: "none"}
	}

	// Standard vmess:// JSON payload (version 2).
	payload := map[string]string{
		"v":    "2",
		"ps":   linkRemark(in, c),
		"add":  srv.Address,
		"port": strconv.Itoa(in.Port),
		"id":   c.UUID,
		"aid":  "0",
		"scy":  "auto",
		"net":  stream.Network,
		"type": "none",
		"tls":  tlsField(stream.Security),
	}
	switch stream.Network {
	case "ws":
		if stream.WS != nil {
			payload["path"] = stream.WS.Path
			payload["host"] = stream.WS.Host
		}
	case "grpc":
		if stream.GRPC != nil {
			payload["path"] = stream.GRPC.ServiceName
		}
	}
	if stream.Security == "tls" && stream.TLS != nil && stream.TLS.ServerName != "" {
		payload["sni"] = stream.TLS.ServerName
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(b), nil
}

func tlsField(security string) string {
	if security == "tls" || security == "reality" {
		return "tls"
	}
	return ""
}
