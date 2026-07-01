package protocols

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
)

func init() { register(trojan{}) }

// trojan implements the Trojan protocol (password-based, per client).
type trojan struct{}

func (trojan) Name() string { return "trojan" }

func (trojan) BuildInbound(in Inbound, clients []Client) (json.RawMessage, error) {
	cs := make([]map[string]any, 0, len(clients))
	for _, c := range clients {
		cs = append(cs, map[string]any{
			"password": c.Password,
			"email":    clientEmail(in.Tag, c),
		})
	}
	settings := map[string]any{"clients": cs}
	return marshal(baseInbound(in, "trojan", settings))
}

func (trojan) BuildLink(srv Server, in Inbound, c Client) (string, error) {
	if srv.Address == "" {
		return "", fmt.Errorf("server address required for link")
	}
	stream, _ := parseStream(in.Stream)
	if stream == nil {
		stream = &Stream{Network: "tcp", Security: "tls"}
	}
	q := stream.linkQuery()

	u := url.URL{
		Scheme:   "trojan",
		User:     url.User(c.Password),
		Host:     net.JoinHostPort(srv.Address, strconv.Itoa(in.Port)),
		RawQuery: q.Encode(),
		Fragment: linkRemark(in, c),
	}
	return u.String(), nil
}
