package tools

import (
	"context"
	"strings"
	"testing"
)

// Covers AC-32.009: search_vector_tool rejects missing/empty query.
func TestSearchVectorToolKnowledge_rejectsEmptyQuery(t *testing.T) {
	tool := NewSearchVectorToolKnowledgeTool(&recordingVectorStore{}, fixedSearchEmbedder{}, 3, 8, 2048, 120)
	_, err := tool.Run(context.Background(), map[string]any{"query": "   "})
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("expected query required error, got %v", err)
	}
}

// Covers AC-32.010: top_k bounds and deterministic ordering for search_vector_tool.
func TestSearchVectorToolKnowledge_topKBoundsAndDeterministicOrder(t *testing.T) {
	store := &recordingVectorStore{searchResults: []searchResultFixture{
		{id: "b", text: "B", score: 0.8},
		{id: "a", text: "A", score: 0.8},
		{id: "c", text: "C", score: 0.2},
	}}
	tool := NewSearchVectorToolKnowledgeTool(store, fixedSearchEmbedder{}, 3, 4, 2048, 120)
	_, err := tool.Run(context.Background(), map[string]any{"query": "q", "top_k": float64(9)})
	if err == nil || !strings.Contains(err.Error(), "top_k must be in 1..4") {
		t.Fatalf("expected top_k bounds error, got %v", err)
	}
	out, err := tool.Run(context.Background(), map[string]any{"query": "q", "top_k": float64(3)})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	posC := strings.Index(out, "c score=0.200000")
	posA := strings.Index(out, "a score=0.800000")
	posB := strings.Index(out, "b score=0.800000")
	if posC <= 0 || posA <= posC || posB <= posA {
		t.Fatalf("unexpected order in output: %q", out)
	}
}

// Covers AC-32.011 and AC-32.012: bounded output with truncation and no store mutation.
func TestSearchVectorSkillKnowledge_outputBoundedReadOnly(t *testing.T) {
	store := &recordingVectorStore{searchResults: []searchResultFixture{
		{id: "s1", text: strings.Repeat("x", 280), score: 0.1},
		{id: "s2", text: strings.Repeat("y", 280), score: 0.2},
		{id: "s3", text: strings.Repeat("z", 280), score: 0.3},
	}}
	tool := NewSearchVectorSkillKnowledgeTool(store, fixedSearchEmbedder{}, 5, 8, 320, 120)
	out, err := tool.Run(context.Background(), map[string]any{"query": "best skill for web output"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(out, "s3 score=0.300000") {
		t.Fatalf("expected bounded output to omit at least one row, got %q", out)
	}
	if len(out) > 320 {
		t.Fatalf("output length %d exceeds cap", len(out))
	}
	if store.addCalls+store.deleteCalls+store.clearCalls != 0 {
		t.Fatal("tool must not mutate vector store")
	}
}
