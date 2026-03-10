package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"pa/internal/allowlist"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/memory"
	"pa/internal/noderunner"
	"pa/internal/scheduler"
	"pa/internal/telegram"
	"pa/internal/tools"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"syscall"
)

// logLevelFromEnv returns the slog level from PA_LOG_LEVEL (e.g. "debug", "info"); default is INFO (REQ-021).
func logLevelFromEnv() slog.Level {
	env := os.Getenv("PA_LOG_LEVEL")
	if env == "" {
		return slog.LevelInfo
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(env)); err != nil {
		return slog.LevelInfo
	}
	return l
}

func main() {
	configPath := flag.String("config", "", "Path to config JSON file")
	verifyNodes := flag.Bool("verify-nodes", false, "Verify SSH access to all configured nodes (run one allowlisted command per node and exit; do not start the bot)")
	verifyNodesCommand := flag.String("verify-nodes-command", "uptime", "Command to run on each node when using -verify-nodes (must be in node allowlist)")
	flag.Parse()
	if *configPath == "" {
		*configPath = os.Getenv("PA_CONFIG_PATH")
	}
	if *configPath == "" {
		*configPath = "./config/config.json"
	}

	logLevel := logLevelFromEnv()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	logger.Info("config loaded", "path", *configPath)

	if *verifyNodes {
		runVerifyNodes(cfg, *configPath, *verifyNodesCommand, logger)
		os.Exit(0)
	}

	adapter, memoryStore, vectorStore, embedder, nodeRunner, err := setup(cfg, *configPath, logger)
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

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(tools.NewRunOnNode(nodeRunner))
	if cleanup := startSchedulerIfConfigured(cfg, adapter, toolRegistry, logger); cleanup != nil {
		defer cleanup()
	}

	logger.Info("starting", "adapter", "telegram")
	if err := core.Run(ctx, cfg, logger, adapter, llmProvider, memoryStore, vectorStore, embedder, nodeRunner); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("run", "error", err)
		os.Exit(1)
	}
}

// startSchedulerIfConfigured loads scheduled tasks and starts the scheduler when paths.scheduled_tasks_path is set. Returns a cleanup function to call on exit, or nil.
func startSchedulerIfConfigured(cfg *config.Config, adapter core.Adapter, toolRegistry *tools.Registry, logger *slog.Logger) func() {
	if cfg.Paths.ScheduledTasksPath == "" {
		return nil
	}
	tasks, err := scheduler.LoadTasks(cfg.Paths.ScheduledTasksPath)
	if err != nil {
		logger.Error("load scheduled tasks", "error", err)
		os.Exit(1)
	}
	if len(tasks) == 0 {
		return nil
	}
	var notifier scheduler.Notifier
	if tg, ok := adapter.(*telegram.Adapter); ok {
		notifier = tg
	}
	sched, err := scheduler.New(tasks, scheduler.Config{
		Registry: toolRegistry,
		Notifier: notifier,
		Logger:   logger,
	})
	if err != nil {
		logger.Error("create scheduler", "error", err)
		os.Exit(1)
	}
	sched.Start()
	logger.Info("scheduler started", "tasks", len(tasks))
	return func() {
		stopCtx := sched.Stop()
		<-stopCtx.Done()
	}
}

// runVerifyNodes loads allowlist and NodeRunner, runs one allowlisted command on each configured node, reports success or failure per node, then exits (REQ-022, AC-032).
func runVerifyNodes(cfg *config.Config, configPath, command string, logger *slog.Logger) {
	if len(cfg.Nodes) == 0 {
		logger.Info("no nodes in config, nothing to verify")
		return
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		logger.Error("allowlist", "error", err)
		os.Exit(1)
	}
	runner := noderunner.New(cfg, al, logger)
	ctx := context.Background()
	for nodeID := range cfg.Nodes {
		logger.Info("verify node", "node_id", nodeID, "command", command)
		stdout, err := runner.RunOnNode(ctx, nodeID, command)
		if err != nil {
			logger.Error("node failed", "node_id", nodeID, "error", err)
			os.Exit(1)
		}
		logger.Info("node OK", "node_id", nodeID)
		if len(stdout) > 0 {
			logger.Info("node output", "node_id", nodeID, "stdout", stdout)
		}
	}
}

// setup creates adapter, memory store, vector store, embedder, and optional node runner from config. Caller must close vectorStore.
func setup(cfg *config.Config, configPath string, logger *slog.Logger) (
	adapter core.Adapter,
	memoryStore *memory.Store,
	vectorStore *sqlite.Store,
	embedder embedding.Embedder,
	nodeRunner core.NodeRunner,
	err error,
) {
	adapter, err = telegram.NewAdapter(cfg, configPath)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if cfg.Paths.MemoryDir != "" {
		if mkErr := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); mkErr != nil {
			return nil, nil, nil, nil, nil, mkErr
		}
		memoryStore, err = memory.NewStore(cfg.Paths.MemoryDir)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}

	embedder, err = embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if mkErr := os.MkdirAll(vecDir, 0o755); mkErr != nil {
		return nil, nil, nil, nil, nil, mkErr
	}
	vectorStore, err = sqlite.New(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if len(cfg.Nodes) > 0 {
		al, alErr := allowlist.NewChecker(cfg)
		if alErr != nil {
			return nil, nil, nil, nil, nil, alErr
		}
		nodeRunner = noderunner.New(cfg, al, logger)
	}

	return adapter, memoryStore, vectorStore, embedder, nodeRunner, nil
}
