package llm

import "context"

// Message is one role/content pair in a conversation (OpenAI chat format).
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // text body
}

// CompletionOptions are optional parameters for a completion call.
type CompletionOptions struct {
	Model       string   `json:"model,omitempty"`      // override config model
	MaxTokens   int      `json:"max_tokens,omitempty"` // max tokens to generate (0 = provider default)
	Temperature *float64 `json:"temperature,omitempty"`
}

// Usage holds token usage returned by the provider.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompletionResult is the result of a single completion call.
type CompletionResult struct {
	Content string `json:"content"` // assistant message text
	Usage   Usage  `json:"usage"`
}

// Provider is the LLM provider interface: one call completes a conversation turn.
type Provider interface {
	// Complete sends messages to the model and returns the assistant reply and usage.
	// opts may be nil to use provider defaults.
	Complete(ctx context.Context, messages []Message, opts *CompletionOptions) (*CompletionResult, error)
}
