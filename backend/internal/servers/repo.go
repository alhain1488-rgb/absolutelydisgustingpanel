// Package servers implements CRUD and SSH-backed operations for managed VPS
// nodes: reachability checks, IP/geo resolution, Xray status and restart, and
// resource metrics. All remote work goes through the ssh.Runner interface.
package servers

import (
	"context"
	"database/sql"
	"errors"
)

// Status values for a server.
const (
	StatusUnknown = "unknown"
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusError   = "error"
)

// ErrNotFound is returned when a server does not exist.
var ErrNotFound = errors.New("server not found")

// Server is a managed VPS node. SSH secrets are stored encrypted and never
// serialized to JSON.
type Server struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Host            string `json:"host"`
	SSHPort         int    `json:"ssh_port"`
	SSHUser         string `json:"ssh_user"`
	SSHAuthMethod   string `json:"ssh_auth_method"`
	SSHSecretEnc    string `json:"-"`
	SSHPassphrase   string `json:"-"`
	XrayConfigPath  string `json:"xray_config_path"`
	XrayServiceName string `json:"xray_service_name"`
	IP              string `json:"ip"`
	GeoCountry      string `json:"geo_country"`
	GeoCity         string `json:"geo_city"`
	GeoASN          string `json:"geo_asn"`
	Status          string `json:"status"`
	LastCheckAt     string `json:"last_check_at,omitempty"`
	LastSyncAt      string `json:"last_sync_at,omitempty"`
	LastSyncError   string `json:"last_sync_error"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// Repo is the server data-access layer.
type Repo struct {
	db *sql.DB
}

// NewRepo builds a server repository.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

const serverCols = `id, name, host, ssh_port, ssh_user, ssh_auth_method, ssh_secret_enc,
	ssh_passphrase_enc, xray_config_path, xray_service_name, ip, geo_country, geo_city,
	geo_asn, status, last_check_at, last_sync_at, last_sync_error, created_at, updated_at`

// Create inserts a server and returns it.
func (r *Repo) Create(ctx context.Context, s *Server) (*Server, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO servers(name, host, ssh_port, ssh_user, ssh_auth_method, ssh_secret_enc,
			ssh_passphrase_enc, xray_config_path, xray_service_name)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.Name, s.Host, s.SSHPort, s.SSHUser, s.SSHAuthMethod, s.SSHSecretEnc,
		s.SSHPassphrase, s.XrayConfigPath, s.XrayServiceName)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

// GetByID loads a server.
func (r *Repo) GetByID(ctx context.Context, id int64) (*Server, error) {
	return r.scanOne(r.db.QueryRowContext(ctx, `SELECT `+serverCols+` FROM servers WHERE id = ?`, id))
}

// List returns all servers ordered by id.
func (r *Repo) List(ctx context.Context) ([]Server, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+serverCols+` FROM servers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Server
	for rows.Next() {
		s, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// Update changes editable fields. SSH secret fields are only overwritten when
// non-empty (so an update without a new secret keeps the old one).
func (r *Repo) Update(ctx context.Context, s *Server) (*Server, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE servers SET name=?, host=?, ssh_port=?, ssh_user=?, ssh_auth_method=?,
			xray_config_path=?, xray_service_name=?,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		 WHERE id=?`,
		s.Name, s.Host, s.SSHPort, s.SSHUser, s.SSHAuthMethod,
		s.XrayConfigPath, s.XrayServiceName, s.ID)
	if err != nil {
		return nil, err
	}
	if s.SSHSecretEnc != "" {
		if _, err := r.db.ExecContext(ctx, `UPDATE servers SET ssh_secret_enc=? WHERE id=?`, s.SSHSecretEnc, s.ID); err != nil {
			return nil, err
		}
	}
	if s.SSHPassphrase != "" {
		if _, err := r.db.ExecContext(ctx, `UPDATE servers SET ssh_passphrase_enc=? WHERE id=?`, s.SSHPassphrase, s.ID); err != nil {
			return nil, err
		}
	}
	return r.GetByID(ctx, s.ID)
}

// Delete removes a server (cascades to its inbounds).
func (r *Repo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM servers WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateCheckResult persists reachability/geo/status after a check.
func (r *Repo) UpdateCheckResult(ctx context.Context, id int64, status, ip, country, city, asn string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE servers SET status=?, ip=?, geo_country=?, geo_city=?, geo_asn=?,
			last_check_at=strftime('%Y-%m-%dT%H:%M:%fZ','now'),
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		 WHERE id=?`,
		status, ip, country, city, asn, id)
	return err
}

// UpdateSyncResult persists the outcome of a config sync.
func (r *Repo) UpdateSyncResult(ctx context.Context, id int64, syncErr string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE servers SET last_sync_at=strftime('%Y-%m-%dT%H:%M:%fZ','now'),
			last_sync_error=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		 WHERE id=?`,
		syncErr, id)
	return err
}

func (r *Repo) scanOne(row *sql.Row) (*Server, error) {
	var s Server
	var port int
	var lastCheck, lastSync sql.NullString
	err := row.Scan(&s.ID, &s.Name, &s.Host, &port, &s.SSHUser, &s.SSHAuthMethod, &s.SSHSecretEnc,
		&s.SSHPassphrase, &s.XrayConfigPath, &s.XrayServiceName, &s.IP, &s.GeoCountry, &s.GeoCity,
		&s.GeoASN, &s.Status, &lastCheck, &lastSync, &s.LastSyncError, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.SSHPort = port
	s.LastCheckAt = lastCheck.String
	s.LastSyncAt = lastSync.String
	return &s, nil
}

func scanRows(rows *sql.Rows) (*Server, error) {
	var s Server
	var port int
	var lastCheck, lastSync sql.NullString
	err := rows.Scan(&s.ID, &s.Name, &s.Host, &port, &s.SSHUser, &s.SSHAuthMethod, &s.SSHSecretEnc,
		&s.SSHPassphrase, &s.XrayConfigPath, &s.XrayServiceName, &s.IP, &s.GeoCountry, &s.GeoCity,
		&s.GeoASN, &s.Status, &lastCheck, &lastSync, &s.LastSyncError, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.SSHPort = port
	s.LastCheckAt = lastCheck.String
	s.LastSyncAt = lastSync.String
	return &s, nil
}
