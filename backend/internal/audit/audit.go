// Package audit records administrative actions to the audit_logs table.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
)

// Entry is a single audit-log record.
type Entry struct {
	ID         int64           `json:"id"`
	AdminID    int64           `json:"admin_id"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   int64           `json:"target_id"`
	Detail     json.RawMessage `json:"detail_json"`
	IP         string          `json:"ip"`
	UserAgent  string          `json:"user_agent"`
	CreatedAt  string          `json:"created_at"`
}

// Recorder writes and lists audit entries.
type Recorder struct {
	db *sql.DB
}

// New builds a Recorder.
func New(db *sql.DB) *Recorder { return &Recorder{db: db} }

// Record persists an admin action. detail may be nil.
func (r *Recorder) Record(ctx context.Context, adminID int64, action, targetType string, targetID int64, detail any, ip, ua string) error {
	detailJSON := []byte("{}")
	if detail != nil {
		b, err := json.Marshal(detail)
		if err == nil {
			detailJSON = b
		}
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_logs(admin_id, action, target_type, target_id, detail_json, ip, user_agent)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		adminID, action, targetType, targetID, string(detailJSON), ip, ua)
	return err
}

// List returns the most recent entries, newest first, up to limit.
func (r *Recorder) List(ctx context.Context, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, admin_id, action, target_type, target_id, detail_json, ip, user_agent, created_at
         FROM audit_logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Entry
	for rows.Next() {
		var e Entry
		var adminID sql.NullInt64
		var detail string
		if err := rows.Scan(&e.ID, &adminID, &e.Action, &e.TargetType, &e.TargetID, &detail, &e.IP, &e.UserAgent, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.AdminID = adminID.Int64
		e.Detail = json.RawMessage(detail)
		out = append(out, e)
	}
	return out, rows.Err()
}
