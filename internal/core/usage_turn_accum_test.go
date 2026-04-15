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
	if got := a.footerLine("full"); got != "*Tokens 42 (in: 30 / out: 12) · full*" {
		t.Fatalf("footerLine(full) = %q, want *Tokens 42 (in: 30 / out: 12) · full*", got)
	}
}

// Covers AC-15.001: footer line format without tier suffix when tier string is empty.
func TestUsageTurnAcc_footerLine_emptyTierOmitsSuffix(t *testing.T) {
	var a usageTurnAcc
	a.add(llm.Usage{PromptTokens: 1, CompletionTokens: 1})
	if got := a.footerLine(""); got != "*Tokens 2 (in: 1 / out: 1)*" {
		t.Fatalf("footerLine(\"\") = %q, want *Tokens 2 (in: 1 / out: 1)*", got)
	}
}

// Covers AC-15.002: zero sums yield no footer line.
func TestUsageTurnAcc_footerLine_emptyWhenZero(t *testing.T) {
	var a usageTurnAcc
	a.add(llm.Usage{})
	if got := a.footerLine("full"); got != "" {
		t.Fatalf("footerLine(full) = %q, want empty", got)
	}
}
