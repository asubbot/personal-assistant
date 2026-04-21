package core

import (
	"context"
	"log/slog"
	"pa/internal/llm"
	"pa/internal/toolcatalog"
	"pa/internal/tools"
	"strings"
	"testing"
)

type fixedSearchVectorToolKnowledgeTool struct{}

func (fixedSearchVectorToolKnowledgeTool) Name() string { return "search_vector_tool" }
func (fixedSearchVectorToolKnowledgeTool) Description() string {
	return "test search_vector_tool"
}

func (fixedSearchVectorToolKnowledgeTool) ParamsSchema() []tools.ParamSpec {
	return []tools.ParamSpec{{Name: "query", Required: true, Type: "string"}}
}

func (fixedSearchVectorToolKnowledgeTool) Run(context.Context, map[string]any) (string, error) {
	return "Tool knowledge hits (top_k=2)\n- web_search score=0.100000 use for external web queries", nil
}

type fixedSearchVectorSkillKnowledgeTool struct{}

func (fixedSearchVectorSkillKnowledgeTool) Name() string { return "search_vector_skill" }
func (fixedSearchVectorSkillKnowledgeTool) Description() string {
	return "test search_vector_skill"
}

func (fixedSearchVectorSkillKnowledgeTool) ParamsSchema() []tools.ParamSpec {
	return []tools.ParamSpec{{Name: "query", Required: true, Type: "string"}}
}

func (fixedSearchVectorSkillKnowledgeTool) Run(context.Context, map[string]any) (string, error) {
	return "Skill knowledge hits (top_k=2)\n- memory-retrieval score=0.100000 use to fetch prior semantic memory", nil
}

// Covers AC-32.017: end-to-end tool loop can use search_vector_tool and produce grounded final answer.
func TestHandleMessage_searchVectorToolLoop_groundedAnswer(t *testing.T) {
	provider := &mockProvider{}
	completeCalls := 0
	provider.CompleteFn = func(_ context.Context, messages []llm.Message, _ *llm.CompletionOptions) (*llm.CompletionResult, error) {
		completeCalls++
		switch completeCalls {
		case 1:
			return &llm.CompletionResult{
				ToolCalls: []llm.ToolCall{
					{ID: "svt1", Name: "search_vector_tool", Arguments: `{"query":"how to search web?"}`},
				},
			}, nil
		case 2:
			toolMsg := messages[len(messages)-1]
			if toolMsg.Role != "tool" || !strings.Contains(toolMsg.Content, "Tool knowledge hits") {
				t.Fatalf("expected tool message with tool knowledge hits, got role=%q content=%q", toolMsg.Role, toolMsg.Content)
			}
			return &llm.CompletionResult{Content: "Use web_search for external web queries."}, nil
		default:
			return &llm.CompletionResult{Content: "unexpected extra call"}, nil
		}
	}

	reg := tools.NewRegistry()
	reg.Register(fixedSearchVectorToolKnowledgeTool{})
	h := &conversationHandler{
		router:                mustRouterSingle(t, provider),
		nativeRegistry:        reg,
		catalog:               &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		logger:                slog.Default(),
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
	}

	reply, err := h.HandleMessage(context.Background(), 1, "", "Which tool should I use for web lookup?")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if completeCalls != 2 {
		t.Fatalf("Complete calls=%d, want 2", completeCalls)
	}
	if !strings.Contains(reply, "web_search") {
		t.Fatalf("reply=%q, want grounded answer from tool context", reply)
	}
}

// Covers AC-32.013: search_vector_skill invocation is logged with redaction policy applied.
func TestHandleMessage_searchVectorSkill_toolInvocationRedactsSensitiveFields(t *testing.T) {
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
					{ID: "svs1", Name: "search_vector_skill", Arguments: `{"query":"secret skill hint"}`},
				},
			}, nil
		}
		return &llm.CompletionResult{Content: "done"}, nil
	}

	reg := tools.NewRegistry()
	reg.Register(fixedSearchVectorSkillKnowledgeTool{})
	h := &conversationHandler{
		router:                mustRouterSingle(t, provider),
		nativeRegistry:        reg,
		catalog:               &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		logger:                logger,
		logRedactor:           redactor,
		maxDynamicSystemRunes: defaultMaxDynamicSystemRunes,
	}
	_, err := h.HandleMessage(context.Background(), 1, "", "Which secret skill should I use?")
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	var argsLog, resLog string
	for _, r := range cap.records {
		if r.msg == "tool invocation" && r.attrs["tool_id"] == "search_vector_skill" {
			argsLog = r.attrs["arguments"]
			resLog = r.attrs["result"]
			break
		}
	}
	if argsLog == "" && resLog == "" {
		t.Fatalf("expected search_vector_skill tool invocation log, records=%+v", cap.records)
	}
	if strings.Contains(argsLog, "secret") || !strings.Contains(argsLog, "[REDACTED]") {
		t.Fatalf("expected redacted arguments, got %q", argsLog)
	}
	if strings.Contains(resLog, "secret") {
		t.Fatalf("expected result without secret leakage, got %q", resLog)
	}
}
