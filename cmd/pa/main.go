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
	"pa/internal/summarize"
	"pa/internal/telegram"
	"pa/internal/tools"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"syscall"
	"time"
)

// configFilePath returns the path to the main config file: PA_CONFIG_DIR (default "./config") joined with config.ConfigFileName.
func configFilePath() string {
	dir := os.Getenv("PA_CONFIG_DIR")
	if dir == "" {
		dir = "./config"
	}
	return filepath.Join(dir, config.ConfigFileName)
}

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
	verifyNodes := flag.Bool("verify-nodes", false, "Verify SSH access to all configured nodes (run one allowlisted command per node and exit; do not start the bot)")
	verifyNodesCommand := flag.String("verify-nodes-command", "uptime", "Command to run on each node when using -verify-nodes (must be in node allowlist)")
	summarizeFlag := flag.String("summarize", "", "Run summarization and exit: YYYY-MM-DD (day), YYYY-MM (month), YYYY (year). No default.")
	flag.Parse()

	configFilePath := configFilePath()

	logLevel := logLevelFromEnv()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	cfg, err := config.Load(configFilePath)
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	logger.Info("config loaded", "path", configFilePath)

	if *verifyNodes {
		runVerifyNodes(cfg, configFilePath, *verifyNodesCommand, logger)
		os.Exit(0)
	}

	if *summarizeFlag != "" {
		runSummarize(cfg, *summarizeFlag, logger)
		return
	}

	adapter, memoryStore, vectorStore, embedder, nodeRunner, err := setup(cfg, configFilePath, logger)
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

	// Only the first LLM provider is used; fallback to next on failure is TBD for a future increment.
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

// runSummarize parses -summarize value (YYYY, YYYY-MM, or YYYY-MM-DD), runs the matching summarization, then exits.
func runSummarize(cfg *config.Config, value string, logger *slog.Logger) {
	scope, err := summarize.ParseSummarizeScope(value)
	if err != nil {
		logger.Error("summarize", "error", err)
		os.Exit(1)
	}
	switch scope.Kind {
	case "day":
		runSummarizeDay(cfg, scope.Day, logger)
	case "month":
		runSummarizeMonth(cfg, scope.Year, scope.Month, logger)
	case "year":
		runSummarizeYear(cfg, scope.Year, logger)
	default:
		logger.Error("summarize: unknown scope", "scope", scope.Kind)
		os.Exit(1)
	}
}

// runSummarizeDay runs day summarization for the given date (UTC), then exits.
func runSummarizeDay(cfg *config.Config, day time.Time, logger *slog.Logger) {
	if err := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); err != nil {
		logger.Error("summarize: mkdir memory", "error", err)
		os.Exit(1)
	}
	memoryStore, err := memory.NewStore(cfg.Paths.MemoryDir)
	if err != nil {
		logger.Error("summarize: memory store", "error", err)
		os.Exit(1)
	}

	llmProvider, err := llm.NewProvider(&cfg.LLMProviders[0])
	if err != nil {
		logger.Error("summarize: llm provider", "error", err)
		os.Exit(1)
	}

	embedder, err := embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		logger.Error("summarize: embedder", "error", err)
		os.Exit(1)
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		logger.Error("summarize: mkdir vector", "error", err)
		os.Exit(1)
	}
	vectorStore, err := sqlite.New(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions)
	if err != nil {
		logger.Error("summarize: vector store", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := vectorStore.Close(); closeErr != nil {
			logger.Error("summarize: close vector store", "error", closeErr)
		}
	}()

	ctx := context.Background()
	err = summarize.Day(ctx, day, summarize.DayConfig{
		LLMLogDir:   cfg.Paths.LLMLogDir,
		LLMProvider: llmProvider,
		MemoryStore: memoryStore,
		Embedder:    embedder,
		VectorStore: vectorStore,
		Logger:      logger,
	})
	if err != nil {
		logger.Error("summarize", "error", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// runSummarizeMonth runs month summarization for the given year/month (UTC), then exits.
func runSummarizeMonth(cfg *config.Config, year int, month int, logger *slog.Logger) {
	if err := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); err != nil {
		logger.Error("summarize: mkdir memory", "error", err)
		os.Exit(1)
	}
	memoryStore, err := memory.NewStore(cfg.Paths.MemoryDir)
	if err != nil {
		logger.Error("summarize: memory store", "error", err)
		os.Exit(1)
	}

	llmProvider, err := llm.NewProvider(&cfg.LLMProviders[0])
	if err != nil {
		logger.Error("summarize: llm provider", "error", err)
		os.Exit(1)
	}

	embedder, err := embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		logger.Error("summarize: embedder", "error", err)
		os.Exit(1)
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		logger.Error("summarize: mkdir vector", "error", err)
		os.Exit(1)
	}
	vectorStore, err := sqlite.New(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions)
	if err != nil {
		logger.Error("summarize: vector store", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := vectorStore.Close(); closeErr != nil {
			logger.Error("summarize: close vector store", "error", closeErr)
		}
	}()

	ctx := context.Background()
	err = summarize.Month(ctx, year, month, summarize.MonthConfig{
		LLMProvider: llmProvider,
		MemoryStore: memoryStore,
		Embedder:    embedder,
		VectorStore: vectorStore,
		Logger:      logger,
	})
	if err != nil {
		logger.Error("summarize", "error", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// runSummarizeYear runs year summarization for the given year, then exits.
func runSummarizeYear(cfg *config.Config, year int, logger *slog.Logger) {
	if err := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); err != nil {
		logger.Error("summarize: mkdir memory", "error", err)
		os.Exit(1)
	}
	memoryStore, err := memory.NewStore(cfg.Paths.MemoryDir)
	if err != nil {
		logger.Error("summarize: memory store", "error", err)
		os.Exit(1)
	}

	llmProvider, err := llm.NewProvider(&cfg.LLMProviders[0])
	if err != nil {
		logger.Error("summarize: llm provider", "error", err)
		os.Exit(1)
	}

	embedder, err := embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		logger.Error("summarize: embedder", "error", err)
		os.Exit(1)
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		logger.Error("summarize: mkdir vector", "error", err)
		os.Exit(1)
	}
	vectorStore, err := sqlite.New(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions)
	if err != nil {
		logger.Error("summarize: vector store", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := vectorStore.Close(); closeErr != nil {
			logger.Error("summarize: close vector store", "error", closeErr)
		}
	}()

	ctx := context.Background()
	err = summarize.Year(ctx, year, summarize.YearConfig{
		LLMProvider: llmProvider,
		MemoryStore: memoryStore,
		Embedder:    embedder,
		VectorStore: vectorStore,
		Logger:      logger,
	})
	if err != nil {
		logger.Error("summarize", "error", err)
		os.Exit(1)
	}
	os.Exit(0)
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
