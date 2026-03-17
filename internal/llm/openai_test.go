package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"pa/internal/config"
	"path/filepath"
	"strings"
	"testing"
)

// Supporting AC-01.015 (US-08): LLM provider construction without api_key_path (e.g. ollama) succeeds.
func TestNewOpenAICompatible_validConfig_noAPIKey(t *testing.T) {
	cfg := &config.LLMProvider{
		Type:     "ollama",
		Endpoint: "http://localhost:11434/v1",
		Model:    "llama3.2",
	}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	if p == nil {
		t.Fatal("NewOpenAICompatible: nil")
	}
}

// Covers AC-01.033 (US-19): LLM provider — missing API key file returns error (startup validation).
func TestNewOpenAICompatible_missingAPIKeyFile(t *testing.T) {
	cfg := &config.LLMProvider{
		Type:       "openai",
		Endpoint:   "https://api.openai.com/v1",
		APIKeyPath: filepath.Join(t.TempDir(), "nonexistent"),
		Model:      "gpt-4",
	}
	_, err := NewOpenAICompatible(cfg)
	if err == nil {
		t.Fatal("NewOpenAICompatible(missing key file): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read api_key_path") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("NewOpenAICompatible: error = %v", err)
	}
}

// Supporting AC-01.001, AC-01.016 (US-01, US-08): Complete success path (contract test).
func TestOpenAICompatible_Complete_success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s, want /chat/completions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Hello"},"index":0}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`))
	}))
	defer server.Close()

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}

	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "Hi"}}
	result, err := p.Complete(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "Hello" {
		t.Errorf("Content = %q, want Hello", result.Content)
	}
	if result.Usage.PromptTokens != 2 || result.Usage.CompletionTokens != 1 || result.Usage.TotalTokens != 3 {
		t.Errorf("Usage = %+v", result.Usage)
	}
	// AC-01.044: successful response sets CompletionResult.Model from config.
	if result.Model != "m" {
		t.Errorf("Model = %q, want m", result.Model)
	}
}

// Covers AC-01.036 (US-08): Complete error path — empty choices returns error; system does not crash.
func TestOpenAICompatible_Complete_emptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Complete(context.Background(), []Message{{Role: "user", Content: "x"}}, nil)
	if err == nil {
		t.Fatal("Complete(empty choices): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "empty choices") {
		t.Errorf("Complete: error = %v", err)
	}
}

// Covers AC-01.036 (US-08): Complete error path — 4xx/5xx returns error; system does not crash.
func TestOpenAICompatible_Complete_apiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer server.Close()

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Complete(context.Background(), []Message{{Role: "user", Content: "x"}}, nil)
	if err == nil {
		t.Fatal("Complete(401): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("Complete: error = %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Complete(401): expected errors.As(APIError), got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("APIError.StatusCode = %d, want 401", apiErr.StatusCode)
	}
}

// Covers LLM fallback: 5xx returns APIError so isRetryable is reliable.
func TestOpenAICompatible_Complete_502_returnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"message":"Bad Gateway"}}`))
	}))
	defer server.Close()

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Complete(context.Background(), []Message{{Role: "user", Content: "x"}}, nil)
	if err == nil {
		t.Fatal("Complete(502): expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Complete(502): expected errors.As(APIError), got %T", err)
	}
	if apiErr.StatusCode != 502 {
		t.Errorf("APIError.StatusCode = %d, want 502", apiErr.StatusCode)
	}
}

// Covers AC-01.036 (US-08): Complete error path — invalid JSON returns error; system does not crash.
func TestOpenAICompatible_Complete_invalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Complete(context.Background(), []Message{{Role: "user", Content: "x"}}, nil)
	if err == nil {
		t.Fatal("Complete(invalid JSON): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "decode") && !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("Complete: error = %v (expect decode/invalid)", err)
	}
}

// Covers AC-01.036 (US-08): Complete error path — canceled context returns error; system does not crash.
func TestOpenAICompatible_Complete_contextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
	}))
	defer server.Close()

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before request

	_, err = p.Complete(ctx, []Message{{Role: "user", Content: "x"}}, nil)
	if err == nil {
		t.Fatal("Complete(canceled ctx): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("Complete: error = %v (expect context canceled)", err)
	}
}

// Covers AC-01.036 (US-08): Complete error path — unreachable server returns error; system does not crash.
func TestOpenAICompatible_Complete_serverUnreachable(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://127.0.0.1:19999", Model: "m"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Complete(context.Background(), []Message{{Role: "user", Content: "x"}}, nil)
	if err == nil {
		t.Fatal("Complete(unreachable): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "request") && !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "refused") {
		t.Logf("Complete: error = %v (connection/request failure)", err)
	}
}

// Covers AC-04.003 (REQ-04.004, REQ-04.005): request includes tools when opts.Tools is set.
func TestOpenAICompatible_Complete_requestIncludesTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var req struct {
			Tools []struct {
				Type     string `json:"type"`
				Function struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				} `json:"function"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(req.Tools) == 0 {
			t.Error("request: expected tools in body, got none")
		}
		if len(req.Tools) > 0 && (req.Tools[0].Type != "function" || req.Tools[0].Function.Name != "run_on_node") {
			t.Errorf("request: tools[0] = %+v", req.Tools[0])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""},"index":0}],"usage":{"prompt_tokens":1,"completion_tokens":0,"total_tokens":1}}`))
	}))
	defer server.Close()

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}

	opts := &CompletionOptions{
		Tools: []ToolDef{
			{Name: "run_on_node", Description: "Run command on node", Parameters: `{"type":"object"}`},
		},
	}
	_, err = p.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, opts)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
}

// Covers AC-04.003 (REQ-04.004, REQ-04.005): response with tool_calls is parsed into result.ToolCalls.
func TestOpenAICompatible_Complete_responseToolCallsParsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call_abc","type":"function","function":{"name":"run_on_node","arguments":"{\"command\":\"uptime\"}"}}]},"index":0}],"usage":{"prompt_tokens":5,"completion_tokens":10,"total_tokens":15}}`))
	}))
	defer server.Close()

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}

	result, err := p.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.ToolCalls == nil {
		t.Fatal("Complete: expected ToolCalls, got nil")
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("Complete: len(ToolCalls) = %d, want 1", len(result.ToolCalls))
	}
	tc := result.ToolCalls[0]
	if tc.ID != "call_abc" || tc.Name != "run_on_node" || tc.Arguments != `{"command":"uptime"}` {
		t.Errorf("Complete: ToolCalls[0] = %+v", tc)
	}
}
