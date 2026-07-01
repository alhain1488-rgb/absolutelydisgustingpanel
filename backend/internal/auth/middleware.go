package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const adminIDKey ctxKey = "admin_id"

// Middleware returns a middleware that requires a valid full session token
// (stage "auth") and puts the admin id in the request context.
func (m *TokenManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := m.claimsFromRequest(r, StageAuth)
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), adminIDKey, claims.AdminID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PendingFromRequest extracts a valid pending (2FA) token's admin id.
func (m *TokenManager) PendingFromRequest(r *http.Request) (int64, bool) {
	claims, ok := m.claimsFromRequest(r, StagePending)
	if !ok {
		return 0, false
	}
	return claims.AdminID, true
}

func (m *TokenManager) claimsFromRequest(r *http.Request, wantStage string) (*Claims, bool) {
	tok := bearerToken(r)
	if tok == "" {
		return nil, false
	}
	claims, err := m.Parse(tok)
	if err != nil {
		return nil, false
	}
	if claims.Stage != wantStage {
		return nil, false
	}
	return claims, true
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// AdminIDFromContext returns the authenticated admin id set by Middleware.
func AdminIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(adminIDKey).(int64)
	return id, ok
}
