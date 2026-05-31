package core

import (
	"context"
	"log/slog"
	"pa/internal/intent"
	"pa/internal/llm"
	"pa/internal/vector"
	"strings"
	"testing"
)

// Covers AC-36.012
func TestHandleMessage_formerFullLitePattern_usesFullAssemblyWithRAG(t *testing.T) {
	const marker = "UNIQUE_RAG_MARKER_EP036"
	big := strings.Repeat("Z", 2000)
	notes := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "notes:2026-05-01:1", Text: marker + "\n" + big}}}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	router := mustRouterSingle(t, provider)
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, []string{`^LITEONLY$`}, 100),
		nil,
	)
	h := testHandlerDeps{
		router:                router,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 200_000,
		memoryVectorTopK:      testMemoryVectorTopK(5),
		classifier:            classifier,
		memVec:                &MemoryVectors{Notes: notes},
		embedder:              &mockEmbedder{vec: []float32{1, 0, 0, 0}},
	}.handler()
	_, err := h.HandleMessage(context.Background(), 1, "", "LITEONLY")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(provider.lastMessages) < 1 {
		t.Fatal("expected system message")
	}
	sys := provider.lastMessages[0].Content
	if !strings.Contains(sys, marker) {
		t.Fatalf("former lite-pattern match should use full path with RAG; system len=%d", len(sys))
	}
}

// Covers AC-18.018
func TestHandleMessage_logsMainLLMPromptAssembled(t *testing.T) {
	var cap captureHandlerWithAttrs
	logger := slog.New(&cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	router := mustRouterSingle(t, provider)
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier([]string{`^hi$`}, nil, 40),
		nil,
	)
	h := testHandlerDeps{
		router:                router,
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		classifier:            classifier,
	}.handler()
	_, err := h.HandleMessage(context.Background(), 1, "", "hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var found bool
	for _, rec := range cap.records {
		if rec.msg != "main llm prompt assembled" {
			continue
		}
		found = true
		for _, key := range []string{"tier", "main_tool_count", "dynamic_tool_selection", "intent_stage"} {
			if _, ok := rec.attrs[key]; !ok {
				t.Errorf("log attrs missing %q: %#v", key, rec.attrs)
			}
		}
	}
	if !found {
		t.Fatalf("expected main llm prompt assembled log, got %#v", cap.records)
	}
}
