// Package auth handles admin login, JWT sessions, TOTP 2FA and the bootstrap
// admin account.
package auth

import (
	"context"
	"database/sql"
	"errors"
)

// Admin is the single panel administrator.
type Admin struct {
	ID            int64  `json:"id"`
	Username      string `json:"username"`
	PasswordHash  string `json:"-"`
	TOTPSecretEnc string `json:"-"`
	TOTPEnabled   bool   `json:"totp_enabled"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ErrNotFound is returned when an admin does not exist.
var ErrNotFound = errors.New("admin not found")

// Repo is the admin data-access layer.
type Repo struct {
	db *sql.DB
}

// NewRepo builds an admin repository.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// GetByUsername loads an admin by username.
func (r *Repo) GetByUsername(ctx context.Context, username string) (*Admin, error) {
	return r.scanOne(r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, totp_secret_enc, totp_enabled, created_at, updated_at
         FROM admins WHERE username = ?`, username))
}

// GetByID loads an admin by id.
func (r *Repo) GetByID(ctx context.Context, id int64) (*Admin, error) {
	return r.scanOne(r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, totp_secret_enc, totp_enabled, created_at, updated_at
         FROM admins WHERE id = ?`, id))
}

// Count returns the number of admins.
func (r *Repo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&n)
	return n, err
}

// Create inserts a new admin and returns it.
func (r *Repo) Create(ctx context.Context, username, passwordHash string) (*Admin, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO admins(username, password_hash) VALUES (?, ?)`, username, passwordHash)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

// SetTOTPSecret stores a (not-yet-enabled) encrypted TOTP secret.
func (r *Repo) SetTOTPSecret(ctx context.Context, id int64, secretEnc string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE admins SET totp_secret_enc = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`,
		secretEnc, id)
	return err
}

// SetTOTPEnabled toggles whether TOTP is required at login.
func (r *Repo) SetTOTPEnabled(ctx context.Context, id int64, enabled bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE admins SET totp_enabled = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`,
		boolToInt(enabled), id)
	return err
}

func (r *Repo) scanOne(row *sql.Row) (*Admin, error) {
	var a Admin
	var enabled int
	err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.TOTPSecretEnc, &enabled, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.TOTPEnabled = enabled != 0
	return &a, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
