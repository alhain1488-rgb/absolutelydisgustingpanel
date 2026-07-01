// Package subscription builds a client's v2ray-style base64 subscription from
// all allowed and enabled inbounds across all servers, and renders QR codes.
package subscription

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/protocols"
)

// ErrNotFound is returned when the subscription token is unknown.
var ErrNotFound = errors.New("subscription not found")

// Builder assembles subscriptions from the database.
type Builder struct {
	db *sql.DB
}

// New builds a subscription Builder.
func New(db *sql.DB) *Builder { return &Builder{db: db} }

// clientRow holds the identity needed to build links.
type clientRow struct {
	id       int64
	name     string
	uuid     string
	password string
	enabled  bool
}

// BuildByToken returns the base64-encoded subscription for a token.
func (b *Builder) BuildByToken(ctx context.Context, token string) (string, error) {
	c, err := b.clientByToken(ctx, token)
	if err != nil {
		return "", err
	}
	links, err := b.linksFor(ctx, c)
	if err != nil {
		return "", err
	}
	joined := strings.Join(links, "\n")
	return base64.StdEncoding.EncodeToString([]byte(joined)), nil
}

// LinksByClientID returns the raw connection URIs for a client (admin view).
func (b *Builder) LinksByClientID(ctx context.Context, clientID int64) ([]string, error) {
	c, err := b.clientByID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return b.linksFor(ctx, c)
}

// QRPNG renders a QR code PNG for the given content.
func QRPNG(content string, size int) ([]byte, error) {
	if size <= 0 {
		size = 256
	}
	return qrcode.Encode(content, qrcode.Medium, size)
}

// linksFor builds the list of URIs for a client. A disabled client yields none.
func (b *Builder) linksFor(ctx context.Context, c *clientRow) ([]string, error) {
	if !c.enabled {
		return []string{}, nil
	}
	rows, err := b.db.QueryContext(ctx, `
		SELECT s.host, s.ip, i.tag, i.protocol, i.listen, i.port,
		       i.settings_json, i.stream_settings_json, i.sniffing_json, i.remark
		FROM client_inbounds ci
		JOIN inbounds i ON i.id = ci.inbound_id AND i.enabled = 1
		JOIN servers  s ON s.id = i.server_id
		WHERE ci.client_id = ? AND ci.enabled = 1
		ORDER BY s.id, i.id`, c.id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	client := protocols.Client{Name: c.name, UUID: c.uuid, Password: c.password}
	var links []string
	for rows.Next() {
		var host, ip, tag, protocol, listen, remark string
		var port int
		var settings, stream, sniffing string
		if err := rows.Scan(&host, &ip, &tag, &protocol, &listen, &port, &settings, &stream, &sniffing, &remark); err != nil {
			return nil, err
		}
		p, ok := protocols.Get(protocol)
		if !ok {
			continue // unknown protocol: skip rather than fail the whole subscription
		}
		in := protocols.Inbound{
			Tag: tag, Protocol: protocol, Listen: listen, Port: port, Remark: remark,
			Settings: json.RawMessage(settings), Stream: json.RawMessage(stream), Sniffing: json.RawMessage(sniffing),
		}
		link, err := p.BuildLink(protocols.Server{Address: address(host, ip)}, in, client)
		if err != nil {
			continue
		}
		links = append(links, link)
	}
	if links == nil {
		links = []string{}
	}
	return links, rows.Err()
}

func (b *Builder) clientByToken(ctx context.Context, token string) (*clientRow, error) {
	return b.scanClient(b.db.QueryRowContext(ctx,
		`SELECT id, name, uuid, password, enabled FROM clients WHERE subscription_token = ?`, token))
}

func (b *Builder) clientByID(ctx context.Context, id int64) (*clientRow, error) {
	return b.scanClient(b.db.QueryRowContext(ctx,
		`SELECT id, name, uuid, password, enabled FROM clients WHERE id = ?`, id))
}

func (b *Builder) scanClient(row *sql.Row) (*clientRow, error) {
	var c clientRow
	var enabled int
	err := row.Scan(&c.id, &c.name, &c.uuid, &c.password, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.enabled = enabled != 0
	return &c, nil
}

// address prefers the configured host, falling back to the resolved IP.
func address(host, ip string) string {
	if strings.TrimSpace(host) != "" {
		return host
	}
	return ip
}
