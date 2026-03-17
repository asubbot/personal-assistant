// Package toolindex builds and holds the tool vector index (REQ-04.018, REQ-04.021).
// At startup, Build populates the vec_tools table from the catalog; Index exposes the store and ready state.
package toolindex

import (
	"context"
	"fmt"
	"pa/internal/embedding"
	"pa/internal/toolcatalog"
	"pa/internal/vector"
	"sort"
	"strings"
	"sync/atomic"
)

// Index holds the tool vector store and whether the index has been built and is ready for search.
type Index struct {
	store vector.Store
	ready atomic.Bool
}

// NewIndex returns an Index that uses the given store. Ready() is false until Build completes.
func NewIndex(store vector.Store) *Index {
	return &Index{store: store}
}

// Store returns the vector store for tool search (vec_tools table).
func (x *Index) Store() vector.Store {
	return x.store
}

// Ready returns true when the tool index has been built and is ready for pre-selection.
func (x *Index) Ready() bool {
	return x.ready.Load()
}

// SetReady sets the ready flag (e.g. after synchronous Build completes in setup).
func (x *Index) SetReady(ready bool) {
	x.ready.Store(ready)
}

// Close closes the underlying vector store. Safe to call multiple times.
func (x *Index) Close() error {
	if x.store == nil {
		return nil
	}
	err := x.store.Close()
	x.store = nil
	return err
}

// Build populates the tool vector store from the catalog: for each tool, build text (id + short_description + triggers),
// embed it (using BatchEmbedder when available; provider chunks by its config batch_size), and add to the store.
func Build(ctx context.Context, catalog *toolcatalog.Catalog, embedder embedding.Embedder, toolStore vector.Store) error {
	if catalog == nil || embedder == nil || toolStore == nil {
		return nil
	}
	ids, texts := orderedToolTexts(catalog)
	if len(ids) == 0 {
		return nil
	}
	if batch, ok := embedder.(embedding.BatchEmbedder); ok {
		return buildBatched(ctx, batch, toolStore, ids, texts)
	}
	return buildSequential(ctx, embedder, toolStore, ids, texts)
}

func orderedToolTexts(catalog *toolcatalog.Catalog) (ids []string, texts []string) {
	keys := make([]string, 0, len(catalog.Tools))
	for id := range catalog.Tools {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	ids = make([]string, 0, len(keys))
	texts = make([]string, 0, len(keys))
	for _, id := range keys {
		t := catalog.Tools[id]
		ids = append(ids, id)
		texts = append(texts, toolIndexText(t))
	}
	return ids, texts
}

func buildBatched(ctx context.Context, batch embedding.BatchEmbedder, toolStore vector.Store, ids, texts []string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	embs, err := batch.EmbedBatch(ctx, texts)
	if err != nil {
		return err
	}
	if len(embs) != len(ids) {
		return fmt.Errorf("toolindex: EmbedBatch result length %d, want %d", len(embs), len(ids))
	}
	for i := range ids {
		if err := toolStore.Add(ctx, ids[i], embs[i], texts[i]); err != nil {
			return err
		}
	}
	return nil
}

func buildSequential(ctx context.Context, embedder embedding.Embedder, toolStore vector.Store, ids, texts []string) error {
	for i := range ids {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		emb, err := embedder.Embed(ctx, texts[i])
		if err != nil {
			return err
		}
		if err := toolStore.Add(ctx, ids[i], emb, texts[i]); err != nil {
			return err
		}
	}
	return nil
}

func toolIndexText(t *toolcatalog.Tool) string {
	parts := []string{t.ID, t.ShortDescription}
	if len(t.Triggers) > 0 {
		parts = append(parts, strings.Join(t.Triggers, " "))
	}
	return strings.Join(parts, " ")
}

// BuildAndSetReady runs Build with the given context; on success sets ready to true.
func (x *Index) BuildAndSetReady(ctx context.Context, catalog *toolcatalog.Catalog, embedder embedding.Embedder) error {
	err := Build(ctx, catalog, embedder, x.store)
	if err == nil {
		x.ready.Store(true)
	}
	return err
}
