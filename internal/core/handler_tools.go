package core

import (
	"context"
	"encoding/json"
	"fmt"
	"pa/internal/cmdsafe"
	"pa/internal/llm"
	"pa/internal/runtimeskills"
	"pa/internal/skillindex"
	"pa/internal/toolcatalog"
	"pa/internal/toolindex"
	"pa/internal/tools"
	"sort"
	"strings"
)

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
