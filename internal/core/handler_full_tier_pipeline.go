package core

import (
	"context"
	"pa/internal/llm"
	"pa/internal/runtimeskills"
)

// fullTierStepOrder documents the fixed assembly order (REQ-41.002, REQ-41.006).
const (
	fullTierStepSelectSkills = iota + 1
	fullTierStepMergeTools
	fullTierStepApplyDynamicCap
	fullTierStepFitTailBudget
	fullTierStepBuildCompletionOptions
)

// fullTierAssembler runs the tier-full tail assembly pipeline (REQ-41.001).
type fullTierAssembler struct {
	h        *conversationHandler
	ctx      context.Context
	userText string
	sysHead  string
	chunks   []string
	messages []llm.Message

	skills     []*runtimeskills.Package
	merged     []string
	sources    map[string]toolOrigin
	tailState  *tailFitState
	dynamicRan bool
}

func newFullTierAssembler(h *conversationHandler, ctx context.Context, userText, sysHead string, chunks []string, messages []llm.Message) *fullTierAssembler {
	return &fullTierAssembler{
		h:        h,
		ctx:      ctx,
		userText: userText,
		sysHead:  sysHead,
		chunks:   append([]string(nil), chunks...),
		messages: messages,
	}
}

func (a *fullTierAssembler) run() (tierMainLLMParams, error) {
	if err := a.stepSelectSkills(); err != nil {
		return tierMainLLMParams{}, err
	}
	return a.runFromSkills()
}

func (a *fullTierAssembler) runFromSkills() (tierMainLLMParams, error) {
	out := tierMainLLMParams{}
	if err := a.stepMergeTools(); err != nil {
		return out, err
	}
	a.stepApplyDynamicCap()
	out.dynamicRan = a.dynamicRan
	a.stepFitTailBudget()
	opts, err := a.stepBuildCompletionOptions()
	if err != nil {
		return out, WrapUserError(UserErrorKindConfiguration, err)
	}
	out.opts = opts
	a.messages[0].Content = a.sysHead + a.h.buildDynamicTailString(a.tailState)
	return out, nil
}

// stepSelectSkills (1): skill selection via selectSkillPackages.
func (a *fullTierAssembler) stepSelectSkills() error {
	skills, err := a.h.selectSkillPackages(a.ctx, a.userText)
	if err != nil {
		return err
	}
	a.skills = skills
	return nil
}

// stepMergeTools (2): tool id merge via mergeSelectedToolIDs.
func (a *fullTierAssembler) stepMergeTools() error {
	merged, sources, err := a.h.mergeSelectedToolIDs(a.ctx, a.userText, a.skills)
	if err != nil {
		return err
	}
	a.merged = merged
	a.sources = sources
	return nil
}

// stepApplyDynamicCap (3): dynamic tool cap via mergedAfterDynamicToolCap.
func (a *fullTierAssembler) stepApplyDynamicCap() {
	a.merged, a.dynamicRan = a.h.mergedAfterDynamicToolCap(a.ctx, a.merged)
}

// stepFitTailBudget (4): tail budget fit via fitDynamicTailToBudget.
func (a *fullTierAssembler) stepFitTailBudget() {
	a.tailState = &tailFitState{
		merged:  append([]string(nil), a.merged...),
		sources: copyToolOriginMap(a.sources),
		chunks:  append([]string(nil), a.chunks...),
		skills:  append([]*runtimeskills.Package(nil), a.skills...),
	}
	a.h.fitDynamicTailToBudget(a.ctx, a.tailState, a.h.llm.maxDynamicSystemRunes)
}

// stepBuildCompletionOptions (5): completion options via completionOptionsMergedCatalogNative.
func (a *fullTierAssembler) stepBuildCompletionOptions() (*llm.CompletionOptions, error) {
	return a.h.completionOptionsMergedCatalogNative(a.tailState.merged)
}
