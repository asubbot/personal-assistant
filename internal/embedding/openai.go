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
	client    *http.Client
	baseURL   string
	apiKey    string
	model     string
	batchSize int // max texts per API request (from config); EmbedBatch chunks internally
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
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	return &OpenAICompatible{
		client:    &http.Client{Timeout: defaultTimeout},
		baseURL:   baseURL,
		apiKey:    apiKey,
		model:     model,
		batchSize: batchSize,
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

// EmbedBatch implements BatchEmbedder. Chunks texts by p.batchSize (config), one API request per chunk; returns vectors in order. Empty texts yields nil, nil.
func (p *OpenAICompatible) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if len(texts) == 1 {
		emb, err := p.Embed(ctx, texts[0])
		if err != nil {
			return nil, err
		}
		return [][]float32{emb}, nil
	}
	var out [][]float32
	for start := 0; start < len(texts); start += p.batchSize {
		end := start + p.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		chunk := texts[start:end]
		body, err := json.Marshal(openAIEmbedRequestBatch{Model: p.model, Input: chunk})
		if err != nil {
			return nil, fmt.Errorf("embedding: encode batch request: %w", err)
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
		embs, err := p.parseBatchResponse(resp, len(chunk))
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, embs...)
	}
	return out, nil
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

func (p *OpenAICompatible) parseBatchResponse(resp *http.Response, wantLen int) ([][]float32, error) {
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
		return nil, fmt.Errorf("embedding: decode batch response: %w", err)
	}
	if len(out.Data) != wantLen {
		return nil, fmt.Errorf("embedding: batch response has %d items, want %d", len(out.Data), wantLen)
	}
	result := make([][]float32, len(out.Data))
	for i := range out.Data {
		result[i] = out.Data[i].Embedding
	}
	return result, nil
}

type openAIEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIEmbedRequestBatch struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}
