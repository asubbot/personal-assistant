package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"pa/internal/config"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewOpenAICompatible_validConfig succeeds without api_key_path (e.g. ollama).
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

// TestNewOpenAICompatible_missingAPIKeyFile returns error when api_key_path is set but file missing.
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

// TestOpenAICompatible_Complete_success parses 200 response and returns content and usage.
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
}

// TestOpenAICompatible_Complete_emptyChoices returns error.
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

// TestOpenAICompatible_Complete_apiError returns error on 4xx/5xx.
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
}

// TestOpenAICompatible_Complete_invalidJSON returns decode error when response body is not valid JSON.
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

// TestOpenAICompatible_Complete_contextCanceled returns error when context is canceled.
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

// TestOpenAICompatible_Complete_serverUnreachable returns error when server does not respond (e.g. connection refused).
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
