package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/toolcatalog"
	"pa/internal/tools"
	"pa/internal/tooltext"
	"pa/internal/vector"
	"strings"
	"testing"
)

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
	vec []float32
	err error
}

func (m *mockEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
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
}

func (m *mockVectorStore) Add(_ context.Context, _ string, _ []float32, chunk string) error {
	m.addChunks = append(m.addChunks, chunk)
	return m.addErr
}

func (m *mockVectorStore) Delete(_ context.Context, _ string) error { return nil }

func (m *mockVectorStore) Clear(_ context.Context) error { return nil }

func (m *mockVectorStore) Search(_ context.Context, _ []float32, _ int) ([]vector.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if m.searchResults != nil {
		return m.searchResults, nil
	}
	return nil, nil
}

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
	// runFunc optional per-call behavior (e.g. EP-006 multi-tool round tests). When set, stdout/err are ignored.
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

// Supporting AC-01.001, REQ-01.001: handler returns provider content to caller.
func TestHandleMessage_returnsProviderContent(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello back"}}
	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger}

	reply, err := h.HandleMessage(context.Background(), 99, "hi")
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
	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger}

	reply, err := h.HandleMessage(context.Background(), 1, "hi")
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if reply != "" {
		t.Errorf("reply = %q, want empty", reply)
	}
}

// Supporting AC-01.001, REQ-01.001: handler passes system and user messages to LLM provider.
func TestHandleMessage_passesSystemAndUserMessages(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger}

	userText := "what is 2+2?"
	_, _ = h.HandleMessage(context.Background(), 42, userText)

	if len(provider.lastMessages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(provider.lastMessages))
	}
	wantSystemPrefix := "You are a helpful assistant. Reply concisely."
	if provider.lastMessages[0].Role != "system" || !strings.HasPrefix(provider.lastMessages[0].Content, wantSystemPrefix) {
		t.Errorf("messages[0] = %+v, want system with prefix %q", provider.lastMessages[0], wantSystemPrefix)
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
	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger}

	for _, text := range []string{"", "  ", "\t\n"} {
		reply, err := h.HandleMessage(context.Background(), 1, text)
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
	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 5}

	// at limit: 5 runes — goes through
	reply, err := h.HandleMessage(context.Background(), 1, "12345")
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
	reply, err = h.HandleMessage(context.Background(), 1, "1234567")
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
	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 0}

	longText := strings.Repeat("a", 10000)
	reply, err := h.HandleMessage(context.Background(), 1, longText)
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
	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger}

	_, _ = h.HandleMessage(context.Background(), 1, "hi")

	var hasLLMCall bool
	for _, r := range cap.records {
		if r.msg == "llm call" && r.level == slog.LevelInfo {
			hasLLMCall = true
			break
		}
	}
	if !hasLLMCall {
		t.Errorf("expected one Info \"llm call\" record, got records: %+v", cap.records)
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
	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger}

	_, _ = h.HandleMessage(context.Background(), 1, "hi")

	var hasRequest, hasCall, hasResponse bool
	for _, r := range cap.records {
		switch r.msg {
		case "llm request":
			hasRequest = true
		case "llm call":
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

	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 6}
	reply, err := h.HandleMessage(context.Background(), 1, cyrillic6)
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
	h5 := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger, maxMessageLength: 5}
	reply, err = h5.HandleMessage(context.Background(), 1, cyrillic6)
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
	h := &conversationHandler{
		router: mustRouterSingle(t, provider),
		logger: logger,
		llmLog: capLog,
		model:  "openai/gpt-4o", // default from first provider
	}

	_, err := h.HandleMessage(context.Background(), 1, "hello")
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
	h := &conversationHandler{
		router: mustRouterSingle(t, provider),
		logger: logger,
		llmLog: capLog,
		model:  "openai/gpt-4o",
	}

	_, err := h.HandleMessage(context.Background(), 1, "hello")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if capLog.lastModel != "openai/gpt-4o" {
		t.Errorf("LLM log entry Model = %q, want openai/gpt-4o (h.model when result.Model empty)", capLog.lastModel)
	}
}

// Covers AC-01.014, REQ-01.007: semantic search results are injected into the system message as relevant past context.
func TestHandleMessage_injectsVectorSearchContextIntoSystemMessage(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	vs := &mockVectorStore{
		searchResults: []vector.SearchResult{{Text: "past mention of bananas"}},
	}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:           mustRouterSingle(t, provider),
		vectorStore:      vs,
		embedder:         emb,
		logger:           logger,
		contextMaxLen:    defaultContextMaxLen,
		vectorSearchTopK: defaultVectorSearchTopK,
	}

	_, err := h.HandleMessage(context.Background(), 1, "what did I say about fruit?")
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

// Covers AC-01.013, REQ-01.007: after successful LLM reply, handler indexes the turn (calls vectorStore.Add with user and assistant text).
func TestHandleMessage_indexTurnCallsAddWithUserAndReply(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "reply text", Usage: llm.Usage{}}}
	vs := &mockVectorStore{}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:           mustRouterSingle(t, provider),
		vectorStore:      vs,
		embedder:         emb,
		logger:           logger,
		contextMaxLen:    defaultContextMaxLen,
		vectorSearchTopK: defaultVectorSearchTopK,
	}

	_, err := h.HandleMessage(context.Background(), 1, "user said this")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(vs.addChunks) != 1 {
		t.Fatalf("Add calls = %d, want 1", len(vs.addChunks))
	}
	wantChunk := "User: user said this\nAssistant: reply text"
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
	h := &conversationHandler{
		router:      mustRouterSingle(t, provider),
		logger:      logger,
		logRedactor: redactor,
	}

	_, err := h.HandleMessage(context.Background(), 1, "user said secret")
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
	h := &conversationHandler{router: mustRouterSingle(t, provider), logger: logger, llmLog: nil}

	reply, err := h.HandleMessage(context.Background(), 1, "hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want ok", reply)
	}
}

// Covers AC-01.014, REQ-01.007: gatherContext includes only whole chunks that fit; when no chunk fits, no context is injected.
func TestHandleMessage_gatherContextTruncatesAtContextMaxLen(t *testing.T) {
	logger := slog.Default()
	// Single chunk too long to fit: nothing is injected (fitted 0/1).
	longText := strings.Repeat("x", defaultContextMaxLen+500)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	vs := &mockVectorStore{
		searchResults: []vector.SearchResult{{Text: longText}},
	}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:           mustRouterSingle(t, provider),
		vectorStore:      vs,
		embedder:         emb,
		logger:           logger,
		contextMaxLen:    defaultContextMaxLen,
		vectorSearchTopK: defaultVectorSearchTopK,
	}

	_, err := h.HandleMessage(context.Background(), 1, "query")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	sysContent := provider.lastMessages[0].Content
	if strings.Contains(sysContent, "Relevant past context") {
		t.Errorf("when no chunk fits, system message must not contain 'Relevant past context'; chunk was too long")
	}

	// Two chunks: first fits, second too long — only first is included, with trailing "..."
	shortChunk := "User: hi\nAssistant: hello"
	vs2 := &mockVectorStore{
		searchResults: []vector.SearchResult{
			{Text: shortChunk},
			{Text: strings.Repeat("y", defaultContextMaxLen)},
		},
	}
	h2 := &conversationHandler{
		router:           mustRouterSingle(t, provider),
		vectorStore:      vs2,
		embedder:         emb,
		logger:           logger,
		contextMaxLen:    defaultContextMaxLen,
		vectorSearchTopK: defaultVectorSearchTopK,
	}
	_, err = h2.HandleMessage(context.Background(), 1, "query")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	sysContent2 := provider.lastMessages[0].Content
	if !strings.Contains(sysContent2, "Relevant past context") {
		t.Errorf("system message must contain 'Relevant past context' when at least one chunk fits")
	}
	if !strings.Contains(sysContent2, shortChunk) {
		t.Errorf("system message must contain the first (fitting) chunk")
	}
	if !strings.Contains(sysContent2, "...") {
		t.Errorf("system message must contain '...' when not all chunks fit")
	}
	prefix := "Use the following context if relevant to the user's message."
	idx := strings.Index(sysContent2, prefix)
	if idx < 0 {
		t.Fatalf("system message missing expected prefix")
	}
	contextBlock := sysContent2[idx+len(prefix):]
	if len(contextBlock) > defaultContextMaxLen+10 {
		t.Errorf("context block length = %d, want at most contextMaxLen+10 (~%d)", len(contextBlock), defaultContextMaxLen+10)
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

	h := &conversationHandler{
		router:           mustRouterSingle(t, provider),
		vectorStore:      vs,
		embedder:         emb,
		logger:           logger,
		contextMaxLen:    defaultContextMaxLen,
		vectorSearchTopK: defaultVectorSearchTopK,
	}

	reply, err := h.HandleMessage(context.Background(), 1, "hi")
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
	h := &conversationHandler{catalog: catalog, nodeRunner: runner, logger: slog.Default()}

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
	h := &conversationHandler{
		catalog:        &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		nativeRegistry: reg,
		nodeRunner:     runner,
		logger:         slog.Default(),
	}
	out, err := h.executeOneToolCall(context.Background(), "run_on_node", `{"node_id":"nas","command":"uptime"}`)
	if err != nil {
		t.Fatalf("executeOneToolCall: %v", err)
	}
	if out != "up" {
		t.Errorf("got %q", out)
	}
}

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
	h := &conversationHandler{catalog: catalog, nodeRunner: runner, logger: slog.Default()}

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
	h := &conversationHandler{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}

	reply, err := h.HandleMessage(context.Background(), 1, "run echo hi")
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
	h := &conversationHandler{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}

	reply, err := h.HandleMessage(context.Background(), 1, "do something")
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
	h := &conversationHandler{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}

	reply, err := h.HandleMessage(context.Background(), 1, "run echo")
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

// Covers REQ-04.006: loop stops after maxToolRounds to avoid infinite loop when provider keeps returning tool_calls.
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
	h := &conversationHandler{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     slog.Default(),
	}

	reply, err := h.HandleMessage(context.Background(), 1, "run")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if callCount > maxToolRounds {
		t.Errorf("HandleMessage: Complete calls = %d, must be at most maxToolRounds=%d", callCount, maxToolRounds)
	}
	_ = reply
}

// Covers AC-04.013, REQ-04.016: tool invocations (id, arguments, result or error) are traceable in logs.
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
	h := &conversationHandler{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     logger,
	}

	_, err := h.HandleMessage(context.Background(), 1, "run echo hi")
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
	h := &conversationHandler{
		router:      mustRouterSingle(t, provider),
		catalog:     catalog,
		nodeRunner:  runner,
		logger:      logger,
		logRedactor: redactor,
	}

	_, err := h.HandleMessage(context.Background(), 1, "run echo")
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

// Covers REQ-01.026: INFO tool invocation error string is redacted (e.g. remote stderr from noderunner).
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
	h := &conversationHandler{
		router:      mustRouterSingle(t, provider),
		catalog:     catalog,
		nodeRunner:  runner,
		logger:      logger,
		logRedactor: redactor,
	}

	_, err := h.HandleMessage(context.Background(), 1, "run echo")
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
	h := &conversationHandler{
		router:     mustRouterSingle(t, provider),
		catalog:    catalog,
		nodeRunner: runner,
		logger:     logger,
	}

	_, _ = h.HandleMessage(context.Background(), 1, "do something")
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

	h := &conversationHandler{
		router:          mustRouterSingle(t, provider),
		catalog:         catalog,
		toolIndex:       ti,
		embedder:        emb,
		toolSearchTopK:  10,
		toolMinCount:    1,
		toolFallbackCap: 50,
		logger:          logger,
	}

	_, err := h.HandleMessage(context.Background(), 1, "check server status")
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
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     logger,
		firstProviderSupportsTools: true,
	}

	_, err := h.HandleMessage(context.Background(), 1, "use alpha")
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
	h := &conversationHandler{catalog: catalog, nodeRunner: runner, logger: slog.Default()}
	_, err := h.executeOneToolCall(context.Background(), "run_echo", `{"msg": "hi;rm -rf /"}`)
	if err == nil {
		t.Fatal("executeOneToolCall: expected error for metacharacter in substituted command")
	}
	if runner.lastCommand != "" {
		t.Errorf("RunOnNode must not run; lastCommand=%q", runner.lastCommand)
	}
}

// Substituted command with a disallowed rune (e.g. tab) must not reach RunOnNode.
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
	h := &conversationHandler{catalog: catalog, nodeRunner: runner, logger: slog.Default()}
	_, err := h.executeOneToolCall(context.Background(), "run_echo", "{\"msg\": \"x\\ty\"}")
	if err == nil {
		t.Fatal("executeOneToolCall: expected error for tab in substituted command")
	}
	if runner.lastCommand != "" {
		t.Errorf("RunOnNode must not run; lastCommand=%q", runner.lastCommand)
	}
}

// Catalog substitution passes cmdsafe gate in handler: INFO log includes tool_id, node_id, remote_command.
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
	h := &conversationHandler{catalog: catalog, nodeRunner: runner, logger: logger}
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

// REQ-04.027–029: text_based + first provider without tools → Hermes in content → execute → follow-up without tools, tool results as user.
// EP-008 AC-08.005 (integration): follow-up Complete keeps ForceJSONOutput so OpenAICompatible can apply REQ-08.005.
//
//nolint:gocyclo // Sequential scenario assertions; clarity over splitting.
func TestHandleMessage_textBasedHermes_toolRoundAndFinalReply(t *testing.T) {
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
	var secondOpts *llm.CompletionOptions
	var secondMsgs []llm.Message
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content: `<tool_call>
{"name": "run_echo", "arguments": {"msg": "hi"}}
</tool_call>`,
				Usage: llm.Usage{},
			}, nil
		}
		secondOpts = opts
		secondMsgs = append([]llm.Message(nil), messages...)
		return &llm.CompletionResult{Content: "Done with echo.", Usage: llm.Usage{}}, nil
	}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		nodeRunner:                 runner,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.New(cap),
		textBasedEnabled:           true,
		firstProviderSupportsTools: false,
	}

	reply, err := h.HandleMessage(context.Background(), 1, "run echo")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "Done with echo." {
		t.Errorf("reply = %q", reply)
	}
	if callCount != 2 {
		t.Errorf("Complete calls = %d, want 2", callCount)
	}
	if secondOpts != nil && len(secondOpts.Tools) > 0 {
		t.Error("follow-up Complete must not pass tools in opts")
	}
	if secondOpts == nil || !secondOpts.ForceJSONOutput {
		t.Errorf("follow-up Complete must keep ForceJSONOutput=true (Hermes JSON hint); opts=%+v", secondOpts)
	}
	var sawUserTool bool
	for _, m := range secondMsgs {
		if m.Role == "user" && strings.Contains(m.Content, "run_echo") && strings.Contains(m.Content, "hello from node") {
			sawUserTool = true
			break
		}
	}
	if !sawUserTool {
		t.Errorf("expected user message with tool result; messages=%+v", secondMsgs)
	}
	var sawHermesLog bool
	for _, r := range cap.records {
		if r.msg == "tool invocation" && r.attrs["invoked_via"] == "hermes" {
			sawHermesLog = true
			break
		}
	}
	if !sawHermesLog {
		t.Errorf("expected tool invocation with invoked_via=hermes; records=%+v", cap.records)
	}
}

// EP-008 AC-08.005 (integration): multi-round Hermes path preserves ForceJSONOutput on every Complete (copyOptsNoTools).
func TestHandleMessage_textBasedHermes_twoToolRounds_preservesForceJSONOnEachComplete(t *testing.T) {
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
	var optsPerCall []*llm.CompletionOptions
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		optsPerCall = append(optsPerCall, opts)
		switch callCount {
		case 1:
			return &llm.CompletionResult{
				Content: `<tool_call>
{"name": "run_echo", "arguments": {"msg": "first"}}
</tool_call>`,
				Usage: llm.Usage{},
			}, nil
		case 2:
			return &llm.CompletionResult{
				ToolCalls: []llm.ToolCall{{ID: "c2", Name: "run_echo", Arguments: `{"msg":"second"}`}},
				Usage:     llm.Usage{},
			}, nil
		default:
			return &llm.CompletionResult{Content: "All tools done.", Usage: llm.Usage{}}, nil
		}
	}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		nodeRunner:                 runner,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.Default(),
		textBasedEnabled:           true,
		firstProviderSupportsTools: false,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "run echo twice")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "All tools done." {
		t.Errorf("reply = %q", reply)
	}
	if callCount != 3 {
		t.Fatalf("Complete calls = %d, want 3", callCount)
	}
	for i, o := range optsPerCall {
		if o == nil || !o.ForceJSONOutput {
			t.Errorf("Complete #%d: want ForceJSONOutput=true, opts=%+v", i+1, o)
		}
		if i > 0 && len(o.Tools) > 0 {
			t.Errorf("Complete #%d: follow-up must omit tools in opts", i+1)
		}
	}
}

func TestHandleMessage_textBasedHermes_invalidMarkup_userMessage(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	provider := &mockProvider{result: &llm.CompletionResult{Content: `<tool_call>broken`, Usage: llm.Usage{}}}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		nodeRunner:                 &mockNodeRunner{},
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.Default(),
		textBasedEnabled:           true,
		firstProviderSupportsTools: false,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "x")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(reply, "Invalid tool call") {
		t.Errorf("reply = %q, want user-facing invalid tool message", reply)
	}
}

// AC-04.023: text-path first completion request includes Hermes format instructions and pre-selected tool ids in system message.
func TestHandleMessage_textBased_systemPromptIncludesHermesAndTools(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo on node", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	var firstSystem string
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, msgs []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		if len(msgs) > 0 && msgs[0].Role == "system" {
			firstSystem = msgs[0].Content
		}
		return &llm.CompletionResult{Content: "No tool call here.", Usage: llm.Usage{}}, nil
	}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		nodeRunner:                 &mockNodeRunner{},
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.Default(),
		textBasedEnabled:           true,
		firstProviderSupportsTools: false,
	}
	_, err := h.HandleMessage(context.Background(), 1, "ping")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(firstSystem, tooltext.FormatDescription) {
		t.Errorf("system prompt missing Hermes format description; len=%d", len(firstSystem))
	}
	if !strings.Contains(firstSystem, "run_echo") || !strings.Contains(firstSystem, "Echo on node") {
		t.Errorf("system prompt missing tool run_echo description; snippet=%.200q...", firstSystem)
	}
}

// AC-04.023/006: Hermes unknown tool id — same validation as native; no RunOnNode.
func TestHandleMessage_textBasedHermes_unknownTool_noRunOnNode(t *testing.T) {
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
				Content: `<tool_call>{"name": "unknown_tool", "arguments": {}}</tool_call>`,
				Usage:   llm.Usage{},
			}, nil
		}
		return &llm.CompletionResult{Content: "Second round.", Usage: llm.Usage{}}, nil
	}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		nodeRunner:                 runner,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.Default(),
		textBasedEnabled:           true,
		firstProviderSupportsTools: false,
	}
	_, err := h.HandleMessage(context.Background(), 1, "x")
	if err != nil {
		t.Fatal(err)
	}
	if runner.lastCommand != "" {
		t.Errorf("RunOnNode must not run for unknown tool; lastCommand=%q", runner.lastCommand)
	}
	if callCount != 2 {
		t.Errorf("Complete calls = %d, want 2", callCount)
	}
}

// AC-04.024: text-path response with no tool_call blocks returns plain assistant text.
func TestHandleMessage_textBasedHermes_plainTextNoBlocks(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	want := "I will answer without using tools."
	provider := &mockProvider{result: &llm.CompletionResult{Content: want, Usage: llm.Usage{}}}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		nodeRunner:                 &mockNodeRunner{},
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.Default(),
		textBasedEnabled:           true,
		firstProviderSupportsTools: false,
	}
	reply, err := h.HandleMessage(context.Background(), 1, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if reply != want {
		t.Errorf("reply = %q, want %q", reply, want)
	}
	if len(provider.lastMessages) != 2 {
		t.Errorf("expected single LLM round (system+user only in first call); messages len issue")
	}
}

// Hermes follow-up after tool round: malformed second response yields error (no silent success).
func TestHandleMessage_textBasedHermes_followUpMalformed_returnsError(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"run_echo": {
				ID: "run_echo", IndexText: "Echo", Template: "echo {{msg}}", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	runner := &mockNodeRunner{stdout: "out"}
	callCount := 0
	provider := &mockProvider{}
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		callCount++
		if callCount == 1 {
			return &llm.CompletionResult{
				Content: `<tool_call>{"name": "run_echo", "arguments": {"msg": "x"}}</tool_call>`,
				Usage:   llm.Usage{},
			}, nil
		}
		return &llm.CompletionResult{Content: `<tool_call>unclosed`, Usage: llm.Usage{}}, nil
	}
	toolStore := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "run_echo", Text: "echo", Score: 0.9}}}
	ti := &mockToolIndex{store: toolStore, ready: true}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		router:                     mustRouterSingle(t, provider),
		catalog:                    catalog,
		nodeRunner:                 runner,
		toolIndex:                  ti,
		embedder:                   emb,
		toolSearchTopK:             10,
		toolMinCount:               1,
		toolFallbackCap:            50,
		logger:                     slog.Default(),
		textBasedEnabled:           true,
		firstProviderSupportsTools: false,
	}
	_, err := h.HandleMessage(context.Background(), 1, "run echo")
	if err == nil {
		t.Fatal("expected error from follow-up Hermes parse failure")
	}
	if !strings.Contains(err.Error(), "follow-up tool_call parse") {
		t.Errorf("err = %v, want follow-up tool_call parse", err)
	}
	if callCount != 2 {
		t.Errorf("Complete calls = %d, want 2", callCount)
	}
}
