package llm

import "context"

// Message is one role/content pair in a conversation (OpenAI chat format).
// For role "tool": ToolCallID is set to match the tool call. For role "assistant" with tool_calls: ToolCalls is set.
type Message struct {
	Role       string     `json:"role"`                   // "system", "user", "assistant", "tool"
	Content    string     `json:"content"`                // text body
	ToolCallID string     `json:"tool_call_id,omitempty"` // for role "tool": id of the tool call this result belongs to
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // for role "assistant": tool calls made by the model
}

// ToolDef is one tool in the completion request (provider-agnostic; core passes these for Tool-calling API).
type ToolDef struct {
	Name        string `json:"name"`                 // tool id/name (e.g. from catalog id)
	Description string `json:"description"`          // index_text for native tool API
	Parameters  string `json:"parameters,omitempty"` // JSON schema or object for arguments; empty = no parameters
}

// ToolCall is one tool call in the completion response (id, name, raw arguments JSON).
type ToolCall struct {
	ID        string `json:"id"`        // call id from the API (for matching in multi-turn)
	Name      string `json:"name"`      // tool name (function name) returned by the model
	Arguments string `json:"arguments"` // raw JSON string of arguments (core parses and validates against catalog)
}

// CompletionOptions are optional parameters for a completion call.
type CompletionOptions struct {
	Model           string          `json:"model,omitempty"`      // override config model
	MaxTokens       int             `json:"max_tokens,omitempty"` // max tokens to generate; 0 means use configured default_max_tokens from llm_providers
	Temperature     *float64        `json:"temperature,omitempty"`
	Tools           []ToolDef       `json:"tools,omitempty"`             // optional; nil or empty = no tools (REQ-04.012)
	ForceJSONOutput bool            `json:"force_json_output,omitempty"` // hint for providers with supports_json_mode to output JSON
	ResponseFormat  *ResponseFormat `json:"response_format,omitempty"`   // explicit per-request response format (Stage C)
}

// ResponseFormat specifies the desired response format for completion requests.
type ResponseFormat struct {
	Type string `json:"type"` // "text", "json_object"
}

// Usage holds token usage returned by the provider.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompletionResult is the result of a single completion call.
type CompletionResult struct {
	Content   string     `json:"content"`              // assistant message text
	Usage     Usage      `json:"usage"`                // token usage
	Model     string     `json:"model,omitempty"`      // optional; which provider/model produced the response (for logging, AC-01.044)
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // optional; tool_calls from the provider (REQ-04.012)
}

// APIError represents an HTTP API error (e.g. 4xx/5xx). Used so isRetryable can reliably detect 5xx.
type APIError struct {
	StatusCode int
	Err        error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// Provider is the LLM provider interface: one call completes a conversation turn.
type Provider interface {
	// Complete sends messages to the model and returns the assistant reply and usage.
	// opts may be nil to use provider defaults.
	Complete(ctx context.Context, messages []Message, opts *CompletionOptions) (*CompletionResult, error)
}
