package core

import (
	"context"
	"pa/internal/runtimeskills"
	"pa/internal/systemprompt"
	"pa/internal/toolcatalog"
	"strings"
	"unicode/utf8"
)

// tailFitState holds mutable pieces of the dynamic system tail (after protected head).
// Order in the final message: TOOLS block, CONTEXT block, SKILLS block (REQ-13.016; wire markers PA_BEGIN_*).
type tailFitState struct {
	merged  []string
	sources map[string]toolOrigin
	chunks  []string // vector memory chunk texts in search order
	skills  []*runtimeskills.Package
}

func formatRetrievedInnerFromChunks(chunks []string) string {
	if len(chunks) == 0 {
		return ""
	}
	prefix := "\n\n---\n\nRelevant past context:\n"
	var buf strings.Builder
	buf.WriteString(prefix)
	for _, t := range chunks {
		buf.WriteString("- ")
		buf.WriteString(t)
		buf.WriteByte('\n')
	}
	return "\n\nUse the following context if relevant to the user's message.\n\n" + buf.String()
}

func (h *conversationHandler) buildDynamicTailString(st *tailFitState) string {
	var b strings.Builder
	if h.catalog != nil && len(st.merged) > 0 {
		if block := toolcatalog.AggregateSystemPrompts(h.catalog, st.merged); block != "" {
			b.WriteString(systemprompt.WrapToolInstructions(block))
		}
	}
	b.WriteString(systemprompt.WrapRetrievedContext(formatRetrievedInnerFromChunks(st.chunks)))
	var sb strings.Builder
	for _, p := range st.skills {
		sb.WriteString(p.PlaybookText())
		sb.WriteString("\n\n")
	}
	b.WriteString(systemprompt.WrapRuntimeSkills(strings.TrimSpace(sb.String())))
	return b.String()
}

func (h *conversationHandler) dynamicTailRunes(st *tailFitState) int {
	return utf8.RuneCountInString(h.buildDynamicTailString(st))
}

// fitDynamicTailToBudget shrinks st in product order until rune count <= maxRunes or no further reduction.
// Order: (1) whole runtime skills from lowest rank, (2) whole retrieved chunks from end,
// (3) catalog tool instructions: vector-only ids from end, then skill-linked (not always), then always_include,
// (5) remove any remaining tool id from end until fit or empty.
func (h *conversationHandler) fitDynamicTailToBudget(ctx context.Context, st *tailFitState, maxRunes int) {
	if maxRunes < 1 {
		return
	}
	before := h.dynamicTailRunes(st)
	if before <= maxRunes {
		return
	}
	trimmed := false
	for h.dynamicTailRunes(st) > maxRunes {
		if !trimDynamicTailOneStep(st) {
			break
		}
		trimmed = true
	}
	after := h.dynamicTailRunes(st)
	if trimmed && h.logger != nil {
		h.logger.InfoContext(ctx, "system dynamic tail trimmed to rune budget",
			"max_dynamic_system_runes", maxRunes,
			"tail_runes_before", before,
			"tail_runes_after", after,
		)
	}
}

func trimDynamicTailOneStep(st *tailFitState) bool {
	if len(st.skills) > 0 {
		st.skills = st.skills[:len(st.skills)-1]
		return true
	}
	if len(st.chunks) > 0 {
		st.chunks = st.chunks[:len(st.chunks)-1]
		return true
	}
	if tryRemoveToolStep4(st) {
		return true
	}
	return tryRemoveToolFromEnd(st)
}

func tryRemoveToolStep4(st *tailFitState) bool {
	if len(st.merged) == 0 || st.sources == nil {
		return false
	}
	for i := len(st.merged) - 1; i >= 0; i-- {
		id := st.merged[i]
		if st.sources[id] == originVector {
			st.merged = append(st.merged[:i], st.merged[i+1:]...)
			delete(st.sources, id)
			return true
		}
	}
	for i := len(st.merged) - 1; i >= 0; i-- {
		id := st.merged[i]
		o := st.sources[id]
		if o&originAlways != 0 {
			continue
		}
		if o&originSkill == 0 {
			continue
		}
		st.merged = append(st.merged[:i], st.merged[i+1:]...)
		delete(st.sources, id)
		return true
	}
	for i := len(st.merged) - 1; i >= 0; i-- {
		id := st.merged[i]
		if st.sources[id]&originAlways == 0 {
			continue
		}
		st.merged = append(st.merged[:i], st.merged[i+1:]...)
		delete(st.sources, id)
		return true
	}
	return false
}

func tryRemoveToolFromEnd(st *tailFitState) bool {
	if len(st.merged) == 0 {
		return false
	}
	i := len(st.merged) - 1
	id := st.merged[i]
	st.merged = st.merged[:i]
	if st.sources != nil {
		delete(st.sources, id)
	}
	return true
}
