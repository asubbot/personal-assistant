package llm

import (
	"context"
	"errors"
	"log/slog"
	"net"
)

// FallbackProvider tries providers in order; on retryable failure it tries the next.
type FallbackProvider struct {
	providers []Provider
	labels    []string // optional; per-provider label for result.Model (e.g. "openai/gpt-4")
	logger    *slog.Logger
}

// NewFallbackProvider builds a provider that calls providers in order and falls back on retryable errors.
// If logger is nil, slog.Default() is used.
func NewFallbackProvider(providers []Provider, labels []string, logger *slog.Logger) *FallbackProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &FallbackProvider{providers: providers, labels: labels, logger: logger}
}

// Complete implements Provider. On success, result.Model is set from labels[i] when labels are provided.
func (f *FallbackProvider) Complete(ctx context.Context, messages []Message, opts *CompletionOptions) (*CompletionResult, error) {
	var lastErr error
	for i, prov := range f.providers {
		result, err := prov.Complete(ctx, messages, opts)
		if err == nil {
			if f.labels != nil && i < len(f.labels) {
				result.Model = f.labels[i]
			}
			return result, nil
		}
		lastErr = err
		if !isRetryable(err) {
			return nil, err
		}
		nextIdx := i + 1
		if nextIdx >= len(f.providers) {
			break
		}
		args := []any{"failed_index", i, "next_index", nextIdx, "error", err}
		if f.labels != nil {
			if i < len(f.labels) {
				args = append(args, "failed_provider", f.labels[i])
			}
			if nextIdx < len(f.labels) {
				args = append(args, "next_provider", f.labels[nextIdx])
			}
		}
		f.logger.Warn("llm provider failed, trying next", args...)
	}
	return nil, lastErr
}

// isRetryable returns true for errors that warrant trying the next provider (network/5xx/timeout).
func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 {
		return true
	}
	return false
}
