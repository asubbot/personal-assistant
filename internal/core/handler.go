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

type handlerToolDeps struct {
	catalog           *toolcatalog.Catalog
	toolIndex         ToolIndex
	skillIndex        SkillIndex
	nativeRegistry    *tools.Registry
	skillPackagesByID map[string]*runtimeskills.Package
	toolsCfg          *config.ToolsConfig
	toolsSelection    *config.ToolsSelection
	toolSearchTopK    int
	toolMinCount      int
	toolFallbackCap   int
	nodeRunner        NodeRunner
	runtimeSkillsCfg  *config.RuntimeSkillsConfig
}

type handlerMemoryDeps struct {
	memVec           *MemoryVectors
	embedder         embedding.Embedder
	memoryVectorTopK config.MemoryVectorConfig
	paLoc            *time.Location
}

type handlerSessionDeps struct {
	sessionCfg   *config.ConversationSessionConfig
	sessionStore *sessionWindowStore
}

type handlerLLMDeps struct {
	router                     *llmrouter.Router
	llmLog                     llmlog.Writer
	model                      string
	firstProviderSupportsTools bool
	logRedactor                func(string) string
	logger                     *slog.Logger
	classifier                 intent.Classifier
	maxMessageLength           int
	maxDynamicSystemRunes      int
}

// conversationHandler implements MessageHandler: vector search, LLM call, optional index (REQ-01.006, REQ-01.007, REQ-01.018).
// Context is built only from vector store (turns and summaries); no full.md day file.
type conversationHandler struct {
	tools                 handlerToolDeps
	memory                handlerMemoryDeps
	session               handlerSessionDeps
	llm                   handlerLLMDeps
	toolResultPromptBytes int // EP-039; stays top-level (single int)
}

// checkUserMessage returns trimmed text, or earlyReply when the message must not reach the LLM.
func (h *conversationHandler) redactLogString(s string) string {
	if h == nil || h.llm.logRedactor == nil {
		return s
	}
	return h.llm.logRedactor(s)
}

func (h *conversationHandler) checkUserMessage(text string) (trimmed string, earlyReply string, reject bool) {
	trimmed = strings.TrimSpace(text)
	if trimmed == "" {
		return "", "Please send a non-empty message.", true
	}
	if h.llm.maxMessageLength > 0 && utf8.RuneCountInString(trimmed) > h.llm.maxMessageLength {
		return "", fmt.Sprintf("Message is too long. Maximum length is %d characters.", h.llm.maxMessageLength), true
	}
	return trimmed, "", false
}

func (h *conversationHandler) sessionMemoryEnabled() bool {
	return h != nil && h.session.sessionCfg != nil && h.session.sessionCfg.Enabled && h.session.sessionStore != nil
}

func (h *conversationHandler) appendSessionIfEnabled(sessionKey, userText, reply string) {
	if !h.sessionMemoryEnabled() {
		return
	}
	k := strings.TrimSpace(sessionKey)
	if k == "" {
		return
	}
	h.session.sessionStore.appendExchange(k, userText, reply, h.session.sessionCfg.MaxSessionExchanges)
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
