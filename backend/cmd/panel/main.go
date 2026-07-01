// Command panel is the entrypoint for the Xray control-panel backend.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/audit"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/auth"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/config"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/crypto"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/db"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/httpapi"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/inbounds"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/logging"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/servers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// No logger yet; fail fast with a clear message.
		_, _ = os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel)

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		logger.Error("database init failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = database.Close() }()

	cipher, err := crypto.NewCipher(cfg.EncryptionKey)
	if err != nil {
		logger.Error("cipher init failed", "error", err)
		os.Exit(1)
	}

	adminRepo := auth.NewRepo(database)
	tokens := auth.NewTokenManager(cfg.JWTSecret)
	authSvc := auth.NewService(adminRepo, tokens, cipher, "Xray Panel")
	auditRec := audit.New(database)
	serversSvc := servers.NewService(servers.NewRepo(database), cipher, servers.NewIPAPIGeolocator())
	inboundsSvc := inbounds.NewService(inbounds.NewRepo(database))

	// Bootstrap the initial admin from configuration on first start.
	if err := authSvc.Bootstrap(context.Background(), cfg.AdminUsername, cfg.AdminPassword); err != nil {
		logger.Error("admin bootstrap failed", "error", err)
		os.Exit(1)
	}

	router := httpapi.NewRouter(httpapi.Deps{
		DB:           database,
		Logger:       logger,
		Auth:         authSvc,
		Audit:        auditRec,
		LoginLimiter: auth.NewRateLimiter(10, time.Minute),
		Servers:      serversSvc,
		Inbounds:     inboundsSvc,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("backend listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
