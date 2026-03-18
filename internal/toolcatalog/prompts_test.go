package toolcatalog

import (
	"strings"
	"testing"
)

// Covers AC-04.026: non-empty system_prompt aggregated per tool id in selection order.
func TestAggregateSystemPrompts_joinsByIdOrder(t *testing.T) {
	c := &Catalog{Tools: map[string]*Tool{
		"a": {ID: "a", IndexText: "ia", Template: "x", NodeID: "n", SystemPrompt: "  first  "},
		"b": {ID: "b", IndexText: "ib", Template: "x", NodeID: "n", SystemPrompt: "second"},
	}}
	s := AggregateSystemPrompts(c, []string{"b", "a"})
	if !strings.Contains(s, "[b]") || !strings.Contains(s, "second") || !strings.Contains(s, "[a]") || !strings.Contains(s, "first") {
		t.Errorf("content: %q", s)
	}
	if strings.Index(s, "[b]") > strings.Index(s, "[a]") {
		t.Errorf("want b before a: %q", s)
	}
}

// Covers AC-04.026: tools without system_prompt omitted from aggregated block.
func TestAggregateSystemPrompts_skipsEmpty(t *testing.T) {
	c := &Catalog{Tools: map[string]*Tool{
		"x": {ID: "x", IndexText: "i", Template: "t", NodeID: "n"},
		"y": {ID: "y", IndexText: "i", Template: "t", NodeID: "n", SystemPrompt: "only"},
	}}
	s := AggregateSystemPrompts(c, []string{"x", "y"})
	if !strings.Contains(s, "[y]") || strings.Contains(s, "[x]") {
		t.Errorf("want only y block: %q", s)
	}
}

// Covers AC-04.027: HermesBody returns hermes_prompt trimmed or falls back to index_text.
func TestTool_HermesBody(t *testing.T) {
	t.Run("hermes_wins", func(t *testing.T) {
		tr := &Tool{HermesPrompt: "  H  ", IndexText: "I"}
		if tr.HermesBody() != "H" {
			t.Errorf("got %q", tr.HermesBody())
		}
	})
	t.Run("fallback_index", func(t *testing.T) {
		tr := &Tool{IndexText: "short"}
		if tr.HermesBody() != "short" {
			t.Errorf("got %q", tr.HermesBody())
		}
	})
}
