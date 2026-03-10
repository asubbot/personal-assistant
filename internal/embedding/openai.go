package embedding

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
	openAIEmbeddingsPath = "/embeddings"
	defaultTimeout       = 30 * time.Second
)

// OpenAICompatible is an OpenAI-compatible embeddings API client (e.g. OpenAI, Ollama with embedding models).
type OpenAICompatible struct {
	client  *http.Client
	baseURL string
	apiKey  string
	model   string
}

// NewOpenAICompatible builds an embedder from EmbeddingProvider config.
func NewOpenAICompatible(cfg *config.EmbeddingProvider) (*OpenAICompatible, error) {
	baseURL := strings.TrimRight(cfg.Endpoint, "/")
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, fmt.Errorf("embedding: model is required")
	}
	var apiKey string
	if strings.TrimSpace(cfg.APIKeyPath) != "" {
		b, err := os.ReadFile(cfg.APIKeyPath)
		if err != nil {
			return nil, fmt.Errorf("embedding: read api_key_path: %w", err)
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

// Embed implements Embedder.
func (p *OpenAICompatible) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(openAIEmbedRequest{Model: p.model, Input: text})
	if err != nil {
		return nil, fmt.Errorf("embedding: encode request: %w", err)
	}
	url := p.baseURL + openAIEmbeddingsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embedding: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return p.parseResponse(resp)
}

func (p *OpenAICompatible) parseResponse(resp *http.Response) ([]float32, error) {
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
		return nil, fmt.Errorf("embedding api %s: %s", resp.Status, msg)
	}
	var out openAIEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("embedding: decode response: %w", err)
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("embedding: empty data")
	}
	return out.Data[0].Embedding, nil
}

type openAIEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}
