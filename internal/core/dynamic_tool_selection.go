package core

import (
	"context"
	"strings"
)

// ApplyDynamicToolCap returns a copy of ordered truncated to at most max entries,
// preserving prefix order (always_include and higher-priority ids stay first).
func ApplyDynamicToolCap(ordered []string, max int) []string {
	if max < 1 {
		out := make([]string, len(ordered))
		copy(out, ordered)
		return out
	}
	if len(ordered) <= max {
		out := make([]string, len(ordered))
		copy(out, ordered)
		return out
	}
	out := make([]string, max)
	copy(out, ordered[:max])
	return out
}

// pickToolsForMainRequest filters unknown tool ids (WARN) then applies max cap (REQ-18.012–REQ-18.014).
func (h *conversationHandler) pickToolsForMainRequest(ctx context.Context, merged []string, max int) []string {
	if h == nil || len(merged) == 0 {
		return merged
	}
	filtered := make([]string, 0, len(merged))
	for _, id := range merged {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if h.mainLLMToolIDValid(id) {
			filtered = append(filtered, id)
			continue
		}
		if h.llm.logger != nil {
			h.llm.logger.WarnContext(ctx, "dynamic tool selection dropped unknown tool id", "tool_id", id)
		}
	}
	return ApplyDynamicToolCap(filtered, max)
}

func (h *conversationHandler) mainLLMToolIDValid(id string) bool {
	if h.tools.catalog != nil {
		if _, ok := h.tools.catalog.Tools[id]; ok {
			return true
		}
	}
	if h.tools.nativeRegistry != nil {
		if _, ok := h.tools.nativeRegistry.Get(id); ok {
			return true
		}
	}
	return false
}
