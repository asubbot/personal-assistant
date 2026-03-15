package core

import (
	"context"
	"errors"
	"log/slog"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/vector"
	"strings"
	"testing"
)

// captureHandler records log records for assertion (AC-031, REQ-021).
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

// captureHandlerWithAttrs records level, message and attrs for assertion (AC-038).
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
}

func (m *mockProvider) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	m.lastMessages = messages
	m.lastOpts = opts
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
	addChunks     []string // chunks passed to Add for assertion (REQ-007)
	searchResults []vector.SearchResult
	searchErr     error
}

func (m *mockVectorStore) Add(_ context.Context, _ string, _ []float32, chunk string) error {
	m.addChunks = append(m.addChunks, chunk)
	return m.addErr
}

func (m *mockVectorStore) Delete(_ context.Context, _ string) error { return nil }

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

// Supporting AC-001, REQ-001: handler returns provider content to caller.
func TestHandleMessage_returnsProviderContent(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello back"}}
	h := &conversationHandler{provider: provider, logger: logger}

	reply, err := h.HandleMessage(context.Background(), 99, "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "hello back" {
		t.Errorf("reply = %q, want %q", reply, "hello back")
	}
}

// Supporting AC-001, REQ-001: handler propagates provider error to caller.
func TestHandleMessage_returnsProviderError(t *testing.T) {
	wantErr := errors.New("provider failed")
	logger := slog.Default()
	provider := &mockProvider{err: wantErr}
	h := &conversationHandler{provider: provider, logger: logger}

	reply, err := h.HandleMessage(context.Background(), 1, "hi")
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if reply != "" {
		t.Errorf("reply = %q, want empty", reply)
	}
}

// Supporting AC-001, REQ-001: handler passes system and user messages to LLM provider.
func TestHandleMessage_passesSystemAndUserMessages(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := &conversationHandler{provider: provider, logger: logger}

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

// Covers AC-002, REQ-001: empty or whitespace message rejected with clear message, no LLM call.
func TestHandleMessage_emptyReturnsRejectionMessage(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "x"}}
	h := &conversationHandler{provider: provider, logger: logger}

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

// Covers AC-002, REQ-001: message over max length rejected, no LLM call.
func TestHandleMessage_rejectsWhenOverMaxLength(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := &conversationHandler{provider: provider, logger: logger, maxMessageLength: 5}

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

// Supporting AC-002, REQ-001: when max length is 0, long message is not truncated and goes to provider.
func TestHandleMessage_noLimit_longMessageGoesToProvider(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := &conversationHandler{provider: provider, logger: logger, maxMessageLength: 0}

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

// Covers AC-031, REQ-021: at INFO level only metadata is logged.
func TestHandleMessage_logsMetadataAtInfo(t *testing.T) {
	cap := &captureHandler{level: slog.LevelInfo}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3}}}
	h := &conversationHandler{provider: provider, logger: logger}

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

// Covers AC-031, REQ-021: at DEBUG level full request and response are logged.
func TestHandleMessage_logsFullRequestResponseAtDebug(t *testing.T) {
	cap := &captureHandler{level: slog.LevelDebug}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello", Usage: llm.Usage{}}}
	h := &conversationHandler{provider: provider, logger: logger}

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

// Covers AC-002, REQ-001: max length enforced by runes.
func TestHandleMessage_maxLength_unicodeRunes(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	// "привет" = 6 runes
	cyrillic6 := "привет"

	h := &conversationHandler{provider: provider, logger: logger, maxMessageLength: 6}
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
	h5 := &conversationHandler{provider: provider, logger: logger, maxMessageLength: 5}
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

// captureLLMLogWriter records the last Log call for assertion (AC-044).
type captureLLMLogWriter struct {
	lastModel string
}

func (c *captureLLMLogWriter) Log(entry *llmlog.Entry) {
	c.lastModel = entry.Model
}

// Covers AC-044, REQ-031, REQ-014: LLM log entry records the model/provider that produced the response (e.g. after fallback).
func TestHandleMessage_llmLogEntryRecordsResultModel(t *testing.T) {
	capLog := &captureLLMLogWriter{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hi", Usage: llm.Usage{}, Model: "ollama/llama3"}}
	h := &conversationHandler{
		provider: provider,
		logger:   logger,
		llmLog:   capLog,
		model:    "openai/gpt-4o", // default from first provider
	}

	_, err := h.HandleMessage(context.Background(), 1, "hello")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if capLog.lastModel != "ollama/llama3" {
		t.Errorf("LLM log entry Model = %q, want ollama/llama3 (result.Model when set)", capLog.lastModel)
	}
}

// Covers AC-044, REQ-031, REQ-014: when provider does not set result.Model, LLM log uses handler default (h.model).
func TestHandleMessage_llmLogEntryUsesDefaultModelWhenResultModelEmpty(t *testing.T) {
	capLog := &captureLLMLogWriter{}
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hi", Usage: llm.Usage{}}}
	h := &conversationHandler{
		provider: provider,
		logger:   logger,
		llmLog:   capLog,
		model:    "openai/gpt-4o",
	}

	_, err := h.HandleMessage(context.Background(), 1, "hello")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if capLog.lastModel != "openai/gpt-4o" {
		t.Errorf("LLM log entry Model = %q, want openai/gpt-4o (h.model when result.Model empty)", capLog.lastModel)
	}
}

// Covers AC-014, REQ-007: semantic search results are injected into the system message as relevant past context.
func TestHandleMessage_injectsVectorSearchContextIntoSystemMessage(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	vs := &mockVectorStore{
		searchResults: []vector.SearchResult{{Text: "past mention of bananas"}},
	}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		provider:    provider,
		vectorStore: vs,
		embedder:    emb,
		logger:      logger,
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

// Covers AC-013, REQ-007: after successful LLM reply, handler indexes the turn (calls vectorStore.Add with user and assistant text).
func TestHandleMessage_indexTurnCallsAddWithUserAndReply(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "reply text", Usage: llm.Usage{}}}
	vs := &mockVectorStore{}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		provider:    provider,
		vectorStore: vs,
		embedder:    emb,
		logger:      logger,
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

// Covers AC-038, REQ-026, REQ-027: at DEBUG level, logRedactor is applied to request/response content before app log.
func TestHandleMessage_logRedactorAppliedInDebugLogs(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelDebug}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "response contains secret", Usage: llm.Usage{}}}
	redactor := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }
	h := &conversationHandler{
		provider:    provider,
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

// Covers AC-018, REQ-015: when LLM log is not configured (llmLog nil), handler does not attempt to write; no panic.
func TestHandleMessage_llmLogNil_succeedsWithoutWrite(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	h := &conversationHandler{provider: provider, logger: logger, llmLog: nil}

	reply, err := h.HandleMessage(context.Background(), 1, "hi")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want ok", reply)
	}
}

// Covers AC-014, REQ-007: gatherContext truncates injected context at contextMaxLen and appends "...".
func TestHandleMessage_gatherContextTruncatesAtContextMaxLen(t *testing.T) {
	logger := slog.Default()
	longText := strings.Repeat("x", contextMaxLen+500)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok", Usage: llm.Usage{}}}
	vs := &mockVectorStore{
		searchResults: []vector.SearchResult{{Text: longText}},
	}
	emb := &mockEmbedder{vec: []float32{0.1}}
	h := &conversationHandler{
		provider:    provider,
		vectorStore: vs,
		embedder:    emb,
		logger:      logger,
	}

	_, err := h.HandleMessage(context.Background(), 1, "query")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	sysContent := provider.lastMessages[0].Content
	if !strings.Contains(sysContent, "Relevant past context") {
		t.Errorf("system message must contain 'Relevant past context'")
	}
	if !strings.Contains(sysContent, "...") {
		t.Errorf("system message must contain truncation suffix '...' when context exceeds contextMaxLen")
	}
	// Injected block (after "Use the following context...") must be at most contextMaxLen+3
	prefix := "Use the following context if relevant to the user's message."
	idx := strings.Index(sysContent, prefix)
	if idx < 0 {
		t.Fatalf("system message missing expected prefix")
	}
	contextBlock := sysContent[idx+len(prefix):]
	if len(contextBlock) > contextMaxLen+10 {
		t.Errorf("context block length = %d, want at most contextMaxLen+3 (~%d)", len(contextBlock), contextMaxLen+3)
	}
}

// Supporting AC-036, AC-037, REQ-025: when indexTurn fails (embedder error), handler still returns reply; system does not crash.
func TestHandleMessage_indexTurnError_stillReturnsReply(t *testing.T) {
	embedErr := errors.New("embed failed")
	cap := &captureHandler{level: slog.LevelError}
	logger := slog.New(cap)
	provider := &mockProvider{result: &llm.CompletionResult{Content: "hello back"}}
	emb := &mockEmbedder{err: embedErr}
	vs := &mockVectorStore{}

	h := &conversationHandler{
		provider:    provider,
		vectorStore: vs,
		embedder:    emb,
		logger:      logger,
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
