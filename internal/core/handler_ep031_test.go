package core

import (
	"context"
	"log/slog"
	"pa/internal/config"
	"pa/internal/llm"
	"pa/internal/toolcatalog"
	"pa/internal/tools"
	"strings"
	"testing"
)

type fixedSearchVectorMemoryTool struct{}

func (fixedSearchVectorMemoryTool) Name() string { return "search_vector_memory" }

func (fixedSearchVectorMemoryTool) Description() string {
	return "test search_vector_memory"
}

func (fixedSearchVectorMemoryTool) ParamsSchema() []tools.ParamSpec {
	return []tools.ParamSpec{{Name: "query", Required: true, Type: "string"}}
}

func (fixedSearchVectorMemoryTool) Run(context.Context, map[string]any) (string, error) {
	return "Vector memory hits (lanes=notes, top_k=3)\n- [notes] notes:2026-04-21:ab score=0.100000 secret deadline is Friday", nil
}

// Covers AC-31.013: end-to-end tool loop can use search_vector_memory and produce grounded final answer.
// Covers AC-31.010: on-demand tool retrieval works when auto-RAG top_k lanes are all zero.
func TestHandleMessage_searchVectorMemoryToolLoop_whenAutoRAGDisabled(t *testing.T) {
	provider := &mockProvider{}
	completeCalls := 0
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		completeCalls++
		switch completeCalls {
		case 1:
			return &llm.CompletionResult{
				ToolCalls: []llm.ToolCall{
					{ID: "svm1", Name: "search_vector_memory", Arguments: `{"query":"when is deadline?"}`},
				},
			}, nil
		case 2:
			toolMsg := messages[len(messages)-1]
			if toolMsg.Role != "tool" || !strings.Contains(toolMsg.Content, "Vector memory hits") {
				t.Fatalf("expected tool message with vector hits, got role=%q content=%q", toolMsg.Role, toolMsg.Content)
			}
			return &llm.CompletionResult{Content: "Your deadline is Friday."}, nil
		default:
			return &llm.CompletionResult{Content: "unexpected extra call"}, nil
		}
	}

	reg := tools.NewRegistry()
	reg.Register(fixedSearchVectorMemoryTool{})
	h := &conversationHandler{
		router:                mustRouterSingle(t, provider),
		nativeRegistry:        reg,
		catalog:               &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		logger:                slog.Default(),
		memoryVectorTopK:      config.MemoryVectorConfig{}, // all lanes zero = auto-RAG disabled
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
	}

	reply, err := h.HandleMessage(context.Background(), 1, "", "When is my deadline?")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if completeCalls != 2 {
		t.Fatalf("Complete calls=%d, want 2", completeCalls)
	}
	if !strings.Contains(reply, "deadline is Friday") {
		t.Fatalf("reply=%q, want grounded answer from tool context", reply)
	}
	if len(provider.lastMessages) > 0 && strings.Contains(provider.lastMessages[0].Content, "Relevant past context:") {
		t.Fatalf("system message must not include auto-RAG context when top_k lanes are zero")
	}
}

// Covers AC-31.009: search_vector_memory invocation is logged with redaction policy applied.
func TestHandleMessage_searchVectorMemory_toolInvocationRedactsSensitiveFields(t *testing.T) {
	cap := &captureHandlerWithAttrs{level: slog.LevelInfo}
	logger := slog.New(cap)
	redactor := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }

	provider := &mockProvider{}
	calls := 0
	provider.CompleteFn = func(_ context.Context, _ []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		calls++
		if calls == 1 {
			return &llm.CompletionResult{
				ToolCalls: []llm.ToolCall{
					{ID: "svm1", Name: "search_vector_memory", Arguments: `{"query":"secret deadline"}`},
				},
			}, nil
		}
		return &llm.CompletionResult{Content: "done"}, nil
	}

	reg := tools.NewRegistry()
	reg.Register(fixedSearchVectorMemoryTool{})
	h := &conversationHandler{
		router:                mustRouterSingle(t, provider),
		nativeRegistry:        reg,
		catalog:               &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		logger:                logger,
		logRedactor:           redactor,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
	}
	_, err := h.HandleMessage(context.Background(), 1, "", "When is my secret deadline?")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	var argsLog, resLog string
	for _, r := range cap.records {
		if r.msg == "tool invocation" && r.attrs["tool_id"] == "search_vector_memory" {
			argsLog = r.attrs["arguments"]
			resLog = r.attrs["result"]
			break
		}
	}
	if argsLog == "" && resLog == "" {
		t.Fatalf("expected search_vector_memory tool invocation log, records=%+v", cap.records)
	}
	if strings.Contains(argsLog, "secret") || !strings.Contains(argsLog, "[REDACTED]") {
		t.Fatalf("expected redacted arguments, got %q", argsLog)
	}
	if strings.Contains(resLog, "secret") || !strings.Contains(resLog, "[REDACTED]") {
		t.Fatalf("expected redacted result, got %q", resLog)
	}
}
