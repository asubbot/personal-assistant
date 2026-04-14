package core

import (
	"fmt"
	"pa/internal/llm"
)

// usageTurnAcc sums API usage across all successful LLM completions in one user turn (EP-015).
type usageTurnAcc struct {
	promptSum, completionSum int
}

func (a *usageTurnAcc) add(u llm.Usage) {
	if a == nil {
		return
	}
	a.promptSum += u.PromptTokens
	a.completionSum += u.CompletionTokens
}

// footerLine returns the EP-015 Telegram footer without a leading newline, or empty when omitted.
// The line is wrapped in *...* so MarkdownToTelegramHTML emits italic (Telegram HTML <i>).
func (a *usageTurnAcc) footerLine() string {
	if a == nil || (a.promptSum == 0 && a.completionSum == 0) {
		return ""
	}
	in, out := a.promptSum, a.completionSum
	total := in + out
	return fmt.Sprintf("*Tokens %d (in: %d / out: %d)*", total, in, out)
}
