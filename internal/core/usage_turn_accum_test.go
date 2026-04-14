package core

import (
	"pa/internal/llm"
	"testing"
)

// Covers AC-15.001: summed API usage produces the prescribed footer line.
func TestUsageTurnAcc_footerLine(t *testing.T) {
	var a usageTurnAcc
	a.add(llm.Usage{PromptTokens: 10, CompletionTokens: 5})
	a.add(llm.Usage{PromptTokens: 20, CompletionTokens: 7})
	if got := a.footerLine(); got != "*Tokens 42 (in: 30 / out: 12)*" {
		t.Fatalf("footerLine() = %q, want *Tokens 42 (in: 30 / out: 12)*", got)
	}
}

// Covers AC-15.002: zero sums yield no footer line.
func TestUsageTurnAcc_footerLine_emptyWhenZero(t *testing.T) {
	var a usageTurnAcc
	a.add(llm.Usage{})
	if got := a.footerLine(); got != "" {
		t.Fatalf("footerLine() = %q, want empty", got)
	}
}
