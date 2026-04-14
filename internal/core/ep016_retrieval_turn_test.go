package core

import (
	"context"
	"log/slog"
	"pa/internal/vector"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Covers AC-16.010: split-table retrieval concatenates notes hits, then summary hits, then turn hits.
func TestGatherRetrievedChunkTexts_splitTableOrder_notesSummaryTurn(t *testing.T) {
	notes := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "notes:2026-04-01:1", Text: "NOTE_LINE"}}}
	summ := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "summary:day:2026-04-01", Text: "SUM_LINE"}}}
	turns := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "turn:2026-04-01:ab", Text: "TURN_LINE"}}}
	h := &conversationHandler{
		memVec:                &MemoryVectors{Notes: notes, Summaries: summ, Legacy: nil, Turns: turns},
		embedder:              &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		logger:                slog.New(slog.DiscardHandler),
		vectorSearchTopK:      5,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
	}
	chunks := h.gatherRetrievedChunkTexts(context.Background(), "query")
	if len(chunks) != 3 {
		t.Fatalf("want 3 chunks, got %d: %#v", len(chunks), chunks)
	}
	joined := strings.Join(chunks, "\n")
	ni := strings.Index(joined, "[notes]")
	si := strings.Index(joined, "[summary:day]")
	ti := strings.Index(joined, "[turn]")
	if ni < 0 || si < 0 || ti < 0 || ni >= si || si >= ti {
		t.Fatalf("expected notes then summary then turn markers; ni=%d si=%d ti=%d\n%s", ni, si, ti, joined)
	}
}

// Covers AC-16.012: legacy vec_items may hold turn rows; merge summary path drops them so assembled context has no legacy turn noise alongside dedicated turns.
func TestGatherRetrievedChunkTexts_legacyTurnRowsNotSurfacedAsSummaryMerge(t *testing.T) {
	leg := &mockVectorStore{
		searchResults: []vector.SearchResult{{ID: "turn:2026-04-01:legacy", Text: "LEGACY_TURN_NOISE"}},
	}
	turns := &mockVectorStore{searchResults: []vector.SearchResult{{ID: "turn:2026-04-01:ab", Text: "DEDICATED_TURN"}}}
	h := &conversationHandler{
		memVec:                &MemoryVectors{Notes: nil, Summaries: nil, Legacy: leg, Turns: turns},
		embedder:              &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		logger:                slog.New(slog.DiscardHandler),
		vectorSearchTopK:      5,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
	}
	chunks := h.gatherRetrievedChunkTexts(context.Background(), "q")
	if len(chunks) != 1 {
		t.Fatalf("want 1 turn chunk, got %d: %#v", len(chunks), chunks)
	}
	if strings.Contains(strings.Join(chunks, ""), "LEGACY_TURN_NOISE") {
		t.Fatalf("legacy turn text must not appear in chunks: %#v", chunks)
	}
	if !strings.Contains(chunks[0], "DEDICATED_TURN") {
		t.Fatalf("expected dedicated turn chunk: %q", chunks[0])
	}
}

// Covers AC-16.013: Telegram message unix time yields event-aligned Date line in indexed turn chunk.
func TestIndexTurn_eventAlignedDateFromTelegramContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "turnvec.sqlite")
	turns, err := sqlite.NewWithTable(path, 4, sqlite.TableTurns)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = turns.Close() }()
	tUnix := time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC).Unix()
	ctx := WithTelegramMessageDate(context.Background(), tUnix)
	h := &conversationHandler{
		memVec:   &MemoryVectors{Turns: turns},
		embedder: &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		paLoc:    time.UTC,
		logger:   slog.New(slog.DiscardHandler),
	}
	if err := h.indexTurn(ctx, "user", "assistant"); err != nil {
		t.Fatal(err)
	}
	res, err := turns.Search(ctx, []float32{1, 0, 0, 0}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || !strings.Contains(res[0].Text, "Date: 2026-04-02") {
		t.Fatalf("unexpected search results: %+v", res)
	}
}

// Covers AC-16.014: without adapter timestamp, indexTurn still writes a valid ISO calendar Date line in pa_timezone.
func TestIndexTurn_fallbackDateLineIsISOYMD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "turnvec2.sqlite")
	turns, err := sqlite.NewWithTable(path, 4, sqlite.TableTurns)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = turns.Close() }()
	h := &conversationHandler{
		memVec:   &MemoryVectors{Turns: turns},
		embedder: &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		paLoc:    time.UTC,
		logger:   slog.New(slog.DiscardHandler),
	}
	ctx := context.Background()
	if err := h.indexTurn(ctx, "u", "a"); err != nil {
		t.Fatal(err)
	}
	res, err := turns.Search(ctx, []float32{1, 0, 0, 0}, 5)
	if err != nil || len(res) != 1 {
		t.Fatalf("search: err=%v n=%d", err, len(res))
	}
	const prefix = "Date: "
	i := strings.Index(res[0].Text, prefix)
	if i < 0 {
		t.Fatalf("missing Date line: %q", res[0].Text)
	}
	rest := strings.TrimSpace(res[0].Text[i+len(prefix):])
	line := strings.SplitN(rest, "\n", 2)[0]
	if len(line) != 10 || line[4] != '-' || line[7] != '-' {
		t.Fatalf("expected YYYY-MM-DD after Date:, got %q", line)
	}
}

// Covers AC-16.015: indexing the same canonical turn twice leaves a single row for the stable id (delete-then-add upsert).
func TestIndexTurn_twiceSameCanonicalPair_oneRow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "turnvec3.sqlite")
	turns, err := sqlite.NewWithTable(path, 4, sqlite.TableTurns)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = turns.Close() }()
	tUnix := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC).Unix()
	ctx := WithTelegramMessageDate(context.Background(), tUnix)
	h := &conversationHandler{
		memVec:   &MemoryVectors{Turns: turns},
		embedder: &mockEmbedder{vec: []float32{1, 0, 0, 0}},
		paLoc:    time.UTC,
		logger:   slog.New(slog.DiscardHandler),
	}
	if err := h.indexTurn(ctx, "hello", "world"); err != nil {
		t.Fatal(err)
	}
	if err := h.indexTurn(ctx, "hello", "world"); err != nil {
		t.Fatal(err)
	}
	res, err := turns.Search(ctx, []float32{1, 0, 0, 0}, 20)
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]int)
	for _, r := range res {
		byID[r.ID]++
	}
	if len(byID) != 1 {
		t.Fatalf("want exactly one distinct turn id, got %v (rows=%d)", byID, len(res))
	}
}
