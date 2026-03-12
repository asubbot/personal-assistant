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

// Run starts the application: wires the adapter to the conversation handler (LLM, memory, vector, optional node runner) and blocks until ctx is cancelled.
// memoryStore, vectorStore, and embedder are optional; when provided, the handler reads memory, runs semantic search, and indexes turns (REQ-006, REQ-007, REQ-018).
// nodeRunner is optional; when provided, tools can run allowlisted commands on nodes via SSH (REQ-004, REQ-005, REQ-013).
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
	handler := &conversationHandler{
		provider:         llmProvider,
		memoryStore:      memoryStore,
		vectorStore:      vectorStore,
		embedder:         embedder,
		nodeRunner:       nodeRunner,
		logger:           logger,
		maxMessageLength: maxLen,
		llmLog:           llmLog,
		model:            model,
		logRedactor:      redactor,
	}
	return adapter.Run(ctx, handler)
}

// buildRedactor returns a redactor from built-in patterns plus config additional_patterns (REQ-027, REQ-028).
func buildRedactor(cfg *config.Config) func(string) string {
	var additional []logredact.Pattern
	if cfg != nil && cfg.LogRedaction != nil {
		for _, p := range cfg.LogRedaction.AdditionalPatterns {
			additional = append(additional, logredact.Pattern{ID: p.ID, Regex: p.Regex, Replacement: p.Replacement})
		}
	}
	return logredact.NewRedactor(additional)
}
