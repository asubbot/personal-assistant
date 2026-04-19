package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"pa/internal/config"
	"strings"
	"time"
)

const openAICompletionsPath = "/chat/completions"

// OpenAICompatible is an OpenAI-compatible HTTP chat completions provider (OpenAI, Ollama with /v1, etc.).
type OpenAICompatible struct {
	client                *http.Client
	baseURL               string
	apiKey                string
	model                 string
	supportsTools         bool // when false, tools are omitted from the request body
	defaultTemperature    float64
	defaultMaxTokens      int
	defaultResponseFormat string // "text" (product)
}

// parseHTTPTimeout enforces the fail-fast contract for llm_providers[].http_timeout
// at the construction site (EP-022, REQ-22.003). The value is also validated at
// config.Load; re-validating here guards direct struct construction in tests.
func parseHTTPTimeout(raw string) (time.Duration, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("llm: http_timeout is required")
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("llm: http_timeout invalid duration %q: %w", raw, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("llm: http_timeout must be > 0, got %s", d)
	}
	return d, nil
}

// NewOpenAICompatible builds a provider from config. Reads API key from api_key_path when set (e.g. for openai/openai-compatible).
func NewOpenAICompatible(cfg *config.LLMProvider) (*OpenAICompatible, error) {
	baseURL := strings.TrimRight(cfg.Endpoint, "/")
	model := strings.TrimSpace(cfg.Model)

	var apiKey string
	if strings.TrimSpace(cfg.APIKeyPath) != "" {
		b, err := os.ReadFile(cfg.APIKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read api_key_path: %w", err)
		}
		apiKey = strings.TrimSpace(string(b))
	}

	st := true
	if cfg.SupportsTools != nil {
		st = *cfg.SupportsTools
	}
	timeout, err := parseHTTPTimeout(cfg.HTTPTimeout)
	if err != nil {
		return nil, err
	}
	return &OpenAICompatible{
		client:                &http.Client{Timeout: timeout},
		baseURL:               baseURL,
		apiKey:                apiKey,
		model:                 model,
		supportsTools:         st,
		defaultTemperature:    cfg.DefaultTemperature,
		defaultMaxTokens:      cfg.DefaultMaxTokens,
		defaultResponseFormat: cfg.DefaultResponseFormat,
	}, nil
}

// Complete implements Provider.
func (p *OpenAICompatible) Complete(ctx context.Context, messages []Message, opts *CompletionOptions) (*CompletionResult, error) {
	model, maxTokens := p.effectiveModelAndMaxTokens(opts)
	body, err := p.buildRequest(model, messages, maxTokens, opts)
	if err != nil {
		return nil, err
	}
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return p.parseResponse(resp)
}

func (p *OpenAICompatible) effectiveModelAndMaxTokens(opts *CompletionOptions) (model string, maxTokens int) {
	model = p.model
	if opts != nil {
		if opts.Model != "" {
			model = opts.Model
		}
		maxTokens = opts.MaxTokens
	}
	// Use defaultMaxTokens if not overridden by opts
	if maxTokens == 0 && p.defaultMaxTokens > 0 {
		maxTokens = p.defaultMaxTokens
	}
	return model, maxTokens
}

func (p *OpenAICompatible) buildRequest(model string, messages []Message, maxTokens int, opts *CompletionOptions) ([]byte, error) {
	oaiMessages := make([]openAIMessage, len(messages))
	for i := range messages {
		oaiMessages[i] = messageToOpenAI(messages[i])
	}
	reqBody := openAIRequest{Model: model, OAIMessages: oaiMessages}
	if maxTokens > 0 {
		reqBody.MaxTokens = &maxTokens
	}
	reqBody.Temperature = p.resolveTemperature(opts)
	if p.supportsTools && opts != nil && len(opts.Tools) > 0 {
		reqBody.Tools = make([]openAITool, len(opts.Tools))
		for i := range opts.Tools {
			t := &opts.Tools[i]
			reqBody.Tools[i] = openAITool{
				Type: "function",
				Function: openAIToolFunction{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  json.RawMessage(t.Parameters),
				},
			}
		}
	}
	reqBody.ResponseFormat = p.resolveResponseFormat(opts)
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	return body, nil
}

// resolveTemperature returns the effective temperature: opts.Temperature > defaultTemperature.
// Since defaultTemperature is required in config, always returns a value.
func (p *OpenAICompatible) resolveTemperature(opts *CompletionOptions) *float64 {
	if opts != nil && opts.Temperature != nil {
		return opts.Temperature
	}
	t := p.defaultTemperature
	return &t
}

// resolveResponseFormat returns the HTTP response_format for the provider (text-only product).
// Explicit type "text" wins; json_object and other hints are ignored and the configured default is used.
func (p *OpenAICompatible) resolveResponseFormat(opts *CompletionOptions) *responseFormat {
	if opts != nil && opts.ResponseFormat != nil {
		if strings.TrimSpace(opts.ResponseFormat.Type) == "text" {
			return &responseFormat{Type: "text"}
		}
	}
	rt := strings.TrimSpace(p.defaultResponseFormat)
	if rt == "" {
		rt = "text"
	}
	return &responseFormat{Type: rt}
}

func (p *OpenAICompatible) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	url := p.baseURL + openAICompletionsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	return resp, nil
}

// decodeAssistantMessageContent parses choices[].message.content from OpenAI-compatible APIs.
// Some providers return a JSON string, others a JSON array of {type,text} parts, or null.
func decodeAssistantMessageContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		if len(parts) == 0 {
			return "", nil
		}
		var b strings.Builder
		for _, p := range parts {
			if p.Type != "" && p.Type != "text" {
				continue
			}
			b.WriteString(p.Text)
		}
		return b.String(), nil
	}
	return "", fmt.Errorf("assistant message content: unsupported JSON shape")
}

// assistantTextFromMessage resolves the visible assistant string when the API splits "thinking" vs answer.
// Ollama OpenAI-compatible: some models (e.g. Gemma 4) return empty decoded content and put text in "reasoning"
// (see ollama/ollama#15288). OpenAI-style models may use reasoning_content when content is empty.
func assistantTextFromMessage(decodedContent string, msg *openAIChoiceMessage) string {
	content := decodedContent
	if strings.TrimSpace(content) == "" && strings.TrimSpace(msg.ReasoningContent) != "" {
		content = msg.ReasoningContent
	}
	if strings.TrimSpace(content) == "" && strings.TrimSpace(msg.Reasoning) != "" {
		content = msg.Reasoning
	}
	return content
}

func (p *OpenAICompatible) parseResponse(resp *http.Response) (*CompletionResult, error) {
	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody.Error.Message
		if msg == "" {
			msg = resp.Status
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Err: fmt.Errorf("api %s: %s", resp.Status, msg)}
	}
	var out openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("api: empty choices")
	}
	msg := &out.Choices[0].Message
	content, err := decodeAssistantMessageContent(msg.Content)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	content = assistantTextFromMessage(content, msg)
	usage := Usage{}
	if out.Usage != nil {
		usage.PromptTokens = out.Usage.PromptTokens
		usage.CompletionTokens = out.Usage.CompletionTokens
		usage.TotalTokens = out.Usage.TotalTokens
	}
	result := &CompletionResult{Content: content, Usage: usage, Model: p.model}
	if len(msg.ToolCalls) > 0 {
		result.ToolCalls = make([]ToolCall, len(msg.ToolCalls))
		for i := range msg.ToolCalls {
			tc := &msg.ToolCalls[i]
			result.ToolCalls[i] = ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: tc.Function.Arguments}
		}
	}
	return result, nil
}

type openAITool struct {
	Type     string             `json:"type"` // "function"
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// openAIMessage is the OpenAI API message shape (supports tool_call_id and tool_calls).
type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

func messageToOpenAI(m Message) openAIMessage {
	out := openAIMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
	if len(m.ToolCalls) > 0 {
		out.ToolCalls = make([]openAIToolCall, len(m.ToolCalls))
		for i := range m.ToolCalls {
			tc := &m.ToolCalls[i]
			out.ToolCalls[i] = openAIToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{Name: tc.Name, Arguments: tc.Arguments},
			}
		}
	}
	return out
}

type openAIRequest struct {
	Model          string          `json:"model"`
	OAIMessages    []openAIMessage `json:"messages"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	Tools          []openAITool    `json:"tools,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"` // "text", "json_object"
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "function"
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIChoiceMessage struct {
	Content          json.RawMessage  `json:"content"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	Reasoning        string           `json:"reasoning,omitempty"` // Ollama: thinking models may use this when content is empty
	ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIChoiceMessage `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
