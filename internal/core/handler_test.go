package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/memory"
	"pa/internal/prompt"
	"pa/internal/toolcatalog"
	"pa/internal/tools"
	"pa/internal/vector"
	"strings"
	"testing"
	"time"
)

func testMemoryVectorTopK(n int) config.MemoryVectorConfig {
	return config.MemoryVectorConfig{NotesTopK: n, SummariesTopK: n, TurnsTopK: n}
}

// captureHandler records log records for assertion (AC-01.031, REQ-01.021).
type captureHandler struct {
	level   slog.Level
	records []struct {
		level slog.Level
		msg   string
	}
}

func (c *captureHandler) Enabled(_ context.Context, level slog.Level) bool { return level >= c.level }
func (c *captureHandler) Handle(_ context.Context, r slog.Record) error {
	c.records = append(c.records, struct {
		level slog.Level
		msg   string
	}{r.Level, r.Message})
	return nil
}
func (c *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *captureHandler) WithGroup(string) slog.Handler      { return c }

// captureHandlerWithAttrs records level, message and attrs for assertion (AC-01.038).
type captureHandlerWithAttrs struct {
	level   slog.Level
	records []struct {
		level slog.Level
		msg   string
		attrs map[string]string
	}
}

func (c *captureHandlerWithAttrs) Enabled(_ context.Context, level slog.Level) bool {
	return level >= c.level
}

func (c *captureHandlerWithAttrs) Handle(_ context.Context, r slog.Record) error {
	attrs := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	c.records = append(c.records, struct {
		level slog.Level
		msg   string
		attrs map[string]string
	}{r.Level, r.Message, attrs})
	return nil
}
func (c *captureHandlerWithAttrs) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *captureHandlerWithAttrs) WithGroup(string) slog.Handler      { return c }

type mockProvider struct {
	result *llm.CompletionResult
	err    error
	// lastCall records the last Complete call (messages and opts) for assertion
	lastMessages []llm.Message
	lastOpts     *llm.CompletionOptions
	// CompleteFn if set overrides the default result/err behaviour (for multi-call tests).
	CompleteFn func(context.Context, []llm.Message, *llm.CompletionOptions) (*llm.CompletionResult, error)
}

func mustRouterSingle(t *testing.T, provider llm.Provider) *llmrouter.Router {
	t.Helper()
	r, err := llmrouter.New([]llm.Provider{provider}, []string{"test/default"}, llmrouter.Config{}, slog.Default())
	if err != nil {
		t.Fatalf("llmrouter.New: %v", err)
	}
	return r
}

func (m *mockProvider) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	m.lastMessages = messages
	m.lastOpts = opts
	if m.CompleteFn != nil {
		return m.CompleteFn(ctx, messages, opts)
	}
	return m.result, m.err
}

type mockEmbedder struct {
	vec        []float32
	err        error
	embedCalls int
}

func (m *mockEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	m.embedCalls++
	if m.err != nil {
		return nil, m.err
	}
	return m.vec, nil
}

type mockVectorStore struct {
	addErr        error
	addChunks     []string // chunks passed to Add for assertion (REQ-01.007)
	searchResults []vector.SearchResult
	searchErr     error
	searchCalls   int
}

func (m *mockVectorStore) Add(_ context.Context, _ string, _ []float32, chunk string) error {
	m.addChunks = append(m.addChunks, chunk)
	return m.addErr
}

func (m *mockVectorStore) Delete(_ context.Context, _ string) error { return nil }

func (m *mockVectorStore) Clear(_ context.Context) error { return nil }

func (m *mockVectorStore) Search(_ context.Context, _ []float32, _ int) ([]vector.SearchResult, error) {
	m.searchCalls++
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if m.searchResults != nil {
		return m.searchResults, nil
	}
	return nil, nil
}

func (m *mockVectorStore) Exists(context.Context, string) (bool, error) { return false, nil }

func (m *mockVectorStore) Close() error { return nil }

// mockToolIndex implements ToolIndex for tests (pre-selection uses Store and Ready).
type mockToolIndex struct {
	store vector.Store
	ready bool
}

func (m *mockToolIndex) Store() vector.Store { return m.store }
func (m *mockToolIndex) Ready() bool         { return m.ready }

// mockNodeRunner records the last RunOnNode call for assertion.
type mockNodeRunner struct {
	lastNodeID  string
	lastCommand string
	stdout      string
	err         error
	// runFunc optional per-call behavior for multi-step tool round tests. When set, stdout/err are ignored.
	runFunc func(ctx context.Context, nodeID, command string) (string, error)
}

func (m *mockNodeRunner) RunOnNode(ctx context.Context, nodeID, command string) (string, error) {
	m.lastNodeID = nodeID
	m.lastCommand = command
	if m.runFunc != nil {
		return m.runFunc(ctx, nodeID, command)
	}
	if m.err != nil {
		return "", m.err
	}
	return m.stdout, nil
}

// Covers AC-38.002, AC-38.014
// Supporting AC-01.001, REQ-01.001: handler returns provider content to caller.
func TestHandleMessage_returnsProviderContent(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello back"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	reply, err := h.HandleMessage(context.Background(), 99, "", "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "hello back" {
		t.Errorf("reply = %q, want %q", reply, "hello back")
	}
}

// Supporting AC-01.001, REQ-01.001: handler propagates provider error to caller.
func TestHandleMessage_returnsProviderError(t *testing.T) {
	wantErr := errors.New("provider failed")
	logger := slog.Default()
	provider := &mockProvider{err: wantErr}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "hi")
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if reply != "" {
		t.Errorf("reply = %q, want empty", reply)
	}
}

// Supporting AC-01.001, REQ-01.001: handler passes system and user messages to LLM provider.
// Covers AC-35.017: system message still begins with prompt.TrustPolicy after the prompt-package merge.
func TestHandleMessage_passesSystemAndUserMessages(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	userText := "what is 2+2?"
	_, _ = h.HandleMessage(context.Background(), 42, "", userText)

	if len(provider.lastMessages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(provider.lastMessages))
	}
	sys := provider.lastMessages[0].Content
	if provider.lastMessages[0].Role != "system" || !strings.HasPrefix(sys, prompt.TrustPolicy) {
		t.Errorf("messages[0] = %+v, want system starting with trust policy", provider.lastMessages[0])
	}
	if !strings.Contains(sys, "Calendar date: ") {
		t.Errorf("system message missing calendar date line: %s", sys)
	}
	wantDate := time.Now().UTC().Format("2006-01-02")
	if !strings.Contains(sys, wantDate) {
		t.Errorf("system message should contain today's UTC date %q", wantDate)
	}
	if !strings.Contains(sys, "You are a helpful assistant. Reply concisely.") {
		t.Errorf("system message missing personality line: %s", sys)
	}
	if provider.lastMessages[1].Role != "user" || provider.lastMessages[1].Content != userText {
		t.Errorf("messages[1] = %+v, want user + %q", provider.lastMessages[1], userText)
	}
	if provider.lastOpts != nil {
		t.Errorf("opts = %v, want nil", provider.lastOpts)
	}
}

// Covers AC-01.002, REQ-01.001: empty or whitespace message rejected with clear message, no LLM call.
func TestHandleMessage_emptyReturnsRejectionMessage(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "x"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	for _, text := range []string{"", "  ", "\t\n"} {
		reply, err := h.HandleMessage(context.Background(), 1, "", text)
		if err != nil {
			t.Errorf("text %q: err = %v", text, err)
		}
		if reply != "Please send a non-empty message." {
			t.Errorf("text %q: reply = %q, want rejection message", text, reply)
		}
	}
	if len(provider.lastMessages) != 0 {
		t.Error("provider.Complete should not be called for empty text")
	}
}

// Covers AC-01.002, REQ-01.001: message over max length rejected, no LLM call.
func TestHandleMessage_rejectsWhenOverMaxLength(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 5}.handler()

	// at limit: 5 runes — goes through
	reply, err := h.HandleMessage(context.Background(), 1, "", "12345")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q", reply)
	}
	if provider.lastMessages[1].Content != "12345" {
		t.Errorf("content = %q, want 12345", provider.lastMessages[1].Content)
	}

	// over limit: 7 runes — rejected, no LLM call
	provider.lastMessages = nil
	reply, err = h.HandleMessage(context.Background(), 1, "", "1234567")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "Message is too long. Maximum length is 5 characters." {
		t.Errorf("reply = %q", reply)
	}
	if len(provider.lastMessages) != 0 {
		t.Error("provider should not be called for over-length message")
	}
}

// Supporting AC-01.002, REQ-01.001: when max length is 0, long message is not truncated and goes to provider.
func TestHandleMessage_noLimit_longMessageGoesToProvider(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 0}.handler()

	longText := strings.Repeat("a", 10000)
	reply, err := h.HandleMessage(context.Background(), 1, "", longText)
	if err != nil {
		t.Fatal(err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q", reply)
	}
	if len(provider.lastMessages) != 2 || provider.lastMessages[1].Content != longText {
		t.Errorf("provider should receive full long message; got content len %d", len(provider.lastMessages[1].Content))
	}
}

// Covers AC-01.031, REQ-01.021: at INFO level only metadata is logged.
func TestHandleMessage_logsMetadataAtInfo(t *testing.T) {
	cap := &captureHandler{level: slog.LevelInfo}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3}}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	_, _ = h.HandleMessage(context.Background(), 1, "", "hi")

	var hasMainLLM bool
	for _, r := range cap.records {
		if r.msg == "main llm completion" && r.level == slog.LevelInfo {
			hasMainLLM = true
			break
		}
	}
	if !hasMainLLM {
		t.Errorf("expected one Info \"main llm completion\" record, got records: %+v", cap.records)
	}
	// No Debug records (request/response) at INFO level
	for _, r := range cap.records {
		if r.level == slog.LevelDebug {
			t.Errorf("at INFO level expected no Debug records, got msg=%q", r.msg)
		}
	}
}

// Covers AC-01.031, REQ-01.021: at DEBUG level full request and response are logged.
func TestHandleMessage_logsFullRequestResponseAtDebug(t *testing.T) {
	cap := &captureHandler{level: slog.LevelDebug}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello", Usage: llm.Usage{}}}
	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger}.handler()

	_, _ = h.HandleMessage(context.Background(), 1, "", "hi")

	var hasRequest, hasCall, hasResponse bool
	for _, r := range cap.records {
		switch r.msg {
		case "llm request":
			hasRequest = true
		case "main llm completion":
			hasCall = true
		case "llm response":
			hasResponse = true
		}
	}
	if !hasRequest {
		t.Errorf("at DEBUG expected \"llm request\" record, got %+v", cap.records)
	}
	if !hasCall {
		t.Errorf("at DEBUG expected \"llm call\" record, got %+v", cap.records)
	}
	if !hasResponse {
		t.Errorf("at DEBUG expected \"llm response\" record, got %+v", cap.records)
	}
}

// Covers AC-01.002, REQ-01.001: max length enforced by runes.
func TestHandleMessage_maxLength_unicodeRunes(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	// "привет" = 6 runes
	cyrillic6 := "привет"

	h := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 6}.handler()
	reply, err := h.HandleMessage(context.Background(), 1, "", cyrillic6)
	if err != nil {
		t.Fatal(err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want ok (at limit)", reply)
	}
	if provider.lastMessages[1].Content != cyrillic6 {
		t.Errorf("content = %q, want %q", provider.lastMessages[1].Content, cyrillic6)
	}

	// limit 5: 6 runes → rejected
	provider.lastMessages = nil
	h5 := testHandlerDeps{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 5}.handler()
	reply, err = h5.HandleMessage(context.Background(), 1, "", cyrillic6)
	if err != nil {
		t.Fatal(err)
	}
	if reply != "Message is too long. Maximum length is 5 characters." {
		t.Errorf("reply = %q", reply)
	}
	if len(provider.lastMessages) != 0 {
		t.Error("provider should not be called when over limit (runes)")
	}
}

// captureLLMLogWriter records the last Log call for assertion (AC-01.044).
type captureLLMLogWriter struct {
	lastModel string
}

func (c *captureLLMLogWriter) Log(entry *llmlog.Entry) {
	c.lastModel = entry.Model
}

// Covers AC-01.044, REQ-01.031, REQ-01.014: LLM log entry records the model/provider that produced the response (e.g. after fallback).
func TestHandleMessage_llmLogEntryRecordsResultModel(t *testing.T) {
	capLog := &captureLLMLogWriter{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hi", Usage: llm.Usage{}, Model: "ollama/llama3"}}
	h := testHandlerDeps{
		router: mustRouterSingle(t, provider),
		logger: logger,
		llmLog: capLog,
		model:  "openai/gpt-4o", // default from first provider
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "hello")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if capLog.lastModel != "ollama/llama3" {
		t.Errorf("LLM log entry Model = %q, want ollama/llama3 (result.Model when set)", capLog.lastModel)
	}
}

// Covers AC-01.044, REQ-01.031, REQ-01.014: when provider does not set result.Model, LLM log uses handler default (h.model).
func TestHandleMessage_llmLogEntryUsesDefaultModelWhenResultModelEmpty(t *testing.T) {
	capLog := &captureLLMLogWriter{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hi", Usage: llm.Usage{}}}
	h := testHandlerDeps{
		router: mustRouterSingle(t, provider),
		logger: logger,
		llmLog: capLog,
		model:  "openai/gpt-4o",
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "hello")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if capLog.lastModel != "openai/gpt-4o" {
		t.Errorf("LLM log entry Model = %q, want openai/gpt-4o (h.model when result.Model empty)", capLog.lastModel)
	}
}

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

// Covers AC-04.007, AC-04.010: valid tool call → substitute template → execute via RunOnNode; allowlist enforced by existing path.
func TestExecuteOneToolCall_ValidCall_RunsViaRunOnNode(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID:        "run_echo",
				IndexText: "Echo on node",
				Template:  "echo {{msg}}",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "hello from node"}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: slog.Default()}.handler()

	stdout, err := h.executeOneToolCall(context.Background(), "run_echo", `{"msg": "hello"}`)
	if err != nil {
		t.Fatalf("executeOneToolCall: %v", err)
	}
	if stdout != "hello from node" {
		t.Errorf("executeOneToolCall: stdout = %q, want hello from node", stdout)
	}
	if runner.lastNodeID != "nas" || runner.lastCommand != "echo hello" {
		t.Errorf("executeOneToolCall: RunOnNode called with (%q, %q), want (nas, echo hello)", runner.lastNodeID, runner.lastCommand)
	}
}

// Covers AC-09.008: native run_on_node dispatch when id not in catalog.
func TestExecuteOneToolCall_nativeRunOnNode(t *testing.T) {
	runner := &mockNodeRunner{stdout: "up"}
	reg := tools.NewRegistry()
	reg.Register(tools.NewRunOnNode(runner))
	h := testHandlerDeps{
		catalog:        &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		nativeRegistry: reg,
		nodeRunner:     runner,
		logger:         slog.Default(),
	}.handler()
	out, err := h.executeOneToolCall(context.Background(), "run_on_node", `{"node_id":"nas","command":"uptime"}`)
	if err != nil {
		t.Fatalf("executeOneToolCall: %v", err)
	}
	if out != "up" {
		t.Errorf("got %q", out)
	}
}

// Covers AC-01.002: traceability for TestRemoteCommandFromRunOnNodeArgs.
func TestRemoteCommandFromRunOnNodeArgs(t *testing.T) {
	if got := remoteCommandFromRunOnNodeArgs("run_on_node", `{"node_id":"nas","command":"  docker ps  "}`); got != "docker ps" {
		t.Errorf("got %q, want docker ps", got)
	}
	if got := remoteCommandFromRunOnNodeArgs("run_echo", `{"command":"x"}`); got != "" {
		t.Errorf("non-native tool: got %q, want empty", got)
	}
	if got := remoteCommandFromRunOnNodeArgs("run_on_node", `not json`); got != "" {
		t.Errorf("invalid json: got %q, want empty", got)
	}
}

// Covers AC-04.006: unknown tool → error, no RunOnNode called.
func TestExecuteOneToolCall_UnknownTool_ReturnsErrorNoRun(t *testing.T) {
	catalog := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}}
	runner := &mockNodeRunner{}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: slog.Default()}.handler()

	_, err := h.executeOneToolCall(context.Background(), "unknown", `{}`)
	if err == nil {
		t.Fatal("executeOneToolCall(unknown tool): expected error, got nil")
	}
	if runner.lastCommand != "" {
		t.Error("executeOneToolCall(unknown tool): RunOnNode must not be called")
	}
}

// Covers AC-04.004, AC-04.008: tool_calls → execution → tool results → provider called again → final reply to user; errors surfaced in chat.
func TestHandleMessage_toolResultLoop_returnsFinalReply(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "hello from node"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "Done. Result: hello from node.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "run echo hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if !strings.Contains(reply, "Done") && !strings.Contains(reply, "hello from node") {
		t.Errorf("HandleMessage: reply = %q, want final reply containing result", reply)
	}
	if callCount != 2 {
		t.Errorf("HandleMessage: provider.Complete calls = %d, want 2 (initial + after tool round)", callCount)
	}
	if runner.lastCommand != "echo hi" {
		t.Errorf("HandleMessage: RunOnNode command = %q, want echo hi", runner.lastCommand)
	}
}

// Covers AC-04.012: changed tool-loop prompt behavior is covered by unit tests.
func TestTruncateToolResultForPrompt(t *testing.T) {
	h := testHandlerDeps{toolResultPromptBytes: maxToolResultPromptBytes}.handler()
	small := "ok"
	if got := h.truncateToolResultForPrompt(small); got != small {
		t.Fatalf("small content changed: got %q", got)
	}
	large := strings.Repeat("a", maxToolResultPromptBytes+73)
	got := h.truncateToolResultForPrompt(large)
	if got == large {
		t.Fatal("expected large content to be truncated")
	}
	if !strings.Contains(got, "[tool output truncated: 73 bytes omitted]") {
		t.Fatalf("missing truncation marker: %q", got[len(got)-80:])
	}
	if len(got) >= len(large) {
		t.Fatalf("expected shorter content after truncation; got=%d want<%d", len(got), len(large))
	}
}

// Covers AC-39.006
func TestTruncateToolResultForPrompt_usesConfiguredLimit(t *testing.T) {
	const customLimit = 4096
	h := testHandlerDeps{toolResultPromptBytes: customLimit}.handler()
	large := strings.Repeat("b", customLimit+37)
	got := h.truncateToolResultForPrompt(large)
	if !strings.Contains(got, "[tool output truncated: 37 bytes omitted]") {
		t.Fatalf("missing truncation marker for configured limit: %q", got[len(got)-80:])
	}
}

// Covers AC-04.004: tool-result loop continues with tool outputs passed to follow-up completion.
func TestHandleMessage_toolResultLoop_largeToolOutput_truncatedForFollowUp(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	largeOut := strings.Repeat("z", maxToolResultPromptBytes+512)
	runner := &mockNodeRunner{stdout: largeOut}
	callCount := 0
	var secondCallMessages []llm.Message
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		secondCallMessages = append([]llm.Message(nil), messages...)
		return &llm.CompletionResult{Content: "done", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, provider),
		catalog:               catalog,
		nodeRunner:            runner,
		logger:                slog.Default(),
		toolResultPromptBytes: maxToolResultPromptBytes,
	}.handler()
	_, err := h.HandleMessage(context.Background(), 1, "", "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var found bool
	for _, m := range secondCallMessages {
		if m.Role != "tool" {
			continue
		}
		if strings.Contains(m.Content, "[tool output truncated:") {
			found = true
			if len(m.Content) >= len(largeOut) {
				t.Fatalf("tool message was not reduced; got=%d want<%d", len(m.Content), len(largeOut))
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected truncated tool message; messages=%+v", secondCallMessages)
	}
}

// Covers AC-04.006, AC-04.008: tool_call with invalid args (unknown tool) → no RunOnNode; error surfaced in chat (in messages to next provider call or in reply).
func TestHandleMessage_toolResultLoop_invalidArgs_noRunOnNode_errorInChat(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	callCount := 0
	var secondCallMessages []llm.Message
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			// Return tool_call with unknown tool id so validation fails.
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "unknown_tool", Arguments: `{"x":"y"}`}},
			}, nil
		}
		secondCallMessages = messages
		return &llm.CompletionResult{Content: "I could not run the tool: unknown tool.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "do something")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if runner.lastCommand != "" {
		t.Errorf("HandleMessage(invalid tool): RunOnNode must not be called; got lastCommand = %q", runner.lastCommand)
	}
	// Error must be visible: either in the tool result message passed to the second provider call, or in the final reply.
	var toolResultContainsError bool
	for _, m := range secondCallMessages {
		if m.Role == "tool" && strings.Contains(m.Content, "unknown tool") {
			toolResultContainsError = true
			break
		}
	}
	if !toolResultContainsError && !strings.Contains(reply, "unknown") {
		t.Errorf("HandleMessage: error must be in tool result or reply; secondCallMessages=%d, reply=%q", len(secondCallMessages), reply)
	}
}

// Covers AC-04.008: execution failure (run_on_node error) surfaced to user in chat (tool result content and/or final reply).
func TestHandleMessage_toolResultLoop_executionError_surfacedInChat(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	execErr := fmt.Errorf("command not in allowlist")
	runner := &mockNodeRunner{err: execErr}
	callCount := 0
	var secondCallMessages []llm.Message
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		secondCallMessages = messages
		return &llm.CompletionResult{Content: "The tool failed: command not in allowlist.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var toolResultContainsError bool
	for _, m := range secondCallMessages {
		if m.Role == "tool" && strings.Contains(m.Content, "allowlist") {
			toolResultContainsError = true
			break
		}
	}
	if !toolResultContainsError {
		t.Errorf("HandleMessage: execution error must appear in tool result message; messages = %v", secondCallMessages)
	}
	if !strings.Contains(reply, "allowlist") && !strings.Contains(reply, "failed") {
		t.Errorf("HandleMessage: reply should surface execution error; got %q", reply)
	}
}

// Covers AC-38.006
// Covers REQ-04.006: loop stops after maxToolRounds to avoid infinite loop when provider keeps returning tool_calls.
// Covers AC-01.002: traceability for TestHandleMessage_toolResultLoop_maxToolRounds_cap.
func TestHandleMessage_toolResultLoop_maxToolRounds_cap(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "ok"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		// Always return tool_calls so the loop would run forever without a cap.
		return &llm.CompletionResult{
			Content:   "",
			Usage:     llm.Usage{},
			ToolCalls: []llm.ToolCall{{ID: "call_x", Name: "run_echo", Arguments: `{"msg": "x"}`}},
		}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}.handler()

	reply, err := h.HandleMessage(context.Background(), 1, "", "run")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if callCount > maxToolRounds {
		t.Errorf("HandleMessage: Complete calls = %d, must be at most maxToolRounds=%d", callCount, maxToolRounds)
	}
	_ = reply
}

// Covers AC-04.013, REQ-04.016, AC-30.010: tool invocations are traceable; invoked_via is never hermes on the native path.
func TestHandleMessage_toolInvocation_loggedWithIdArgumentsAndResult(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "hello from node"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "Done.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     logger,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "run echo hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var found bool
	for _, r := range cap.records {
		if r.msg != "tool invocation" {
			continue
		}
		found = true
		if r.attrs["tool_id"] != "run_echo" {
			t.Errorf("tool invocation log: tool_id = %q, want run_echo", r.attrs["tool_id"])
		}
		if r.attrs["arguments"] != `{"msg": "hi"}` {
			t.Errorf("tool invocation log: arguments = %q", r.attrs["arguments"])
		}
		if r.attrs["result"] != "hello from node" {
			t.Errorf("tool invocation log: result = %q, want hello from node", r.attrs["result"])
		}
		if r.attrs["invoked_via"] != "tool_calls" {
			t.Errorf("tool invocation log: invoked_via = %q, want tool_calls", r.attrs["invoked_via"])
		}
		break
	}
	if !found {
		t.Errorf("expected one Info \"tool invocation\" record; got records: %+v", cap.records)
	}
}

// Covers REQ-01.026: logRedactor applies to INFO tool invocation attrs (arguments, result, error).
// Covers AC-01.002: traceability for TestHandleMessage_toolInvocation_redactsInfoLogAttrs.
func TestHandleMessage_toolInvocation_redactsInfoLogAttrs(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	redactor := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "node says secret"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "secret"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "Done.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:      mustRouterSingle(t, provider),
		catalog:     catalog,
		nodeRunner:  runner,
		logger:      logger,
		logRedactor: redactor,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var argsLog, resLog string
	for _, r := range cap.records {
		if r.msg == "tool invocation" {
			argsLog = r.attrs["arguments"]
			resLog = r.attrs["result"]
			break
		}
	}
	if argsLog == "" && resLog == "" {
		t.Fatalf("expected tool invocation log; records=%+v", cap.records)
	}
	if strings.Contains(argsLog, "secret") || !strings.Contains(argsLog, "[REDACTED]") {
		t.Errorf("arguments attr should be redacted; got %q", argsLog)
	}
	if strings.Contains(resLog, "secret") || !strings.Contains(resLog, "[REDACTED]") {
		t.Errorf("result attr should be redacted; got %q", resLog)
	}
}

// Covers AC-16.019: write_memory tool invocation arguments are passed through the same log redactor as other native tools.
func TestHandleMessage_writeMemory_toolInvocation_redactsArguments(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	redactor := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }
	memDir := t.TempDir()
	store, err := memory.NewStore(memDir, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	reg := tools.NewRegistry()
	reg.Register(tools.NewWriteMemoryTool(store, nil, nil, 4096, 1<<20))
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "w1", Name: "write_memory", Arguments: `{"text":"note secret body","date":"2026-04-14"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "saved", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		catalog:                    &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		nativeRegistry:             reg,
		logger:                     logger,
		logRedactor:                redactor,
		firstProviderSupportsTools: true,
		maxDynamicSystemRunes:      defaultMaxDynamicSystemRunes,
		memoryVectorTopK:           testMemoryVectorTopK(10),
	}.handler()
	_, err = h.HandleMessage(context.Background(), 1, "", "remember this")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var argsLog string
	for _, r := range cap.records {
		if r.msg == "tool invocation" && r.attrs["tool_id"] == "write_memory" {
			argsLog = r.attrs["arguments"]
			break
		}
	}
	if argsLog == "" {
		t.Fatalf("expected write_memory tool invocation log; records=%+v", cap.records)
	}
	if strings.Contains(argsLog, "secret") || !strings.Contains(argsLog, "[REDACTED]") {
		t.Errorf("write_memory arguments should be redacted; got %q", argsLog)
	}
}

// Covers REQ-01.026: INFO tool invocation error string is redacted (e.g. remote stderr from noderunner).
// Covers AC-01.002: traceability for TestHandleMessage_toolInvocation_redactsErrorAttr.
func TestHandleMessage_toolInvocation_redactsErrorAttr(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	redactor := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{err: errors.New("stderr: secret failure")}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg": "hi"}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "after tool", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:      mustRouterSingle(t, provider),
		catalog:     catalog,
		nodeRunner:  runner,
		logger:      logger,
		logRedactor: redactor,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	var errLog string
	for _, r := range cap.records {
		if r.msg == "tool invocation" && r.attrs["error"] != "" {
			errLog = r.attrs["error"]
			break
		}
	}
	if errLog == "" {
		t.Fatalf("expected tool invocation with error attr; records=%+v", cap.records)
	}
	if strings.Contains(errLog, "secret") || !strings.Contains(errLog, "[REDACTED]") {
		t.Errorf("error attr should be redacted; got %q", errLog)
	}
}

// Covers AC-04.013: tool invocation that fails (e.g. validation) is logged with error.
func TestHandleMessage_toolInvocation_loggedWithError(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content:   "",
				Usage:     llm.Usage{},
				ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "unknown_tool", Arguments: `{}`}},
			}, nil
		}
		return &llm.CompletionResult{Content: "Tool failed.", Usage: llm.Usage{}}, nil
	}
	h := testHandlerDeps{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     logger,
	}.handler()

	_, _ = h.HandleMessage(context.Background(), 1, "", "do something")
	var found bool
	for _, r := range cap.records {
		if r.msg == "tool invocation" && r.attrs["tool_id"] == "unknown_tool" {
			found = true
			if r.attrs["error"] == "" {
				t.Error("tool invocation (failed): expected error attr in log")
			}
			if r.attrs["invoked_via"] != "tool_calls" {
				t.Errorf("invoked_via = %q, want tool_calls", r.attrs["invoked_via"])
			}
			break
		}
	}
	if !found {
		t.Errorf("expected \"tool invocation\" record with error; got records: %+v", cap.records)
	}
}

// Covers AC-04.003, AC-04.015: when toolIndex and catalog are set, completion request includes pre-selected tools in provider format.
func TestHandleMessage_requestContainsPreselectedTools(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_uptime": {
				ID:        "run_uptime",
				IndexText: "Run uptime on the node",
				Template:  "uptime",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "node_id", Type: "string", Required: true}},
			},
		},
	}
	// Tool index returns run_uptime from search (or fallback will include it from catalog).
	toolStore := &mockVectorStore{
		searchResults: []vector.SearchResult{{ID: "run_uptime", Text: "uptime", Score: 0.9}},
	}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}

	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     logger,
		firstProviderSupportsTools: true,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "check server status")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if provider.lastOpts == nil {
		t.Fatal("expected Complete to be called with non-nil opts (tools)")
	}
	if len(provider.lastOpts.Tools) == 0 {
		t.Errorf("expected pre-selected tools in request; got opts.Tools empty")
	}
	// First tool should be from catalog (pre-selection returned run_uptime).
	found := false
	for _, td := range provider.lastOpts.Tools {
		if td.Name == "run_uptime" {
			found = true
			if td.Description != "Run uptime on the node" {
				t.Errorf("ToolDef Description = %q, want Run uptime on the node", td.Description)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected tool run_uptime in opts.Tools; got %v", provider.lastOpts.Tools)
	}
}

// Covers AC-30.001: with catalog tools selected, native tool defs are attached on the completion path (REQ-30.002, REQ-30.016).
// AC-04.026 / REQ-04.032: first system message includes per-tool [id] blocks for non-empty system_prompt when tools are selected.
func TestHandleMessage_firstSystemMessage_includesSystemPromptSections(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	const marker = "UNIQUE_SYSTEM_PROMPT_MARKER_8264"
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"alpha_tool": {
				ID:           "alpha_tool",
				IndexText:    "Alpha capability for testing",
				SystemPrompt: marker + "\nSecond line of rules.",
				Template:     "echo alpha",
				NodeID:       "nas",
				Arguments:    []toolcatalog.ArgumentRule{},
			},
		},
	}
	toolStore := &mockVectorStore{
		searchResults: []vector.SearchResult{{ID: "alpha_tool", Text: "alpha", Score: 0.9}},
	}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     logger,
		firstProviderSupportsTools: true,
	}.handler()

	_, err := h.HandleMessage(context.Background(), 1, "", "use alpha")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(provider.lastMessages) < 1 {
		t.Fatal("expected at least one message to provider")
	}
	sys := provider.lastMessages[0]
	if sys.Role != "system" {
		t.Fatalf("first message role = %q, want system", sys.Role)
	}
	if !strings.Contains(sys.Content, "Tool instructions:") {
		t.Errorf("system message missing Tool instructions header: %q", sys.Content)
	}
	if !strings.Contains(sys.Content, "[alpha_tool]") {
		t.Errorf("system message missing [alpha_tool] section: %q", sys.Content)
	}
	if !strings.Contains(sys.Content, marker) {
		t.Errorf("system message missing system_prompt body: %q", sys.Content)
	}
	if provider.lastOpts == nil || len(provider.lastOpts.Tools) == 0 {
		t.Fatalf("expected native tool defs in completion options, got opts=%v", provider.lastOpts)
	}
}

// Covers AC-30.002: assistant free-text `<tool_call>` without native tool_calls does not execute catalog tools.
func TestHandleMessage_fakeToolCallMarkupWithoutNativeToolCalls_noToolExecution(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	provider := &mockProvider{result: &llm.CompletionResult{
		Content: `<tool_call>{"name":"run_echo","arguments":{"msg":"x"}}</tool_call>`,
		Usage:   llm.Usage{},
	}}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := testHandlerDeps{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		nodeRunner:                 runner,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.Default(),
		firstProviderSupportsTools: true,
	}.handler()
	if _, err := h.HandleMessage(context.Background(), 1, "", "run echo"); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if runner.lastCommand != "" {
		t.Fatalf("RunOnNode must not run for markup-only assistant text; lastCommand=%q", runner.lastCommand)
	}
}

// Covers AC-04.029 (REQ-04.031): substituted command must pass cmdsafe.ValidateRemoteCommand before RunOnNode (e.g. `;` rejected as disallowed rune).
func TestExecuteOneToolCall_substitutedCommandWithMetachar_noRunOnNode(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID:        "run_echo",
				IndexText: "Echo",
				Template:  "echo {{msg}}",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: slog.Default()}.handler()
	_, err := h.executeOneToolCall(context.Background(), "run_echo", `{"msg": "hi;rm -rf /"}`)
	if err == nil {
		t.Fatal("executeOneToolCall: expected error for metacharacter in substituted command")
	}
	if runner.lastCommand != "" {
		t.Errorf("RunOnNode must not run; lastCommand=%q", runner.lastCommand)
	}
}

// Substituted command with a disallowed rune (e.g. tab) must not reach RunOnNode.
// Covers AC-01.002: traceability for TestExecuteOneToolCall_substitutedCommandWithDisallowedRune_noRunOnNode.
func TestExecuteOneToolCall_substitutedCommandWithDisallowedRune_noRunOnNode(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID:        "run_echo",
				IndexText: "Echo",
				Template:  "echo {{msg}}",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: slog.Default()}.handler()
	_, err := h.executeOneToolCall(context.Background(), "run_echo", "{\"msg\": \"x\\ty\"}")
	if err == nil {
		t.Fatal("executeOneToolCall: expected error for tab in substituted command")
	}
	if runner.lastCommand != "" {
		t.Errorf("RunOnNode must not run; lastCommand=%q", runner.lastCommand)
	}
}

// Catalog substitution passes cmdsafe gate in handler: INFO log includes tool_id, node_id, remote_command.
// Covers AC-01.002: traceability for TestExecuteOneToolCall_catalogCmdsafeRejection_logsRemoteCommand.
func TestExecuteOneToolCall_catalogCmdsafeRejection_logsRemoteCommand(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID:        "run_echo",
				IndexText: "Echo",
				Template:  "echo {{msg}}",
				NodeID:    "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{}
	h := testHandlerDeps{catalog: catalog, nodeRunner: runner, logger: logger}.handler()
	_, err := h.executeOneToolCall(context.Background(), "run_echo", `{"msg": "bad;cmd"}`)
	if err == nil {
		t.Fatal("executeOneToolCall: expected cmdsafe error")
	}
	if runner.lastCommand != "" {
		t.Errorf("RunOnNode must not run; lastCommand=%q", runner.lastCommand)
	}
	var found bool
	for _, rec := range cap.records {
		if rec.msg != "catalog tool remote command rejected" {
			continue
		}
		found = true
		if rec.attrs["tool_id"] != "run_echo" || rec.attrs["node_id"] != "nas" {
			t.Errorf("attrs = %v", rec.attrs)
		}
		if !strings.Contains(rec.attrs["remote_command"], "bad") {
			t.Errorf("remote_command = %q", rec.attrs["remote_command"])
		}
		break
	}
	if !found {
		t.Fatalf("expected catalog tool remote command rejected log; records=%+v", cap.records)
	}
}

// Covers AC-14.006, AC-14.010, AC-14.012: prior exchange appears between system and current user, oldest first.
func TestHandleMessage_sessionMemory_injectsHistoryBetweenSystemAndUser(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "first reply"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "chat-1", "hello"); err != nil {
		t.Fatalf("first turn: %v", err)
	}
	p.result = &llm.CompletionResult{Content: "second reply"}
	if _, err := h.HandleMessage(ctx, 1, "chat-1", "follow up"); err != nil {
		t.Fatalf("second turn: %v", err)
	}
	msgs := p.lastMessages
	if len(msgs) != 4 {
		t.Fatalf("len(messages) = %d, want 4", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[1].Role != "user" || msgs[1].Content != "hello" ||
		msgs[2].Role != "assistant" || msgs[2].Content != "first reply" ||
		msgs[3].Role != "user" || msgs[3].Content != "follow up" {
		t.Errorf("messages: %#v", msgs)
	}
}

// Covers AC-14.007: when session memory off, only system + one user message.
func TestHandleMessage_sessionDisabled_singleUserAfterSystem(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "chat-1", "hi"); err != nil {
		t.Fatal(err)
	}
	if len(p.lastMessages) != 2 || p.lastMessages[0].Role != "system" || p.lastMessages[1].Role != "user" {
		t.Errorf("got %#v", p.lastMessages)
	}
}

// Covers AC-14.003: distinct session keys keep separate windows.
func TestHandleMessage_sessionMemory_distinctKeysIsolated(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "r1"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "100", "only A"); err != nil {
		t.Fatal(err)
	}
	p.result = &llm.CompletionResult{Content: "r2"}
	if _, err := h.HandleMessage(ctx, 1, "200", "only B"); err != nil {
		t.Fatal(err)
	}
	if len(p.lastMessages) != 2 {
		t.Fatalf("second chat should have no history, got %d msgs", len(p.lastMessages))
	}
	if p.lastMessages[1].Content != "only B" {
		t.Errorf("user msg = %q", p.lastMessages[1].Content)
	}
}

// Covers AC-14.008: cap drops oldest exchange.
func TestHandleMessage_sessionMemory_capEvictsOldest(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "r"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 1},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	for _, text := range []string{"m1", "m2", "m3"} {
		if _, err := h.HandleMessage(ctx, 1, "k", text); err != nil {
			t.Fatal(err)
		}
	}
	msgs := p.lastMessages
	if len(msgs) != 4 {
		t.Fatalf("want 4 msgs (system + 1 pair + current), got %d", len(msgs))
	}
	if msgs[1].Content != "m2" || msgs[2].Content != "r" {
		t.Errorf("expected window u2/a2 before m3, got %#v", msgs)
	}
}

// Covers AC-14.009: empty user message does not grow session window.
func TestHandleMessage_sessionMemory_emptyUser_noAppend(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "r"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "k", "ok"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleMessage(ctx, 1, "k", "   "); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleMessage(ctx, 1, "k", "after"); err != nil {
		t.Fatal(err)
	}
	msgs := p.lastMessages
	if len(msgs) != 4 {
		t.Fatalf("want one stored exchange before 'after', got %d msgs", len(msgs))
	}
	if msgs[1].Content != "ok" {
		t.Errorf("first user in history = %q", msgs[1].Content)
	}
}

// Covers AC-14.009: over max length rejection does not append.
func TestHandleMessage_sessionMemory_overMaxLength_noAppend(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	p := &mockProvider{result: &llm.CompletionResult{Content: "r"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxMessageLength:      3,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "k", "ok"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleMessage(ctx, 1, "k", "toobig"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.HandleMessage(ctx, 1, "k", "z"); err != nil {
		t.Fatal(err)
	}
	msgs := p.lastMessages
	if len(msgs) != 4 || msgs[1].Content != "ok" {
		t.Errorf("unexpected messages %#v", msgs)
	}
}

// Covers AC-14.011: vector path unchanged with session (no hits → still system + history + user).
func TestHandleMessage_sessionMemory_withVectorStoreEmpty_coexists(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	vec := &mockVectorStore{}
	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}
	p := &mockProvider{result: &llm.CompletionResult{Content: "r1"}}
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		memVec:                SingleStoreMemoryVectors(vec),
		embedder:              emb,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "k", "one"); err != nil {
		t.Fatal(err)
	}
	p.result = &llm.CompletionResult{Content: "r2"}
	if _, err := h.HandleMessage(ctx, 1, "k", "two"); err != nil {
		t.Fatal(err)
	}
	if len(p.lastMessages) < 4 {
		t.Fatalf("expected history + user, got %d", len(p.lastMessages))
	}
	if !strings.Contains(p.lastMessages[0].Content, "Host rules in this message") {
		t.Error("expected merged system first")
	}
}

// Covers AC-14.013: DEBUG logs redact session history content like other user text.
func TestHandleMessage_sessionMemory_debugLogsRedactHistoryUserText(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelDebug}
	logger := slog.New(cap)
	secret := "SECRET_TOKEN_XYZ"
	p := &mockProvider{result: &llm.CompletionResult{Content: "r1"}}
	redact := func(s string) string { return strings.ReplaceAll(s, secret, "[REDACTED]") }
	h := testHandlerDeps{
		router:                mustRouterSingle(t, p),
		logger:                logger,
		maxDynamicSystemRunes: 4000,
		memoryVectorTopK:      testMemoryVectorTopK(10),
		logRedactor:           redact,
		sessionCfg:            &config.ConversationSessionConfig{Enabled: true, MaxSessionExchanges: 10},
		sessionStore:          newSessionWindowStore(),
	}.handler()
	ctx := context.Background()
	if _, err := h.HandleMessage(ctx, 1, "k", "hello "+secret); err != nil {
		t.Fatal(err)
	}
	p.result = &llm.CompletionResult{Content: "r2"}
	if _, err := h.HandleMessage(ctx, 1, "k", "next"); err != nil {
		t.Fatal(err)
	}
	for _, r := range cap.records {
		if c, ok := r.attrs["content"]; ok && strings.Contains(c, secret) {
			t.Errorf("debug log leaked secret in content attr: %q", c)
		}
	}
}
