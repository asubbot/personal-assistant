package core

import (
	"context"
	"log/slog"
	"pa/internal/toolcatalog"
	"testing"
)

// Covers AC-18.013
func TestApplyDynamicToolCap_truncatesPreservingPrefix(t *testing.T) {
	in := []string{"a", "b", "c", "d"}
	got := ApplyDynamicToolCap(in, 2)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("ApplyDynamicToolCap = %#v, want [a b]", got)
	}
}

// Covers AC-18.012, AC-18.019 (unknown id dropped)
func TestPickToolsForMainRequest_filtersUnknownAndCaps(t *testing.T) {
	cat := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"keep_a": {ID: "keep_a"},
			"keep_b": {ID: "keep_b"},
		},
	}
	h := testHandlerDeps{
		catalog: cat,
		logger:  slog.New(slog.DiscardHandler),
	}.handler()
	ctx := context.Background()
	merged := []string{"keep_a", "bogus", "keep_b", "keep_a"}
	out := h.pickToolsForMainRequest(ctx, merged, 2)
	if len(out) != 2 || out[0] != "keep_a" || out[1] != "keep_b" {
		t.Fatalf("pickToolsForMainRequest = %#v, want [keep_a keep_b]", out)
	}
}
