package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"pa/internal/config"
	"time"
)

// observabilityHTTPHandler returns the mux for health and readiness (EP-029). cfg.ObservabilityHTTP must be non-nil.
func observabilityHTTPHandler(cfg *config.Config, app *paApplication, logger *slog.Logger) http.Handler {
	o := cfg.ObservabilityHTTP
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+o.HealthPath, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(logger, w, http.StatusOK, map[string]string{"status": "alive"})
	})
	mux.HandleFunc("GET "+o.ReadinessPath, func(w http.ResponseWriter, r *http.Request) {
		body := app.evalReadiness(r.Context())
		status := http.StatusOK
		if !body.Ready {
			status = http.StatusServiceUnavailable
		}
		writeJSON(logger, w, status, body)
	})
	return mux
}

// startObservabilityHTTPServer binds the configured listener and serves health/readiness until ctx is done.
// Returns a function that shuts down the server (caller should defer it). No-op when cfg.ObservabilityHTTP is nil.
func startObservabilityHTTPServer(ctx context.Context, cfg *config.Config, app *paApplication, logger *slog.Logger) func() {
	if cfg == nil || cfg.ObservabilityHTTP == nil || app == nil {
		return func() {}
	}
	o := cfg.ObservabilityHTTP
	srv := &http.Server{
		Addr:              o.ListenAddress,
		Handler:           observabilityHTTPHandler(cfg, app, logger),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		if logger != nil {
			logger.Info("observability http listening", "addr", o.ListenAddress, "health", o.HealthPath, "readiness", o.ReadinessPath)
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed && logger != nil {
			logger.Error("observability http server", "error", err)
		}
	}()

	return func() {
		// WithoutCancel keeps request-scoped values but ignores parent cancellation so Shutdown still gets a deadline.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && logger != nil {
			logger.Warn("observability http shutdown", "error", err)
		}
	}
}
