package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
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
	"pa/internal/memoryjob"
	"pa/internal/noderunner"
	"pa/internal/scheduler"
	"pa/internal/skillindex"
	"pa/internal/ssh"
	"pa/internal/summarize"
	"pa/internal/telegram"
	"pa/internal/toolindex"
	"pa/internal/tools"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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

// buildAppLLM constructs one set of LLM providers from cfg and a summarize-only adapter backed by the same
// provider slice. Conversation (core) and summarization (memoryjob, -summarize CLI) share these instances;
// each llm.Provider uses an http.Client safe for concurrent use.
func buildAppLLM(cfg *config.Config, logger *slog.Logger) ([]llm.Provider, []string, llm.Provider, error) {
	providers, labels, err := buildLLMProviders(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	summarizeLLM, err := llmrouter.NewProviderAdapter(providers, labels, llmrouter.SummarizeRouterConfig(cfg), logger)
	if err != nil {
		return nil, nil, nil, err
	}
	return providers, labels, summarizeLLM, nil
}

// configFilePath returns the path to the main config file: PA_CONFIG_DIR (default "./.config") joined with config.ConfigFileName.
func configFilePath() string {
	dir := os.Getenv("PA_CONFIG_DIR")
	if dir == "" {
		dir = "./.config"
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
	clearContextOnStart := flag.Bool("clear-context-on-start", false, "Clear conversation turn context index (vec_turns) before starting the bot; does not affect vec_tools or memory files")
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
		logger.Info("cleared conversation context (vec_turns) before start")
	}

	if err := runServer(cfg, configFilePath, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("run", "error", err)
		os.Exit(1)
	}
}

// runServer sets up adapter, stores, LLM, scheduler and runs the core until context is canceled.
//
//nolint:gocyclo // sequential startup wiring and teardown branches
func runServer(cfg *config.Config, configPath string, logger *slog.Logger) error {
	adapter, memoryStore, memVec, embedder, nodeRunner, toolIndex, skillIndex, err := setup(cfg, configPath, logger)
	if err != nil {
		return err
	}
	warnIfNodesSSHUnreachable(context.Background(), cfg, logger)
	defer func() {
		if memVec != nil {
			if closeErr := memVec.Close(); closeErr != nil {
				logger.Error("close memory vector stores", "error", closeErr)
			}
		}
		if toolIndex != nil {
			if closeErr := toolIndex.Close(); closeErr != nil {
				logger.Error("close tool index", "error", closeErr)
			}
		}
		if skillIndex != nil {
			if closeErr := skillIndex.Close(); closeErr != nil {
				logger.Error("close skill index", "error", closeErr)
			}
		}
	}()

	llmProviders, llmLabels, summarizeLLM, err := buildAppLLM(cfg, logger)
	if err != nil {
		return err
	}
	logLLMStartupInfo(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var memJob *memoryjob.Runner
	if memoryStore != nil && strings.TrimSpace(cfg.Paths.LLMLogDir) != "" &&
		embedder != nil && memVec != nil && memVec.Summaries != nil {
		paLoc, locErr := config.PALocation(cfg)
		if locErr != nil {
			return fmt.Errorf("memory summarization pa_timezone: %w", locErr)
		}
		memJob = memoryjob.Start(ctx, memoryjob.Deps{
			Cfg:         cfg,
			Loc:         paLoc,
			Memory:      memoryStore,
			Vector:      memVec.Summaries,
			Embedder:    embedder,
			LLMProvider: summarizeLLM,
			Logger:      logger,
		})
		logger.Info("memory summarization worker started")
	}

	toolRegistry := tools.NewRegistry()
	if nodeRunner != nil {
		toolRegistry.Register(tools.NewRunOnNode(nodeRunner))
	}
	absCatalog, err := filepath.Abs(cfg.Paths.ToolCatalogPath)
	if err != nil {
		return fmt.Errorf("tool catalog path: %w", err)
	}
	var createToolMu sync.Mutex
	if cfg.ToolCatalog != nil {
		toolRegistry.Register(tools.NewCreateTool(&createToolMu, cfg.ToolCatalog, absCatalog, cfg, embedder, toolIndex, logger))
	}
	registerWebToolsIfEnabled(cfg, toolRegistry, logger)
	if err := registerMemoryToolsIfEnabled(cfg, toolRegistry, memoryStore, memVec, embedder); err != nil {
		return err
	}
	if cleanup := startSchedulerIfConfigured(cfg, adapter, toolRegistry, logger); cleanup != nil {
		defer cleanup()
	}

	defer func() {
		if memJob != nil {
			memJob.Stop()
			<-memJob.Done()
		}
	}()

	logger.Info("starting", "adapter", "telegram")
	var ti core.ToolIndex = toolIndex
	var si core.SkillIndex = skillIndex
	return core.Run(ctx, cfg, logger, adapter, llmProviders, llmLabels, memoryStore, memVec, embedder, nodeRunner, ti, si, toolRegistry)
}

const sshStartupCheckTimeout = 5 * time.Second

// warnIfNodesSSHUnreachable tries SSH dial and handshake for each configured node; logs a warning on failure and never returns an error.
func warnIfNodesSSHUnreachable(ctx context.Context, cfg *config.Config, logger *slog.Logger) {
	if cfg == nil || len(cfg.Nodes) == 0 {
		return
	}
	ids := make([]string, 0, len(cfg.Nodes))
	for id := range cfg.Nodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, nodeID := range ids {
		checkCtx, cancel := context.WithTimeout(ctx, sshStartupCheckTimeout)
		err := ssh.VerifyDialAndHandshake(checkCtx, cfg, nodeID)
		cancel()
		if err != nil {
			logger.Warn("ssh startup check failed", "node_id", nodeID, "error", err)
		}
	}
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
		d := scope.Day.UTC().Format("2006-01-02")
		logger.Info("summarize: starting", "scope", "day", "day", d)
		return runSummarizeDay(cfg, scope.Day, logger)
	case "month":
		logger.Info("summarize: starting", "scope", "month", "year", scope.Year, "month", scope.Month)
		return runSummarizeMonth(cfg, scope.Year, scope.Month, logger)
	case "year":
		logger.Info("summarize: starting", "scope", "year", "year", scope.Year)
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
	loc, err := config.PALocation(cfg)
	if err != nil {
		return fmt.Errorf("summarize: pa_timezone: %w", err)
	}
	logger.Info("summarize: opening memory store", "memory_dir", cfg.Paths.MemoryDir)
	memoryStore, err := memory.NewStore(cfg.Paths.MemoryDir, loc)
	if err != nil {
		return fmt.Errorf("summarize: memory store: %w", err)
	}

	logger.Info("summarize: building llm provider")
	_, _, llmProvider, err := buildAppLLM(cfg, logger)
	if err != nil {
		return fmt.Errorf("summarize: llm provider: %w", err)
	}

	logger.Info("summarize: building embedder")
	embedder, err := embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return fmt.Errorf("summarize: embedder: %w", err)
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		return fmt.Errorf("summarize: mkdir vector: %w", err)
	}
	logger.Info("summarize: opening vector store", "path", cfg.Paths.VectorIndexPath)
	vectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableSummaries)
	if err != nil {
		return fmt.Errorf("summarize: vector store: %w", err)
	}
	defer func() {
		if closeErr := vectorStore.Close(); closeErr != nil {
			logger.Error("summarize: close vector store", "error", closeErr)
		}
	}()

	dayCal := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc)
	dayStr := dayCal.In(loc).Format("2006-01-02")
	logger.Info("summarize: running day pipeline", "day", dayStr, "llm_log_dir", cfg.Paths.LLMLogDir)
	ctx := context.Background()
	if err := summarize.Day(ctx, dayCal, summarize.DayConfig{
		LLMLogDir:   cfg.Paths.LLMLogDir,
		LLMProvider: llmProvider,
		MemoryStore: memoryStore,
		Embedder:    embedder,
		VectorStore: vectorStore,
		Logger:      logger,
		Loc:         loc,
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
	loc, err := config.PALocation(cfg)
	if err != nil {
		return fmt.Errorf("summarize: pa_timezone: %w", err)
	}
	logger.Info("summarize: opening memory store", "memory_dir", cfg.Paths.MemoryDir)
	memoryStore, err := memory.NewStore(cfg.Paths.MemoryDir, loc)
	if err != nil {
		return fmt.Errorf("summarize: memory store: %w", err)
	}

	logger.Info("summarize: building llm provider")
	_, _, llmProvider, err := buildAppLLM(cfg, logger)
	if err != nil {
		return fmt.Errorf("summarize: llm provider: %w", err)
	}

	logger.Info("summarize: building embedder")
	embedder, err := embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return fmt.Errorf("summarize: embedder: %w", err)
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		return fmt.Errorf("summarize: mkdir vector: %w", err)
	}
	logger.Info("summarize: opening vector store", "path", cfg.Paths.VectorIndexPath)
	vectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableSummaries)
	if err != nil {
		return fmt.Errorf("summarize: vector store: %w", err)
	}
	defer func() {
		if closeErr := vectorStore.Close(); closeErr != nil {
			logger.Error("summarize: close vector store", "error", closeErr)
		}
	}()

	logger.Info("summarize: running month pipeline", "year", year, "month", month)
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
	loc, err := config.PALocation(cfg)
	if err != nil {
		return fmt.Errorf("summarize: pa_timezone: %w", err)
	}
	logger.Info("summarize: opening memory store", "memory_dir", cfg.Paths.MemoryDir)
	memoryStore, err := memory.NewStore(cfg.Paths.MemoryDir, loc)
	if err != nil {
		return fmt.Errorf("summarize: memory store: %w", err)
	}

	logger.Info("summarize: building llm provider")
	_, _, llmProvider, err := buildAppLLM(cfg, logger)
	if err != nil {
		return fmt.Errorf("summarize: llm provider: %w", err)
	}

	logger.Info("summarize: building embedder")
	embedder, err := embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return fmt.Errorf("summarize: embedder: %w", err)
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if err := os.MkdirAll(vecDir, 0o755); err != nil {
		return fmt.Errorf("summarize: mkdir vector: %w", err)
	}
	logger.Info("summarize: opening vector store", "path", cfg.Paths.VectorIndexPath)
	vectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableSummaries)
	if err != nil {
		return fmt.Errorf("summarize: vector store: %w", err)
	}
	defer func() {
		if closeErr := vectorStore.Close(); closeErr != nil {
			logger.Error("summarize: close vector store", "error", closeErr)
		}
	}()

	logger.Info("summarize: running year pipeline", "year", year)
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

// newSkillIndex builds vec_skills synchronously when runtime skills are enabled (EP-013).
func newSkillIndex(cfg *config.Config, embedder embedding.Embedder, logger *slog.Logger) (*skillindex.Index, error) {
	if cfg.RuntimeSkills == nil || !cfg.RuntimeSkills.Enabled || len(cfg.RuntimeSkillPackages) == 0 {
		return nil, nil
	}
	store, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableSkills)
	if err != nil {
		return nil, err
	}
	idx := skillindex.NewIndex(store)
	if err := idx.BuildAndSetReady(context.Background(), cfg.RuntimeSkillPackages, embedder); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("skill index: %w", err)
	}
	logger.Info("runtime skill index ready", "packages", len(cfg.RuntimeSkillPackages))
	return idx, nil
}

// openMemoryVectorBundle opens EP-016 split memory tables on the configured vector DB path.
func openMemoryVectorBundle(cfg *config.Config) (*core.MemoryVectors, error) {
	dim := cfg.Embedding.Dimensions
	path := cfg.Paths.VectorIndexPath
	summ, err := sqlite.NewWithTable(path, dim, sqlite.TableSummaries)
	if err != nil {
		return nil, err
	}
	turns, err := sqlite.NewWithTable(path, dim, sqlite.TableTurns)
	if err != nil {
		_ = summ.Close()
		return nil, err
	}
	notes, err := sqlite.NewWithTable(path, dim, sqlite.TableNotes)
	if err != nil {
		_ = summ.Close()
		_ = turns.Close()
		return nil, err
	}
	return &core.MemoryVectors{
		Summaries: summ,
		Turns:     turns,
		Notes:     notes,
	}, nil
}

// setup creates adapter, memory store, memory vector bundle, embedder, and optional node runner from config. Caller must close memVec via MemoryVectors.Close.
//
//nolint:gocyclo // many optional subsystems; each branch is independent
func setup(cfg *config.Config, configPath string, logger *slog.Logger) (
	adapter core.Adapter,
	memoryStore *memory.Store,
	memVec *core.MemoryVectors,
	embedder embedding.Embedder,
	nodeRunner core.NodeRunner,
	toolIndex *toolindex.Index,
	skillIndex *skillindex.Index,
	err error,
) {
	adapter, err = telegram.NewAdapter(cfg, configPath)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	if cfg.Paths.MemoryDir != "" {
		if mkErr := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); mkErr != nil {
			return nil, nil, nil, nil, nil, nil, nil, mkErr
		}
		loc, locErr := config.PALocation(cfg)
		if locErr != nil {
			return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("memory store timezone: %w", locErr)
		}
		memoryStore, err = memory.NewStore(cfg.Paths.MemoryDir, loc)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, err
		}
	}

	embedder, err = embedding.NewEmbedder(cfg.Embedding)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	if mkErr := os.MkdirAll(vecDir, 0o755); mkErr != nil {
		return nil, nil, nil, nil, nil, nil, nil, mkErr
	}
	memVec, err = openMemoryVectorBundle(cfg)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	toolIndex, err = newToolIndex(cfg, embedder, logger)
	if err != nil {
		if memVec != nil {
			_ = memVec.Close()
		}
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	skillIndex, err = newSkillIndex(cfg, embedder, logger)
	if err != nil {
		if toolIndex != nil {
			_ = toolIndex.Close()
		}
		if memVec != nil {
			_ = memVec.Close()
		}
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	if len(cfg.Nodes) > 0 {
		al, alErr := allowlist.NewChecker(cfg)
		if alErr != nil {
			return nil, nil, nil, nil, nil, nil, nil, alErr
		}
		nr := noderunner.New(cfg, al, logger)
		nr.SetLogRedactor(core.BuildLogRedactor(cfg))
		nodeRunner = nr
	}

	return adapter, memoryStore, memVec, embedder, nodeRunner, toolIndex, skillIndex, nil
}

func registerWebToolsIfEnabled(cfg *config.Config, reg *tools.Registry, logger *slog.Logger) {
	if cfg == nil || cfg.WebTools == nil || !cfg.WebTools.Enabled {
		return
	}
	webHTTP := &http.Client{
		Timeout: 0,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	reg.Register(tools.NewWebSearchTool(cfg.WebTools, webHTTP, nil))
	reg.Register(tools.NewWebFetchTool(&cfg.WebTools.Fetch, webHTTP))
	logger.Info("web tools enabled", "search_provider", cfg.WebTools.Search.Provider)
}

func registerMemoryToolsIfEnabled(cfg *config.Config, reg *tools.Registry, memoryStore *memory.Store, memVec *core.MemoryVectors, embedder embedding.Embedder) error {
	if reg == nil {
		return fmt.Errorf("memory tools: registry is nil")
	}
	if memoryStore == nil {
		return fmt.Errorf("memory tools: memory store is required")
	}
	span, outBytes := readMemoryLimits(cfg)
	reg.Register(tools.NewReadMemoryTool(memoryStore, span, outBytes))

	// write_memory is a core feature and must be available in full mode at startup.
	if !writeMemoryRuntimeReady(memVec, embedder) {
		return fmt.Errorf("memory tools: write_memory requires notes vector and embedding provider")
	}
	maxAppend, maxFile := writeMemoryLimits(writeMemoryConfig(cfg))
	reg.Register(tools.NewWriteMemoryTool(memoryStore, memVec.Notes, embedder, maxAppend, maxFile))
	return nil
}

func readMemoryLimits(cfg *config.Config) (span, outBytes int) {
	span, outBytes = 31, 256*1024
	if cfg == nil || cfg.ReadMemory == nil {
		return span, outBytes
	}
	if cfg.ReadMemory.MaxSpanDays != 0 {
		span = cfg.ReadMemory.MaxSpanDays
	}
	if cfg.ReadMemory.MaxOutputBytes != 0 {
		outBytes = cfg.ReadMemory.MaxOutputBytes
	}
	return span, outBytes
}

func writeMemoryRuntimeReady(memVec *core.MemoryVectors, embedder embedding.Embedder) bool {
	return memVec != nil && memVec.Notes != nil && embedder != nil
}

func writeMemoryConfig(cfg *config.Config) *config.WriteMemoryConfig {
	if cfg == nil {
		return nil
	}
	return cfg.WriteMemory
}

func writeMemoryLimits(wm *config.WriteMemoryConfig) (maxAppend, maxFile int) {
	maxAppend, maxFile = config.WriteMemoryDefaultMaxAppendBytes, config.WriteMemoryDefaultMaxFileBytes
	if wm == nil {
		return maxAppend, maxFile
	}
	if wm.MaxAppendBytes != 0 {
		maxAppend = wm.MaxAppendBytes
	}
	if wm.MaxFileBytes != 0 {
		maxFile = wm.MaxFileBytes
	}
	return maxAppend, maxFile
}

// clearConversationContext deletes turn rows from vec_turns. vec_summaries, vec_notes, and vec_tools are unchanged.
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
	ctx := context.Background()
	turns, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableTurns)
	if err != nil {
		return err
	}
	defer func() { _ = turns.Close() }()
	return turns.Clear(ctx)
}
