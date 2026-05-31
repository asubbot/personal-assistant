package core

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/embedding"
	"pa/internal/intent"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/runtimeskills"
	"pa/internal/toolcatalog"
	"pa/internal/tools"
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
	// toolsSelection holds the EP-037 tools.selection block (required in config; carries the
	// main-LLM tool cap). nil is only the nil-config/test fallback and disables the cap.
	toolsSelection *config.ToolsSelection
	// toolResultPromptBytes caps tool-result bytes in the main LLM prompt (EP-039); defaults to maxToolResultPromptBytes when unset.
	toolResultPromptBytes int
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
