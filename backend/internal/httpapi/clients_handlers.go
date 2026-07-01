package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/auth"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/clients"
)

type clientHandler struct {
	deps Deps
}

func mountClients(r chi.Router, deps Deps) {
	h := &clientHandler{deps: deps}
	r.Route("/api/clients", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Delete("/", h.delete)
			r.Post("/enable", h.enable)
			r.Post("/disable", h.disable)
			r.Put("/inbounds", h.setInbounds)
			r.Post("/rotate-token", h.rotateToken)
			r.Get("/links", h.links)
			r.Get("/qrcode", h.qrcode)
			r.Get("/config", h.config)
		})
	})
}

func (h *clientHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.deps.Clients.Repo().List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list clients failed")
		return
	}
	if list == nil {
		list = []clients.Client{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *clientHandler) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name   string `json:"name"`
		Remark string `json:"remark"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.deps.Clients.Create(r.Context(), in.Name, in.Remark)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(r, "create", "client", c.ID, map[string]any{"name": c.Name})
	writeJSON(w, http.StatusCreated, c)
}

func (h *clientHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	c, err := h.deps.Clients.Repo().GetByID(r.Context(), id)
	if h.handleErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *clientHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var in struct {
		Name   string `json:"name"`
		Remark string `json:"remark"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.deps.Clients.Update(r.Context(), id, in.Name, in.Remark)
	if h.handleErr(w, err) {
		return
	}
	h.audit(r, "update", "client", c.ID, nil)
	writeJSON(w, http.StatusOK, c)
}

func (h *clientHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.deps.Clients.Repo().Delete(r.Context(), id); h.handleErr(w, err) {
		return
	}
	h.audit(r, "delete", "client", id, nil)
	h.fireSyncAll()
	w.WriteHeader(http.StatusNoContent)
}

func (h *clientHandler) enable(w http.ResponseWriter, r *http.Request)  { h.setEnabled(w, r, true) }
func (h *clientHandler) disable(w http.ResponseWriter, r *http.Request) { h.setEnabled(w, r, false) }

func (h *clientHandler) setEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.deps.Clients.SetEnabled(r.Context(), id, enabled); h.handleErr(w, err) {
		return
	}
	action := "disable"
	if enabled {
		action = "enable"
	}
	h.audit(r, action, "client", id, nil)
	h.fireSyncAll()
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": enabled})
}

func (h *clientHandler) setInbounds(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var in struct {
		InboundIDs []int64 `json:"inbound_ids"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.deps.Clients.SetInbounds(r.Context(), id, in.InboundIDs)
	if h.handleErr(w, err) {
		return
	}
	h.audit(r, "set_inbounds", "client", id, map[string]any{"inbound_ids": in.InboundIDs})
	h.fireSyncAll()
	writeJSON(w, http.StatusOK, c)
}

func (h *clientHandler) rotateToken(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	c, err := h.deps.Clients.RotateToken(r.Context(), id)
	if h.handleErr(w, err) {
		return
	}
	h.audit(r, "rotate_token", "client", id, nil)
	writeJSON(w, http.StatusOK, c)
}

func (h *clientHandler) links(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if h.deps.Subscription == nil {
		writeError(w, http.StatusNotImplemented, "subscription not configured")
		return
	}
	list, err := h.deps.Subscription.LinksByClientID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *clientHandler) qrcode(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	c, err := h.deps.Clients.Repo().GetByID(r.Context(), id)
	if h.handleErr(w, err) {
		return
	}
	png, err := subscriptionQR(h.deps.SubBaseURL, c.SubscriptionToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "qr generation failed")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png)
}

func (h *clientHandler) config(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	c, err := h.deps.Clients.Repo().GetByID(r.Context(), id)
	if h.handleErr(w, err) {
		return
	}
	sub, err := h.deps.Subscription.BuildByToken(r.Context(), c.SubscriptionToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"subscription.txt\"")
	_, _ = w.Write([]byte(sub))
}

func (h *clientHandler) handleErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, clients.ErrNotFound) {
		writeError(w, http.StatusNotFound, "client not found")
		return true
	}
	writeError(w, http.StatusInternalServerError, err.Error())
	return true
}

func (h *clientHandler) audit(r *http.Request, action, targetType string, targetID int64, detail any) {
	if h.deps.Audit == nil {
		return
	}
	adminID, _ := auth.AdminIDFromContext(r.Context())
	_ = h.deps.Audit.Record(r.Context(), adminID, action, targetType, targetID, detail, clientIP(r), r.UserAgent())
}

// fireSyncAll pushes config to all servers in the background, best-effort.
// Errors are recorded on each server's last_sync_error.
func (h *clientHandler) fireSyncAll() {
	if h.deps.Sync == nil || h.deps.Servers == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		list, err := h.deps.Servers.Repo().List(ctx)
		if err != nil {
			return
		}
		for _, s := range list {
			_, _ = h.deps.Sync.SyncServer(ctx, s.ID)
		}
	}()
}
