package core

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/llm"
	"pa/internal/vector"
	"strings"
	"testing"
	"time"
)

// Covers AC-01.014, REQ-01.007, AC-02.013: semantic search injects relevant past context without invoking read_memory (baseline vector path).
func TestHandleMessage_injectsVectorSearchContextIntoSystemMessage(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	summ := &mockVectorStore{
		searchResults: []vector.SearchResult{{ID: "summary:day:2099-01-01", Text: "past mention of bananas"}},
	}
	turnM := &mockVectorStore{}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, provider),
		memVec:                &MemoryVectors{Summaries: summ, Turns: turnM},
		embedder:              emb,
		logger:                logger,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
		memoryVectorTopK:      testMemoryVectorTopK(10),
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "what did I say about fruit?")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	sysContent := provider.lastMessages[0].Content
	if !strings.Contains(sysContent, "Relevant past context") {
		t.Errorf("system message must contain 'Relevant past context'; got: %s", sysContent)
	}
	if !strings.Contains(sysContent, "past mention of bananas") {
		t.Errorf("system message must contain search result text; got: %s", sysContent)
	}
}

// Covers AC-01.013, REQ-01.007, AC-02.008, AC-02.009: indexTurn stores Date line and [turn] label in vector chunk text.
func TestHandleMessage_indexTurnCallsAddWithUserAndReply(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "reply text", Usage: llm.Usage{}}}
	vs := &mockVectorStore{}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, provider),
		memVec:                SingleStoreMemoryVectors(vs),
		embedder:              emb,
		logger:                logger,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
		memoryVectorTopK:      testMemoryVectorTopK(10),
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "user said this")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(vs.addChunks) != 1 {
		t.Fatalf("Add calls = %d, want 1", len(vs.addChunks))
	}
	dateStr := time.Now().UTC().Format("2006-01-02")
	wantChunk := "Date: " + dateStr + "\n[turn]\nUser: user said this\nAssistant: reply text"
	if vs.addChunks[0] != wantChunk {
		t.Errorf("Add chunk = %q, want %q", vs.addChunks[0], wantChunk)
	}
}

// Covers AC-01.038, REQ-01.026, REQ-01.027: at DEBUG level, logRedactor is applied to request/response content before app log.
func TestHandleMessage_logRedactorAppliedInDebugLogs(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelDebug}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "response contains secret", Usage: llm.Usage{}}}
	redactor := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }
	h := testHandlerDeps{
		router:      mustRouterSingle(t, provider),
		logger:      logger,
		logRedactor: redactor,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "user said secret")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var foundRedacted bool
	for _, r := range cap.records {
		if c, ok := r.attrs["content"]; ok && strings.Contains(c, "[REDACTED]") && !strings.Contains(c, "secret") {
			foundRedacted = true
			break
		}
	}
	if !foundRedacted {
		t.Errorf("expected DEBUG log record with redacted content (no raw 'secret'); got records: %+v", cap.records)
	}
}

// Covers AC-01.018, REQ-01.015: when LLM log is not configured (llmLog nil), handler does not attempt to write; no panic.
func TestHandleMessage_llmLogNil_succeedsWithoutWrite(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger, llmLog: nil}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want ok", reply)
	}
}

// Covers AC-01.014, REQ-01.007: retrieved memory uses whole chunks only; max_dynamic_system_runes trims the dynamic tail (drops trailing chunks first).
func TestHandleMessage_gatherContextTailFitsWholeChunksOnly(t *testing.T) {
	logger := slog.Default()
	// Single chunk exceeds dynamic tail budget: chunk is dropped; no retrieved section.
	longText := strings.Repeat("x", defaultMaxDynamicSystemRunes+500)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	vs := &mockVectorStore{
		searchResults: []vector.SearchResult{{Text: longText}},
	}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, provider),
		memVec:                SingleStoreMemoryVectors(vs),
		embedder:              emb,
		logger:                logger,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
		memoryVectorTopK:      testMemoryVectorTopK(10),
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "query")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	sysContent := provider.lastMessages[0].Content
	if strings.Contains(sysContent, "Relevant past context") {
		t.Errorf("when chunk does not fit tail budget, system message must not contain 'Relevant past context'")
	}

	// Two chunks: tail fit drops the oversized second chunk; first remains (no mid-chunk truncation).
	shortChunk := "User: hi\nAssistant: hello"
	shortChunkLabeled := "[turn]\n" + shortChunk
	longY := strings.Repeat("y", defaultMaxDynamicSystemRunes)
	vs2 := &mockVectorStore{
		searchResults: []vector.SearchResult{
			{Text: shortChunk},
			{Text: longY},
		},
	}
	h2 := testHandlerDeps{
		router:                mustRouterSingle(t, provider),
		memVec:                SingleStoreMemoryVectors(vs2),
		embedder:              emb,
		logger:                logger,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
		memoryVectorTopK:      testMemoryVectorTopK(10),
	}.handler()
	_, err = h2.HandleMessage(context.Background(), 1, "", "query")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	sysContent2 := provider.lastMessages[0].Content
	if !strings.Contains(sysContent2, "Relevant past context") {
		t.Errorf("system message must contain 'Relevant past context' when the first chunk remains after tail fit")
	}
	if !strings.Contains(sysContent2, shortChunkLabeled) && !strings.Contains(sysContent2, shortChunk) {
		t.Errorf("system message must contain the first chunk")
	}
	if strings.Contains(sysContent2, longY[:200]) {
		t.Errorf("system message must not contain the dropped second chunk")
	}
}

// Covers AC-02.009: vector search prefixes retrieved summary chunks with [summary:day] from stable id prefix.
func TestHandleMessage_vectorSearchPrefixesSummaryDayLabel(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	stored := "Date: 2026-03-01\n[summary:day]\nRemembered fact."
	vs := &mockVectorStore{
		searchResults: []vector.SearchResult{{ID: "summary:day:2026-03-01", Text: stored}},
	}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, provider),
		memVec:                SingleStoreMemoryVectors(vs),
		embedder:              emb,
		logger:                logger,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
		memoryVectorTopK:      testMemoryVectorTopK(10),
	}.handler()
	_, err := h.HandleMessage(context.Background(), 1, "", "what did we save?")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	sys := provider.lastMessages[0].Content
	if !strings.Contains(sys, "[summary:day]") || !strings.Contains(sys, "Remembered fact.") {
		t.Errorf("system message should include labeled summary chunk; got:\n%s", sys)
	}
	if c := strings.Count(sys, "[summary:day]"); c != 1 {
		t.Errorf("want exactly one [summary:day] marker (no duplicate retrieval prefix), got %d in:\n%s", c, sys)
	}
}

// Covers AC-02.009: retrievalChunkWithLabel avoids duplicating an embedded type line already present in stored vector text.
func TestRetrievalChunkWithLabel_noDuplicateWhenBodyHasMarker(t *testing.T) {
	stored := "Date: 2026-03-01\n[summary:day]\nBody"
	got := retrievalChunkWithLabel("summary:day", stored)
	if got != stored {
		t.Fatalf("got %q, want unchanged body", got)
	}
	raw := "plain snippet without marker"
	got2 := retrievalChunkWithLabel("turn", raw)
	want2 := "[turn]\n" + raw
	if got2 != want2 {
		t.Fatalf("got %q, want %q", got2, want2)
	}
}

// Supporting AC-01.036, AC-01.037, REQ-01.025: when indexTurn fails (embedder error), handler still returns reply; system does not crash.
func TestHandleMessage_indexTurnError_stillReturnsReply(t *testing.T) {
	embedErr := errors.New("embed failed")
	cap := &captureHandler{level: slog.LevelError}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello back"}}
	emb := &mockEmbedder{err: embedErr}
	vs := &mockVectorStore{}

	h := testHandlerDeps{
		router:                mustRouterSingle(t, provider),
		memVec:                SingleStoreMemoryVectors(vs),
		embedder:              emb,
		logger:                logger,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
		memoryVectorTopK:      testMemoryVectorTopK(10),
	}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "hi")
	if err != nil {
		t.Fatalf("HandleMessage err = %v, want nil (caller must not see index error)", err)
	}
	if reply != "hello back" {
		t.Errorf("reply = %q, want %q", reply, "hello back")
	}

	var hasIndexTurnError bool
	for _, r := range cap.records {
		if r.msg == "index turn" && r.level == slog.LevelError {
			hasIndexTurnError = true
			break
		}
	}
	if !hasIndexTurnError {
		t.Errorf("expected Error \"index turn\" record, got records: %+v", cap.records)
	}
}
