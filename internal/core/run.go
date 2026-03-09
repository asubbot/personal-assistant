package core

import (
	"context"
	"log/slog"
	"pa/internal/config"
)

// Run starts the application (Telegram, scheduler, etc.). For now it returns nil immediately.
// When the run loop is implemented (task 3.x), Run will block until ctx is cancelled (SIGINT/SIGTERM) for graceful shutdown.
func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	_ = ctx
	_ = cfg
	_ = logger
	return nil
}
