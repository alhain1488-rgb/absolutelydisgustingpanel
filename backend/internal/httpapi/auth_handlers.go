package httpapi

import (
	"errors"
	"net"
	"net/http"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/auth"
)

type authHandler struct {
	deps Deps
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	// Rate-limit by client IP to slow brute force.
	if h.deps.LoginLimiter != nil && !h.deps.LoginLimiter.Allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many login attempts, try again later")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.deps.Auth.Login(r.Context(), req.Username, req.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	if !res.Need2FA {
		h.record(r, 0, "login", "admin", 0, map[string]any{"username": req.Username})
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *authHandler) verify2FA(w http.ResponseWriter, r *http.Request) {
	adminID, ok := h.deps.Auth.Tokens().PendingFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid or expired pending token")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	token, err := h.deps.Auth.CompleteTOTP(r.Context(), adminID, req.Code)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid code")
		return
	}
	h.record(r, adminID, "login_2fa", "admin", adminID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *authHandler) setup2FA(w http.ResponseWriter, r *http.Request) {
	adminID, _ := auth.AdminIDFromContext(r.Context())
	setup, err := h.deps.Auth.SetupTOTP(r.Context(), adminID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "totp setup failed")
		return
	}
	h.record(r, adminID, "2fa_setup", "admin", adminID, nil)
	writeJSON(w, http.StatusOK, setup)
}

func (h *authHandler) enable2FA(w http.ResponseWriter, r *http.Request) {
	adminID, _ := auth.AdminIDFromContext(r.Context())
	var req struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.deps.Auth.EnableTOTP(r.Context(), adminID, req.Code); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid code")
		return
	}
	h.record(r, adminID, "2fa_enable", "admin", adminID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	adminID, _ := auth.AdminIDFromContext(r.Context())
	h.record(r, adminID, "logout", "admin", adminID, nil)
	// Stateless JWT: the client discards the token.
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	adminID, _ := auth.AdminIDFromContext(r.Context())
	admin, err := h.deps.Auth.AdminRepo().GetByID(r.Context(), adminID)
	if err != nil {
		writeError(w, http.StatusNotFound, "admin not found")
		return
	}
	writeJSON(w, http.StatusOK, admin)
}

// record writes an audit entry if a recorder is configured, best-effort.
func (h *authHandler) record(r *http.Request, adminID int64, action, targetType string, targetID int64, detail any) {
	if h.deps.Audit == nil {
		return
	}
	_ = h.deps.Audit.Record(r.Context(), adminID, action, targetType, targetID, detail, clientIP(r), r.UserAgent())
}

func clientIP(r *http.Request) string {
	// chi's RealIP middleware normalizes RemoteAddr from X-Forwarded-For.
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
