package protocols

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
)

func init() { register(shadowsocks{}) }

// shadowsocks implements Shadowsocks. The inbound carries a single method +
// password (generated at inbound creation) shared by everyone granted access —
// classic SS has no per-user identity. This keeps configs valid for xray -test
// and links trivial to build; per-client separation is provided by the other
// protocols. See docs/QUESTIONS.md.
type shadowsocks struct{}

func (shadowsocks) Name() string { return "shadowsocks" }

const defaultSSMethod = "aes-256-gcm"

func (shadowsocks) BuildInbound(in Inbound, _ []Client) (json.RawMessage, error) {
	s := parseSettings(in.Settings)
	method := s.Method
	if method == "" {
		method = defaultSSMethod
	}
	settings := map[string]any{
		"method":   method,
		"password": s.Password,
		"network":  "tcp,udp",
	}
	return marshal(baseInbound(in, "shadowsocks", settings))
}

func (shadowsocks) BuildLink(srv Server, in Inbound, _ Client) (string, error) {
	if srv.Address == "" {
		return "", fmt.Errorf("server address required for link")
	}
	s := parseSettings(in.Settings)
	method := s.Method
	if method == "" {
		method = defaultSSMethod
	}
	userinfo := base64.RawURLEncoding.EncodeToString([]byte(method + ":" + s.Password))
	host := net.JoinHostPort(srv.Address, strconv.Itoa(in.Port))
	remark := in.Remark
	if remark == "" {
		remark = in.Tag
	}
	return fmt.Sprintf("ss://%s@%s#%s", userinfo, host, remark), nil
}
