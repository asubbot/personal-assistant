package core

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/logredact"
	"pa/internal/memory"
	"pa/internal/vector"
)

// ToolIndex provides the tool vector store and ready state for tool pre-selection (step 3.1). Optional; when nil, no tool index is used.
type ToolIndex interface {
	Store() vector.Store
	Ready() bool
}

// Run starts the application: wires the adapter to the conversation handler (LLM, vector, optional node runner) and blocks until ctx is cancelled.
// memoryStore is not used by the handler (context is from vector only); vectorStore and embedder are optional; when provided, the handler runs semantic search and indexes turns (REQ-01.006, REQ-01.007, REQ-01.018).
// nodeRunner is optional; when provided, tools can run allowlisted commands on nodes via SSH (REQ-01.004, REQ-01.005, REQ-01.013).
// toolIndex is optional; when provided and Ready(), step 3.1 will use it for tool pre-selection.
// providers and providerLabels define the ordered LLM chain used by the unified router.
func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger, adapter Adapter, providers []llm.Provider, providerLabels []string, memoryStore *memory.Store, vectorStore vector.Store, embedder embedding.Embedder, nodeRunner NodeRunner, toolIndex ToolIndex) error {
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
	router, err := llmrouter.New(providers, providerLabels, llmrouter.Config{
		Escalation: escalationFromConfig(cfg),
	}, logger)
	if err != nil {
		return err
	}
	handler, err := newRunConversationHandler(cfg, logger, redactor, router, vectorStore, embedder, nodeRunner, toolIndex)
	if err != nil {
		return err
	}
	return adapter.Run(ctx, handler)
}

func newRunConversationHandler(cfg *config.Config, logger *slog.Logger, redactor func(string) string, router *llmrouter.Router, vectorStore vector.Store, embedder embedding.Embedder, nodeRunner NodeRunner, toolIndex ToolIndex) (*conversationHandler, error) {
	maxLen := 0
	if cfg != nil && cfg.Telegram.MaxMessageLength > 0 {
		maxLen = cfg.Telegram.MaxMessageLength
	}
	llmLog, model, err := openLLMLogIfConfigured(cfg, logger, redactor)
	if err != nil {
		return nil, err
	}
	var ctxMaxLen, topK int
	var toolTopK, toolMin, toolCap int
	if cfg != nil {
		ctxMaxLen, topK = conversationContextParams(cfg)
		toolTopK, toolMin, toolCap = toolPreSelectionParams(cfg)
	} else {
		// core.Run allows nil config only for narrow tests; match historical implicit defaults.
		ctxMaxLen, topK = 4000, 10
		toolTopK, toolMin, toolCap = 10, 1, 50
	}
	firstSupportsTools, textBased := firstProviderTextToolFlags(cfg)
	h := &conversationHandler{
		router:                     router,
		escalation:                 escalationFromConfig(cfg),
		vectorStore:                vectorStore,
		embedder:                   embedder,
		nodeRunner:                 nodeRunner,
		toolIndex:                  toolIndex,
		toolSearchTopK:             toolTopK,
		toolMinCount:               toolMin,
		toolFallbackCap:            toolCap,
		logger:                     logger,
		maxMessageLength:           maxLen,
		contextMaxLen:              ctxMaxLen,
		vectorSearchTopK:           topK,
		llmLog:                     llmLog,
		model:                      model,
		logRedactor:                redactor,
		textBasedEnabled:           textBased,
		firstProviderSupportsTools: firstSupportsTools,
	}
	if cfg != nil && cfg.ToolCatalog != nil {
		h.catalog = cfg.ToolCatalog
	}
	return h, nil
}

func escalationFromConfig(cfg *config.Config) *config.LLMEscalationConfig {
	return cfg.ToolsLLMEscalation()
}

func openLLMLogIfConfigured(cfg *config.Config, logger *slog.Logger, redactor func(string) string) (llmlog.Writer, string, error) {
	if cfg == nil || cfg.Paths.LLMLogDir == "" {
		return nil, "", nil
	}
	w, err := llmlog.NewWriter(cfg.Paths.LLMLogDir, logger, llmlog.Redactor(redactor))
	if err != nil {
		return nil, "", fmt.Errorf("core: llm log writer: %w", err)
	}
	model := ""
	if len(cfg.LLMProviders) > 0 {
		idx := 0
		if esc := cfg.ToolsLLMEscalation(); esc != nil && esc.Enabled && esc.BaselineIndex < len(cfg.LLMProviders) {
			idx = esc.BaselineIndex
		}
		model = cfg.LLMProviders[idx].Model
	}
	return w, model, nil
}

func firstProviderTextToolFlags(cfg *config.Config) (firstSupportsTools, textBased bool) {
	firstSupportsTools = true
	if cfg == nil {
		return firstSupportsTools, textBased
	}
	idx := 0
	if esc := cfg.ToolsLLMEscalation(); esc != nil && esc.Enabled && esc.BaselineIndex < len(cfg.LLMProviders) {
		idx = esc.BaselineIndex
	}
	if len(cfg.LLMProviders) > idx && cfg.LLMProviders[idx].SupportsTools != nil {
		firstSupportsTools = *cfg.LLMProviders[idx].SupportsTools
	}
	if cfg.Tools != nil {
		textBased = cfg.Tools.TextBasedEnabled
	}
	return firstSupportsTools, textBased
}

// toolPreSelectionParams returns tool pre-selection from config (validated at config.Load).
func toolPreSelectionParams(cfg *config.Config) (topK, minCount, fallbackCap int) {
	return cfg.ToolPreSelection.ToolSearchTopK, cfg.ToolPreSelection.ToolMinCount, cfg.ToolPreSelection.ToolFallbackCap
}

// conversationContextParams returns conversation context limits from config (validated at config.Load).
func conversationContextParams(cfg *config.Config) (contextMaxLen, vectorSearchTopK int) {
	return cfg.ConversationContext.InjectedContextMaxChars, cfg.ConversationContext.VectorSearchTopK
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
