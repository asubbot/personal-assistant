package core

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"strings"
	"testing"
)

// No AC: core.Run contract — nil adapter returns error.
func TestRun_nilAdapter_returnsError(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "x"}}

	err := Run(context.Background(), cfg, logger, nil, provider, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error when adapter is nil")
	}
	if err.Error() != "core: adapter is nil" {
		t.Errorf("err = %q, want %q", err.Error(), "core: adapter is nil")
	}
}

// No AC: core.Run contract — nil provider returns error.
func TestRun_nilProvider_returnsError(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.Default()
	adapter := &capturingAdapter{}

	err := Run(context.Background(), cfg, logger, adapter, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error when provider is nil")
	}
	if err.Error() != "core: llm provider is nil" {
		t.Errorf("err = %q, want %q", err.Error(), "core: llm provider is nil")
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

// No AC: core.Run contract — adapter.Run is called with non-nil handler.
func TestRun_callsAdapterRunWithHandler(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	adapter := &capturingAdapter{}

	err := Run(context.Background(), cfg, logger, adapter, provider, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if adapter.handler == nil {
		t.Fatal("adapter.handler was not set")
	}

	reply, err := adapter.handler.HandleMessage(context.Background(), 1, "test")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want %q", reply, "ok")
	}
}

// No AC: core.Run with nil config does not panic; handler gets zero max length (no limit).
func TestRun_cfgNil_noPanic_handlerGetsZeroMaxLength(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	adapter := &capturingAdapter{}

	err := Run(context.Background(), nil, logger, adapter, provider, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Run(cfg=nil): %v", err)
	}
	if adapter.handler == nil {
		t.Fatal("adapter.handler was not set")
	}
	// Handler should have maxMessageLength 0 (no limit); long message goes through
	longText := strings.Repeat("x", 5000)
	reply, err := adapter.handler.HandleMessage(context.Background(), 1, longText)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want ok (no limit when cfg nil)", reply)
	}
}

// Ensure capturingAdapter implements Adapter.
var _ Adapter = (*capturingAdapter)(nil)
