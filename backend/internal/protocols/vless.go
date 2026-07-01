package protocols

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
)

func init() { register(vless{}) }

// vless implements VLESS, covering both REALITY and TLS via streamSettings.
type vless struct{}

func (vless) Name() string { return "vless" }

func (vless) BuildInbound(in Inbound, clients []Client) (json.RawMessage, error) {
	s := parseSettings(in.Settings)
	cs := make([]map[string]any, 0, len(clients))
	for _, c := range clients {
		entry := map[string]any{"id": c.UUID, "email": clientEmail(in.Tag, c)}
		if s.Flow != "" {
			entry["flow"] = s.Flow
		}
		cs = append(cs, entry)
	}
	settings := map[string]any{
		"clients":    cs,
		"decryption": "none",
	}
	return marshal(baseInbound(in, "vless", settings))
}

func (vless) BuildLink(srv Server, in Inbound, c Client) (string, error) {
	stream, _ := parseStream(in.Stream)
	if stream == nil {
		stream = &Stream{Network: "tcp", Security: "none"}
	}
	s := parseSettings(in.Settings)
	q := stream.linkQuery()
	q.Set("encryption", "none")
	if s.Flow != "" {
		q.Set("flow", s.Flow)
	}

	u := url.URL{
		Scheme:   "vless",
		User:     url.User(c.UUID),
		Host:     net.JoinHostPort(srv.Address, strconv.Itoa(in.Port)),
		RawQuery: q.Encode(),
		Fragment: linkRemark(in, c),
	}
	if srv.Address == "" {
		return "", fmt.Errorf("server address required for link")
	}
	return u.String(), nil
}
