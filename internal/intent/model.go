package intent

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/llm"
	"regexp"
	"strings"
	"time"
)

const classificationPromptTemplate = `Classify the following user message into exactly one tier. Reply with a single token on the first line: simple, full_lite, or full.

Tiers:
- simple: casual greeting, ping, short acknowledgment, chitchat (no tools or memory needed)
- full_lite: normal conversation without long-term memory retrieval in the prompt; tools may still apply when configured
- full: question requiring knowledge, memory (RAG), tools, or a detailed response

User message:
<<<%s>>>

Reply with exactly one word on the first line: simple, full_lite, or full`

const defaultModelTimeout = 5 * time.Second

// ModelClassifier sends a minimal prompt to a cheap LLM for ambiguous cases (REQ-17.007–REQ-17.009, EP-018 three-way).
type ModelClassifier struct {
	provider llm.Provider
	logger   *slog.Logger
	timeout  time.Duration
}

// NewModelClassifier creates a model-stage classifier.
// timeout <= 0 uses defaultModelTimeout (5s).
func NewModelClassifier(provider llm.Provider, logger *slog.Logger, timeout time.Duration) *ModelClassifier {
	if timeout <= 0 {
		timeout = defaultModelTimeout
	}
	return &ModelClassifier{provider: provider, logger: logger, timeout: timeout}
}

// Classify sends the message to the classification provider and parses the response.
// Returns (tier, error); error means the response was unparseable or the call failed.
func (m *ModelClassifier) Classify(ctx context.Context, message string) (Tier, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	prompt := fmt.Sprintf(classificationPromptTemplate, message)
	msgs := []llm.Message{{Role: "user", Content: prompt}}
	opts := &llm.CompletionOptions{MaxTokens: 16}

	result, err := m.provider.Complete(ctx, msgs, opts)
	if err != nil {
		return "", fmt.Errorf("classification model call: %w", err)
	}

	if m.logger != nil {
		m.logger.Info("intent classification model usage",
			"component", "intent_classifier_model",
			"prompt_tokens", result.Usage.PromptTokens,
			"completion_tokens", result.Usage.CompletionTokens,
		)
	}

	return parseTierResponse(result.Content)
}

var thinkBlockRe = regexp.MustCompile(`(?s)<think>.*?</think>`)

func parseTierResponse(content string) (Tier, error) {
	cleaned := thinkBlockRe.ReplaceAllString(content, "")
	s := strings.TrimSpace(strings.ToLower(cleaned))
	if t, ok := tierFromSingleLineOrPrefix(s); ok {
		return t, nil
	}
	// Multi-line bodies (e.g. Ollama Gemma "reasoning" with prose then a label on the last line).
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if t, ok := tierFromSingleLineOrPrefix(line); ok {
			return t, nil
		}
	}
	return "", fmt.Errorf("unparseable classification response: %q", content)
}

// tierFromSingleLineOrPrefix maps one line (or whole single-line body) to a tier. full_lite is checked before full.
func tierFromSingleLineOrPrefix(s string) (Tier, bool) {
	if strings.HasPrefix(s, "simple") {
		return TierSimple, true
	}
	if strings.HasPrefix(s, "full_lite") || strings.HasPrefix(s, "full-lite") {
		return TierFullLite, true
	}
	if strings.HasPrefix(s, "full") {
		return TierFull, true
	}
	return "", false
}
