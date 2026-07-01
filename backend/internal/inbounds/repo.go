// Package inbounds implements CRUD for inbounds within a server. Protocol and
// transport parameters live in JSON columns; REALITY keys and Shadowsocks
// passwords are generated on the backend at creation time.
package inbounds

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// ErrNotFound is returned when an inbound does not exist.
var ErrNotFound = errors.New("inbound not found")

// Inbound is a listening endpoint on a server.
type Inbound struct {
	ID        int64           `json:"id"`
	ServerID  int64           `json:"server_id"`
	Tag       string          `json:"tag"`
	Protocol  string          `json:"protocol"`
	Listen    string          `json:"listen"`
	Port      int             `json:"port"`
	Settings  json.RawMessage `json:"settings_json"`
	Stream    json.RawMessage `json:"stream_settings_json"`
	Sniffing  json.RawMessage `json:"sniffing_json"`
	Remark    string          `json:"remark"`
	Enabled   bool            `json:"enabled"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

// Repo is the inbound data-access layer.
type Repo struct {
	db *sql.DB
}

// NewRepo builds an inbound repository.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

const cols = `id, server_id, tag, protocol, listen, port, settings_json,
	stream_settings_json, sniffing_json, remark, enabled, created_at, updated_at`

// Create inserts an inbound and returns it.
func (r *Repo) Create(ctx context.Context, in *Inbound) (*Inbound, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO inbounds(server_id, tag, protocol, listen, port, settings_json,
			stream_settings_json, sniffing_json, remark, enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.ServerID, in.Tag, in.Protocol, in.Listen, in.Port, jsonOr(in.Settings),
		jsonOr(in.Stream), jsonOr(in.Sniffing), in.Remark, boolToInt(in.Enabled))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

// GetByID loads an inbound.
func (r *Repo) GetByID(ctx context.Context, id int64) (*Inbound, error) {
	return scanOne(r.db.QueryRowContext(ctx, `SELECT `+cols+` FROM inbounds WHERE id = ?`, id))
}

// ListByServer returns inbounds of a server.
func (r *Repo) ListByServer(ctx context.Context, serverID int64) ([]Inbound, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+cols+` FROM inbounds WHERE server_id = ? ORDER BY id`, serverID)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

// Update changes editable fields.
func (r *Repo) Update(ctx context.Context, in *Inbound) (*Inbound, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE inbounds SET tag=?, protocol=?, listen=?, port=?, settings_json=?,
			stream_settings_json=?, sniffing_json=?, remark=?, enabled=?,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		 WHERE id=?`,
		in.Tag, in.Protocol, in.Listen, in.Port, jsonOr(in.Settings),
		jsonOr(in.Stream), jsonOr(in.Sniffing), in.Remark, boolToInt(in.Enabled), in.ID)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, in.ID)
}

// Delete removes an inbound.
func (r *Repo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM inbounds WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanOne(row *sql.Row) (*Inbound, error) {
	in, err := scan(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return in, err
}

func collect(rows *sql.Rows) ([]Inbound, error) {
	defer func() { _ = rows.Close() }()
	var out []Inbound
	for rows.Next() {
		in, err := scan(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *in)
	}
	return out, rows.Err()
}

// scan reads a row via the provided Scan func (works for *sql.Row and *sql.Rows).
func scan(scanFn func(dest ...any) error) (*Inbound, error) {
	var in Inbound
	var enabled int
	var settings, stream, sniffing string
	err := scanFn(&in.ID, &in.ServerID, &in.Tag, &in.Protocol, &in.Listen, &in.Port,
		&settings, &stream, &sniffing, &in.Remark, &enabled, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		return nil, err
	}
	in.Enabled = enabled != 0
	in.Settings = json.RawMessage(settings)
	in.Stream = json.RawMessage(stream)
	in.Sniffing = json.RawMessage(sniffing)
	return &in, nil
}

func jsonOr(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "{}"
	}
	return string(raw)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
