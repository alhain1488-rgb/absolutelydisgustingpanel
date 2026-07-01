// Package clients implements CRUD for clients, their access grants to inbounds
// (the many-to-many client_inbounds relation) and subscription-token handling.
package clients

import (
	"context"
	"database/sql"
	"errors"
)

// ErrNotFound is returned when a client does not exist.
var ErrNotFound = errors.New("client not found")

// Client is a subscriber that may be granted access to specific inbounds.
type Client struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	UUID              string  `json:"uuid"`
	Password          string  `json:"password"`
	SubscriptionToken string  `json:"subscription_token"`
	Enabled           bool    `json:"enabled"`
	Remark            string  `json:"remark"`
	InboundIDs        []int64 `json:"inbound_ids"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

// Repo is the client data-access layer.
type Repo struct {
	db *sql.DB
}

// NewRepo builds a client repository.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

const cols = `id, name, uuid, password, subscription_token, enabled, remark, created_at, updated_at`

// Create inserts a client.
func (r *Repo) Create(ctx context.Context, c *Client) (*Client, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO clients(name, uuid, password, subscription_token, enabled, remark)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.Name, c.UUID, c.Password, c.SubscriptionToken, boolToInt(c.Enabled), c.Remark)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

// GetByID loads a client with its granted inbound IDs.
func (r *Repo) GetByID(ctx context.Context, id int64) (*Client, error) {
	c, err := scanOne(r.db.QueryRowContext(ctx, `SELECT `+cols+` FROM clients WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	if c.InboundIDs, err = r.grantedInboundIDs(ctx, c.ID); err != nil {
		return nil, err
	}
	return c, nil
}

// GetByToken loads a client by subscription token.
func (r *Repo) GetByToken(ctx context.Context, token string) (*Client, error) {
	return scanOne(r.db.QueryRowContext(ctx, `SELECT `+cols+` FROM clients WHERE subscription_token = ?`, token))
}

// List returns all clients (without inbound IDs, for the index view).
func (r *Repo) List(ctx context.Context) ([]Client, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+cols+` FROM clients ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Client
	for rows.Next() {
		c, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// UpdateInfo changes name/remark.
func (r *Repo) UpdateInfo(ctx context.Context, id int64, name, remark string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE clients SET name=?, remark=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=?`,
		name, remark, id)
	return err
}

// SetEnabled toggles a client on/off.
func (r *Repo) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE clients SET enabled=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=?`,
		boolToInt(enabled), id)
	return err
}

// RotateToken sets a new subscription token.
func (r *Repo) RotateToken(ctx context.Context, id int64, token string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE clients SET subscription_token=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=?`,
		token, id)
	return err
}

// Delete removes a client (cascades to client_inbounds).
func (r *Repo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM clients WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetInbounds replaces the client's grants with the given inbound IDs.
func (r *Repo) SetInbounds(ctx context.Context, clientID int64, inboundIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM client_inbounds WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	for _, inboundID := range inboundIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO client_inbounds(client_id, inbound_id, enabled) VALUES (?, ?, 1)`,
			clientID, inboundID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repo) grantedInboundIDs(ctx context.Context, clientID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT inbound_id FROM client_inbounds WHERE client_id = ? AND enabled = 1 ORDER BY inbound_id`, clientID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanOne(row *sql.Row) (*Client, error) {
	c, err := scan(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func scanRows(rows *sql.Rows) (*Client, error) { return scan(rows.Scan) }

func scan(scanFn func(dest ...any) error) (*Client, error) {
	var c Client
	var enabled int
	err := scanFn(&c.ID, &c.Name, &c.UUID, &c.Password, &c.SubscriptionToken, &enabled, &c.Remark, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c.Enabled = enabled != 0
	return &c, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
