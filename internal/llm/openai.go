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

const (
	openAICompletionsPath = "/chat/completions"
	defaultTimeout        = 60 * time.Second
)

// OpenAICompatible is an OpenAI-compatible HTTP chat completions provider (OpenAI, Ollama with /v1, etc.).
type OpenAICompatible struct {
	client                *http.Client
	baseURL               string
	apiKey                string
	model                 string
	supportsTools         bool // when false, tools are omitted from the request body
	supportsJSONMode      bool // when true, provider supports response_format: json_object
	defaultTemperature    float64
	defaultMaxTokens      int
	defaultResponseFormat string // "text" or "json_object"
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
	return &OpenAICompatible{
		client:                &http.Client{Timeout: defaultTimeout},
		baseURL:               baseURL,
		apiKey:                apiKey,
		model:                 model,
		supportsTools:         st,
		supportsJSONMode:      cfg.SupportsJSONMode,
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

// resolveResponseFormat returns the effective response format: explicit ResponseFormat (when valid
// for this provider) > ForceJSONOutput > defaultResponseFormat.
// json_object is only emitted when supportsJSONMode is true. Empty or unknown explicit types are ignored.
// Since defaultResponseFormat is required in config, always returns a value.
func (p *OpenAICompatible) resolveResponseFormat(opts *CompletionOptions) *responseFormat {
	if opts != nil && opts.ResponseFormat != nil {
		t := strings.TrimSpace(opts.ResponseFormat.Type)
		switch t {
		case "text":
			return &responseFormat{Type: "text"}
		case "json_object":
			if p.supportsJSONMode {
				return &responseFormat{Type: "json_object"}
			}
			// Caller asked for JSON mode but provider does not support it — fall through.
		default:
			// Empty or unknown type — fall through.
		}
	}
	if p.supportsJSONMode && opts != nil && opts.ForceJSONOutput {
		return &responseFormat{Type: "json_object"}
	}
	return &responseFormat{Type: p.defaultResponseFormat}
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
	content := msg.Content
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

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
