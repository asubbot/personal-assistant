package core

import (
	"context"
	"fmt"
	"pa/internal/intent"
	"pa/internal/llm"
	"pa/internal/runtimeskills"
	"strings"
)

// tierMainLLMParams holds the outcome of tier-specific main-LLM prompt assembly (EP-026).
type tierMainLLMParams struct {
	opts       *llm.CompletionOptions
	dynamicRan bool
}

func copyToolOriginMap(sources map[string]toolOrigin) map[string]toolOrigin {
	if len(sources) == 0 {
		return nil
	}
	out := make(map[string]toolOrigin, len(sources))
	for k, v := range sources {
		out[k] = v
	}
	return out
}

// buildMainTurnMessagesPreTail returns session key, tier, intent stage, optional retrieval chunks,
// and the base message stack (system + optional session memory + user) before tier tail assembly.
func (h *conversationHandler) buildMainTurnMessagesPreTail(ctx context.Context, userID int64, sessionKey, userText string) (sk string, tier intent.Tier, intentStage string, chunks []string, messages []llm.Message) {
	sk = strings.TrimSpace(sessionKey)
	if sk == "" {
		sk = fmt.Sprintf("uid:%d", userID)
	}
	tier = intent.TierFull
	intentStage = "disabled"
	if h.llm.classifier != nil {
		classResult := h.llm.classifier.Classify(ctx, userText)
		tier = classResult.Tier
		intentStage = classResult.Stage
		if h.llm.logger != nil {
			h.llm.logger.InfoContext(ctx, "intent classified",
				"tier", string(tier),
				"stage", classResult.Stage,
				"message_len", classResult.MessageLen,
			)
		}
	}
	if tier == intent.TierFull {
		chunks = h.gatherRetrievedChunkTexts(ctx, userText)
	}
	sysHead := h.systemStaticHead()
	messages = []llm.Message{
		{Role: "system", Content: sysHead},
	}
	if h.sessionMemoryEnabled() {
		for _, ex := range h.session.sessionStore.snapshot(sk) {
			messages = append(messages, llm.Message{Role: "user", Content: ex.user}, llm.Message{Role: "assistant", Content: ex.assistant})
		}
	}
	messages = append(messages, llm.Message{Role: "user", Content: userText})
	return sk, tier, intentStage, chunks, messages
}

// assembleTierMainLLMParams dispatches to tier builders after base messages are constructed.
func (h *conversationHandler) assembleTierMainLLMParams(ctx context.Context, tier intent.Tier, userText, sysHead string, chunks []string, messages []llm.Message) (tierMainLLMParams, error) {
	switch tier {
	case intent.TierFull:
		return h.buildTierFullMainPrompt(ctx, userText, sysHead, chunks, messages)
	default:
		return h.buildTierSimpleMainPrompt(), nil
	}
}

func (h *conversationHandler) buildTierSimpleMainPrompt() tierMainLLMParams {
	return tierMainLLMParams{}
}

func (h *conversationHandler) buildTierFullMainPrompt(ctx context.Context, userText, sysHead string, chunks []string, messages []llm.Message) (tierMainLLMParams, error) {
	return newFullTierAssembler(h, ctx, userText, sysHead, chunks, messages).run()
}

func (h *conversationHandler) mergedAfterDynamicToolCap(ctx context.Context, merged []string) (picked []string, dynamicRan bool) {
	if h.tools.toolsSelection == nil || !h.tools.toolsSelection.Enabled || len(merged) == 0 {
		return merged, false
	}
	return h.pickToolsForMainRequest(ctx, merged, h.tools.toolsSelection.MaxToolsForLLMRequest), true
}

// mergeTailMergedToolsAndOptions delegates to fullTierAssembler from step 2 when skills are pre-selected.
func (h *conversationHandler) mergeTailMergedToolsAndOptions(ctx context.Context, userText, sysHead string, skills []*runtimeskills.Package, chunks []string, messages []llm.Message) (tierMainLLMParams, error) {
	a := newFullTierAssembler(h, ctx, userText, sysHead, chunks, messages)
	a.skills = skills
	return a.runFromSkills()
}
