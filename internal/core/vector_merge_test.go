package core

import (
	"context"
	"pa/internal/vector"
	"testing"
)

// Covers AC-16.011: summary merge path keeps only summary-prefixed ids from search results.
func TestMergeSummarySearch_filtersNonSummaryRows(t *testing.T) {
	ctx := context.Background()
	summ := &mockVectorStore{
		searchResults: []vector.SearchResult{
			{ID: "turn-legacy", Text: "turn noise"},
			{ID: "summary:day:2026-04-01", Text: "day summary body"},
		},
	}
	out, err := mergeSummarySearch(ctx, summ, []float32{1, 0, 0, 0}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID != "summary:day:2026-04-01" {
		t.Fatalf("got %+v, want single summary:day row", out)
	}
}
