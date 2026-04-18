package core

import (
	"context"
	"fmt"
	"pa/internal/intent"
	"pa/internal/llm"
	"pa/internal/runtimeskills"
	"pa/internal/tooltext"
	"strings"
)

// tierMainLLMParams holds the outcome of tier-specific main-LLM prompt assembly (EP-026).
type tierMainLLMParams struct {
	opts       *llm.CompletionOptions
	textPath   bool
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
	if h.classifier != nil {
		classResult := h.classifier.Classify(ctx, userText)
		tier = classResult.Tier
		intentStage = classResult.Stage
		if h.logger != nil {
			h.logger.InfoContext(ctx, "intent classified",
				"tier", string(tier),
				"stage", classResult.Stage,
				"message_len", classResult.MessageLen,
			)
		}
	}
	if tier == intent.TierFull {
		chunks = h.gatherRetrievedChunkTexts(ctx, userText)
	}
	hasRet := len(chunks) > 0
	sysHead := h.systemStaticHead(hasRet)
	messages = []llm.Message{
		{Role: "system", Content: sysHead},
	}
	if h.sessionMemoryEnabled() {
		for _, ex := range h.sessionStore.snapshot(sk) {
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
	case intent.TierFullLite:
		return h.buildTierFullLiteMainPrompt(ctx, userText, sysHead, messages)
	default:
		return h.buildTierSimpleMainPrompt(), nil
	}
}

func (h *conversationHandler) buildTierSimpleMainPrompt() tierMainLLMParams {
	return tierMainLLMParams{}
}

func (h *conversationHandler) buildTierFullMainPrompt(ctx context.Context, userText, sysHead string, chunks []string, messages []llm.Message) (tierMainLLMParams, error) {
	skills, err := h.selectSkillPackages(ctx, userText)
	if err != nil {
		return tierMainLLMParams{}, err
	}
	return h.mergeTailMergedToolsAndOptions(ctx, userText, sysHead, skills, chunks, messages, false)
}

func (h *conversationHandler) buildTierFullLiteMainPrompt(ctx context.Context, userText, sysHead string, messages []llm.Message) (tierMainLLMParams, error) {
	return h.mergeTailMergedToolsAndOptions(ctx, userText, sysHead, nil, nil, messages, true)
}

func (h *conversationHandler) mergedAfterDynamicToolCap(ctx context.Context, merged []string, dynamicPickRequiresTextBased bool) (picked []string, dynamicRan bool) {
	if h.toolsDynamic == nil || !h.toolsDynamic.Enabled || len(merged) == 0 {
		return merged, false
	}
	if dynamicPickRequiresTextBased && !h.textBasedEnabled {
		return merged, false
	}
	return h.pickToolsForMainRequest(ctx, merged, h.toolsDynamic.MaxToolsForLLMRequest), true
}

func (h *conversationHandler) includeHermesForMainTail(merged []string, couldTextPath bool) bool {
	if !couldTextPath || h.catalog == nil {
		return false
	}
	inner := tooltext.InstructionsForCatalogToolsPlusNative(h.catalog, merged, h.nativeToolDefs())
	return strings.TrimSpace(inner) != ""
}

// mergeTailMergedToolsAndOptions implements the shared full / full_lite tail path.
// When dynamicPickRequiresTextBased is true, dynamic tool capping runs only if h.textBasedEnabled (full_lite semantics).
func (h *conversationHandler) mergeTailMergedToolsAndOptions(ctx context.Context, userText, sysHead string, skills []*runtimeskills.Package, chunks []string, messages []llm.Message, dynamicPickRequiresTextBased bool) (tierMainLLMParams, error) {
	out := tierMainLLMParams{}
	merged, sources, errMer := h.mergeSelectedToolIDs(ctx, userText, skills)
	if errMer != nil {
		return out, errMer
	}
	merged, out.dynamicRan = h.mergedAfterDynamicToolCap(ctx, merged, dynamicPickRequiresTextBased)
	couldTextPath := h.textBasedEnabled && !h.firstProviderSupportsTools
	includeHermes := h.includeHermesForMainTail(merged, couldTextPath)
	tailState := &tailFitState{
		merged:        append([]string(nil), merged...),
		sources:       copyToolOriginMap(sources),
		chunks:        append([]string(nil), chunks...),
		skills:        append([]*runtimeskills.Package(nil), skills...),
		includeHermes: includeHermes,
		textPath:      couldTextPath,
	}
	h.fitDynamicTailToBudget(ctx, tailState, h.maxDynamicSystemRunes)
	opts, err := h.completionOptionsMergedCatalogNative(tailState.merged)
	if err != nil {
		return out, WrapUserError(UserErrorKindConfiguration, err)
	}
	out.opts = opts
	out.textPath = couldTextPath && opts != nil && len(opts.Tools) > 0
	if opts != nil && out.textPath {
		opts.ForceJSONOutput = true
	}
	messages[0].Content = sysHead + h.buildDynamicTailString(tailState)
	return out, nil
}
