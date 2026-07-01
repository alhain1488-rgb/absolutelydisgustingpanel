package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/auth"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/servers"
)

type serverHandler struct {
	deps Deps
}

func mountServers(r chi.Router, deps Deps) {
	h := &serverHandler{deps: deps}
	r.Route("/api/servers", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Delete("/", h.delete)
			r.Post("/check", h.check)
			r.Post("/restart-xray", h.restartXray)
			r.Get("/stats", h.stats)
		})
	})
}

// serverInput is the request body for create/update.
type serverInput struct {
	Name            string `json:"name"`
	Host            string `json:"host"`
	SSHPort         int    `json:"ssh_port"`
	SSHUser         string `json:"ssh_user"`
	SSHAuthMethod   string `json:"ssh_auth_method"`
	SSHSecret       string `json:"ssh_secret"`
	SSHPassphrase   string `json:"ssh_passphrase"`
	XrayConfigPath  string `json:"xray_config_path"`
	XrayServiceName string `json:"xray_service_name"`
}

func (in serverInput) toCreateInput() servers.CreateInput {
	return servers.CreateInput{
		Name:            in.Name,
		Host:            in.Host,
		SSHPort:         in.SSHPort,
		SSHUser:         in.SSHUser,
		SSHAuthMethod:   in.SSHAuthMethod,
		SSHSecret:       in.SSHSecret,
		SSHPassphrase:   in.SSHPassphrase,
		XrayConfigPath:  in.XrayConfigPath,
		XrayServiceName: in.XrayServiceName,
	}
}

func (h *serverHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.deps.Servers.Repo().List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list servers failed")
		return
	}
	if list == nil {
		list = []servers.Server{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *serverHandler) create(w http.ResponseWriter, r *http.Request) {
	var in serverInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	srv, err := h.deps.Servers.Create(r.Context(), in.toCreateInput())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(r, "create", "server", srv.ID, map[string]any{"name": srv.Name, "host": srv.Host})
	writeJSON(w, http.StatusCreated, srv)
}

func (h *serverHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	srv, err := h.deps.Servers.Repo().GetByID(r.Context(), id)
	if h.handleErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, srv)
}

func (h *serverHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var in serverInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	srv, err := h.deps.Servers.Update(r.Context(), id, in.toCreateInput())
	if h.handleErr(w, err) {
		return
	}
	h.audit(r, "update", "server", srv.ID, map[string]any{"name": srv.Name})
	writeJSON(w, http.StatusOK, srv)
}

func (h *serverHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.deps.Servers.Repo().Delete(r.Context(), id); h.handleErr(w, err) {
		return
	}
	h.audit(r, "delete", "server", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *serverHandler) check(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	srv, err := h.deps.Servers.Check(r.Context(), id)
	if h.handleErr(w, err) {
		return
	}
	h.audit(r, "check", "server", id, map[string]any{"status": srv.Status})
	writeJSON(w, http.StatusOK, srv)
}

func (h *serverHandler) restartXray(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.deps.Servers.RestartXray(r.Context(), id); err != nil {
		if errors.Is(err, servers.ErrNotFound) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusBadGateway, "restart failed: "+err.Error())
		return
	}
	h.audit(r, "restart_xray", "server", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

func (h *serverHandler) stats(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	st, err := h.deps.Servers.Stats(r.Context(), id)
	if err != nil {
		if errors.Is(err, servers.ErrNotFound) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusBadGateway, "stats failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// handleErr writes a 404 for not-found and 500 otherwise; returns true if it wrote.
func (h *serverHandler) handleErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, servers.ErrNotFound) {
		writeError(w, http.StatusNotFound, "server not found")
		return true
	}
	writeError(w, http.StatusInternalServerError, err.Error())
	return true
}

func (h *serverHandler) audit(r *http.Request, action, targetType string, targetID int64, detail any) {
	if h.deps.Audit == nil {
		return
	}
	adminID, _ := auth.AdminIDFromContext(r.Context())
	_ = h.deps.Audit.Record(r.Context(), adminID, action, targetType, targetID, detail, clientIP(r), r.UserAgent())
}

// idParam parses the {id} URL parameter.
func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}
