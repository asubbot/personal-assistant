package core

import (
	"fmt"
	"pa/internal/llm"
	"strings"
)

// usageTurnAcc sums API usage across all successful LLM completions in one user turn (EP-015).
// round counts successful main-LLM completions in the turn (1-based), for per-call usage logs.
type usageTurnAcc struct {
	promptSum, completionSum int
	round                    int
}

func (a *usageTurnAcc) add(u llm.Usage) {
	if a == nil {
		return
	}
	a.round++
	a.promptSum += u.PromptTokens
	a.completionSum += u.CompletionTokens
}

// footerLine returns the EP-015 Telegram footer without a leading newline, or empty when omitted.
// When tier is non-empty, it is appended after an interpunct so the user sees which intent tier ran.
// The line is wrapped in *...* so MarkdownToTelegramHTML emits italic (Telegram HTML <i>).
func (a *usageTurnAcc) footerLine(tier string) string {
	if a == nil || (a.promptSum == 0 && a.completionSum == 0) {
		return ""
	}
	in, out := a.promptSum, a.completionSum
	total := in + out
	tier = strings.TrimSpace(tier)
	if tier != "" {
		return fmt.Sprintf("*Tokens %d (in: %d / out: %d) · %s*", total, in, out, tier)
	}
	return fmt.Sprintf("*Tokens %d (in: %d / out: %d)*", total, in, out)
}
