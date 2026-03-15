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
	client  *http.Client
	baseURL string
	apiKey  string
	model   string
}

// NewOpenAICompatible builds a provider from config. Reads API key from api_key_path when set (e.g. for openai/openai-compatible).
func NewOpenAICompatible(cfg *config.LLMProvider) (*OpenAICompatible, error) {
	baseURL := strings.TrimRight(cfg.Endpoint, "/")
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-3.5-turbo" // fallback for compatibility
	}

	var apiKey string
	if strings.TrimSpace(cfg.APIKeyPath) != "" {
		b, err := os.ReadFile(cfg.APIKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read api_key_path: %w", err)
		}
		apiKey = strings.TrimSpace(string(b))
	}

	return &OpenAICompatible{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
	}, nil
}

// Complete implements Provider.
func (p *OpenAICompatible) Complete(ctx context.Context, messages []Message, opts *CompletionOptions) (*CompletionResult, error) {
	model, maxTokens := p.effectiveModelAndMaxTokens(opts)
	body, err := p.buildRequest(model, messages, maxTokens)
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
	return model, maxTokens
}

func (p *OpenAICompatible) buildRequest(model string, messages []Message, maxTokens int) ([]byte, error) {
	reqBody := openAIRequest{Model: model, Messages: messages}
	if maxTokens > 0 {
		reqBody.MaxTokens = &maxTokens
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	return body, nil
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
	content := out.Choices[0].Message.Content
	usage := Usage{}
	if out.Usage != nil {
		usage.PromptTokens = out.Usage.PromptTokens
		usage.CompletionTokens = out.Usage.CompletionTokens
		usage.TotalTokens = out.Usage.TotalTokens
	}
	return &CompletionResult{Content: content, Usage: usage, Model: p.model}, nil
}

type openAIRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens *int      `json:"max_tokens,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
