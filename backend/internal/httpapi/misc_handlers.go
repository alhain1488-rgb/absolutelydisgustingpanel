package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/audit"
)

// mountMisc registers dashboard/logs/settings endpoints that are available so
// far. It grows across phases.
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
}
