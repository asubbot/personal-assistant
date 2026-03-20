package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"pa/internal/allowlist"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/memory"
	"pa/internal/noderunner"
	"pa/internal/scheduler"
	"pa/internal/summarize"
	"pa/internal/telegram"
	"pa/internal/toolindex"
	"pa/internal/tools"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// buildLLMProviders builds one Provider per cfg.LLMProviders entry and parallel labels (Type/Model).
func buildLLMProviders(cfg *config.Config) ([]llm.Provider, []string, error) {
	if len(cfg.LLMProviders) == 0 {
		return nil, nil, fmt.Errorf("no llm providers configured")
	}
	var providers []llm.Provider
	var labels []string
	for i := range cfg.LLMProviders {
		p, err := llm.NewProvider(&cfg.LLMProviders[i])
		if err != nil {
			return nil, nil, err
		}
		providers = append(providers, p)
		typ := strings.TrimSpace(strings.ToLower(cfg.LLMProviders[i].Type))
		model := strings.TrimSpace(cfg.LLMProviders[i].Model)
		if model == "" {
			model = "default"
		}
		labels = append(labels, typ+"/"+model)
	}
	return providers, labels, nil
}

// newLLMProvider builds a provider adapter backed by unified llmrouter transport routing.
func newLLMProvider(cfg *config.Config, logger *slog.Logger) (llm.Provider, error) {
	providers, labels, err := buildLLMProviders(cfg)
	if err != nil {
		return nil, err
	}
	return llmrouter.NewProviderAdapter(providers, labels, logger)
}

// newLLMForConversation returns the ordered providers and labels for the unified router in core.
func newLLMForConversation(cfg *config.Config) (providers []llm.Provider, labels []string, err error) {
	return buildLLMProviders(cfg)
}

// configFilePath returns the path to the main config file: PA_CONFIG_DIR (default "./config") joined with config.ConfigFileName.
func configFilePath() string {
	dir := os.Getenv("PA_CONFIG_DIR")
	if dir == "" {
		dir = "./config"
	}
	return filepath.Join(dir, config.ConfigFileName)
}

// logLevelFromEnv returns the slog level from PA_LOG_LEVEL (e.g. "debug", "info"); default is INFO (REQ-01.021).
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
	clearContextOnStart := flag.Bool("clear-context-on-start", false, "Clear conversation context index (vec_items) before starting the bot; does not affect vec_tools or memory files")
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
		if err := runVerifyNodes(cfg, *verifyNodesCommand, logger); err != nil {
			logger.Error("verify-nodes", "error", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *summarizeFlag != "" {
		if err := runSummarize(cfg, *summarizeFlag, logger); err != nil {
			logger.Error("summarize", "error", err)
			os.Exit(1)
		}
		return
	}

	if *clearContextOnStart {
		if err := clearConversationContext(cfg); err != nil {
			logger.Error("clear-context-on-start", "error", err)
			os.Exit(1)
		}
		logger.Info("cleared conversation context index (vec_items) before start")
	}

	if err := runServer(cfg, configFilePath, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("run", "error", err)
		os.Exit(1)
	}
}

// runServer sets up adapter, stores, LLM, scheduler and runs the core until context is canceled.
func runServer(cfg *config.Config, configPath string, logger *slog.Logger) error {
	adapter, memoryStore, vectorStore, embedder, nodeRunner, toolIndex, err := setup(cfg, configPath, logger)
	if err != nil {
		return err
	}
	defer func() {
		if vectorStore != nil {
			if closeErr := vectorStore.Close(); closeErr != nil {
				logger.Error("close vector store", "error", closeErr)
			}
		}
		if toolIndex != nil {
			if closeErr := toolIndex.Close(); closeErr != nil {
				logger.Error("close tool index", "error", closeErr)
			}
		}
	}()

	llmProviders, llmLabels, err := newLLMForConversation(cfg)
	if err != nil {
		return err
	}
	logLLMStartupInfo(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(tools.NewRunOnNode(nodeRunner))
	if cleanup := startSchedulerIfConfigured(cfg, adapter, toolRegistry, logger); cleanup != nil {
		defer cleanup()
	}

	logger.Info("starting", "adapter", "telegram")
	var ti core.ToolIndex = toolIndex
	return core.Run(ctx, cfg, logger, adapter, llmProviders, llmLabels, memoryStore, vectorStore, embedder, nodeRunner, ti)
}

func logLLMStartupInfo(cfg *config.Config, logger *slog.Logger) {
	model := cfg.LLMProviders[0].Model
	if model == "" {
		model = "default"
	}
	if esc := cfg.ToolsLLMEscalation(); esc != nil && esc.Enabled {
		bi := esc.BaselineIndex
		if bi >= 0 && bi < len(cfg.LLMProviders) {
			model = cfg.LLMProviders[bi].Model
			if model == "" {
				model = "default"
			}
		}
		logger.Info("llm escalation", "enabled", true, "baseline_index", bi, "model", model)
		return
	}
	logger.Info("llm model", "model", model)
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

// runSummarize parses -summarize value (YYYY, YYYY-MM, or YYYY-MM-DD), runs the matching summarization. Caller exits on error.
func runSummarize(cfg *config.Config, value string, logger *slog.Logger) error {
	scope, err := summarize.ParseSummarizeScope(value)
	if err != nil {
		return err
	}
	if err := llmlog.PruneRetention(cfg.Paths.LLMLogDir, cfg.Paths.LLMLogRetentionDays, logger); err != nil {
		logger.Error("prune llm logs", "error", err)
	}
	switch scope.Kind {
	case "day":
		return runSummarizeDay(cfg, scope.Day, logger)
	case "month":
		return runSummarizeMonth(cfg, scope.Year, scope.Month, logger)
	case "year":
		return runSummarizeYear(cfg, scope.Year, logger)
	default:
		return fmt.Errorf("summarize: unknown scope %q", scope.Kind)
	}
}

// runSummarizeDay runs day summarization for the given date (UTC). Caller exits on error.
func runSummarizeDay(cfg *config.Config, day time.Time, logger *slog.Logger) error {
	if err := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); err != nil {
		return fmt.Errorf("summarize: mkdir memory: %w", err)
	}
	memoryStore, err := memory.NewStore(cfg.Paths.MemoryDir)
	if err != nil {
		return fmt.Errorf("summarize: memory store: %w", err)
	}

	llmProvider, err := newLLMProvider(cfg, logger)
	if err != nil {
		return fmt.Errorf("summarize: llm provider: %w", err)
	}

	embedder, err := embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return fmt.Errorf("summarize: embedder: %w", err)
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		return fmt.Errorf("summarize: mkdir vector: %w", err)
	}
	vectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableMemory)
	if err != nil {
		return fmt.Errorf("summarize: vector store: %w", err)
	}
	defer func() {
		if closeErr := vectorStore.Close(); closeErr != nil {
			logger.Error("summarize: close vector store", "error", closeErr)
		}
	}()

	ctx := context.Background()
	if err := summarize.Day(ctx, day, summarize.DayConfig{
		LLMLogDir:   cfg.Paths.LLMLogDir,
		LLMProvider: llmProvider,
		MemoryStore: memoryStore,
		Embedder:    embedder,
		VectorStore: vectorStore,
		Logger:      logger,
	}); err != nil {
		return fmt.Errorf("summarize: %w", err)
	}
	return nil
}

// runSummarizeMonth runs month summarization for the given year/month (UTC). Caller exits on error.
func runSummarizeMonth(cfg *config.Config, year int, month int, logger *slog.Logger) error {
	if err := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); err != nil {
		return fmt.Errorf("summarize: mkdir memory: %w", err)
	}
	memoryStore, err := memory.NewStore(cfg.Paths.MemoryDir)
	if err != nil {
		return fmt.Errorf("summarize: memory store: %w", err)
	}

	llmProvider, err := newLLMProvider(cfg, logger)
	if err != nil {
		return fmt.Errorf("summarize: llm provider: %w", err)
	}

	embedder, err := embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return fmt.Errorf("summarize: embedder: %w", err)
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		return fmt.Errorf("summarize: mkdir vector: %w", err)
	}
	vectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableMemory)
	if err != nil {
		return fmt.Errorf("summarize: vector store: %w", err)
	}
	defer func() {
		if closeErr := vectorStore.Close(); closeErr != nil {
			logger.Error("summarize: close vector store", "error", closeErr)
		}
	}()

	ctx := context.Background()
	if err := summarize.Month(ctx, year, month, summarize.MonthConfig{
		LLMProvider: llmProvider,
		MemoryStore: memoryStore,
		Embedder:    embedder,
		VectorStore: vectorStore,
		Logger:      logger,
	}); err != nil {
		return fmt.Errorf("summarize: %w", err)
	}
	return nil
}

// runSummarizeYear runs year summarization for the given year. Caller exits on error.
func runSummarizeYear(cfg *config.Config, year int, logger *slog.Logger) error {
	if err := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); err != nil {
		return fmt.Errorf("summarize: mkdir memory: %w", err)
	}
	memoryStore, err := memory.NewStore(cfg.Paths.MemoryDir)
	if err != nil {
		return fmt.Errorf("summarize: memory store: %w", err)
	}

	llmProvider, err := newLLMProvider(cfg, logger)
	if err != nil {
		return fmt.Errorf("summarize: llm provider: %w", err)
	}

	embedder, err := embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return fmt.Errorf("summarize: embedder: %w", err)
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		return fmt.Errorf("summarize: mkdir vector: %w", err)
	}
	vectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableMemory)
	if err != nil {
		return fmt.Errorf("summarize: vector store: %w", err)
	}
	defer func() {
		if closeErr := vectorStore.Close(); closeErr != nil {
			logger.Error("summarize: close vector store", "error", closeErr)
		}
	}()

	ctx := context.Background()
	if err := summarize.Year(ctx, year, summarize.YearConfig{
		LLMProvider: llmProvider,
		MemoryStore: memoryStore,
		Embedder:    embedder,
		VectorStore: vectorStore,
		Logger:      logger,
	}); err != nil {
		return fmt.Errorf("summarize: %w", err)
	}
	return nil
}

// runVerifyNodes loads allowlist and NodeRunner, runs one allowlisted command on each configured node, reports success or failure (REQ-01.022, AC-01.032). Returns error on allowlist load failure or any node failure.
func runVerifyNodes(cfg *config.Config, command string, logger *slog.Logger) error {
	if len(cfg.Nodes) == 0 {
		logger.Info("no nodes in config, nothing to verify")
		return nil
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		return fmt.Errorf("allowlist: %w", err)
	}
	runner := noderunner.New(cfg, al, logger)
	runner.SetLogRedactor(core.BuildLogRedactor(cfg))
	ctx := context.Background()
	for nodeID := range cfg.Nodes {
		logger.Info("verify node", "node_id", nodeID, "command", command)
		stdout, err := runner.RunOnNode(ctx, nodeID, command)
		if err != nil {
			return fmt.Errorf("node %q: %w", nodeID, err)
		}
		logger.Info("node OK", "node_id", nodeID)
		if stdout != "" {
			logger.Info("node output", "node_id", nodeID, "stdout", stdout)
		}
	}
	return nil
}

// newToolIndex creates the tool vector store and starts building the index from the catalog in the background. Caller must close the returned Index.
// Requires cfg.ToolCatalog to be non-nil; returns error otherwise. Logs INFO on success, ERROR on build failure.
func newToolIndex(cfg *config.Config, embedder embedding.Embedder, logger *slog.Logger) (*toolindex.Index, error) {
	if cfg.ToolCatalog == nil {
		return nil, fmt.Errorf("tool catalog is required")
	}
	toolVectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableTools)
	if err != nil {
		return nil, err
	}
	idx := toolindex.NewIndex(toolVectorStore)
	catalog := cfg.ToolCatalog
	go func() {
		err := idx.BuildAndSetReady(context.Background(), catalog, embedder)
		toolindex.LogBuildOutcome(logger, len(catalog.Tools), err)
	}()
	return idx, nil
}

// setup creates adapter, memory store, vector store, embedder, and optional node runner from config. Caller must close vectorStore.
func setup(cfg *config.Config, configPath string, logger *slog.Logger) (
	adapter core.Adapter,
	memoryStore *memory.Store,
	vectorStore *sqlite.Store,
	embedder embedding.Embedder,
	nodeRunner core.NodeRunner,
	toolIndex *toolindex.Index,
	err error,
) {
	adapter, err = telegram.NewAdapter(cfg, configPath)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	if cfg.Paths.MemoryDir != "" {
		if mkErr := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); mkErr != nil {
			return nil, nil, nil, nil, nil, nil, mkErr
		}
		memoryStore, err = memory.NewStore(cfg.Paths.MemoryDir)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
	}

	embedder, err = embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if mkErr := os.MkdirAll(vecDir, 0o755); mkErr != nil {
		return nil, nil, nil, nil, nil, nil, mkErr
	}
	vectorStore, err = sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableMemory)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	toolIndex, err = newToolIndex(cfg, embedder, logger)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	if len(cfg.Nodes) > 0 {
		al, alErr := allowlist.NewChecker(cfg)
		if alErr != nil {
			return nil, nil, nil, nil, nil, nil, alErr
		}
		nr := noderunner.New(cfg, al, logger)
		nr.SetLogRedactor(core.BuildLogRedactor(cfg))
		nodeRunner = nr
	}

	return adapter, memoryStore, vectorStore, embedder, nodeRunner, toolIndex, nil
}

// clearConversationContext deletes all rows from vec_items (semantic context for the LLM). vec_tools and memory/ files are unchanged.
func clearConversationContext(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("clear context: config is nil")
	}
	if cfg.Embedding == nil || cfg.Embedding.Dimensions <= 0 {
		return fmt.Errorf("clear context: embedding.dimensions must be positive")
	}
	if strings.TrimSpace(cfg.Paths.VectorIndexPath) == "" {
		return fmt.Errorf("clear context: paths.vector_index_path is required")
	}
	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		return fmt.Errorf("clear context: mkdir: %w", err)
	}
	store, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableMemory)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()
	return store.Clear(context.Background())
}
