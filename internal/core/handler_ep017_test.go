package core

import (
	"bytes"
	"context"
	"log/slog"
	"pa/internal/intent"
	"pa/internal/llm"
	"strings"
	"testing"
)

// Covers AC-38.002, AC-38.011, AC-38.018
// Covers AC-17.002, AC-36.003, AC-36.011
// Supporting AC-18.002 (simple tier unchanged vs EP-017)
func TestHandleMessage_SimpleTier_NoToolsNoRAG(t *testing.T) {
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hi there!"}}
	router := mustRouterSingle(t, provider)
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier([]string{`^привет$`}, nil, 40),
		nil,
	)
	h := &conversationHandler{
		router:                router,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		classifier:            classifier,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "", "привет")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "hi there!" {
		t.Errorf("reply = %q, want %q", reply, "hi there!")
	}
	if provider.lastOpts != nil {
		t.Errorf("simple tier: opts should be nil (no tools), got %+v", provider.lastOpts)
	}
}

// Covers AC-17.003, AC-36.011
func TestHandleMessage_FullTier_IncludesFullPromptPath(t *testing.T) {
	provider := &mockProvider{result: &llm.CompletionResult{Content: "full response"}}
	router := mustRouterSingle(t, provider)
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, []string{`напомни`}, 40),
		nil,
	)
	h := &conversationHandler{
		router:                router,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		classifier:            classifier,
	}
	_, err := h.HandleMessage(context.Background(), 1, "", "напомни что было вчера")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
}

// Covers AC-17.014
func TestHandleMessage_ClassifierDisabled_FullPath(t *testing.T) {
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	router := mustRouterSingle(t, provider)
	h := &conversationHandler{
		router:                router,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		classifier:            nil,
	}
	_, err := h.HandleMessage(context.Background(), 1, "", "привет")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
}

// Covers AC-17.012
func TestHandleMessage_SimpleTier_SkipsToolsAndRAG(t *testing.T) {
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hi"}}
	router := mustRouterSingle(t, provider)
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier([]string{`^привет$`}, nil, 40),
		nil,
	)
	h := &conversationHandler{
		router:                router,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		classifier:            classifier,
	}
	_, err := h.HandleMessage(context.Background(), 1, "", "привет")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if provider.lastOpts != nil {
		t.Errorf("simple tier should have nil opts (no tools), got non-nil")
	}
}

// Covers AC-17.013
func TestHandleMessage_FullTier_SameAsBaseline(t *testing.T) {
	providerWithClassifier := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	routerWith := mustRouterSingle(t, providerWithClassifier)
	classifier := intent.NewCascadeClassifier(nil, nil)
	hWith := &conversationHandler{
		router:                routerWith,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		classifier:            classifier,
	}
	_, err := hWith.HandleMessage(context.Background(), 1, "", "напомни")
	if err != nil {
		t.Fatalf("HandleMessage with classifier: %v", err)
	}

	providerWithout := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	routerWithout := mustRouterSingle(t, providerWithout)
	hWithout := &conversationHandler{
		router:                routerWithout,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		classifier:            nil,
	}
	_, err = hWithout.HandleMessage(context.Background(), 1, "", "напомни")
	if err != nil {
		t.Fatalf("HandleMessage without classifier: %v", err)
	}

	if len(providerWithClassifier.lastMessages) != len(providerWithout.lastMessages) {
		t.Errorf("message count differs: with=%d, without=%d",
			len(providerWithClassifier.lastMessages), len(providerWithout.lastMessages))
	}
	for i := range providerWithClassifier.lastMessages {
		if i >= len(providerWithout.lastMessages) {
			break
		}
		if providerWithClassifier.lastMessages[i].Content != providerWithout.lastMessages[i].Content {
			t.Errorf("message[%d] content differs", i)
		}
	}
}

// Covers AC-17.017
func TestHandleMessage_SimpleTier_FooterOnlyMainTokens(t *testing.T) {
	provider := &mockProvider{result: &llm.CompletionResult{
		Content: "hello!",
		Usage:   llm.Usage{PromptTokens: 100, CompletionTokens: 5},
	}}
	router := mustRouterSingle(t, provider)
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier([]string{`^hi$`}, nil, 40),
		nil,
	)
	h := &conversationHandler{
		router:                router,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		classifier:            classifier,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "", "hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(reply, "in: 100") {
		t.Errorf("footer should contain main model token count, got: %s", reply)
	}
}

// Covers AC-17.016
func TestHandleMessage_ClassificationLogsInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	router := mustRouterSingle(t, provider)
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier([]string{`^hi$`}, nil, 40),
		nil,
	)
	h := &conversationHandler{
		router:                router,
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		classifier:            classifier,
	}
	_, err := h.HandleMessage(context.Background(), 1, "", "hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	logOutput := buf.String()
	for _, key := range []string{"tier=", "stage=", "message_len="} {
		if !strings.Contains(logOutput, key) {
			t.Errorf("INFO log should contain %q, got: %s", key, logOutput)
		}
	}
}
