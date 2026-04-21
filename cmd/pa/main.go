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
	"pa/internal/intent"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/memory"
	"pa/internal/noderunner"
	"pa/internal/skillindex"
	"pa/internal/ssh"
	"pa/internal/summarize"
	"pa/internal/toolindex"
	"pa/internal/tools"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"slices"
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

func paEnvIsDevelopment() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PA_ENV")), "development")
}

// warnSensitiveLLMLogging emits one WARN when application logging is at debug and PA_ENV
// is not set to "development" (REQ-24.008). Full LLM bodies in logs are enabled at debug.
func warnSensitiveLLMLogging(logger *slog.Logger, level slog.Level) {
	if logger == nil || level != slog.LevelDebug {
		return
	}
	if paEnvIsDevelopment() {
		return
	}
	logger.Warn("sensitive diagnostic logging: PA_LOG_LEVEL=debug may log full LLM request and response bodies in application logs; treat logs as highly sensitive. Use PA_LOG_LEVEL=info in production, or set PA_ENV=development only on trusted diagnostic hosts.")
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
	warnSensitiveLLMLogging(logger, logLevel)
	cfg, err := config.Load(configFilePath)
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	logger.Info("config loaded", "path", configFilePath)
	warnBaselineOmitsNativeToolsWithCatalog(cfg, logger)

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

// runServer constructs the application from config, then runs the adapter until context is canceled.
func runServer(cfg *config.Config, configPath string, logger *slog.Logger) error {
	app, err := newPAApplication(cfg, configPath, logger)
	if err != nil {
		return err
	}
	defer app.Close()
	defer app.stopMemorySummarization()

	warnIfNodesSSHUnreachable(context.Background(), cfg, logger)

	if err := app.startLLMProviders(); err != nil {
		return err
	}
	logLLMStartupInfo(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.maybeStartMemorySummarization(ctx); err != nil {
		return err
	}

	toolRegistry, err := app.buildToolRegistry()
	if err != nil {
		return err
	}
	handler, err := app.buildMessageHandler(ctx, toolRegistry)
	if err != nil {
		return err
	}

	shutdownObservability := startObservabilityHTTPServer(ctx, cfg, app, logger)
	defer shutdownObservability()

	logger.Info("starting", "adapter", "telegram", "model", mainConversationModelName(cfg))
	return app.infra.Adapter.Run(ctx, handler)
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

func baselineProviderSupportsTools(cfg *config.Config) bool {
	if cfg == nil {
		return true
	}
	idx := 0
	if esc := cfg.ToolsLLMEscalation(); esc != nil && esc.Enabled && esc.BaselineIndex < len(cfg.LLMProviders) {
		idx = esc.BaselineIndex
	}
	if len(cfg.LLMProviders) > idx && cfg.LLMProviders[idx].SupportsTools != nil {
		return *cfg.LLMProviders[idx].SupportsTools
	}
	return true
}

// warnBaselineOmitsNativeToolsWithCatalog logs once when the baseline LLM omits native tools while the catalog defines tools (REQ-30.009).
func warnBaselineOmitsNativeToolsWithCatalog(cfg *config.Config, logger *slog.Logger) {
	if cfg == nil || logger == nil {
		return
	}
	if baselineProviderSupportsTools(cfg) {
		return
	}
	if cfg.ToolCatalog == nil || len(cfg.ToolCatalog.Tools) == 0 {
		return
	}
	logger.Warn("native tool calling is disabled for the baseline LLM (supports_tools false) while the tool catalog defines tools; conversation tools will not run in completion requests")
}

// mainConversationModelName returns the configured chat model id for the active baseline provider
// (first provider, or tools.llm_escalation.baseline_index when escalation is enabled).
func mainConversationModelName(cfg *config.Config) string {
	if cfg == nil || len(cfg.LLMProviders) == 0 {
		return "unknown"
	}
	idx := 0
	if esc := cfg.ToolsLLMEscalation(); esc != nil && esc.Enabled {
		bi := esc.BaselineIndex
		if bi >= 0 && bi < len(cfg.LLMProviders) {
			idx = bi
		}
	}
	m := cfg.LLMProviders[idx].Model
	if m == "" {
		return "default"
	}
	return m
}

func logLLMStartupInfo(cfg *config.Config, logger *slog.Logger) {
	model := mainConversationModelName(cfg)
	if esc := cfg.ToolsLLMEscalation(); esc != nil && esc.Enabled {
		bi := esc.BaselineIndex
		logger.Info("llm escalation", "enabled", true, "baseline_index", bi, "model", model)
		return
	}
	logger.Info("llm model", "model", model)
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
	vectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableSummaries, cfg.VectorStoreReliability.ToPolicy())
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
	vectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableSummaries, cfg.VectorStoreReliability.ToPolicy())
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
	vectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableSummaries, cfg.VectorStoreReliability.ToPolicy())
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
	toolVectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableTools, cfg.VectorStoreReliability.ToPolicy())
	if err != nil {
		return nil, err
	}
	idx := toolindex.NewIndex(toolVectorStore)
	catalog := cfg.ToolCatalog
	go func() {
		t0 := time.Now()
		err := idx.BuildAndSetReady(context.Background(), catalog, embedder)
		toolindex.LogBuildOutcome(logger, len(catalog.Tools), time.Since(t0), err)
	}()
	return idx, nil
}

// newSkillIndex builds vec_skills synchronously when runtime skills are enabled (EP-013).
func newSkillIndex(cfg *config.Config, embedder embedding.Embedder, logger *slog.Logger) (*skillindex.Index, error) {
	if cfg.RuntimeSkills == nil || !cfg.RuntimeSkills.Enabled || len(cfg.RuntimeSkillPackages) == 0 {
		return nil, nil
	}
	store, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableSkills, cfg.VectorStoreReliability.ToPolicy())
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
	vecPolicy := cfg.VectorStoreReliability.ToPolicy()
	summ, err := sqlite.NewWithTable(path, dim, sqlite.TableSummaries, vecPolicy)
	if err != nil {
		return nil, err
	}
	turns, err := sqlite.NewWithTable(path, dim, sqlite.TableTurns, vecPolicy)
	if err != nil {
		_ = summ.Close()
		return nil, err
	}
	notes, err := sqlite.NewWithTable(path, dim, sqlite.TableNotes, vecPolicy)
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

func registerWebToolsIfEnabled(cfg *config.Config, reg *tools.Registry, logger *slog.Logger) error {
	if cfg == nil || cfg.WebTools == nil || !cfg.WebTools.Enabled {
		return nil
	}
	timeout, err := time.ParseDuration(strings.TrimSpace(cfg.WebTools.HTTPTimeout))
	if err != nil || timeout <= 0 {
		return fmt.Errorf("web tools: invalid http_timeout %q (validated at config.Load; unreachable unless bypassed)", cfg.WebTools.HTTPTimeout)
	}
	webHTTP := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	reg.Register(tools.NewWebSearchTool(cfg.WebTools, webHTTP, nil))
	reg.Register(tools.NewWebFetchTool(&cfg.WebTools.Fetch, webHTTP))
	logger.Info("web tools enabled", "search_provider", cfg.WebTools.Search.Provider, "http_timeout", timeout)
	return nil
}

func registerMemoryToolsIfEnabled(cfg *config.Config, reg *tools.Registry, memoryStore *memory.Store, memVec *core.MemoryVectors, embedder embedding.Embedder) error {
	if reg == nil {
		return fmt.Errorf("memory tools: registry is nil")
	}
	if memoryStore == nil {
		return fmt.Errorf("memory tools: memory store is required")
	}
	if cfg.ReadMemory == nil || cfg.WriteMemory == nil {
		return fmt.Errorf("memory tools: read_memory and write_memory are required in config")
	}
	rm := cfg.ReadMemory
	wm := cfg.WriteMemory
	reg.Register(tools.NewReadMemoryTool(memoryStore, rm.MaxSpanDays, rm.MaxOutputBytes))

	// write_memory is a core feature and must be available in full mode at startup.
	if !writeMemoryRuntimeReady(memVec, embedder) {
		return fmt.Errorf("memory tools: write_memory requires notes vector and embedding provider")
	}
	reg.Register(tools.NewWriteMemoryTool(memoryStore, memVec.Notes, embedder, wm.MaxAppendBytes, wm.MaxFileBytes))
	searchMemoryCfg := cfg.VectorSearchToolSettings("search_vector_memory")
	// search_vector_memory is read-only semantic retrieval over vector memory lanes.
	reg.Register(tools.NewSearchVectorMemoryTool(memVec.Notes, memVec.Summaries, memVec.Turns, embedder, searchMemoryCfg.DefaultTopK, searchMemoryCfg.MaxTopK, searchMemoryCfg.MaxOutputBytes, searchMemoryCfg.SnippetRunes))
	return nil
}

func writeMemoryRuntimeReady(memVec *core.MemoryVectors, embedder embedding.Embedder) bool {
	return memVec != nil && memVec.Notes != nil && embedder != nil
}

func registerKnowledgeToolsIfEnabled(cfg *config.Config, reg *tools.Registry, toolIdx *toolindex.Index, skillIdx *skillindex.Index, embedder embedding.Embedder) {
	if cfg == nil || reg == nil || embedder == nil {
		return
	}
	toolCfg := cfg.VectorSearchToolSettings("search_vector_tool")
	if toolCfg.Enabled && toolIdx != nil && toolIdx.Store() != nil {
		reg.Register(tools.NewSearchVectorToolKnowledgeTool(toolIdx.Store(), embedder, toolCfg.DefaultTopK, toolCfg.MaxTopK, toolCfg.MaxOutputBytes, toolCfg.SnippetRunes))
	}
	skillCfg := cfg.VectorSearchToolSettings("search_vector_skill")
	if skillCfg.Enabled && skillIdx != nil && skillIdx.Store() != nil {
		reg.Register(tools.NewSearchVectorSkillKnowledgeTool(skillIdx.Store(), embedder, skillCfg.DefaultTopK, skillCfg.MaxTopK, skillCfg.MaxOutputBytes, skillCfg.SnippetRunes))
	}
}

// buildIntentClassifier constructs the EP-017 cascade classifier from config. Returns nil when disabled.
func buildIntentClassifier(cfg *config.Config, logger *slog.Logger) (intent.Classifier, error) {
	ic := cfg.IntentClassifier
	if ic == nil || !ic.Enabled {
		return nil, nil
	}
	var heuristic *intent.HeuristicClassifier
	if ic.Heuristic != nil {
		heuristic = intent.NewHeuristicClassifier(
			ic.Heuristic.SimplePatterns,
			ic.Heuristic.FullPatterns,
			ic.Heuristic.FullLitePatterns,
			ic.Heuristic.MaxSimpleLen,
		)
	}
	var model *intent.ModelClassifier
	if ic.ModelStage != nil && ic.ModelStage.Enabled {
		provCfg := &config.LLMProvider{
			Type:                  ic.ModelStage.Type,
			Endpoint:              ic.ModelStage.Endpoint,
			APIKeyPath:            ic.ModelStage.APIKeyPath,
			Model:                 ic.ModelStage.Model,
			DefaultTemperature:    ic.ModelStage.DefaultTemperature,
			DefaultMaxTokens:      ic.ModelStage.DefaultMaxTokens,
			DefaultResponseFormat: "text",
			SupportsTools:         boolPtr(false),
			HTTPTimeout:           ic.ModelStage.HTTPTimeout,
		}
		provider, err := llm.NewProvider(provCfg)
		if err != nil {
			return nil, fmt.Errorf("intent classifier model provider: %w", err)
		}
		var timeout time.Duration
		if ic.ModelStage.Timeout != "" {
			var parseErr error
			timeout, parseErr = time.ParseDuration(ic.ModelStage.Timeout)
			if parseErr != nil {
				return nil, fmt.Errorf("intent classifier model timeout: %w", parseErr)
			}
		}
		model = intent.NewModelClassifier(provider, logger, timeout)
	}
	attrs := []any{"heuristic", heuristic != nil, "model_stage", model != nil}
	if ic.ModelStage != nil && ic.ModelStage.Enabled {
		if m := strings.TrimSpace(ic.ModelStage.Model); m != "" {
			attrs = append(attrs, "classifier_model", m)
		}
	}
	logger.Info("intent classifier enabled", attrs...)
	return intent.NewCascadeClassifier(heuristic, model, logger), nil
}

func boolPtr(v bool) *bool { return &v }

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
	turns, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableTurns, cfg.VectorStoreReliability.ToPolicy())
	if err != nil {
		return err
	}
	defer func() { _ = turns.Close() }()
	return turns.Clear(ctx)
}
