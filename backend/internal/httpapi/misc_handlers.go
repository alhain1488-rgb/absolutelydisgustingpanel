package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/audit"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/servers"
)

// mountMisc registers dashboard, logs and settings endpoints.
func mountMisc(r chi.Router, deps Deps) {
	if deps.Audit != nil {
		r.Get("/api/logs", func(w http.ResponseWriter, req *http.Request) {
			limit := 100
			if v := req.URL.Query().Get("limit"); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					limit = n
				}
			}
			entries, err := deps.Audit.List(req.Context(), limit)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "list logs failed")
				return
			}
			if entries == nil {
				entries = []audit.Entry{}
			}
			writeJSON(w, http.StatusOK, entries)
		})
	}

	if deps.Servers != nil && deps.Clients != nil {
		r.Get("/api/dashboard/summary", func(w http.ResponseWriter, req *http.Request) {
			dashboardSummary(w, req, deps)
		})
	}

	if deps.DB != nil {
		r.Get("/api/settings", func(w http.ResponseWriter, req *http.Request) {
			settings, err := loadSettings(req.Context(), deps.DB)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "load settings failed")
				return
			}
			writeJSON(w, http.StatusOK, settings)
		})
		r.Put("/api/settings", func(w http.ResponseWriter, req *http.Request) {
			var in map[string]string
			if err := decodeJSON(req, &in); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			for k, v := range in {
				if strings.HasPrefix(k, "sync_hash_") {
					continue // internal key, not user-editable
				}
				if _, err := deps.DB.ExecContext(req.Context(),
					`INSERT INTO settings(key, value) VALUES (?, ?)
					 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
					writeError(w, http.StatusInternalServerError, "save settings failed")
					return
				}
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
	}
}

func dashboardSummary(w http.ResponseWriter, req *http.Request, deps Deps) {
	ctx := req.Context()
	serverList, err := deps.Servers.Repo().List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "summary failed")
		return
	}
	clientList, err := deps.Clients.Repo().List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "summary failed")
		return
	}

	var serversOnline, clientsEnabled int
	var lastSync string
	for _, s := range serverList {
		if s.Status == servers.StatusOnline {
			serversOnline++
		}
		if s.LastSyncAt > lastSync {
			lastSync = s.LastSyncAt
		}
	}
	for _, c := range clientList {
		if c.Enabled {
			clientsEnabled++
		}
	}
	if serverList == nil {
		serverList = []servers.Server{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"clients_total":   len(clientList),
		"clients_enabled": clientsEnabled,
		"servers_total":   len(serverList),
		"servers_online":  serversOnline,
		"last_sync_at":    lastSync,
		"servers":         serverList,
	})
}

func loadSettings(ctx context.Context, database *sql.DB) (map[string]string, error) {
	rows, err := database.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		if strings.HasPrefix(k, "sync_hash_") {
			continue // hide internal keys
		}
		out[k] = v
	}
	return out, rows.Err()
}
