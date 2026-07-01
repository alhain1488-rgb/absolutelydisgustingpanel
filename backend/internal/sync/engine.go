// Package sync builds the full config.json for a server from the database and
// pushes it over SSH: backup → write → xray -test → restart. It is idempotent:
// a config whose hash matches the last successful push is skipped.
package sync

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/protocols"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/servers"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/xrayconfig"
)

// Engine performs configuration synchronization to servers.
type Engine struct {
	db      *sql.DB
	servers *servers.Service
}

// NewEngine builds a sync Engine.
func NewEngine(db *sql.DB, serversSvc *servers.Service) *Engine {
	return &Engine{db: db, servers: serversSvc}
}

// Result reports the outcome of a sync.
type Result struct {
	ServerID int64  `json:"server_id"`
	Changed  bool   `json:"changed"`
	Skipped  bool   `json:"skipped"`
	Error    string `json:"error,omitempty"`
}

// BuildConfig assembles the config.json for a server from its enabled inbounds
// and the enabled clients granted to each.
func (e *Engine) BuildConfig(ctx context.Context, serverID int64) (json.RawMessage, error) {
	items, err := e.collectItems(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return xrayconfig.BuildConfig(items)
}

// SyncServer builds and pushes the config to a single server.
func (e *Engine) SyncServer(ctx context.Context, serverID int64) (*Result, error) {
	res := &Result{ServerID: serverID}

	srv, err := e.servers.Repo().GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	cfg, err := e.BuildConfig(ctx, serverID)
	if err != nil {
		return e.fail(ctx, res, err)
	}

	// Idempotency: skip when the config hash is unchanged.
	hash := hashConfig(cfg)
	prev, _ := e.getSetting(ctx, hashKey(serverID))
	if prev == hash {
		res.Skipped = true
		return res, nil
	}

	runner, err := e.servers.RunnerFor(srv)
	if err != nil {
		return e.fail(ctx, res, err)
	}

	if err := e.push(ctx, runner, srv, cfg); err != nil {
		return e.fail(ctx, res, err)
	}

	if err := e.setSetting(ctx, hashKey(serverID), hash); err != nil {
		return e.fail(ctx, res, err)
	}
	if err := e.servers.Repo().UpdateSyncResult(ctx, serverID, ""); err != nil {
		return nil, err
	}
	res.Changed = true
	return res, nil
}

// push performs backup → write → xray -test → restart, restoring on test failure.
func (e *Engine) push(ctx context.Context, runner ssh.Runner, srv *servers.Server, cfg json.RawMessage) error {
	path := srv.XrayConfigPath

	// Ensure directory and back up the current config if present.
	if _, _, err := runner.Run(ctx, fmt.Sprintf("mkdir -p \"$(dirname %s)\"", shellQuote(path))); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}
	if _, _, err := runner.Run(ctx, fmt.Sprintf("if [ -f %s ]; then cp %s %s.bak; fi", shellQuote(path), shellQuote(path), shellQuote(path))); err != nil {
		return fmt.Errorf("backup config: %w", err)
	}

	// Write the new config.
	if err := runner.Upload(ctx, path, cfg, 0o644); err != nil {
		return fmt.Errorf("upload config: %w", err)
	}

	// Validate; on failure, restore the backup and abort.
	if err := ssh.TestConfig(ctx, runner, path); err != nil {
		_, _, _ = runner.Run(ctx, fmt.Sprintf("if [ -f %s.bak ]; then cp %s.bak %s; fi", shellQuote(path), shellQuote(path), shellQuote(path)))
		return err
	}

	// Restart the service.
	if err := ssh.RestartXray(ctx, runner, srv.XrayServiceName); err != nil {
		return err
	}
	return nil
}

// collectItems gathers enabled inbounds and their enabled granted clients.
func (e *Engine) collectItems(ctx context.Context, serverID int64) ([]xrayconfig.InboundWithClients, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT id, tag, protocol, listen, port, settings_json, stream_settings_json, sniffing_json, remark
		FROM inbounds WHERE server_id = ? AND enabled = 1 ORDER BY id`, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	type inboundRow struct {
		id int64
		in protocols.Inbound
	}
	var inbounds []inboundRow
	for rows.Next() {
		var id, port int64
		var tag, protocol, listen, remark string
		var settings, stream, sniffing string
		if err := rows.Scan(&id, &tag, &protocol, &listen, &port, &settings, &stream, &sniffing, &remark); err != nil {
			return nil, err
		}
		inbounds = append(inbounds, inboundRow{
			id: id,
			in: protocols.Inbound{
				Tag: tag, Protocol: protocol, Listen: listen, Port: int(port), Remark: remark,
				Settings: json.RawMessage(settings), Stream: json.RawMessage(stream), Sniffing: json.RawMessage(sniffing),
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := make([]xrayconfig.InboundWithClients, 0, len(inbounds))
	for _, ib := range inbounds {
		grantees, err := e.clientsForInbound(ctx, ib.id)
		if err != nil {
			return nil, err
		}
		items = append(items, xrayconfig.InboundWithClients{Inbound: ib.in, Clients: grantees})
	}
	return items, nil
}

func (e *Engine) clientsForInbound(ctx context.Context, inboundID int64) ([]protocols.Client, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT c.name, c.uuid, c.password
		FROM client_inbounds ci
		JOIN clients c ON c.id = ci.client_id AND c.enabled = 1
		WHERE ci.inbound_id = ? AND ci.enabled = 1
		ORDER BY c.id`, inboundID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []protocols.Client
	for rows.Next() {
		var c protocols.Client
		if err := rows.Scan(&c.Name, &c.UUID, &c.Password); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (e *Engine) fail(ctx context.Context, res *Result, err error) (*Result, error) {
	res.Error = err.Error()
	_ = e.servers.Repo().UpdateSyncResult(ctx, res.ServerID, err.Error())
	return res, nil
}

func (e *Engine) getSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := e.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (e *Engine) setSetting(ctx context.Context, key, value string) error {
	_, err := e.db.ExecContext(ctx,
		`INSERT INTO settings(key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func hashConfig(cfg json.RawMessage) string {
	sum := sha256.Sum256(cfg)
	return hex.EncodeToString(sum[:])
}

func hashKey(serverID int64) string { return fmt.Sprintf("sync_hash_%d", serverID) }

// shellQuote single-quotes a string for safe shell embedding.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
