package core

import (
	"context"
	"pa/internal/summarize"
	"pa/internal/vector"
)

// appendUniqueSummaryHits appends summary-vector rows from hits that pass IsSummaryVectorID and are not in seen.
// When stopAfter > 0, stops once len(out) >= stopAfter (counting only newly appended from this call's perspective is handled by caller passing growing out).
func appendUniqueSummaryHits(hits []vector.SearchResult, seen map[string]struct{}, out []vector.SearchResult, stopAfter int) []vector.SearchResult {
	for _, x := range hits {
		if !summarize.IsSummaryVectorID(x.ID) {
			continue
		}
		if _, ok := seen[x.ID]; ok {
			continue
		}
		seen[x.ID] = struct{}{}
		out = append(out, x)
		if stopAfter > 0 && len(out) >= stopAfter {
			break
		}
	}
	return out
}

func mergeSummarySearch(ctx context.Context, sum, leg vector.Store, query []float32, topK int) ([]vector.SearchResult, error) {
	if topK < 1 {
		topK = 1
	}
	seen := make(map[string]struct{})
	var out []vector.SearchResult
	if sum != nil {
		r, err := sum.Search(ctx, query, topK)
		if err != nil {
			return nil, err
		}
		out = appendUniqueSummaryHits(r, seen, out, 0)
	}
	if leg != nil {
		r, err := leg.Search(ctx, query, topK*4)
		if err != nil {
			return nil, err
		}
		out = appendUniqueSummaryHits(r, seen, out, topK*2)
	}
	return out, nil
}
