package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/subscription"
)

func mountSubscription(r chi.Router, deps Deps) {
	r.Get("/sub/{token}", func(w http.ResponseWriter, req *http.Request) {
		// Rate-limit the public endpoint per client IP.
		if deps.SubLimiter != nil && !deps.SubLimiter.Allow(clientIP(req)) {
			writeError(w, http.StatusTooManyRequests, "rate limited")
			return
		}
		token := chi.URLParam(req, "token")
		sub, err := deps.Subscription.BuildByToken(req.Context(), token)
		if err != nil {
			if errors.Is(err, subscription.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "subscription failed")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		// Standard subscription header used by some clients.
		w.Header().Set("Profile-Update-Interval", "12")
		_, _ = w.Write([]byte(sub))
	})
}

// subscriptionQR builds a QR PNG for the subscription URL of a token.
func subscriptionQR(baseURL, token string) ([]byte, error) {
	return subscription.QRPNG(subscriptionURL(baseURL, token), 256)
}

// subscriptionURL joins the base URL and token into the public /sub URL.
func subscriptionURL(baseURL, token string) string {
	return strings.TrimRight(baseURL, "/") + "/sub/" + token
}
