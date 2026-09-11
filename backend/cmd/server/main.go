package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mainstarkov/veil-browser/backend/internal/config"
	"github.com/mainstarkov/veil-browser/backend/internal/httpapi"
	"github.com/mainstarkov/veil-browser/backend/internal/session"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	store := session.NewStore(cfg.SessionTTL)
	api := httpapi.New(store, httpapi.Options{
		AllowedOrigin: cfg.AllowedOrigin,
		SecureCookies: cfg.SecureCookies,
	})

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           api,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go removeExpiredSessions(ctx, store, cfg.CleanupInterval)

	go func() {
		slog.Info("Veil API started", "address", cfg.Address, "origin", cfg.AllowedOrigin)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}

func removeExpiredSessions(ctx context.Context, store *session.Store, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			store.DeleteExpired()
		}
	}
}
