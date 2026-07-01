// Package httpapi wires the HTTP router, middleware and handlers.
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/audit"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/auth"
)

// Deps holds the dependencies handlers need.
type Deps struct {
	DB           *sql.DB
	Logger       *slog.Logger
	Auth         *auth.Service
	Audit        *audit.Recorder
	LoginLimiter *auth.RateLimiter
}

// NewRouter builds the top-level HTTP handler.
func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(realIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(requestLogger(deps.Logger))

	// Liveness.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Readiness.
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if deps.DB != nil {
			ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
			defer cancel()
			if err := deps.DB.PingContext(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db unavailable"})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	if deps.Auth != nil {
		mountAuth(r, deps)
	}

	return r
}

// mountAuth registers the /api/auth/* routes.
func mountAuth(r chi.Router, deps Deps) {
	h := &authHandler{deps: deps}
	tokens := deps.Auth.Tokens()

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", h.login)
		r.Post("/2fa/verify", h.verify2FA)

		// Authenticated sub-routes.
		r.Group(func(r chi.Router) {
			r.Use(tokens.Middleware)
			r.Post("/2fa/setup", h.setup2FA)
			r.Post("/2fa/enable", h.enable2FA)
			r.Post("/logout", h.logout)
			r.Get("/me", h.me)
		})
	})
}

// realIP rewrites RemoteAddr from proxy headers set by our trusted front proxy
// (Caddy). The panel is only ever exposed through Caddy, which strips
// client-supplied X-Forwarded-For and sets it to the real client IP.
func realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			r.RemoteAddr = net.JoinHostPort(xrip, "0")
		} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.IndexByte(xff, ','); i >= 0 {
				xff = xff[:i]
			}
			r.RemoteAddr = net.JoinHostPort(strings.TrimSpace(xff), "0")
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			if logger != nil {
				logger.Info("http_request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration_ms", time.Since(start).Milliseconds(),
				)
			}
		})
	}
}
