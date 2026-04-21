package tools

import (
	"context"
	"pa/internal/vector"
	"strings"
	"testing"
)

type recordingVectorStore struct {
	searchResults []searchResultFixture
	searchErr     error
	searchCalls   int
	addCalls      int
	deleteCalls   int
	clearCalls    int
}

type searchResultFixture struct {
	id    string
	text  string
	score float64
}

func (s *recordingVectorStore) Add(context.Context, string, []float32, string) error {
	s.addCalls++
	return nil
}
func (s *recordingVectorStore) Delete(context.Context, string) error { s.deleteCalls++; return nil }
func (s *recordingVectorStore) Clear(context.Context) error          { s.clearCalls++; return nil }
func (s *recordingVectorStore) Exists(context.Context, string) (bool, error) {
	return false, nil
}
func (s *recordingVectorStore) Close() error { return nil }
func (s *recordingVectorStore) Search(context.Context, []float32, int) ([]vector.SearchResult, error) {
	s.searchCalls++
	if s.searchErr != nil {
		return nil, s.searchErr
	}
	out := make([]vector.SearchResult, 0, len(s.searchResults))
	for _, r := range s.searchResults {
		out = append(out, vector.SearchResult{ID: r.id, Text: r.text, Score: r.score})
	}
	return out, nil
}

type fixedSearchEmbedder struct{}

func (fixedSearchEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{1, 0, 0, 0}, nil
}

// Covers AC-31.002: search_vector_memory rejects missing/empty query.
func TestSearchVectorMemoryTool_rejectsEmptyQuery(t *testing.T) {
	tool := NewSearchVectorMemoryTool(&recordingVectorStore{}, &recordingVectorStore{}, &recordingVectorStore{}, fixedSearchEmbedder{}, 3, 8, 2048)
	_, err := tool.Run(context.Background(), map[string]any{"query": "   "})
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("expected query required error, got %v", err)
	}
}

// Covers AC-31.003 and AC-31.007: omitted lanes searches all lanes and performs read-only searches.
func TestSearchVectorMemoryTool_defaultLanesSearchAllReadOnly(t *testing.T) {
	notes := &recordingVectorStore{searchResults: []searchResultFixture{{id: "n1", text: "note", score: 0.2}}}
	summ := &recordingVectorStore{searchResults: []searchResultFixture{{id: "s1", text: "summary", score: 0.3}}}
	turns := &recordingVectorStore{searchResults: []searchResultFixture{{id: "t1", text: "turn", score: 0.4}}}
	tool := NewSearchVectorMemoryTool(notes, summ, turns, fixedSearchEmbedder{}, 3, 8, 2048)
	out, err := tool.Run(context.Background(), map[string]any{"query": "deadline"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, want := range []string{"[notes]", "[summaries]", "[turns]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing lane %s: %q", want, out)
		}
	}
	if notes.searchCalls != 1 || summ.searchCalls != 1 || turns.searchCalls != 1 {
		t.Fatalf("search calls notes=%d summaries=%d turns=%d", notes.searchCalls, summ.searchCalls, turns.searchCalls)
	}
	if notes.addCalls+notes.deleteCalls+notes.clearCalls+summ.addCalls+summ.deleteCalls+summ.clearCalls+turns.addCalls+turns.deleteCalls+turns.clearCalls != 0 {
		t.Fatal("tool must not mutate stores")
	}
}

// Covers AC-31.004: invalid lane is rejected with explicit error.
func TestSearchVectorMemoryTool_rejectsInvalidLane(t *testing.T) {
	tool := NewSearchVectorMemoryTool(&recordingVectorStore{}, &recordingVectorStore{}, &recordingVectorStore{}, fixedSearchEmbedder{}, 3, 8, 2048)
	_, err := tool.Run(context.Background(), map[string]any{
		"query": "q",
		"lanes": []any{"notes", "bad-lane"},
	})
	if err == nil || !strings.Contains(err.Error(), `invalid lane`) {
		t.Fatalf("expected invalid lane error, got %v", err)
	}
}

// Covers AC-31.005: top_k bounds and deterministic within-lane ordering.
func TestSearchVectorMemoryTool_topKBoundsAndDeterministicOrder(t *testing.T) {
	notes := &recordingVectorStore{searchResults: []searchResultFixture{
		{id: "b", text: "B", score: 0.8},
		{id: "a", text: "A", score: 0.8},
		{id: "c", text: "C", score: 0.2},
	}}
	tool := NewSearchVectorMemoryTool(notes, nil, nil, fixedSearchEmbedder{}, 3, 4, 2048)

	_, err := tool.Run(context.Background(), map[string]any{"query": "q", "top_k": float64(10)})
	if err == nil || !strings.Contains(err.Error(), "top_k must be in 1..4") {
		t.Fatalf("expected top_k bounds error, got %v", err)
	}

	out, err := tool.Run(context.Background(), map[string]any{"query": "q", "lanes": []any{"notes"}, "top_k": float64(3)})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	posC := strings.Index(out, " c score=0.200000")
	posA := strings.Index(out, " a score=0.800000")
	posB := strings.Index(out, " b score=0.800000")
	if posC <= 0 || posA <= posC || posB <= posA {
		t.Fatalf("unexpected order in output: %q", out)
	}
}

// Covers AC-31.006: output is bounded and reports truncation when limit is reached.
func TestSearchVectorMemoryTool_outputBoundedWithTruncationFooter(t *testing.T) {
	notes := &recordingVectorStore{searchResults: []searchResultFixture{
		{id: "n1", text: strings.Repeat("x", 300), score: 0.1},
		{id: "n2", text: strings.Repeat("y", 300), score: 0.2},
		{id: "n3", text: strings.Repeat("z", 300), score: 0.3},
	}}
	tool := NewSearchVectorMemoryTool(notes, nil, nil, fixedSearchEmbedder{}, 5, 8, 360)
	out, err := tool.Run(context.Background(), map[string]any{"query": "q", "lanes": []any{"notes"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "[truncated:") {
		t.Fatalf("expected truncation footer, got %q", out)
	}
	if len(out) > 360 {
		t.Fatalf("output length %d exceeds cap", len(out))
	}
}
