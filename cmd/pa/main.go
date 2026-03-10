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
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/memory"
	"pa/internal/telegram"
	"pa/internal/vector/sqlite"
	"path/filepath"
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
	logger.Info("config loaded", "path", *configPath)

	adapter, memoryStore, vectorStore, embedder, err := setup(cfg, *configPath, logger)
	if err != nil {
		logger.Error("setup", "error", err)
		os.Exit(1)
	}
	defer func() {
		if vectorStore != nil {
			if closeErr := vectorStore.Close(); closeErr != nil {
				logger.Error("close vector store", "error", closeErr)
			}
		}
	}()

	llmProvider, err := llm.NewProvider(&cfg.LLMProviders[0])
	if err != nil {
		logger.Error("create llm provider", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting", "adapter", "telegram")
	if err := core.Run(ctx, cfg, logger, adapter, llmProvider, memoryStore, vectorStore, embedder); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("run", "error", err)
		os.Exit(1)
	}
}

// setup creates adapter, memory store, vector store, and embedder from config. Caller must close vectorStore.
func setup(cfg *config.Config, configPath string, _ *slog.Logger) (
	adapter core.Adapter,
	memoryStore *memory.Store,
	vectorStore *sqlite.Store,
	embedder embedding.Embedder,
	err error,
) {
	adapter, err = telegram.NewAdapter(cfg, configPath)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if cfg.Paths.MemoryDir != "" {
		if mkErr := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); mkErr != nil {
			return nil, nil, nil, nil, mkErr
		}
		memoryStore, err = memory.NewStore(cfg.Paths.MemoryDir)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	embedder, err = embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if mkErr := os.MkdirAll(vecDir, 0o755); mkErr != nil {
		return nil, nil, nil, nil, mkErr
	}
	vectorStore, err = sqlite.New(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return adapter, memoryStore, vectorStore, embedder, nil
}
