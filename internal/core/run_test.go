package core

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"testing"
)

func TestRun_nilAdapter_returnsError(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "x"}}

	err := Run(context.Background(), cfg, logger, nil, provider)
	if err == nil {
		t.Fatal("expected error when adapter is nil")
	}
	if err.Error() != "core: adapter is nil" {
		t.Errorf("err = %q, want %q", err.Error(), "core: adapter is nil")
	}
}

func TestRun_nilProvider_returnsError(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.Default()
	adapter := &capturingAdapter{}

	err := Run(context.Background(), cfg, logger, adapter, nil)
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

func TestRun_callsAdapterRunWithHandler(t *testing.T) {
	cfg := &config.Config{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	adapter := &capturingAdapter{}

	err := Run(context.Background(), cfg, logger, adapter, provider)
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

// Ensure capturingAdapter implements Adapter.
var _ Adapter = (*capturingAdapter)(nil)
