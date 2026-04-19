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
	"time"
)

// Covers AC-01.036: traceability for TestOpenAICompatible_Complete_contentAsTextPartsArray.
func TestOpenAICompatible_Complete_contentAsTextPartsArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":[{"type":"text","text":"Hello array"}]}}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`))
	}))
	defer server.Close()
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "Hello array" {
		t.Errorf("Content = %q, want Hello array", result.Content)
	}
}

// Covers AC-01.036: traceability for TestOpenAICompatible_Complete_reasoningContentFallback.
func TestOpenAICompatible_Complete_reasoningContentFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","reasoning_content":"Fallback text"}}],"usage":{}}`))
	}))
	defer server.Close()
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "Fallback text" {
		t.Errorf("Content = %q, want Fallback text", result.Content)
	}
}

// Covers AC-01.036: Ollama OpenAI-compatible — Gemma and other thinking models may leave content empty and use "reasoning".
func TestOpenAICompatible_Complete_ollamaReasoningFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":null,"reasoning":"full_lite"}}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`))
	}))
	defer server.Close()
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "full_lite" {
		t.Errorf("Content = %q, want full_lite", result.Content)
	}
}

// Covers AC-01.036: traceability for TestDecodeAssistantMessageContent_variants.
func TestDecodeAssistantMessageContent_variants(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{`"hello"`, "hello"},
		{`null`, ""},
		{`[]`, ""},
		{`[{"type":"text","text":"a"},{"type":"text","text":"b"}]`, "ab"},
		{`[{"text":"x"}]`, "x"},
	}
	for _, tt := range tests {
		got, err := decodeAssistantMessageContent(json.RawMessage(tt.raw))
		if err != nil {
			t.Fatalf("decodeAssistantMessageContent(%q): %v", tt.raw, err)
		}
		if got != tt.want {
			t.Errorf("decodeAssistantMessageContent(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

// Supporting AC-01.015 (US-08): LLM provider construction without api_key_path (e.g. ollama) succeeds.
func TestNewOpenAICompatible_validConfig_noAPIKey(t *testing.T) {
	cfg := &config.LLMProvider{
		Type:                  "ollama",
		Endpoint:              "http://localhost:11434/v1",
		Model:                 "llama3.2",
		DefaultTemperature:    0.3,
		DefaultMaxTokens:      1024,
		DefaultResponseFormat: "text",
		HTTPTimeout:           "60s",
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
		Type:                  "openai",
		Endpoint:              "https://api.openai.com/v1",
		APIKeyPath:            filepath.Join(t.TempDir(), "nonexistent"),
		Model:                 "gpt-4",
		DefaultTemperature:    0.3,
		DefaultMaxTokens:      1024,
		DefaultResponseFormat: "text",
		HTTPTimeout:           "60s",
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

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
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

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
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

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
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
// Covers AC-01.036: traceability for TestOpenAICompatible_Complete_502_returnsAPIError.
func TestOpenAICompatible_Complete_502_returnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"message":"Bad Gateway"}}`))
	}))
	defer server.Close()

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
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

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
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

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
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
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://127.0.0.1:19999", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
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

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
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

// Covers AC-04.028 (REQ-04.034, REQ-04.026): supports_tools false omits tools from HTTP body even when opts.Tools is set.
func TestOpenAICompatible_Complete_supportsToolsFalse_omitsTools(t *testing.T) {
	var raw []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		raw, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"index":0}],"usage":{}}`))
	}))
	defer server.Close()

	f := false
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", SupportsTools: &f, DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}
	opts := &CompletionOptions{Tools: []ToolDef{{Name: "x", Description: "y"}}}
	_, err = p.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"tools"`) {
		t.Errorf("body should not contain tools key; body=%s", string(raw))
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

	cfg := &config.LLMProvider{Type: "ollama", Endpoint: server.URL, Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
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

// buildRequestToolMessagesReq is the shape of the request body for buildRequest serialization test.
type buildRequestToolMessagesReq struct {
	Messages []struct {
		Role       string `json:"role"`
		Content    string `json:"content"`
		ToolCallID string `json:"tool_call_id"`
		ToolCalls  []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Fn   struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"messages"`
}

func assertBuildRequestToolMessages(t *testing.T, req *buildRequestToolMessagesReq) {
	t.Helper()
	if len(req.Messages) != 2 {
		t.Fatalf("messages length = %d, want 2", len(req.Messages))
	}
	m0 := req.Messages[0]
	if m0.Role != "assistant" || len(m0.ToolCalls) != 1 {
		t.Fatalf("first message: role=%q tool_calls=%d", m0.Role, len(m0.ToolCalls))
	}
	tc := m0.ToolCalls[0]
	if tc.ID != "call_1" || tc.Type != "function" || tc.Fn.Name != "run_echo" || tc.Fn.Arguments != `{"msg":"hi"}` {
		t.Errorf("first message tool_calls[0] = %+v", tc)
	}
	m1 := req.Messages[1]
	if m1.Role != "tool" || m1.ToolCallID != "call_1" || m1.Content != "hello from node" {
		t.Errorf("second message: role=%q tool_call_id=%q content=%q", m1.Role, m1.ToolCallID, m1.Content)
	}
}

// Covers tool-result loop (AC-04.004): request body serializes Message with tool_call_id and tool_calls in OpenAI format.
func TestOpenAICompatible_buildRequest_serializesToolMessagesAndAssistantToolCalls(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	messages := []Message{
		{Role: "assistant", Content: "", ToolCalls: []ToolCall{{ID: "call_1", Name: "run_echo", Arguments: `{"msg":"hi"}`}}},
		{Role: "tool", Content: "hello from node", ToolCallID: "call_1"},
	}
	body, err := p.buildRequest("m", messages, 0, nil)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req buildRequestToolMessagesReq
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal request body: %v", err)
	}
	assertBuildRequestToolMessages(t, &req)
}

// EP-008 AC-08.001 / REQ-08.001: default_temperature included in HTTP body when request does not override.
func TestOpenAICompatible_buildRequest_withDefaultTemperature(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.7, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	body, err := p.buildRequest("m", []Message{{Role: "user", Content: "hi"}}, 0, nil)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		Temperature *float64 `json:"temperature"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.Temperature == nil || *req.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", req.Temperature)
	}
}

// EP-008 AC-08.002 / REQ-08.002: CompletionOptions.Temperature overrides provider default in HTTP body.
func TestOpenAICompatible_buildRequest_withOverrideTemperature(t *testing.T) {
	overrideTemp := 0.3
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.7, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	opts := &CompletionOptions{Temperature: &overrideTemp}
	body, err := p.buildRequest("m", []Message{{Role: "user", Content: "hi"}}, 0, opts)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		Temperature *float64 `json:"temperature"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.Temperature == nil || *req.Temperature != 0.3 {
		t.Errorf("Temperature = %v, want 0.3 (override)", req.Temperature)
	}
}

// EP-008 AC-08.003 / REQ-08.003: default_max_tokens in HTTP body when opts.MaxTokens is unset/zero.
func TestOpenAICompatible_buildRequest_withDefaultMaxTokens(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	// Use effectiveModelAndMaxTokens to get the default value
	model, maxTokens := p.effectiveModelAndMaxTokens(nil)
	if maxTokens != 1024 {
		t.Errorf("maxTokens = %d, want 1024", maxTokens)
	}
	body, err := p.buildRequest(model, []Message{{Role: "user", Content: "hi"}}, maxTokens, nil)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		MaxTokens *int `json:"max_tokens"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.MaxTokens == nil || *req.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %v, want 1024", req.MaxTokens)
	}
}

// EP-008 AC-08.004 / REQ-08.004: CompletionOptions.MaxTokens overrides provider default in HTTP body.
func TestOpenAICompatible_buildRequest_withOverrideMaxTokens(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	opts := &CompletionOptions{MaxTokens: 512}
	model, maxTokens := p.effectiveModelAndMaxTokens(opts)
	if maxTokens != 512 {
		t.Errorf("maxTokens = %d, want 512 (override)", maxTokens)
	}
	body, err := p.buildRequest(model, []Message{{Role: "user", Content: "hi"}}, maxTokens, opts)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		MaxTokens *int `json:"max_tokens"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.MaxTokens == nil || *req.MaxTokens != 512 {
		t.Errorf("MaxTokens = %v, want 512 (override)", req.MaxTokens)
	}
}

// Covers AC-30.007, AC-30.008: default completion path uses text response_format only; nil opts use configured default.
func TestOpenAICompatible_buildRequest_nilOpts_usesDefaultResponseFormat(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	body, err := p.buildRequest("m", []Message{{Role: "user", Content: "hi"}}, 0, nil)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		ResponseFormat *struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != "text" {
		t.Errorf("ResponseFormat = %v, want type=text", req.ResponseFormat)
	}
}

// Covers AC-30.008: explicit per-request ResponseFormat type text is honored in the HTTP body.
func TestOpenAICompatible_buildRequest_explicitResponseFormatText(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	opts := &CompletionOptions{ResponseFormat: &ResponseFormat{Type: "text"}}
	body, err := p.buildRequest("m", []Message{{Role: "user", Content: "hi"}}, 0, opts)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		ResponseFormat *struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != "text" {
		t.Errorf("ResponseFormat = %v, want type=text", req.ResponseFormat)
	}
}

// Covers AC-30.008: json_object hint in opts is ignored; body uses text default.
func TestOpenAICompatible_buildRequest_explicitJSONObject_ignoredWithoutJSONMode(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	opts := &CompletionOptions{ResponseFormat: &ResponseFormat{Type: "json_object"}}
	body, err := p.buildRequest("m", []Message{{Role: "user", Content: "hi"}}, 0, opts)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		ResponseFormat *struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != "text" {
		t.Errorf("ResponseFormat = %v, want type=text (unsupported json_object ignored)", req.ResponseFormat)
	}
}

// EP-008: whitespace-only explicit ResponseFormat.Type treated as unset; default format chain applies.
// Covers AC-01.036: traceability for TestOpenAICompatible_buildRequest_emptyExplicitResponseFormatType_usesDefault.
func TestOpenAICompatible_buildRequest_emptyExplicitResponseFormatType_usesDefault(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	opts := &CompletionOptions{ResponseFormat: &ResponseFormat{Type: "   "}}
	body, err := p.buildRequest("m", []Message{{Role: "user", Content: "hi"}}, 0, opts)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		ResponseFormat *struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != "text" {
		t.Errorf("ResponseFormat = %v, want type=text (empty explicit type ignored)", req.ResponseFormat)
	}
}

// Covers AC-30.008: empty explicit ResponseFormat.Type falls through to configured default (text).
func TestOpenAICompatible_buildRequest_emptyExplicitType_usesDefaultText(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "text", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	opts := &CompletionOptions{ResponseFormat: &ResponseFormat{Type: ""}}
	body, err := p.buildRequest("m", []Message{{Role: "user", Content: "hi"}}, 0, opts)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		ResponseFormat *struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != "text" {
		t.Errorf("ResponseFormat = %v, want type=text", req.ResponseFormat)
	}
}

// Covers AC-30.008: explicit ResponseFormat type text overrides a non-text default on the provider struct (tests only).
func TestOpenAICompatible_buildRequest_explicitOverridesDefault(t *testing.T) {
	cfg := &config.LLMProvider{Type: "ollama", Endpoint: "http://localhost", Model: "m", DefaultTemperature: 0.3, DefaultMaxTokens: 1024, DefaultResponseFormat: "json_object", HTTPTimeout: "60s"}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	opts := &CompletionOptions{ResponseFormat: &ResponseFormat{Type: "text"}}
	body, err := p.buildRequest("m", []Message{{Role: "user", Content: "hi"}}, 0, opts)
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	var req struct {
		ResponseFormat *struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != "text" {
		t.Errorf("ResponseFormat = %v, want type=text (explicit override of default)", req.ResponseFormat)
	}
}

// Covers AC-22.004 (EP-022): NewOpenAICompatible propagates the configured
// http_timeout to *http.Client.Timeout verbatim.
func TestNewOpenAICompatible_HTTPTimeout_AppliedToClient(t *testing.T) {
	cfg := &config.LLMProvider{
		Type: "ollama", Endpoint: "http://localhost", Model: "m",
		DefaultTemperature: 0.3, DefaultMaxTokens: 1024,
		DefaultResponseFormat: "text",
		HTTPTimeout:           "45s",
	}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	if got, want := p.client.Timeout, 45*time.Second; got != want {
		t.Fatalf("httpClient.Timeout = %s, want %s", got, want)
	}
}

// Covers AC-22.007 (EP-022): invalid http_timeout string is rejected at the
// construction site with a named error.
func TestNewOpenAICompatible_HTTPTimeout_InvalidRejected(t *testing.T) {
	cfg := &config.LLMProvider{
		Type: "ollama", Endpoint: "http://localhost", Model: "m",
		DefaultTemperature: 0.3, DefaultMaxTokens: 1024,
		DefaultResponseFormat: "text",
		HTTPTimeout:           "not-a-duration",
	}
	_, err := NewOpenAICompatible(cfg)
	if err == nil || !strings.Contains(err.Error(), "http_timeout") {
		t.Fatalf("expected http_timeout error, got %v", err)
	}
}

// Covers AC-22.008 (EP-022): zero http_timeout is rejected.
func TestNewOpenAICompatible_HTTPTimeout_ZeroRejected(t *testing.T) {
	cfg := &config.LLMProvider{
		Type: "ollama", Endpoint: "http://localhost", Model: "m",
		DefaultTemperature: 0.3, DefaultMaxTokens: 1024,
		DefaultResponseFormat: "text",
		HTTPTimeout:           "0s",
	}
	_, err := NewOpenAICompatible(cfg)
	if err == nil || !strings.Contains(err.Error(), "http_timeout") {
		t.Fatalf("expected http_timeout error, got %v", err)
	}
}

// Covers AC-22.008 (EP-022): empty http_timeout is rejected.
func TestNewOpenAICompatible_HTTPTimeout_EmptyRejected(t *testing.T) {
	cfg := &config.LLMProvider{
		Type: "ollama", Endpoint: "http://localhost", Model: "m",
		DefaultTemperature: 0.3, DefaultMaxTokens: 1024,
		DefaultResponseFormat: "text",
		HTTPTimeout:           "",
	}
	_, err := NewOpenAICompatible(cfg)
	if err == nil || !strings.Contains(err.Error(), "http_timeout") {
		t.Fatalf("expected http_timeout error, got %v", err)
	}
}
