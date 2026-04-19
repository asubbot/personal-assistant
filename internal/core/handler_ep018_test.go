package core

import (
	"context"
	"log/slog"
	"pa/internal/intent"
	"pa/internal/llm"
	"pa/internal/vector"
	"strings"
	"testing"
	"unicode/utf8"
)

// Covers AC-18.004
func TestHandleMessage_FullLite_skipsRAGInSystemMessage(t *testing.T) {
	const marker = "UNIQUE_RAG_MARKER_EP018"
	big := strings.Repeat("Z", 2000)
	notes := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "notes:2026-04-01:1", Text: marker + "\n" + big}}}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	router := mustRouterSingle(t, provider)
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, nil, []string{`^LITEONLY$`}, 100),
		nil,
		nil,
	)
	h := &conversationHandler{
		router:                router,
		logger:                slog.Default(),
		maxDynamicSystemRunes: 200_000,
		memoryVectorTopK:      testMemoryVectorTopK(5),
		classifier:            classifier,
		memVec:                &MemoryVectors{Notes: notes},
		embedder:              &mockEmbedder{vec: []float32{1, 0, 0, 0}},
	}
	_, err := h.HandleMessage(context.Background(), 1, "", "LITEONLY")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(provider.lastMessages) < 1 {
		t.Fatal("expected system message")
	}
	sys := provider.lastMessages[0].Content
	if strings.Contains(sys, marker) {
		t.Fatalf("full_lite system message should not contain RAG marker; got len=%d", len(sys))
	}
}

// Covers AC-18.020
func TestHandleMessage_FullLite_systemPromptRunesLowerThanFullWithRAG(t *testing.T) {
	const marker = "EP018_RUNE_COMPARE"
	big := strings.Repeat("W", 8000)
	notes := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "notes:2026-04-01:9", Text: marker + big}}}
	embed := &mockEmbedder{vec: []float32{1, 0, 0, 0}}

	run := func(tierMsg string, clf intent.Classifier) int {
		t.Helper()
		p := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
		r := mustRouterSingle(t, p)
		h := &conversationHandler{
			router:                r,
			logger:                slog.Default(),
			maxDynamicSystemRunes: 200_000,
			memoryVectorTopK:      testMemoryVectorTopK(5),
			classifier:            clf,
			memVec:                &MemoryVectors{Notes: notes},
			embedder:              embed,
		}
		_, err := h.HandleMessage(context.Background(), 1, "", tierMsg)
		if err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		if len(p.lastMessages) < 1 {
			t.Fatal("expected system message")
		}
		return utf8.RuneCountInString(p.lastMessages[0].Content)
	}

	fullRunes := run("FULLRUNE", intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, []string{`^FULLRUNE$`}, nil, 100),
		nil,
		nil,
	))
	liteRunes := run("LITERUNE", intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier(nil, nil, []string{`^LITERUNE$`}, 100),
		nil,
		nil,
	))
	if fullRunes <= liteRunes {
		t.Fatalf("expected full > lite runes; full=%d lite=%d", fullRunes, liteRunes)
	}
	reduction := float64(fullRunes-liteRunes) / float64(fullRunes)
	if reduction < 0.15 {
		t.Fatalf("expected >= 15%% reduction from full to full_lite; full=%d lite=%d reduction=%.3f",
			fullRunes, liteRunes, reduction)
	}
}

// Covers AC-18.018
func TestHandleMessage_logsMainLLMPromptAssembled(t *testing.T) {
	var cap captureHandlerWithAttrs
	logger := slog.New(&cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	router := mustRouterSingle(t, provider)
	classifier := intent.NewCascadeClassifier(
		intent.NewHeuristicClassifier([]string{`^hi$`}, nil, nil, 40),
		nil,
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
