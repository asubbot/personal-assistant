package core

import (
	"context"
	"log/slog"
	"os"
	"pa/internal/intent"
	"pa/internal/llm"
	"strings"
	"testing"
)

// Covers AC-26.001, AC-26.002, AC-26.003. Supporting AC-26.005, AC-26.006: exercised with full package tests and make check.
func TestTierMainPromptBuilders_simpleTierUnchanged(t *testing.T) {
	ctx := context.Background()
	h := &conversationHandler{logger: slog.New(slog.DiscardHandler)}
	sysHead := "system-head"
	userText := "hello"
	msgs := []llm.Message{{Role: "system", Content: sysHead}, {Role: "user", Content: userText}}
	got, err := h.assembleTierMainLLMParams(ctx, intent.TierSimple, userText, sysHead, nil, msgs)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if got.opts != nil || got.dynamicRan {
		t.Fatalf("simple tier params = %+v, want empty", got)
	}
	if msgs[0].Content != sysHead {
		t.Fatalf("system content = %q, want unchanged %q", msgs[0].Content, sysHead)
	}
}

// Covers AC-26.001, AC-26.002, AC-26.003
func TestTierMainPromptBuilders_fullLiteNilCatalog(t *testing.T) {
	ctx := context.Background()
	h := &conversationHandler{logger: slog.New(slog.DiscardHandler)}
	sysHead := "system-head"
	userText := "hello"
	msgs := []llm.Message{{Role: "system", Content: sysHead}, {Role: "user", Content: userText}}
	got, err := h.assembleTierMainLLMParams(ctx, intent.TierFullLite, userText, sysHead, nil, msgs)
	if err != nil {
		t.Fatalf("assemble full_lite: %v", err)
	}
	if got.opts != nil {
		t.Fatalf("opts = %v, want nil when catalog nil", got.opts)
	}
}

// Covers AC-26.001, AC-26.002, AC-26.003
func TestTierMainPromptBuilders_fullNilCatalog(t *testing.T) {
	ctx := context.Background()
	h := &conversationHandler{logger: slog.New(slog.DiscardHandler)}
	sysHead := "system-head"
	userText := "hello"
	msgs := []llm.Message{{Role: "system", Content: sysHead}, {Role: "user", Content: userText}}
	got, err := h.assembleTierMainLLMParams(ctx, intent.TierFull, userText, sysHead, nil, msgs)
	if err != nil {
		t.Fatalf("assemble full: %v", err)
	}
	if got.opts != nil {
		t.Fatalf("opts = %v, want nil when catalog nil", got.opts)
	}
}

// Covers AC-26.001
func TestTierMainPromptBuilders_explicitEntryPoints(t *testing.T) {
	h := &conversationHandler{logger: slog.New(slog.DiscardHandler)}
	ctx := context.Background()
	sysHead := "h"
	msgs := []llm.Message{{Role: "system", Content: sysHead}}
	if p := h.buildTierSimpleMainPrompt(); p.opts != nil || p.dynamicRan {
		t.Fatalf("buildTierSimpleMainPrompt = %+v", p)
	}
	if _, err := h.buildTierFullLiteMainPrompt(ctx, "u", sysHead, msgs); err != nil {
		t.Fatalf("buildTierFullLiteMainPrompt: %v", err)
	}
	msgs2 := []llm.Message{{Role: "system", Content: sysHead}}
	if _, err := h.buildTierFullMainPrompt(ctx, "u", sysHead, nil, msgs2); err != nil {
		t.Fatalf("buildTierFullMainPrompt: %v", err)
	}
}

// Covers AC-26.004
func TestEP026_HandlerGoHasNoGocycloNolint(t *testing.T) {
	raw, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read handler.go: %v", err)
	}
	if strings.Contains(string(raw), "//nolint:gocyclo") {
		t.Fatal("handler.go must not contain //nolint:gocyclo (EP-026)")
	}
}
