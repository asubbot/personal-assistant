package core

import (
	"context"
	"log/slog"
	"os"
	"pa/internal/config"
	"pa/internal/llm"
	"path/filepath"
	"strings"
	"testing"
)

// minimalConfigForRun satisfies config fields required by newRunConversationHandler when tests bypass config.Load.
func minimalConfigForRun() *config.Config {
	return &config.Config{
		Tools: &config.ToolsConfig{
			Selection: &config.ToolsSelection{
				ToolSearchTopK: 10, ToolMinCount: 1, ToolFallbackCap: 50,
				Enabled: false, MaxToolsForLLMRequest: 0,
			},
		},
		LogRedaction:        &config.LogRedaction{},
		PATimezone:          "UTC",
		ConversationContext: &config.ConversationContextConfig{MaxDynamicSystemRunes: 4000, MemoryVector: config.MemoryVectorConfig{NotesTopK: 10, SummariesTopK: 10, TurnsTopK: 10}},
	}
}

// Covers AC-01.003 (US-02): core.Run with nil adapter returns error and does not start serving.
func TestRun_nilAdapter_returnsError(t *testing.T) {
	cfg := minimalConfigForRun()
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "x"}}

	err := Run(context.Background(), cfg, logger, nil, []llm.Provider{provider}, []string{"test/default"}, nil, nil, nil, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error when adapter is nil")
	}
	if err.Error() != "core: adapter is nil" {
		t.Errorf("err = %q, want %q", err.Error(), "core: adapter is nil")
	}
}

// Covers AC-01.003 (US-02): core.Run with no providers returns error and does not start serving.
func TestRun_nilProvider_returnsError(t *testing.T) {
	cfg := minimalConfigForRun()
	logger := slog.Default()
	adapter := &capturingAdapter{}

	err := Run(context.Background(), cfg, logger, adapter, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error when providers are empty")
	}
	if err.Error() != "core: providers are required" {
		t.Errorf("err = %q, want %q", err.Error(), "core: providers are required")
	}
}

// capturingAdapter implements Adapter by storing the handler and returning nil from Run.
type capturingAdapter struct {
	handler MessageHandler
}

func (a *capturingAdapter) Run(ctx context.Context, handler MessageHandler) error {
	a.handler = handler
	return nil
}

// Covers AC-37.012: handler construction reads pre-selection from cfg.Tools.Selection only.
func TestRun_wiresToolsSelectionFromConfig(t *testing.T) {
	cfg := minimalConfigForRun()
	cfg.Tools.Selection.Enabled = true
	cfg.Tools.Selection.MaxToolsForLLMRequest = 3
	adapter := &capturingAdapter{}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	if err := Run(context.Background(), cfg, slog.Default(), adapter, []llm.Provider{provider}, []string{"test/default"}, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	h, ok := adapter.handler.(*conversationHandler)
	if !ok {
		t.Fatalf("handler type %T", adapter.handler)
	}
	if h.toolsSelection == nil || !h.toolsSelection.Enabled || h.toolsSelection.MaxToolsForLLMRequest != 3 {
		t.Fatalf("toolsSelection = %#v", h.toolsSelection)
	}
	if h.toolSearchTopK != cfg.Tools.Selection.ToolSearchTopK {
		t.Fatalf("toolSearchTopK = %d want %d", h.toolSearchTopK, cfg.Tools.Selection.ToolSearchTopK)
	}
}

// Covers AC-38.015
// Covers AC-01.003 (US-02): core.Run calls adapter.Run with non-nil handler (valid wiring).
func TestRun_callsAdapterRunWithHandler(t *testing.T) {
	cfg := minimalConfigForRun()
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	adapter := &capturingAdapter{}

	err := Run(context.Background(), cfg, logger, adapter, []llm.Provider{provider}, []string{"test/default"}, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if adapter.handler == nil {
		t.Fatal("adapter.handler was not set")
	}

	reply, err := adapter.handler.HandleMessage(context.Background(), 1, "", "test")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want %q", reply, "ok")
	}
}

// Covers AC-01.003 (US-02): core.Run with nil config does not panic; handler gets zero max length (no limit).
func TestRun_cfgNil_noPanic_handlerGetsZeroMaxLength(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	adapter := &capturingAdapter{}

	err := Run(context.Background(), nil, logger, adapter, []llm.Provider{provider}, []string{"test/default"}, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Run(cfg=nil): %v", err)
	}
	if adapter.handler == nil {
		t.Fatal("adapter.handler was not set")
	}
	// Handler should have maxMessageLength 0 (no limit); long message goes through
	longText := strings.Repeat("x", 5000)
	reply, err := adapter.handler.HandleMessage(context.Background(), 1, "", longText)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want ok (no limit when cfg nil)", reply)
	}
}

// Ensure capturingAdapter implements Adapter.
var _ Adapter = (*capturingAdapter)(nil)

// Covers AC-01.028, REQ-01.017 (unit): config points to file with known fake secret; built LLM context (messages sent to provider) must not contain it.
func TestRun_builtLLMContextDoesNotContainConfigSecret(t *testing.T) {
	const fakeSecret = "fake-secret-unit-12345"
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(secretPath, []byte(fakeSecret), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	cfg := &config.Config{
		Version:  1,
		Telegram: config.Telegram{TokenPath: secretPath},
		Tools: &config.ToolsConfig{
			Selection: &config.ToolsSelection{
				ToolSearchTopK: 10, ToolMinCount: 1, ToolFallbackCap: 50,
				Enabled: false, MaxToolsForLLMRequest: 0,
			},
		},
		LogRedaction:        &config.LogRedaction{},
		PATimezone:          "UTC",
		ConversationContext: &config.ConversationContextConfig{MaxDynamicSystemRunes: 4000, MemoryVector: config.MemoryVectorConfig{NotesTopK: 10, SummariesTopK: 10, TurnsTopK: 10}},
	}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	adapter := &capturingAdapter{}

	err := Run(context.Background(), cfg, logger, adapter, []llm.Provider{provider}, []string{"test/default"}, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if adapter.handler == nil {
		t.Fatal("adapter.handler was not set")
	}

	_, _ = adapter.handler.HandleMessage(context.Background(), 1, "", "hello")

	for i, m := range provider.lastMessages {
		if strings.Contains(m.Content, fakeSecret) {
			t.Errorf("message[%d] (role=%q) must not contain config secret; content contains %q", i, m.Role, fakeSecret)
		}
	}
}

// Covers wiring validation: labels must match providers length.
// Covers AC-01.003: traceability for TestRun_labelsLengthMismatch_returnsError.
func TestRun_labelsLengthMismatch_returnsError(t *testing.T) {
	cfg := minimalConfigForRun()
	logger := slog.Default()
	p := &mockProvider{result: &llm.CompletionResult{Content: "x"}}
	adapter := &capturingAdapter{}
	err := Run(context.Background(), cfg, logger, adapter, []llm.Provider{p}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error when labels length mismatches providers")
	}
	if !strings.Contains(err.Error(), "provider labels length") {
		t.Errorf("err = %q", err.Error())
	}
}

// Covers provider chain mode: first provider in chain can handle completion.
// Covers AC-01.003: traceability for TestRun_providerChain_wiresHandler.
func TestRun_providerChain_wiresHandler(t *testing.T) {
	cfg := minimalConfigForRun()
	logger := slog.Default()
	p := &mockProvider{result: &llm.CompletionResult{Content: "from-chain"}}
	adapter := &capturingAdapter{}
	err := Run(context.Background(), cfg, logger, adapter, []llm.Provider{p}, []string{"openai/gpt"}, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	reply, err := adapter.handler.HandleMessage(context.Background(), 1, "", "hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "from-chain" {
		t.Errorf("reply = %q, want from-chain", reply)
	}
}
