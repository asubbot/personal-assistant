package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/intent"
	"pa/internal/llm"
	"pa/internal/llmrouter"
	"pa/internal/prompt"
	"time"
	"unicode/utf8"
)

// genRequestID returns a short unique id for LLM log entries (16 hex chars from 8 random bytes).
func genRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func (h *conversationHandler) onRouteEvent(ctx context.Context, e llmrouter.Event) {
	if e.Action == llmrouter.ActionSwitchNextTransport {
		h.logger.WarnContext(ctx, "llm provider failed, trying next", e.LogAttrs()...)
		return
	}
	h.logger.WarnContext(ctx, "llm routing stop", e.LogAttrs()...)
}

func (h *conversationHandler) completeViaRouter(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	rs := h.router.NewState()
	result, err := h.router.Complete(ctx, rs, messages, opts, func(e llmrouter.Event) {
		h.onRouteEvent(ctx, e)
	})
	if err != nil {
		h.logger.Error("llm complete", "error", err)
	}
	return result, err
}

// completeAt runs Complete through the unified router. usageAcc receives API usage on success when non-nil (EP-015).
func (h *conversationHandler) completeAt(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions, usageAcc *usageTurnAcc) (*llm.CompletionResult, error) {
	if h.logger.Enabled(ctx, slog.LevelDebug) {
		h.logLLMRequest(ctx, messages)
	}
	if h.router == nil {
		return nil, fmt.Errorf("core: llm router is nil")
	}
	result, err := h.completeViaRouter(ctx, messages, opts)
	if err != nil {
		return nil, err
	}
	if usageAcc != nil && result != nil {
		usageAcc.add(result.Usage)
		h.logMainLLMCompletion(ctx, usageAcc.round, len(messages), result)
	}
	return result, nil
}

// systemStaticHead returns the fixed prefix (trust + marker semantics, calendar date, personality).
// The date line is YYYY-MM-DD only (no clock time) in pa_timezone so prompt text stays stable within a calendar day for caching.
// Web output guidance lives in the optional runtime skill `web-output-brief` (skills_dir) when selected by vector search.
func (h *conversationHandler) systemStaticHead() string {
	dateStr := todayCalendarDateInPALocation(h)
	dateLine := "Calendar date: " + dateStr + "\n\n"
	personality := "You are a helpful assistant. Reply concisely.\n\n"
	return prompt.TrustPolicy + "\n\n" + dateLine + personality
}

func todayCalendarDateInPALocation(h *conversationHandler) string {
	loc := time.UTC
	if h != nil && h.paLoc != nil {
		loc = h.paLoc
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func (h *conversationHandler) finishAfterFirstLLM(ctx context.Context, requestID, sessionKey, userText string, start time.Time, messages []llm.Message, result *llm.CompletionResult, opts *llm.CompletionOptions, usageAcc *usageTurnAcc) (string, error) {
	if len(result.ToolCalls) == 0 {
		h.handleLLMSuccess(ctx, requestID, messages, result, userText, time.Since(start))
		h.appendSessionIfEnabled(sessionKey, userText, result.Content)
		return result.Content, nil
	}
	messages, result, err := h.runToolResultLoop(ctx, messages, result, opts, usageAcc)
	if err != nil {
		return "", err
	}
	h.handleLLMSuccess(ctx, requestID, messages, result, userText, time.Since(start))
	h.appendSessionIfEnabled(sessionKey, userText, result.Content)
	return result.Content, nil
}

func paLocationFromConfig(cfg *config.Config) *time.Location {
	if cfg == nil {
		return time.UTC
	}
	l, err := config.PALocation(cfg)
	if err != nil {
		return time.UTC
	}
	return l
}

func (h *conversationHandler) logMainLLMPromptAssembled(ctx context.Context, tier intent.Tier, opts *llm.CompletionOptions, dynamicRan bool, intentStage string) {
	if h == nil || h.logger == nil {
		return
	}
	n := 0
	if opts != nil {
		n = len(opts.Tools)
	}
	h.logger.InfoContext(ctx, "main llm prompt assembled",
		"tier", string(tier),
		"main_tool_count", n,
		"dynamic_tool_selection", dynamicRan,
		"intent_stage", intentStage,
	)
}

// runToolResultLoop continues until no tool_calls or max rounds (REQ-04.006).
func (h *conversationHandler) runToolResultLoop(ctx context.Context, messages []llm.Message, result *llm.CompletionResult, opts *llm.CompletionOptions, usageAcc *usageTurnAcc) ([]llm.Message, *llm.CompletionResult, error) {
	for rounds := 1; len(result.ToolCalls) > 0 && rounds < maxToolRounds; rounds++ {
		messages = h.appendToolRound(ctx, messages, result)
		var err error
		result, err = h.completeAt(ctx, messages, opts, usageAcc)
		if err != nil {
			h.logger.Error("llm complete", "error", err)
			return nil, nil, err
		}
	}
	return messages, result, nil
}

func (h *conversationHandler) truncateToolResultForPrompt(content string) string {
	limit := maxToolResultPromptBytes
	if h != nil && h.toolResultPromptBytes > 0 {
		limit = h.toolResultPromptBytes
	}
	if len(content) <= limit {
		return content
	}
	for limit > 0 && !utf8.ValidString(content[:limit]) {
		limit--
	}
	if limit < 1 {
		return "[tool output truncated: content omitted]"
	}
	truncated := content[:limit]
	omitted := len(content) - len(truncated)
	return fmt.Sprintf("%s\n\n[tool output truncated: %d bytes omitted]", truncated, omitted)
}

func (h *conversationHandler) appendToolRound(ctx context.Context, messages []llm.Message, result *llm.CompletionResult) []llm.Message {
	messages = append(messages, llm.Message{Role: "assistant", Content: result.Content, ToolCalls: result.ToolCalls})
	for _, tc := range result.ToolCalls {
		stdout, execErr := h.executeOneToolCall(ctx, tc.Name, tc.Arguments)
		content := stdout
		if execErr != nil {
			content = execErr.Error()
		}
		if h.logger != nil {
			argsLog := h.redactLogString(tc.Arguments)
			remoteCmd := remoteCommandFromRunOnNodeArgs(tc.Name, tc.Arguments)
			if execErr != nil {
				attrs := []any{"tool_id", tc.Name, "arguments", argsLog, "invoked_via", "tool_calls", "error", h.redactLogString(execErr.Error())}
				if remoteCmd != "" {
					attrs = append(attrs, "remote_command", h.redactLogString(remoteCmd))
				}
				h.logger.InfoContext(ctx, "tool invocation", attrs...)
			} else {
				attrs := []any{"tool_id", tc.Name, "arguments", argsLog, "invoked_via", "tool_calls", "result", h.redactLogString(stdout)}
				if remoteCmd != "" {
					attrs = append(attrs, "remote_command", h.redactLogString(remoteCmd))
				}
				h.logger.InfoContext(ctx, "tool invocation", attrs...)
			}
		}
		messages = append(messages, llm.Message{Role: "tool", Content: h.truncateToolResultForPrompt(content), ToolCallID: tc.ID})
	}
	return messages
}

// logLLMRequest logs the full request at DEBUG (REQ-01.021). Content may be truncated and redacted (REQ-01.026).
func (h *conversationHandler) logLLMRequest(ctx context.Context, messages []llm.Message) {
	for i, m := range messages {
		content := m.Content
		if len(content) > logTruncateMaxLen {
			content = content[:logTruncateMaxLen] + "...[truncated]"
		}
		if h.logRedactor != nil {
			content = h.logRedactor(content)
		}
		h.logger.DebugContext(ctx, "llm request", "index", i, "role", m.Role, "content_len", len(m.Content), "content", content)
	}
}

// logMainLLMCompletion logs per-completion usage for the main chat model at INFO (REQ-01.021: metadata only).
// One line per successful Complete (including each tool-follow-up round).
func (h *conversationHandler) logMainLLMCompletion(ctx context.Context, round, messageCount int, result *llm.CompletionResult) {
	if h == nil || h.logger == nil || result == nil {
		return
	}
	model := result.Model
	if model == "" {
		model = h.model
	}
	h.logger.InfoContext(ctx, "main llm completion",
		"round", round,
		"message_count", messageCount,
		"response_len", len(result.Content),
		"tool_calls", len(result.ToolCalls),
		"prompt_tokens", result.Usage.PromptTokens,
		"completion_tokens", result.Usage.CompletionTokens,
		"total_tokens", result.Usage.TotalTokens,
		"model", model,
	)
}

// logLLMResponse logs the full response at DEBUG (REQ-01.021). Content may be truncated and redacted (REQ-01.026).
func (h *conversationHandler) logLLMResponse(ctx context.Context, result *llm.CompletionResult) {
	content := result.Content
	if len(content) > logTruncateMaxLen {
		content = content[:logTruncateMaxLen] + "...[truncated]"
	}
	if h.logRedactor != nil {
		content = h.logRedactor(content)
	}
	h.logger.DebugContext(ctx, "llm response", "content", content, "content_len", len(result.Content), "usage", result.Usage)
}
