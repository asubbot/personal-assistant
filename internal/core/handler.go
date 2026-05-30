package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"pa/internal/cmdsafe"
	"pa/internal/config"
	"pa/internal/embedding"
	"pa/internal/intent"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/prompt"
	"pa/internal/runtimeskills"
	"pa/internal/skillindex"
	"pa/internal/summarize"
	"pa/internal/toolcatalog"
	"pa/internal/toolindex"
	"pa/internal/tools"
	"pa/internal/vector"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultMaxDynamicSystemRunes = 4000 // tests: handler.maxDynamicSystemRunes when simulating production defaults
	logTruncateMaxLen            = 2000 // max chars per message/response when logging at DEBUG (REQ-01.021)
	maxToolRounds                = 10   // max request–tool-result rounds to avoid infinite loop (REQ-04.006)
	maxToolResultPromptBytes     = 8 << 10
)

// genRequestID returns a short unique id for LLM log entries (16 hex chars from 8 random bytes).
func genRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// conversationHandler implements MessageHandler: vector search, LLM call, optional index (REQ-01.006, REQ-01.007, REQ-01.018).
// Context is built only from vector store (turns and summaries); no full.md day file.
type conversationHandler struct {
	router                     *llmrouter.Router
	memVec                     *MemoryVectors // optional; EP-016 split memory vectors (nil disables retrieval and turn indexing)
	embedder                   embedding.Embedder
	nodeRunner                 NodeRunner // optional; for tools that run allowlisted commands on nodes (REQ-01.004, REQ-01.005, REQ-01.013)
	toolIndex                  ToolIndex  // optional; for tool pre-selection when Ready() (step 3.1)
	skillIndex                 SkillIndex // optional; vec_skills when Ready() (EP-013)
	runtimeSkillsCfg           *config.RuntimeSkillsConfig
	toolsCfg                   *config.ToolsConfig // optional; always_include and other tools.* JSON (EP-013)
	skillPackagesByID          map[string]*runtimeskills.Package
	nativeRegistry             *tools.Registry // optional; native tools (run_on_node, create_tool) when not overridden by catalog id
	catalog                    *toolcatalog.Catalog
	toolSearchTopK             int
	toolMinCount               int
	toolFallbackCap            int
	logger                     *slog.Logger
	maxMessageLength           int
	maxDynamicSystemRunes      int                       // max UTF-8 runes for dynamic system tail (from config; >= 1 when using loaded config)
	memoryVectorTopK           config.MemoryVectorConfig // per-lane vector top-K for retrieved memory chunks (from config; 0 disables lane)
	llmLog                     llmlog.Writer             // optional; when set, each LLM call is logged as JSONL
	model                      string                    // configured model name for LLM log entries
	logRedactor                func(string) string       // optional; redacts content in DEBUG app logs and INFO tool-invocation logs (REQ-01.026)
	firstProviderSupportsTools bool                      // baseline LLM provider sends tools in HTTP (supports_tools)
	// EP-014 sliding session memory (optional).
	sessionCfg   *config.ConversationSessionConfig
	sessionStore *sessionWindowStore
	// paLoc is pa_timezone for vector turn dates (EP-002); nil means UTC.
	paLoc *time.Location
	// classifier is the optional EP-017 intent classifier; nil = disabled (always full tier).
	classifier intent.Classifier
	// toolsDynamic is optional EP-018 main-LLM tool cap; nil = disabled for both tiers.
	toolsDynamic *config.ToolDynamicSelection
}

// checkUserMessage returns trimmed text, or earlyReply when the message must not reach the LLM.
func (h *conversationHandler) redactLogString(s string) string {
	if h == nil || h.logRedactor == nil {
		return s
	}
	return h.logRedactor(s)
}

func (h *conversationHandler) checkUserMessage(text string) (trimmed string, earlyReply string, reject bool) {
	trimmed = strings.TrimSpace(text)
	if trimmed == "" {
		return "", "Please send a non-empty message.", true
	}
	if h.maxMessageLength > 0 && utf8.RuneCountInString(trimmed) > h.maxMessageLength {
		return "", fmt.Sprintf("Message is too long. Maximum length is %d characters.", h.maxMessageLength), true
	}
	return trimmed, "", false
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

func (h *conversationHandler) sessionMemoryEnabled() bool {
	return h != nil && h.sessionCfg != nil && h.sessionCfg.Enabled && h.sessionStore != nil
}

func (h *conversationHandler) appendSessionIfEnabled(sessionKey, userText, reply string) {
	if !h.sessionMemoryEnabled() {
		return
	}
	k := strings.TrimSpace(sessionKey)
	if k == "" {
		return
	}
	h.sessionStore.appendExchange(k, userText, reply, h.sessionCfg.MaxSessionExchanges)
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

// HandleMessage sends the user message to the LLM and returns the assistant reply.
// Runs semantic search, injects context into the LLM call, then indexes the turn (REQ-01.006, REQ-01.007, REQ-01.018).
func (h *conversationHandler) HandleMessage(ctx context.Context, userID int64, sessionKey string, text string) (string, error) {
	userText, early, stop := h.checkUserMessage(text)
	if stop {
		return early, nil
	}
	EnterUserTurn()
	defer LeaveUserTurn()
	sk, tier, intentStage, chunks, messages := h.buildMainTurnMessagesPreTail(ctx, userID, sessionKey, userText)
	sysHead := messages[0].Content

	tierParams, err := h.assembleTierMainLLMParams(ctx, tier, userText, sysHead, chunks, messages)
	if err != nil {
		return "", err
	}
	opts := tierParams.opts
	dynamicRan := tierParams.dynamicRan
	h.logMainLLMPromptAssembled(ctx, tier, opts, dynamicRan, intentStage)
	requestID := genRequestID()
	start := time.Now()
	var usageAcc usageTurnAcc
	result, err := h.completeAt(ctx, messages, opts, &usageAcc)
	if err != nil {
		return "", err
	}
	reply, err := h.finishAfterFirstLLM(ctx, requestID, sk, userText, start, messages, result, opts, &usageAcc)
	if err != nil {
		return "", err
	}
	if line := usageAcc.footerLine(string(tier)); line != "" && strings.TrimSpace(reply) != "" {
		return reply + "\n" + line, nil
	}
	return reply, nil
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

func truncateToolResultForPrompt(content string) string {
	if len(content) <= maxToolResultPromptBytes {
		return content
	}
	limit := maxToolResultPromptBytes
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
		messages = append(messages, llm.Message{Role: "tool", Content: truncateToolResultForPrompt(content), ToolCallID: tc.ID})
	}
	return messages
}

// mergeSelectedToolIDs merges tools.always_include, skill-linked, and vector-selected catalog tool ids (EP-013).
// When the tool index is nil or the catalog is nil, it returns nil, nil (native-only or no tools path).
func (h *conversationHandler) mergeSelectedToolIDs(ctx context.Context, userText string, skills []*runtimeskills.Package) (merged []string, sources map[string]toolOrigin, err error) {
	if h.catalog == nil {
		return nil, nil, nil
	}
	topK := h.toolSearchTopK
	if h.runtimeSkillsCfg != nil && h.runtimeSkillsCfg.Enabled && h.runtimeSkillsCfg.ToolVectorTopKCap > 0 && topK > h.runtimeSkillsCfg.ToolVectorTopKCap {
		topK = h.runtimeSkillsCfg.ToolVectorTopKCap
	}
	if h.toolIndex == nil {
		return nil, nil, nil
	}
	ids, err := toolindex.SelectToolIDs(ctx, h.embedder, h.toolIndex.Store(), h.toolIndex.Ready(), h.catalog, userText, topK, h.toolMinCount, h.toolFallbackCap, h.logger)
	if err != nil {
		h.logger.Error("tool pre-selection", "error", err)
		return nil, nil, err
	}
	merged, sources = mergeToolIDs(h.toolsCfg, h.runtimeSkillsCfg, skills, ids)
	return merged, sources, nil
}

func (h *conversationHandler) selectSkillPackages(ctx context.Context, userText string) ([]*runtimeskills.Package, error) {
	if h.runtimeSkillsCfg == nil || !h.runtimeSkillsCfg.Enabled || h.skillIndex == nil || !h.skillIndex.Ready() || len(h.skillPackagesByID) == 0 {
		return nil, nil
	}
	k := h.runtimeSkillsCfg.MaxSkillsPerTurn
	ids, err := skillindex.SearchSkillIDs(ctx, h.embedder, h.skillIndex.Store(), userText, k)
	if err != nil {
		return nil, err
	}
	var out []*runtimeskills.Package
	for _, id := range ids {
		if p, ok := h.skillPackagesByID[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (h *conversationHandler) completionOptionsMergedCatalogNative(ids []string) (*llm.CompletionOptions, error) {
	if !h.firstProviderSupportsTools {
		return nil, nil
	}
	toolDefsForLLM, err := toolcatalog.BuildToolDefs(h.catalog, ids)
	if err != nil {
		return nil, err
	}
	if len(toolDefsForLLM) == 0 && h.nativeRegistry == nil {
		return nil, nil
	}
	toolDefs := make([]llm.ToolDef, 0, len(toolDefsForLLM)+4)
	for i := range toolDefsForLLM {
		toolDefs = append(toolDefs, llm.ToolDef{
			Name:        toolDefsForLLM[i].Name,
			Description: toolDefsForLLM[i].Description,
			Parameters:  toolDefsForLLM[i].Parameters,
		})
	}
	toolDefs = append(toolDefs, h.nativeToolDefs()...)
	if len(toolDefs) == 0 {
		return nil, nil
	}
	return &llm.CompletionOptions{Tools: toolDefs}, nil
}

// nativeToolDefs returns LLM defs for registered native tools whose names are not already in the catalog.
func (h *conversationHandler) nativeToolDefs() []llm.ToolDef {
	if h.nativeRegistry == nil || h.catalog == nil {
		return nil
	}
	names := h.nativeRegistry.List()
	sort.Strings(names)
	var out []llm.ToolDef
	for _, name := range names {
		if _, inCat := h.catalog.Tools[name]; inCat {
			continue
		}
		nt, ok := h.nativeRegistry.Get(name)
		if !ok {
			continue
		}
		out = append(out, tools.LLMDefFromTool(nt))
	}
	return out
}

// executeOneToolCall dispatches catalog tools or native registry tools (REQ-04.009, REQ-04.010, EP-009).
// Returns stdout or an error message string (deterministic) for validation/execution failures.
func (h *conversationHandler) executeOneToolCall(ctx context.Context, toolID, argsJSON string) (stdout string, err error) {
	if h.catalog != nil {
		if _, ok := h.catalog.Tools[toolID]; ok {
			return h.executeCatalogToolCall(ctx, toolID, argsJSON)
		}
	}
	if h.nativeRegistry != nil {
		if nt, ok := h.nativeRegistry.Get(toolID); ok {
			params, err := parseToolArgumentsJSON(argsJSON)
			if err != nil {
				return "", err
			}
			out, err := nt.Run(ctx, params)
			if err != nil {
				return "", fmt.Errorf("tool %q: %w", toolID, err)
			}
			return out, nil
		}
	}
	return "", fmt.Errorf("tool catalog: unknown tool %q", toolID)
}

func parseToolArgumentsJSON(argsJSON string) (map[string]any, error) {
	s := strings.TrimSpace(argsJSON)
	if s == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, fmt.Errorf("tool arguments JSON: %w", err)
	}
	return m, nil
}

// remoteCommandFromRunOnNodeArgs extracts the command field from native run_on_node tool JSON for INFO logs (correlate with noderunner remote_command).
func remoteCommandFromRunOnNodeArgs(toolID, argsJSON string) string {
	if toolID != "run_on_node" {
		return ""
	}
	s := strings.TrimSpace(argsJSON)
	if s == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return ""
	}
	c, _ := m["command"].(string)
	return strings.TrimSpace(c)
}

func (h *conversationHandler) executeCatalogToolCall(ctx context.Context, toolID, argsJSON string) (stdout string, err error) {
	tool, args, err := toolcatalog.ValidateToolCall(h.catalog, toolID, argsJSON)
	if err != nil {
		return "", err
	}
	command, err := toolcatalog.Substitute(tool.Template, args)
	if err != nil {
		return "", fmt.Errorf("tool %q: %w", toolID, err)
	}
	if err := cmdsafe.ValidateRemoteCommand(command); err != nil {
		if h.logger != nil {
			h.logger.InfoContext(ctx, "catalog tool remote command rejected", "tool_id", toolID, "node_id", tool.NodeID, "remote_command", h.redactLogString(command), "error", err)
		}
		return "", fmt.Errorf("tool %q: %w", toolID, err)
	}
	if h.nodeRunner == nil {
		return "", fmt.Errorf("tool %q: no node runner configured", toolID)
	}
	return h.nodeRunner.RunOnNode(ctx, tool.NodeID, command)
}

// handleLLMSuccess logs the LLM call, optionally writes to llmLog, and indexes the turn (REQ-01.018, REQ-01.007).
func (h *conversationHandler) handleLLMSuccess(ctx context.Context, requestID string, messages []llm.Message, result *llm.CompletionResult, userText string, duration time.Duration) {
	if h.llmLog != nil {
		model := h.model
		if result.Model != "" {
			model = result.Model
		}
		h.llmLog.Log(&llmlog.Entry{
			RequestID:       requestID,
			Messages:        messages,
			Model:           model,
			ResponseContent: result.Content,
			Usage:           result.Usage,
			DurationMs:      duration.Milliseconds(),
		})
	}
	if h.logger.Enabled(ctx, slog.LevelDebug) {
		h.logLLMResponse(ctx, result)
	}
	if h.memVec != nil && h.memVec.Turns != nil && h.embedder != nil {
		if err := h.indexTurn(ctx, userText, result.Content); err != nil {
			h.logger.Error("index turn", "error", err)
		}
	}
}

// retrievalChunkWithLabel prepends a type line for the LLM when stored vector text does not already carry it
// (summaries and indexed turns embed `[summary:*]` / `[turn]` after the Date line — avoid duplicating the label).
func retrievalChunkWithLabel(label, body string) string {
	marker := "\n[" + label + "]\n"
	if strings.Contains(body, marker) || strings.HasPrefix(strings.TrimSpace(body), "["+label+"]\n") {
		return body
	}
	return "[" + label + "]\n" + body
}

// gatherRetrievedChunkTexts returns non-empty vector memory chunk texts in search order (REQ-01.006, REQ-01.007).
// The dynamic system tail fitter may drop trailing chunks to satisfy max_dynamic_system_runes.
func (h *conversationHandler) gatherRetrievedChunkTexts(ctx context.Context, userText string) []string {
	mv := h.memVec
	cfgTop := h.memoryVectorTopK
	if mv == nil || !mv.anyNonNil() || h.embedder == nil {
		return nil
	}
	if cfgTop.NotesTopK == 0 && cfgTop.SummariesTopK == 0 && cfgTop.TurnsTopK == 0 {
		return nil
	}
	queryEmbedding, err := h.embedder.Embed(ctx, userText)
	if err != nil {
		h.logger.Error("embed query", "error", err)
		return nil
	}
	return h.gatherSplitTableChunks(ctx, queryEmbedding, cfgTop)
}

func (h *conversationHandler) labeledChunksFromResults(results []vector.SearchResult) []string {
	var chunks []string
	for _, r := range results {
		if strings.TrimSpace(r.Text) == "" {
			continue
		}
		label := summarize.VectorChunkLabel(r.ID)
		chunks = append(chunks, retrievalChunkWithLabel(label, r.Text))
	}
	return chunks
}

func (h *conversationHandler) gatherSplitTableChunks(ctx context.Context, queryEmbedding []float32, topK config.MemoryVectorConfig) []string {
	mv := h.memVec
	var chunks []string
	if mv.Notes != nil && topK.NotesTopK > 0 {
		r, err := mv.Notes.Search(ctx, queryEmbedding, topK.NotesTopK)
		if err != nil {
			h.logger.Error("vector search notes", "error", err)
		} else {
			chunks = append(chunks, h.labeledChunksFromResults(r)...)
		}
	}
	if topK.SummariesTopK > 0 {
		sr, err := mergeSummarySearch(ctx, mv.Summaries, queryEmbedding, topK.SummariesTopK)
		if err != nil {
			h.logger.Error("vector search summaries", "error", err)
			return nil
		}
		chunks = append(chunks, h.labeledChunksFromResults(sr)...)
	}
	if mv.Turns != nil && topK.TurnsTopK > 0 {
		r, err := mv.Turns.Search(ctx, queryEmbedding, topK.TurnsTopK)
		if err != nil {
			h.logger.Error("vector search turns", "error", err)
			return nil
		}
		chunks = append(chunks, h.labeledChunksFromResults(r)...)
	}
	h.logger.DebugContext(ctx, "context chunks", "non_empty", len(chunks))
	return chunks
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

// indexTurn adds the user message and assistant reply to the turn vector store (REQ-01.007, EP-016 dedup).
func (h *conversationHandler) indexTurn(ctx context.Context, userText, reply string) error {
	if h.memVec == nil || h.memVec.Turns == nil || h.embedder == nil {
		return nil
	}
	loc := time.UTC
	if h.paLoc != nil {
		loc = h.paLoc
	}
	dateStr := eventAlignedTurnDate(ctx, loc)
	chunk := "Date: " + dateStr + "\n[turn]\nUser: " + userText + "\nAssistant: " + reply
	if prompt.TextContainsForbiddenMarkerLine(chunk) {
		return fmt.Errorf("indexTurn: chunk contains forbidden PA marker line")
	}
	emb, err := h.embedder.Embed(ctx, chunk)
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(canonicalizeTurnPair(userText, reply)))
	// First 12 bytes of the digest (24 hex chars) keep ids short; collision risk for same-day dedup is negligible.
	id := fmt.Sprintf("turn:%s:%x", dateStr, sum[:12])
	_ = h.memVec.Turns.Delete(ctx, id)
	return h.memVec.Turns.Add(ctx, id, emb, chunk)
}

func eventAlignedTurnDate(ctx context.Context, paLoc *time.Location) string {
	if u := TelegramMessageDateUnix(ctx); u > 0 {
		return time.Unix(u, 0).In(paLoc).Format("2006-01-02")
	}
	return time.Now().In(paLoc).Format("2006-01-02")
}

func canonicalizeTurnPair(userText, reply string) string {
	u := canonicalizeTurnText(userText)
	a := canonicalizeTurnText(reply)
	return u + "\n---\n" + a
}

func canonicalizeTurnText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Join(strings.Fields(s), " ")
}
