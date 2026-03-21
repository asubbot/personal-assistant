package tooltext

import (
	"pa/internal/llm"
	"pa/internal/toolcatalog"
	"strings"
	"testing"
)

// Covers AC-04.024: assistant text with no tool_call blocks → empty parse, no execution path from parser.
func TestParseHermesToolCalls_empty(t *testing.T) {
	calls, err := ParseHermesToolCalls("plain text only")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Errorf("calls = %d, want 0", len(calls))
	}
}

// Covers AC-04.023, AC-04.028: valid Hermes markup → parsed tool id and arguments for validation/execution path.
func TestParseHermesToolCalls_one(t *testing.T) {
	s := `Thinking.
<tool_call>
{"name": "run_echo", "arguments": {"msg": "x"}}
</tool_call>`
	calls, err := ParseHermesToolCalls(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("len = %d", len(calls))
	}
	if calls[0].Name != "run_echo" || calls[0].Arguments != `{"msg": "x"}` {
		t.Errorf("call = %+v", calls[0])
	}
	if calls[0].ID == "" {
		t.Error("expected synthetic id")
	}
}

// Covers AC-04.024: unclosed tool_call → parse error, no valid tool_calls extracted.
func TestParseHermesToolCalls_unclosed(t *testing.T) {
	_, err := ParseHermesToolCalls(`<tool_call>{"name":"x"}`)
	if err == nil {
		t.Fatal("expected error")
	}
}

// Covers AC-04.024: invalid JSON inside tool_call → parse error.
func TestParseHermesToolCalls_invalidJSON(t *testing.T) {
	_, err := ParseHermesToolCalls(`<tool_call>not json</tool_call>`)
	if err == nil {
		t.Fatal("expected error")
	}
}

// Covers AC-04.023: prompt instructions include tool ids, format, and parameters for text-based invocation.
func TestInstructionsForTools_containsFormat(t *testing.T) {
	s := InstructionsForTools([]llm.ToolDef{{Name: "t1", Description: "d1", Parameters: `{"type":"object"}`}})
	if !strings.Contains(s, "t1") || !strings.Contains(s, "tool_call") {
		t.Errorf("instructions: %s", s)
	}
}

// Covers AC-04.027: Hermes tool list uses hermes_prompt when non-empty for that tool.
func TestInstructionsForCatalogTools_hermesPromptOverridesIndex(t *testing.T) {
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{
		"t1": {ID: "t1", IndexText: "short", HermesPrompt: "LONG HERMES", Template: "x", NodeID: "n"},
	}}
	s := InstructionsForCatalogTools(cat, []string{"t1"})
	if !strings.Contains(s, "LONG HERMES") || strings.Contains(s, "- t1: short") {
		t.Errorf("want hermes line for t1: %s", s)
	}
}

// Covers AC-04.027: when hermes_prompt empty, tool line uses index_text.
func TestInstructionsForCatalogTools_fallbackIndexText(t *testing.T) {
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{
		"t1": {ID: "t1", IndexText: "only index", Template: "x", NodeID: "n"},
	}}
	s := InstructionsForCatalogTools(cat, []string{"t1"})
	if !strings.Contains(s, "only index") {
		t.Errorf("instructions: %s", s)
	}
}

// Covers AC-04.023, AC-04.028: multiple valid tool_call blocks → multiple parsed calls.
func TestParseHermesToolCalls_multipleBlocks(t *testing.T) {
	s := `<tool_call>{"name": "a", "arguments": {"x": 1}}</tool_call>
<tool_call>
{"name": "b", "arguments": {}}
</tool_call>`
	calls, err := ParseHermesToolCalls(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("len = %d, want 2", len(calls))
	}
	if calls[0].Name != "a" || calls[1].Name != "b" {
		t.Errorf("names = %q, %q", calls[0].Name, calls[1].Name)
	}
	if calls[0].Arguments != `{"x": 1}` {
		t.Errorf("args0 = %q", calls[0].Arguments)
	}
}

func TestSuspectedBrokenHermesMarkup_pseudoBlock(t *testing.T) {
	s := `-tool_call>{"name":"run_echo","arguments":{"msg":"x"}}</tool_call>`
	calls, err := ParseHermesToolCalls(s)
	if err != nil || len(calls) != 0 {
		t.Fatalf("parse: err=%v len=%d", err, len(calls))
	}
	if !SuspectedBrokenHermesMarkup(s) {
		t.Fatal("expected suspected true for -tool_call>…</tool_call>")
	}
}

func TestSuspectedBrokenHermesMarkup_reversedMarkers(t *testing.T) {
	s := `note </tool_call> then -tool_call>{}`
	calls, err := ParseHermesToolCalls(s)
	if err != nil || len(calls) != 0 {
		t.Fatalf("parse: err=%v len=%d", err, len(calls))
	}
	if SuspectedBrokenHermesMarkup(s) {
		t.Fatal("expected false when first tool_call> is not before first </tool_call>")
	}
}

func TestSuspectedBrokenHermesMarkup_noCloseTag(t *testing.T) {
	s := `-tool_call>{"name":"x"}`
	calls, err := ParseHermesToolCalls(s)
	if err != nil || len(calls) != 0 {
		t.Fatalf("parse: err=%v len=%d", err, len(calls))
	}
	if SuspectedBrokenHermesMarkup(s) {
		t.Fatal("expected false without </tool_call>")
	}
}

func TestSuspectedBrokenHermesMarkup_plainText(t *testing.T) {
	if SuspectedBrokenHermesMarkup("plain only") {
		t.Fatal("expected false")
	}
}

func TestSuspectedBrokenHermesMarkup_emptyTrim(t *testing.T) {
	if SuspectedBrokenHermesMarkup("   \n\t  ") {
		t.Fatal("expected false for whitespace-only")
	}
}

func TestSuspectedBrokenHermesMarkup_validHermesNotUsedWithEmptyParse(t *testing.T) {
	s := `<tool_call>
{"name": "run_echo", "arguments": {"msg": "x"}}
</tool_call>`
	calls, err := ParseHermesToolCalls(s)
	if err != nil || len(calls) != 1 {
		t.Fatalf("parse: err=%v len=%d", err, len(calls))
	}
	if SuspectedBrokenHermesMarkup(s) {
		t.Fatal("expected false when exact <tool_call> is present (valid path uses parsed calls)")
	}
}
