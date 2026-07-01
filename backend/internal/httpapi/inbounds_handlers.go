package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/auth"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/inbounds"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/servers"
)

type inboundHandler struct {
	deps Deps
}

func mountInbounds(r chi.Router, deps Deps) {
	h := &inboundHandler{deps: deps}
	r.Get("/api/servers/{id}/inbounds", h.listByServer)
	r.Post("/api/servers/{id}/inbounds", h.create)
	r.Route("/api/inbounds/{id}", func(r chi.Router) {
		r.Get("/", h.get)
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})
}

func (h *inboundHandler) listByServer(w http.ResponseWriter, r *http.Request) {
	serverID, ok := idParam(w, r)
	if !ok {
		return
	}
	list, err := h.deps.Inbounds.Repo().ListByServer(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list inbounds failed")
		return
	}
	if list == nil {
		list = []inbounds.Inbound{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *inboundHandler) create(w http.ResponseWriter, r *http.Request) {
	serverID, ok := idParam(w, r)
	if !ok {
		return
	}
	// Ensure the server exists for a clean 404 instead of an FK error.
	if h.deps.Servers != nil {
		if _, err := h.deps.Servers.Repo().GetByID(r.Context(), serverID); err != nil {
			if errors.Is(err, servers.ErrNotFound) {
				writeError(w, http.StatusNotFound, "server not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	var in inbounds.Input
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	created, err := h.deps.Inbounds.Create(r.Context(), serverID, in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(r, "create", "inbound", created.ID, map[string]any{"tag": created.Tag, "protocol": created.Protocol, "server_id": serverID})
	h.fireSyncServer(serverID)
	writeJSON(w, http.StatusCreated, created)
}

func (h *inboundHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	in, err := h.deps.Inbounds.Repo().GetByID(r.Context(), id)
	if h.handleErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func (h *inboundHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var in inbounds.Input
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.deps.Inbounds.Update(r.Context(), id, in)
	if h.handleErr(w, err) {
		return
	}
	h.audit(r, "update", "inbound", updated.ID, map[string]any{"tag": updated.Tag})
	h.fireSyncServer(updated.ServerID)
	writeJSON(w, http.StatusOK, updated)
}

func (h *inboundHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	// Capture the server id before deletion so we can resync it.
	var serverID int64
	if existing, err := h.deps.Inbounds.Repo().GetByID(r.Context(), id); err == nil {
		serverID = existing.ServerID
	}
	if err := h.deps.Inbounds.Repo().Delete(r.Context(), id); h.handleErr(w, err) {
		return
	}
	h.audit(r, "delete", "inbound", id, nil)
	if serverID != 0 {
		h.fireSyncServer(serverID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *inboundHandler) handleErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, inbounds.ErrNotFound) {
		writeError(w, http.StatusNotFound, "inbound not found")
		return true
	}
	writeError(w, http.StatusInternalServerError, err.Error())
	return true
}

func (h *inboundHandler) audit(r *http.Request, action, targetType string, targetID int64, detail any) {
	if h.deps.Audit == nil {
		return
	}
	adminID, _ := auth.AdminIDFromContext(r.Context())
	_ = h.deps.Audit.Record(r.Context(), adminID, action, targetType, targetID, detail, clientIP(r), r.UserAgent())
}

// fireSyncServer pushes config to one server in the background, best-effort.
func (h *inboundHandler) fireSyncServer(serverID int64) {
	if h.deps.Sync == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		_, _ = h.deps.Sync.SyncServer(ctx, serverID)
	}()
}
