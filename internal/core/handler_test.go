package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"pa/internal/llm"
	"pa/internal/memory"
	"pa/internal/vector"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	searchResults []vector.SearchResult
	searchErr     error
}

func (m *mockVectorStore) Add(_ context.Context, _ string, _ []float32, _ string) error {
	return m.addErr
}

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

// Supporting AC-001 (US-01): handler returns provider content to caller.
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

// Supporting AC-001 (US-01): handler propagates provider error to caller.
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

// Supporting AC-001 (US-01): handler passes system and user messages to LLM provider.
func TestHandleMessage_passesSystemAndUserMessages(t *testing.T) {
	logger := slog.Default()
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	h := &conversationHandler{provider: provider, logger: logger}

	userText := "what is 2+2?"
	_, _ = h.HandleMessage(context.Background(), 42, userText)

	if len(provider.lastMessages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(provider.lastMessages))
	}
	wantSystemPrefix := "You are a helpful assistant. Reply concisely. You have access to relevant past context"
	if provider.lastMessages[0].Role != "system" || !strings.Contains(provider.lastMessages[0].Content, wantSystemPrefix) {
		t.Errorf("messages[0] = %+v, want system with %q", provider.lastMessages[0], wantSystemPrefix)
	}
	if provider.lastMessages[1].Role != "user" || provider.lastMessages[1].Content != userText {
		t.Errorf("messages[1] = %+v, want user + %q", provider.lastMessages[1], userText)
	}
	if provider.lastOpts != nil {
		t.Errorf("opts = %v, want nil", provider.lastOpts)
	}
}

// TestHandleMessage_emptyReturnsRejectionMessage covers AC-002 (unit): empty or whitespace message rejected with clear message, no LLM call.
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

// TestHandleMessage_rejectsWhenOverMaxLength covers AC-002 (unit): message over max length rejected, no LLM call.
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

// Supporting AC-002 (US-01): when max length is 0, long message is not truncated and goes to provider.
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

// TestHandleMessage_logsMetadataAtInfo covers AC-031 (REQ-021): at INFO level only metadata is logged.
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

// TestHandleMessage_logsFullRequestResponseAtDebug covers AC-031 (REQ-021): at DEBUG level full request and response are logged.
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

// TestHandleMessage_maxLength_unicodeRunes covers AC-002 (unit): max length enforced by runes.
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

// Supporting AC-036 (US-08): when memory read fails, handler still returns reply and includes vector context; system does not crash.
func TestHandleMessage_memoryReadError_stillIncludesVectorContext(t *testing.T) {
	// Create memory store root with today's path as a directory so ReadDay fails (os.ReadFile on dir returns error).
	rootDir := t.TempDir()
	now := time.Now().UTC()
	y, m, d := now.Year(), int(now.Month()), now.Day()
	dayPath := filepath.Join(rootDir, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", m), fmt.Sprintf("%02d.md", d))
	if err := os.MkdirAll(filepath.Dir(dayPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Mkdir(dayPath, 0o755); err != nil {
		t.Fatalf("mkdir day path as dir: %v", err)
	}

	memStore, err := memory.NewStore(rootDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	// Verify ReadDay fails (path is a directory, not a file).
	_, err = memStore.ReadDay(context.Background(), now)
	if err == nil {
		t.Fatal("ReadDay must fail when path is a directory")
	}

	vecStore := &mockVectorStore{
		searchResults: []vector.SearchResult{{Text: "User said: my favorite color is blue"}},
	}
	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}
	provider := &mockProvider{result: &llm.CompletionResult{Content: "ok"}}
	logger := slog.Default()

	h := &conversationHandler{
		provider:    provider,
		memoryStore: memStore,
		vectorStore: vecStore,
		embedder:    emb,
		logger:      logger,
	}

	reply, err := h.HandleMessage(context.Background(), 1, "what is my favorite color?")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply = %q, want %q", reply, "ok")
	}

	sys := provider.lastMessages[0].Content
	if !strings.Contains(sys, "Relevant past context:") {
		t.Errorf("system message must contain vector context; got:\n%s", sys)
	}
	if strings.Contains(sys, "Relevant memory (today)") {
		t.Errorf("system message must NOT contain memory content when ReadDay fails; got:\n%s", sys)
	}
}

// Supporting AC-036, AC-037 (US-08, US-07): when indexTurn fails (embedder error), handler still returns reply; system does not crash.
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
