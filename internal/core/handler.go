package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"pa/internal/cmdsafe"
	"pa/internal/config"
	"pa/internal/core/toolfailure"
	"pa/internal/embedding"
	"pa/internal/escalationpolicy"
	"pa/internal/llm"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/promptmarkers"
	"pa/internal/runtimeskills"
	"pa/internal/skillindex"
	"pa/internal/summarize"
	"pa/internal/systemprompt"
	"pa/internal/toolcatalog"
	"pa/internal/toolindex"
	"pa/internal/tools"
	"pa/internal/tooltext"
	"pa/internal/vector"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultMaxDynamicSystemRunes = 4000 // tests: handler.maxDynamicSystemRunes when simulating production defaults
	defaultVectorSearchTopK      = 10   // tests: explicit handler.vectorSearchTopK when simulating production defaults
	logTruncateMaxLen            = 2000 // max chars per message/response when logging at DEBUG (REQ-01.021)
	maxToolRounds                = 10   // max request–tool-result rounds to avoid infinite loop (REQ-04.006)
)

// genRequestID returns a short unique id for LLM log entries (16 hex chars from 8 random bytes).
func genRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// llmTurnState holds active provider index and escalation count for one user message (EP-006).
type llmTurnState struct {
	activeIdx int
	escUsed   int
}

// conversationHandler implements MessageHandler: vector search, LLM call, optional index (REQ-01.006, REQ-01.007, REQ-01.018).
// Context is built only from vector store (turns and summaries); no full.md day file.
type conversationHandler struct {
	router                *llmrouter.Router
	escalation            *config.LLMEscalationConfig
	vectorStore           vector.Store // optional; for semantic search and indexing
	embedder              embedding.Embedder
	nodeRunner            NodeRunner // optional; for tools that run allowlisted commands on nodes (REQ-01.004, REQ-01.005, REQ-01.013)
	toolIndex             ToolIndex  // optional; for tool pre-selection when Ready() (step 3.1)
	skillIndex            SkillIndex // optional; vec_skills when Ready() (EP-013)
	runtimeSkillsCfg      *config.RuntimeSkillsConfig
	toolsCfg              *config.ToolsConfig // optional; always_include and other tools.* JSON (EP-013)
	skillPackagesByID     map[string]*runtimeskills.Package
	nativeRegistry        *tools.Registry // optional; native tools (run_on_node, create_tool) when not overridden by catalog id
	catalog               *toolcatalog.Catalog
	toolSearchTopK        int
	toolMinCount          int
	toolFallbackCap       int
	logger                *slog.Logger
	maxMessageLength      int
	maxDynamicSystemRunes int                 // max UTF-8 runes for dynamic system tail (from config; >= 1 when using loaded config)
	vectorSearchTopK      int                 // vector search top-K for retrieved chunks (from config; >= 1 when using loaded config)
	llmLog                llmlog.Writer       // optional; when set, each LLM call is logged as JSONL
	model                 string              // configured model name for LLM log entries
	logRedactor           func(string) string // optional; redacts content in DEBUG app logs and INFO tool-invocation logs (REQ-01.026)
	// textBasedEnabled + firstProviderSupportsTools: when true + false, Hermes text tool path (REQ-04.027–029).
	textBasedEnabled           bool
	firstProviderSupportsTools bool // true if first LLM provider sends tools in HTTP (supports_tools)
	// EP-014 sliding session memory (optional).
	sessionCfg   *config.ConversationSessionConfig
	sessionStore *sessionWindowStore
	// paLoc is pa_timezone for vector turn dates (EP-002); nil means UTC.
	paLoc *time.Location
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

func (h *conversationHandler) escalationEnabled() bool {
	if h.escalation == nil || !h.escalation.Enabled {
		return false
	}
	return h.router != nil
}

func activeIndex(st *llmTurnState) int {
	if st == nil {
		return 0
	}
	return st.activeIdx
}

func (h *conversationHandler) onRouteEvent(ctx context.Context, e llmrouter.Event) {
	if e.Action == llmrouter.ActionEscalatePolicy {
		h.logger.InfoContext(ctx, "llm tool escalation", e.LogAttrs()...)
		return
	}
	if e.Action == llmrouter.ActionSwitchNextTransport {
		h.logger.WarnContext(ctx, "llm provider failed, trying next", e.LogAttrs()...)
		return
	}
	h.logger.WarnContext(ctx, "llm routing stop", e.LogAttrs()...)
}

func (h *conversationHandler) completeViaRouter(ctx context.Context, st *llmTurnState, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	rs := &llmrouter.State{ActiveIndex: 0, EscUsed: 0}
	if st != nil {
		rs.ActiveIndex = st.activeIdx
		rs.EscUsed = st.escUsed
	}
	result, err := h.router.Complete(ctx, rs, messages, opts, func(e llmrouter.Event) {
		h.onRouteEvent(ctx, e)
	})
	if st != nil {
		st.activeIdx = rs.ActiveIndex
		st.escUsed = rs.EscUsed
	}
	if err != nil {
		h.logger.Error("llm complete", "error", err, "provider_index", activeIndex(st))
	}
	return result, err
}

// completeAt runs Complete through the unified router.
func (h *conversationHandler) completeAt(ctx context.Context, st *llmTurnState, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	if h.logger.Enabled(ctx, slog.LevelDebug) {
		h.logLLMRequest(ctx, messages)
	}
	if h.router == nil {
		return nil, fmt.Errorf("core: llm router is nil")
	}
	return h.completeViaRouter(ctx, st, messages, opts)
}

// failureClass is e.g. tool_execution, hermes_parse (REQ-06.010, REQ-06.016).
func (h *conversationHandler) maybeEscalate(ctx context.Context, st *llmTurnState, hadQualifyingFailure bool, failureClass string) {
	if st == nil || !h.escalationEnabled() || !hadQualifyingFailure {
		return
	}
	if failureClass == "" {
		failureClass = "tool_execution"
	}
	if st.escUsed >= h.escalation.MaxPerUserMessage {
		return
	}
	if h.router == nil {
		return
	}
	rs := &llmrouter.State{ActiveIndex: st.activeIdx, EscUsed: st.escUsed}
	escalated := h.router.OnQualifyingFailure(rs, llmrouter.PhaseToolFailure, failureClass, func(e llmrouter.Event) {
		h.onRouteEvent(ctx, e)
	})
	if escalated {
		st.activeIdx = rs.ActiveIndex
		st.escUsed = rs.EscUsed
	}
}

// textToolModeAfterFirstCompletion sets result.ToolCalls from Hermes when applicable. plainDone: finish with result.Content. invalidFormatReply: user chat message when markup is broken.
func textToolModeAfterFirstCompletion(textPath bool, result *llm.CompletionResult, opts *llm.CompletionOptions) (textToolMode, plainDone bool, invalidFormatReply string) {
	if !textPath || len(result.ToolCalls) > 0 || opts == nil || len(opts.Tools) == 0 {
		return false, false, ""
	}
	calls, perr := tooltext.ParseHermesToolCalls(result.Content)
	if perr != nil {
		return false, true, "Invalid tool call format in the assistant response. Please try again."
	}
	if len(calls) == 0 {
		if tooltext.SuspectedBrokenHermesMarkup(result.Content) {
			return false, true, "Invalid tool call format in the assistant response. Please try again."
		}
		return false, true, ""
	}
	result.ToolCalls = calls
	return true, false, ""
}

// systemStaticHead returns the fixed prefix (trust, marker supplement, personality).
// hasRetrieved reflects whether vector search returned at least one non-empty chunk before tail fitting (REQ-13.016).
func (h *conversationHandler) systemStaticHead(hasRetrieved bool) string {
	personality := "You are a helpful assistant. Reply concisely."
	if hasRetrieved {
		personality = "You are a personal assistant. Reply concisely."
	}
	return systemprompt.TrustPolicy + "\n\n" + systemprompt.MarkerSupplement + "\n\n" + personality + "\n\n"
}

func (h *conversationHandler) finishAfterFirstLLM(ctx context.Context, requestID, sessionKey, userText string, start time.Time, messages []llm.Message, result *llm.CompletionResult, opts *llm.CompletionOptions, textPath bool, st *llmTurnState) (string, error) {
	for {
		textToolMode, plainDone, invalidReply := textToolModeAfterFirstCompletion(textPath, result, opts)
		if invalidReply != "" {
			if !h.escalationEnabled() || st == nil {
				return invalidReply, nil
			}
			prev := activeIndex(st)
			h.maybeEscalate(ctx, st, true, "hermes_parse")
			if activeIndex(st) == prev {
				return invalidReply, nil
			}
			var err error
			result, err = h.completeAt(ctx, st, messages, opts)
			if err != nil {
				return "", err
			}
			continue
		}
		if plainDone {
			h.handleLLMSuccess(ctx, requestID, messages, result, userText, time.Since(start))
			h.appendSessionIfEnabled(sessionKey, userText, result.Content)
			return result.Content, nil
		}
		var err error
		messages, result, err = h.runToolResultLoop(ctx, messages, result, opts, textToolMode, st)
		if err != nil {
			return "", err
		}
		h.handleLLMSuccess(ctx, requestID, messages, result, userText, time.Since(start))
		h.appendSessionIfEnabled(sessionKey, userText, result.Content)
		return result.Content, nil
	}
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

// HandleMessage sends the user message to the LLM and returns the assistant reply.
// Runs semantic search, injects context into the LLM call, then indexes the turn (REQ-01.006, REQ-01.007, REQ-01.018).
//
//nolint:gocyclo // tail assembly, tool merge, options, and first completion share one linear flow
func (h *conversationHandler) HandleMessage(ctx context.Context, userID int64, sessionKey string, text string) (string, error) {
	userText, early, stop := h.checkUserMessage(text)
	if stop {
		return early, nil
	}
	EnterUserTurn()
	defer LeaveUserTurn()
	sk := strings.TrimSpace(sessionKey)
	if sk == "" {
		sk = fmt.Sprintf("uid:%d", userID)
	}
	chunks := h.gatherRetrievedChunkTexts(ctx, userText)
	hasRet := len(chunks) > 0
	sysHead := h.systemStaticHead(hasRet)
	messages := []llm.Message{
		{Role: "system", Content: sysHead},
	}
	if h.sessionMemoryEnabled() {
		for _, ex := range h.sessionStore.snapshot(sk) {
			messages = append(messages, llm.Message{Role: "user", Content: ex.user}, llm.Message{Role: "assistant", Content: ex.assistant})
		}
	}
	messages = append(messages, llm.Message{Role: "user", Content: userText})
	skills, err := h.selectSkillPackages(ctx, userText)
	if err != nil {
		return "", err
	}
	merged, sources, err := h.mergeSelectedToolIDs(ctx, userText, skills)
	if err != nil {
		return "", err
	}
	couldTextPath := h.textBasedEnabled && !h.firstProviderSupportsTools
	includeHermes := false
	if couldTextPath && h.catalog != nil {
		inner := tooltext.InstructionsForCatalogToolsPlusNative(h.catalog, merged, h.nativeToolDefs())
		includeHermes = strings.TrimSpace(inner) != ""
	}
	mergedCopy := append([]string(nil), merged...)
	var sourcesCopy map[string]toolOrigin
	if len(sources) > 0 {
		sourcesCopy = make(map[string]toolOrigin, len(sources))
		for k, v := range sources {
			sourcesCopy[k] = v
		}
	}
	tailState := &tailFitState{
		merged:        mergedCopy,
		sources:       sourcesCopy,
		chunks:        append([]string(nil), chunks...),
		skills:        append([]*runtimeskills.Package(nil), skills...),
		includeHermes: includeHermes,
		textPath:      couldTextPath,
	}
	h.fitDynamicTailToBudget(ctx, tailState, h.maxDynamicSystemRunes)
	opts, err := h.completionOptionsMergedCatalogNative(tailState.merged)
	if err != nil {
		return "", err
	}
	textPath := couldTextPath && opts != nil && len(opts.Tools) > 0
	if opts != nil && textPath {
		opts.ForceJSONOutput = true
	}
	messages[0].Content = sysHead + h.buildDynamicTailString(tailState)
	requestID := genRequestID()
	start := time.Now()
	var st *llmTurnState
	if h.router != nil {
		rs := h.router.NewState()
		st = &llmTurnState{activeIdx: rs.ActiveIndex, escUsed: rs.EscUsed}
	}
	result, err := h.completeAt(ctx, st, messages, opts)
	if err != nil {
		return "", err
	}
	return h.finishAfterFirstLLM(ctx, requestID, sk, userText, start, messages, result, opts, textPath, st)
}

// resolveHermesFollowUpCompletion parses result.Content into tool calls for text-tool follow-ups; may escalate and re-Complete (REQ-06.016).
func (h *conversationHandler) resolveHermesFollowUpCompletion(ctx context.Context, st *llmTurnState, messages []llm.Message, optsFollow *llm.CompletionOptions, result *llm.CompletionResult) (*llm.CompletionResult, error) {
	cur := result
	for {
		calls, perr := tooltext.ParseHermesToolCalls(cur.Content)
		if perr == nil && len(calls) == 0 && tooltext.SuspectedBrokenHermesMarkup(cur.Content) {
			perr = fmt.Errorf("suspected broken Hermes markup")
		}
		if perr == nil {
			cur.ToolCalls = calls
			return cur, nil
		}
		if st == nil || !h.escalationEnabled() {
			return nil, fmt.Errorf("follow-up tool_call parse: %w", perr)
		}
		prev := activeIndex(st)
		h.maybeEscalate(ctx, st, true, "hermes_parse")
		if activeIndex(st) == prev {
			return nil, fmt.Errorf("follow-up tool_call parse: %w", perr)
		}
		next, err := h.completeAt(ctx, st, messages, optsFollow)
		if err != nil {
			h.logger.Error("llm complete", "error", err)
			return nil, err
		}
		cur = next
	}
}

// runToolResultLoop continues until no tool_calls or max rounds (REQ-04.006). textToolMode: follow-up without HTTP tools; tool results as user messages (REQ-04.029).
// st is non-nil when EP-006 escalation is enabled; after a qualifying tool failure the active provider may advance before the next Complete.
func (h *conversationHandler) runToolResultLoop(ctx context.Context, messages []llm.Message, result *llm.CompletionResult, opts *llm.CompletionOptions, textToolMode bool, st *llmTurnState) ([]llm.Message, *llm.CompletionResult, error) {
	optsFollow := opts
	if textToolMode {
		optsFollow = copyOptsNoTools(opts)
	}
	for rounds := 1; len(result.ToolCalls) > 0 && rounds < maxToolRounds; rounds++ {
		var qual bool
		messages, qual = h.appendToolRound(ctx, messages, result, textToolMode)
		h.maybeEscalate(ctx, st, qual, "tool_execution")
		var err error
		result, err = h.completeAt(ctx, st, messages, optsFollow)
		if err != nil {
			h.logger.Error("llm complete", "error", err)
			return nil, nil, err
		}
		if textToolMode && len(result.ToolCalls) == 0 {
			result, err = h.resolveHermesFollowUpCompletion(ctx, st, messages, optsFollow, result)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	return messages, result, nil
}

func copyOptsNoTools(o *llm.CompletionOptions) *llm.CompletionOptions {
	if o == nil {
		return nil
	}
	return &llm.CompletionOptions{
		Model:           o.Model,
		MaxTokens:       o.MaxTokens,
		Temperature:     o.Temperature,
		ForceJSONOutput: o.ForceJSONOutput,
		ResponseFormat:  o.ResponseFormat,
	}
}

func (h *conversationHandler) appendToolRound(ctx context.Context, messages []llm.Message, result *llm.CompletionResult, textToolMode bool) ([]llm.Message, bool) {
	qualifyingFailure := false
	if textToolMode {
		messages = append(messages, llm.Message{Role: "assistant", Content: result.Content})
	} else {
		messages = append(messages, llm.Message{Role: "assistant", Content: result.Content, ToolCalls: result.ToolCalls})
	}
	for _, tc := range result.ToolCalls {
		stdout, execErr := h.executeOneToolCall(ctx, tc.Name, tc.Arguments)
		content := stdout
		if execErr != nil {
			content = execErr.Error()
			if toolfailure.QualifiesForEscalation(execErr) {
				qualifyingFailure = true
			}
		}
		if h.logger != nil {
			src := "tool_calls"
			if textToolMode {
				src = "hermes"
			}
			argsLog := h.redactLogString(tc.Arguments)
			remoteCmd := remoteCommandFromRunOnNodeArgs(tc.Name, tc.Arguments)
			if execErr != nil {
				attrs := []any{"tool_id", tc.Name, "arguments", argsLog, "invoked_via", src, "error", h.redactLogString(execErr.Error())}
				if remoteCmd != "" {
					attrs = append(attrs, "remote_command", h.redactLogString(remoteCmd))
				}
				h.logger.InfoContext(ctx, "tool invocation", attrs...)
			} else {
				attrs := []any{"tool_id", tc.Name, "arguments", argsLog, "invoked_via", src, "result", h.redactLogString(stdout)}
				if remoteCmd != "" {
					attrs = append(attrs, "remote_command", h.redactLogString(remoteCmd))
				}
				h.logger.InfoContext(ctx, "tool invocation", attrs...)
			}
		}
		if textToolMode {
			line := fmt.Sprintf("Tool %s (call_id %s) result:\n%s", tc.Name, tc.ID, content)
			messages = append(messages, llm.Message{Role: "user", Content: line})
		} else {
			messages = append(messages, llm.Message{Role: "tool", Content: content, ToolCallID: tc.ID})
		}
	}
	return messages, qualifyingFailure
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
				return "", toolfailure.NoEscalate(err)
			}
			out, err := nt.Run(ctx, params)
			if err != nil {
				wrapped := fmt.Errorf("tool %q: %w", toolID, err)
				if toolID == "create_tool" {
					return "", toolfailure.NoEscalate(wrapped)
				}
				return "", toolfailure.MayEscalate(wrapped)
			}
			return out, nil
		}
	}
	return "", toolfailure.NoEscalate(fmt.Errorf("tool catalog: unknown tool %q", toolID))
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
		return "", escalationpolicy.WrapCatalogValidateError(err)
	}
	command, err := toolcatalog.Substitute(tool.Template, args)
	if err != nil {
		return "", toolfailure.MayEscalate(fmt.Errorf("tool %q: %w", toolID, err))
	}
	if err := cmdsafe.ValidateRemoteCommand(command); err != nil {
		if h.logger != nil {
			h.logger.InfoContext(ctx, "catalog tool remote command rejected", "tool_id", toolID, "node_id", tool.NodeID, "remote_command", h.redactLogString(command), "error", err)
		}
		return "", toolfailure.NoEscalate(fmt.Errorf("tool %q: %w", toolID, err))
	}
	if h.nodeRunner == nil {
		return "", toolfailure.NoEscalate(fmt.Errorf("tool %q: no node runner configured", toolID))
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
	h.logLLMMetadata(ctx, len(messages), result)
	if h.logger.Enabled(ctx, slog.LevelDebug) {
		h.logLLMResponse(ctx, result)
	}
	if h.vectorStore != nil && h.embedder != nil {
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
	topK := h.vectorSearchTopK
	if h.vectorStore == nil || h.embedder == nil {
		return nil
	}
	queryEmbedding, err := h.embedder.Embed(ctx, userText)
	if err != nil {
		h.logger.Error("embed query", "error", err)
		return nil
	}
	results, err := h.vectorStore.Search(ctx, queryEmbedding, topK)
	if err != nil {
		h.logger.Error("vector search", "error", err)
		return nil
	}
	if len(results) == 0 {
		return nil
	}
	var chunks []string
	for _, r := range results {
		if strings.TrimSpace(r.Text) == "" {
			continue
		}
		label := summarize.VectorChunkLabel(r.ID)
		chunks = append(chunks, retrievalChunkWithLabel(label, r.Text))
	}
	h.logger.DebugContext(ctx, "context chunks", "non_empty", len(chunks), "total", len(results))
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

// logLLMMetadata logs message count, response length, and usage at INFO (REQ-01.021).
func (h *conversationHandler) logLLMMetadata(ctx context.Context, messageCount int, result *llm.CompletionResult) {
	h.logger.InfoContext(ctx, "llm call", "message_count", messageCount, "response_len", len(result.Content), "prompt_tokens", result.Usage.PromptTokens, "completion_tokens", result.Usage.CompletionTokens, "total_tokens", result.Usage.TotalTokens)
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

// indexTurn adds the user message and assistant reply to the vector store for future semantic search (REQ-01.007).
func (h *conversationHandler) indexTurn(ctx context.Context, userText, reply string) error {
	loc := time.UTC
	if h != nil && h.paLoc != nil {
		loc = h.paLoc
	}
	dateStr := time.Now().In(loc).Format("2006-01-02")
	chunk := "Date: " + dateStr + "\n[turn]\nUser: " + userText + "\nAssistant: " + reply
	if promptmarkers.TextContainsForbiddenMarkerLine(chunk) {
		return fmt.Errorf("indexTurn: chunk contains forbidden PA marker line")
	}
	emb, err := h.embedder.Embed(ctx, chunk)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return h.vectorStore.Add(ctx, id, emb, chunk)
}
