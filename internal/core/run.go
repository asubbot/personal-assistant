package core

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/embedding"
	"pa/internal/intent"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/logredact"
	"pa/internal/memory"
	"pa/internal/runtimeskills"
	"pa/internal/tools"
	"pa/internal/vector"
)

// ToolIndex provides the tool vector store and ready state for tool pre-selection (step 3.1). Optional; when nil, no tool index is used.
type ToolIndex interface {
	Store() vector.Store
	Ready() bool
}

// SkillIndex provides vec_skills store and ready state (EP-013). Optional; when nil, runtime skills selection is skipped.
type SkillIndex interface {
	Store() vector.Store
	Ready() bool
	Close() error
}

// Run starts the application: wires the adapter to the conversation handler (LLM, vector, optional node runner) and blocks until ctx is cancelled.
// memoryStore is used by native memory tools; memVec and embedder are optional; when provided, the handler runs semantic search and indexes turns (REQ-01.006, REQ-01.007, REQ-01.018, EP-016).
// nodeRunner is optional; when provided, tools can run allowlisted commands on nodes via SSH (REQ-01.004, REQ-01.005, REQ-01.013).
// toolIndex is optional; when provided and Ready(), step 3.1 will use it for tool pre-selection.
// providers and providerLabels define the ordered LLM chain used by the unified router.
// skillIndex is optional; when provided and Ready(), runtime skills are selected per message (EP-013).
// nativeRegistry is optional; when non-nil, native tools (e.g. run_on_node, create_tool) are merged into LLM tool lists and dispatch.
// classifier is optional (EP-017); when non-nil, classifies messages before prompt assembly.
func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger, adapter Adapter, providers []llm.Provider, providerLabels []string, memoryStore *memory.Store, memVec *MemoryVectors, embedder embedding.Embedder, nodeRunner NodeRunner, toolIndex ToolIndex, skillIndex SkillIndex, nativeRegistry *tools.Registry, classifier intent.Classifier) error {
	if adapter == nil {
		return fmt.Errorf("core: adapter is nil")
	}
	if len(providers) == 0 {
		return fmt.Errorf("core: providers are required")
	}
	if len(providerLabels) != len(providers) {
		return fmt.Errorf("core: provider labels length %d != providers length %d", len(providerLabels), len(providers))
	}
	redactor := buildRedactor(cfg)
	router, err := llmrouter.New(providers, providerLabels, llmrouter.Config{}, logger)
	if err != nil {
		return err
	}
	handler, err := BuildMessageHandler(cfg, logger, redactor, router, memVec, embedder, nodeRunner, toolIndex, skillIndex, nativeRegistry, classifier)
	if err != nil {
		return err
	}
	return adapter.Run(ctx, handler)
}

// BuildMessageHandler constructs a configured conversational message handler.
func BuildMessageHandler(cfg *config.Config, logger *slog.Logger, redactor func(string) string, router *llmrouter.Router, memVec *MemoryVectors, embedder embedding.Embedder, nodeRunner NodeRunner, toolIndex ToolIndex, skillIndex SkillIndex, nativeRegistry *tools.Registry, classifier intent.Classifier) (MessageHandler, error) {
	return newRunConversationHandler(cfg, logger, redactor, router, memVec, embedder, nodeRunner, toolIndex, skillIndex, nativeRegistry, classifier)
}

func newRunConversationHandler(cfg *config.Config, logger *slog.Logger, redactor func(string) string, router *llmrouter.Router, memVec *MemoryVectors, embedder embedding.Embedder, nodeRunner NodeRunner, toolIndex ToolIndex, skillIndex SkillIndex, nativeRegistry *tools.Registry, classifier intent.Classifier) (*conversationHandler, error) {
	maxLen := 0
	if cfg != nil && cfg.Telegram.MaxMessageLength > 0 {
		maxLen = cfg.Telegram.MaxMessageLength
	}
	llmLog, model, err := openLLMLogIfConfigured(cfg, logger, redactor)
	if err != nil {
		return nil, err
	}
	var maxDynRunes int
	var memVecTopK config.MemoryVectorConfig
	var toolTopK, toolMin, toolCap int
	if cfg != nil {
		maxDynRunes, memVecTopK = conversationContextParams(cfg)
		toolTopK, toolMin, toolCap = toolPreSelectionParams(cfg)
	} else {
		// core.Run allows nil config only for narrow tests; match historical implicit defaults.
		maxDynRunes = 4000
		memVecTopK = config.MemoryVectorConfig{NotesTopK: 10, SummariesTopK: 10, TurnsTopK: 10}
		toolTopK, toolMin, toolCap = 10, 1, 50
	}
	firstSupportsTools := baselineProviderSupportsTools(cfg)
	byID := make(map[string]*runtimeskills.Package)
	var rs *config.RuntimeSkillsConfig
	var tc *config.ToolsConfig
	var toolDynSel *config.ToolDynamicSelection
	var sessCfg *config.ConversationSessionConfig
	var sessStore *sessionWindowStore
	paLoc := paLocationFromConfig(cfg)
	if cfg != nil {
		rs = cfg.RuntimeSkills
		tc = cfg.Tools
		if tc != nil {
			toolDynSel = tc.DynamicSelection
		}
		for _, p := range cfg.RuntimeSkillPackages {
			byID[p.ID] = p
		}
		if cfg.ConversationSession != nil && cfg.ConversationSession.Enabled {
			sessCfg = cfg.ConversationSession
			sessStore = newSessionWindowStore()
		}
	}
	h := &conversationHandler{
		router:                     router,
		memVec:                     memVec,
		embedder:                   embedder,
		nodeRunner:                 nodeRunner,
		toolIndex:                  toolIndex,
		skillIndex:                 skillIndex,
		runtimeSkillsCfg:           rs,
		toolsCfg:                   tc,
		skillPackagesByID:          byID,
		nativeRegistry:             nativeRegistry,
		toolSearchTopK:             toolTopK,
		toolMinCount:               toolMin,
		toolFallbackCap:            toolCap,
		logger:                     logger,
		maxMessageLength:           maxLen,
		maxDynamicSystemRunes:      maxDynRunes,
		memoryVectorTopK:           memVecTopK,
		llmLog:                     llmLog,
		model:                      model,
		logRedactor:                redactor,
		firstProviderSupportsTools: firstSupportsTools,
		sessionCfg:                 sessCfg,
		sessionStore:               sessStore,
		paLoc:                      paLoc,
		classifier:                 classifier,
		toolsDynamic:               toolDynSel,
	}
	if cfg != nil && cfg.ToolCatalog != nil {
		h.catalog = cfg.ToolCatalog
	}
	return h, nil
}

func openLLMLogIfConfigured(cfg *config.Config, logger *slog.Logger, redactor func(string) string) (llmlog.Writer, string, error) {
	if cfg == nil || cfg.Paths.LLMLogDir == "" {
		return nil, "", nil
	}
	calLoc, _ := config.PALocation(cfg)
	w, err := llmlog.NewWriter(cfg.Paths.LLMLogDir, logger, llmlog.Redactor(redactor), calLoc)
	if err != nil {
		return nil, "", fmt.Errorf("core: llm log writer: %w", err)
	}
	model := ""
	if len(cfg.LLMProviders) > 0 {
		model = cfg.LLMProviders[0].Model
	}
	return w, model, nil
}

func baselineProviderSupportsTools(cfg *config.Config) bool {
	if cfg == nil {
		return true
	}
	if len(cfg.LLMProviders) > 0 && cfg.LLMProviders[0].SupportsTools != nil {
		return *cfg.LLMProviders[0].SupportsTools
	}
	return true
}

// toolPreSelectionParams returns tool pre-selection from config (validated at config.Load).
func toolPreSelectionParams(cfg *config.Config) (topK, minCount, fallbackCap int) {
	return cfg.ToolPreSelection.ToolSearchTopK, cfg.ToolPreSelection.ToolMinCount, cfg.ToolPreSelection.ToolFallbackCap
}

// conversationContextParams returns conversation context limits from config (validated at config.Load).
func conversationContextParams(cfg *config.Config) (maxDynamicSystemRunes int, memVec config.MemoryVectorConfig) {
	cc := cfg.ConversationContext
	return cc.MaxDynamicSystemRunes, cc.MemoryVector
}

// buildRedactor returns a redactor from built-in patterns plus config additional_patterns (REQ-01.027, REQ-01.028).
func buildRedactor(cfg *config.Config) func(string) string {
	var additional []logredact.Pattern
	if cfg != nil && cfg.LogRedaction != nil {
		for _, p := range cfg.LogRedaction.AdditionalPatterns {
			additional = append(additional, logredact.Pattern{ID: p.ID, Regex: p.Regex, Replacement: p.Replacement})
		}
	}
	return logredact.NewRedactor(additional)
}

// BuildLogRedactor returns the application log redactor from config (same as handler and tool-invocation INFO logs). Callers such as cmd/pa may attach it to noderunner for consistent redaction of remote stream fragments in app logs (REQ-01.026).
func BuildLogRedactor(cfg *config.Config) func(string) string {
	return buildRedactor(cfg)
}
