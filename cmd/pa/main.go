package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"pa/internal/config"
	"pa/internal/core"
	"syscall"
)

func main() {
	configPath := flag.String("config", "", "Path to config JSON file")
	flag.Parse()
	if *configPath == "" {
		*configPath = os.Getenv("PA_CONFIG_PATH")
	}
	if *configPath == "" {
		*configPath = "./config.json"
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := core.Run(ctx, cfg, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("run", "error", err)
		os.Exit(1)
	}
}
