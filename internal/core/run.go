package core

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/embedding"
	"pa/internal/llm"
	"pa/internal/llmlog"
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
func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger, adapter Adapter, llmProvider llm.Provider, memoryStore *memory.Store, vectorStore vector.Store, embedder embedding.Embedder, nodeRunner NodeRunner, toolIndex ToolIndex) error {
	if adapter == nil {
		return fmt.Errorf("core: adapter is nil")
	}
	if llmProvider == nil {
		return fmt.Errorf("core: llm provider is nil")
	}
	redactor := buildRedactor(cfg)
	handler, err := newRunConversationHandler(cfg, logger, redactor, llmProvider, vectorStore, embedder, nodeRunner, toolIndex)
	if err != nil {
		return err
	}
	return adapter.Run(ctx, handler)
}

func newRunConversationHandler(cfg *config.Config, logger *slog.Logger, redactor func(string) string, llmProvider llm.Provider, vectorStore vector.Store, embedder embedding.Embedder, nodeRunner NodeRunner, toolIndex ToolIndex) (*conversationHandler, error) {
	maxLen := 0
	if cfg != nil && cfg.Telegram.MaxMessageLength > 0 {
		maxLen = cfg.Telegram.MaxMessageLength
	}
	llmLog, model, err := openLLMLogIfConfigured(cfg, logger, redactor)
	if err != nil {
		return nil, err
	}
	ctxMaxLen, topK := conversationContextParams(cfg)
	toolTopK, toolMin, toolCap := toolPreSelectionParams(cfg)
	firstSupportsTools, textBased := firstProviderTextToolFlags(cfg)
	h := &conversationHandler{
		provider:                   llmProvider,
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
		model = cfg.LLMProviders[0].Model
	}
	return w, model, nil
}

func firstProviderTextToolFlags(cfg *config.Config) (firstSupportsTools, textBased bool) {
	firstSupportsTools = true
	if cfg == nil {
		return firstSupportsTools, textBased
	}
	if len(cfg.LLMProviders) > 0 && cfg.LLMProviders[0].SupportsTools != nil {
		firstSupportsTools = *cfg.LLMProviders[0].SupportsTools
	}
	if cfg.Tools != nil {
		textBased = cfg.Tools.TextBasedEnabled
	}
	return firstSupportsTools, textBased
}

// toolPreSelectionParams returns tool_search_top_k, tool_min_count, tool_fallback_cap from config; 0 means use toolindex defaults.
func toolPreSelectionParams(cfg *config.Config) (topK, minCount, fallbackCap int) {
	if cfg == nil || cfg.ToolPreSelection == nil {
		return 0, 0, 0
	}
	return cfg.ToolPreSelection.ToolSearchTopK, cfg.ToolPreSelection.ToolMinCount, cfg.ToolPreSelection.ToolFallbackCap
}

// conversationContextParams returns injected context max chars and vector search top-K from config; zero means use defaults.
func conversationContextParams(cfg *config.Config) (contextMaxLen, vectorSearchTopK int) {
	contextMaxLen = defaultContextMaxLen
	vectorSearchTopK = defaultVectorSearchTopK
	if cfg != nil && cfg.ConversationContext != nil {
		if cfg.ConversationContext.InjectedContextMaxChars > 0 {
			contextMaxLen = cfg.ConversationContext.InjectedContextMaxChars
		}
		if cfg.ConversationContext.VectorSearchTopK > 0 {
			vectorSearchTopK = cfg.ConversationContext.VectorSearchTopK
		}
	}
	return contextMaxLen, vectorSearchTopK
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
