package core

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"pa/internal/llmrouter"
	"pa/internal/vector"
	"testing"
)

func testMemoryVectorTopK(n int) config.MemoryVectorConfig {
	return uniformMemoryVectorConfig(n)
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
	// never-assigned-field: safe — nil is the default successful Add behavior for this test double
	addErr        error
	addChunks     []string // chunks passed to Add for assertion (REQ-01.007)
	searchResults []vector.SearchResult
	// never-assigned-field: safe — nil is the default successful Search behavior for this test double
	searchErr   error
	searchCalls int
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
	// never-assigned-field: safe — nil disables the optional per-call test override
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
