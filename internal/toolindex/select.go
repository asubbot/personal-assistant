// Package toolindex: SelectToolIDs implements tool pre-selection and fallback (REQ-04.019, REQ-04.020).

package toolindex

import (
	"context"
	"log/slog"
	"pa/internal/embedding"
	"pa/internal/toolcatalog"
	"pa/internal/vector"
	"sort"
)

// SelectToolIDs returns tool IDs for the given query: either from vector search (top-k) or fallback (sorted catalog IDs up to cap).
// catalog nil or empty → empty slice. When index is not ready or store is nil, or search returns fewer than minTools, fallback is used.
// topK, minTools, and fallbackCap must be positive (enforced at config load for production paths).
func SelectToolIDs(
	ctx context.Context,
	embedder embedding.Embedder,
	toolStore vector.Store,
	indexReady bool,
	catalog *toolcatalog.Catalog,
	query string,
	topK, minTools, fallbackCap int,
	logger *slog.Logger,
) ([]string, error) {
	if catalog == nil || len(catalog.Tools) == 0 {
		return nil, nil
	}

	doFallback := func(reason string) []string {
		logFallbackReason(ctx, logger, reason)
		return sortedCatalogIDs(catalog, fallbackCap)
	}

	if !indexReady || toolStore == nil {
		// fallback-return: safe — REQ-04.023: bounded catalog subset while index builds
		return doFallback("index not ready"), nil
	}

	ids, err := searchToolIDs(ctx, embedder, toolStore, query, topK)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		// fallback-return: safe — REQ-04.020: empty search → capped catalog IDs
		return doFallback("empty result"), nil
	}
	if len(ids) < minTools {
		// fallback-return: safe — REQ-04.020: below minTools → capped catalog IDs
		return doFallback("below minimum"), nil
	}
	// fallback-return: safe — success path after pre-selection; not a silent degrade
	return ids, nil
}

func logFallbackReason(ctx context.Context, logger *slog.Logger, reason string) {
	if logger != nil && logger.Enabled(ctx, slog.LevelDebug) {
		logger.DebugContext(ctx, "tool pre-selection: using fallback", slog.String("reason", reason))
	}
}

func searchToolIDs(ctx context.Context, embedder embedding.Embedder, toolStore vector.Store, query string, topK int) ([]string, error) {
	queryEmb, err := embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	results, err := toolStore.Search(ctx, queryEmb, topK)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.ID)
	}
	return ids, nil
}

// sortedCatalogIDs returns sorted catalog tool IDs, at most cap.
func sortedCatalogIDs(catalog *toolcatalog.Catalog, cap int) []string {
	keys := make([]string, 0, len(catalog.Tools))
	for id := range catalog.Tools {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	if cap <= 0 || len(keys) <= cap {
		return keys
	}
	return keys[:cap]
}
