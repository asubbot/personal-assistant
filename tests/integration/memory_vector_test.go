//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/llm"
	"pa/internal/memory"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockLLMWithMessages captures the last messages passed to Complete (for asserting on system message content).
type mockLLMWithMessages struct {
	content      string
	LastMessages []llm.Message
}

func (m *mockLLMWithMessages) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	m.LastMessages = messages
	return &llm.CompletionResult{Content: m.content}, nil
}

// mockEmbedder returns a fixed vector for every text (for vector store integration tests).
type mockEmbedder struct {
	vec []float32
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return m.vec, nil
}

const vectorTestDimensions = 4

// TestMemoryStore_injectsTodayMemory (integration) validates AC-011, AC-012:
// core.Run with memory store reads from configured path and injects "Relevant memory (today):" into the LLM system message.
func TestMemoryStore_injectsTodayMemory(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	y, m, d := now.Year(), int(now.Month()), now.Day()
	calPath := filepath.Join(dir, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", m), fmt.Sprintf("%02d.md", d))
	if err := os.MkdirAll(filepath.Dir(calPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	const dayContent = "Pre-written day note for integration test."
	if err := os.WriteFile(calPath, []byte(dayContent), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	store, err := memory.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	provider := &mockLLMWithMessages{content: "ok"}
	adapter := &fakeAdapter{userID: 1, text: "hello", done: make(chan result, 1)}
	cfg := &config.Config{}
	logger := slog.Default()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = core.Run(ctx, cfg, logger, adapter, provider, store, nil, nil)
		close(done)
	}()

	select {
	case res := <-adapter.done:
		if res.err != nil {
			t.Fatalf("handler error: %v", res.err)
		}
	case <-time.After(integrationTimeout):
		t.Fatalf("no reply within %v", integrationTimeout)
	}

	cancel()
	<-done

	if len(provider.LastMessages) < 1 || provider.LastMessages[0].Role != "system" {
		t.Fatalf("expected system message, got %d messages", len(provider.LastMessages))
	}
	sys := provider.LastMessages[0].Content
	if !strings.Contains(sys, "Relevant memory (today):") {
		t.Errorf("system message missing Relevant memory (today):\n%s", sys)
	}
	if !strings.Contains(sys, dayContent) {
		t.Errorf("system message missing day content %q:\n%s", dayContent, sys)
	}
}

// TestVectorStore_injectsPastContext (integration) validates AC-013, AC-014:
// after one turn is indexed, a second message gets "Relevant past context:" from vector search in the system message.
func TestVectorStore_injectsPastContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pa_vectors.sqlite")
	vecStore, err := sqlite.New(path, vectorTestDimensions)
	if err != nil {
		t.Fatalf("vector store New: %v", err)
	}
	defer func() { _ = vecStore.Close() }()

	emb := &mockEmbedder{vec: []float32{1, 0, 0, 0}}
	provider := &mockLLMWithMessages{content: "Noted: March 15."}
	cfg := &config.Config{}
	logger := slog.Default()

	// First run: user states a fact -> indexTurn adds it to vector store.
	adapter1 := &fakeAdapter{userID: 1, text: "My project deadline is March 15.", done: make(chan result, 1)}
	ctx1, cancel1 := context.WithCancel(context.Background())
	done1 := make(chan struct{})
	go func() {
		_ = core.Run(ctx1, cfg, logger, adapter1, provider, nil, vecStore, emb)
		close(done1)
	}()

	select {
	case res := <-adapter1.done:
		if res.err != nil {
			t.Fatalf("first run handler error: %v", res.err)
		}
	case <-time.After(integrationTimeout):
		t.Fatalf("first run: no reply within %v", integrationTimeout)
	}
	cancel1()
	<-done1

	// Second run: same vector store and embedder; question should retrieve the indexed chunk.
	adapter2 := &fakeAdapter{userID: 1, text: "When is my deadline?", done: make(chan result, 1)}
	ctx2, cancel2 := context.WithCancel(context.Background())
	done2 := make(chan struct{})
	go func() {
		_ = core.Run(ctx2, cfg, logger, adapter2, provider, nil, vecStore, emb)
		close(done2)
	}()

	select {
	case res := <-adapter2.done:
		if res.err != nil {
			t.Fatalf("second run handler error: %v", res.err)
		}
	case <-time.After(integrationTimeout):
		t.Fatalf("second run: no reply within %v", integrationTimeout)
	}
	cancel2()
	<-done2

	if len(provider.LastMessages) < 1 || provider.LastMessages[0].Role != "system" {
		t.Fatalf("expected system message, got %d messages", len(provider.LastMessages))
	}
	sys := provider.LastMessages[0].Content
	if !strings.Contains(sys, "Relevant past context:") {
		t.Errorf("system message missing Relevant past context:\n%s", sys)
	}
	if !strings.Contains(sys, "March 15") {
		t.Errorf("system message should contain indexed chunk (March 15):\n%s", sys)
	}
}
