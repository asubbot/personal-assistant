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

// Run starts the application: wires the adapter to the conversation handler (LLM, vector, optional node runner) and blocks until ctx is cancelled.
// memoryStore is not used by the handler (context is from vector only); vectorStore and embedder are optional; when provided, the handler runs semantic search and indexes turns (REQ-01.006, REQ-01.007, REQ-01.018).
// nodeRunner is optional; when provided, tools can run allowlisted commands on nodes via SSH (REQ-01.004, REQ-01.005, REQ-01.013).
func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger, adapter Adapter, llmProvider llm.Provider, memoryStore *memory.Store, vectorStore vector.Store, embedder embedding.Embedder, nodeRunner NodeRunner) error {
	if adapter == nil {
		return fmt.Errorf("core: adapter is nil")
	}
	if llmProvider == nil {
		return fmt.Errorf("core: llm provider is nil")
	}
	maxLen := 0
	if cfg != nil && cfg.Telegram.MaxMessageLength > 0 {
		maxLen = cfg.Telegram.MaxMessageLength
	}
	redactor := buildRedactor(cfg)
	var llmLog llmlog.Writer
	var model string
	if cfg != nil && cfg.Paths.LLMLogDir != "" {
		var err error
		llmLog, err = llmlog.NewWriter(cfg.Paths.LLMLogDir, logger, llmlog.Redactor(redactor))
		if err != nil {
			return fmt.Errorf("core: llm log writer: %w", err)
		}
		if len(cfg.LLMProviders) > 0 {
			model = cfg.LLMProviders[0].Model
		}
	}
	ctxMaxLen, topK := conversationContextParams(cfg)
	handler := &conversationHandler{
		provider:         llmProvider,
		vectorStore:      vectorStore,
		embedder:         embedder,
		nodeRunner:       nodeRunner,
		logger:           logger,
		maxMessageLength: maxLen,
		contextMaxLen:    ctxMaxLen,
		vectorSearchTopK: topK,
		llmLog:           llmLog,
		model:            model,
		logRedactor:      redactor,
	}
	return adapter.Run(ctx, handler)
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
