package tooltext

import (
	"pa/internal/llm"
	"strings"
	"testing"
)

func TestParseHermesToolCalls_empty(t *testing.T) {
	calls, err := ParseHermesToolCalls("plain text only")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Errorf("calls = %d, want 0", len(calls))
	}
}

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

func TestParseHermesToolCalls_unclosed(t *testing.T) {
	_, err := ParseHermesToolCalls(`<tool_call>{"name":"x"}`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseHermesToolCalls_invalidJSON(t *testing.T) {
	_, err := ParseHermesToolCalls(`<tool_call>not json</tool_call>`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstructionsForTools_containsFormat(t *testing.T) {
	s := InstructionsForTools([]llm.ToolDef{{Name: "t1", Description: "d1", Parameters: `{"type":"object"}`}})
	if !strings.Contains(s, "t1") || !strings.Contains(s, "tool_call") {
		t.Errorf("instructions: %s", s)
	}
}

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
