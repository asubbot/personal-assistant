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
	if h.tools.catalog == nil {
		return nil, nil, nil
	}
	topK := h.tools.toolSearchTopK
	if h.tools.runtimeSkillsCfg != nil && h.tools.runtimeSkillsCfg.Enabled && h.tools.runtimeSkillsCfg.ToolVectorTopKCap > 0 && topK > h.tools.runtimeSkillsCfg.ToolVectorTopKCap {
		topK = h.tools.runtimeSkillsCfg.ToolVectorTopKCap
	}
	if h.tools.toolIndex == nil {
		return nil, nil, nil
	}
	ids, err := toolindex.SelectToolIDs(ctx, h.memory.embedder, h.tools.toolIndex.Store(), h.tools.toolIndex.Ready(), h.tools.catalog, userText, topK, h.tools.toolMinCount, h.tools.toolFallbackCap, h.llm.logger)
	if err != nil {
		h.llm.logger.Error("tool pre-selection", "error", err)
		return nil, nil, err
	}
	merged, sources = mergeToolIDs(h.tools.toolsCfg, h.tools.runtimeSkillsCfg, skills, ids)
	return merged, sources, nil
}

func (h *conversationHandler) selectSkillPackages(ctx context.Context, userText string) ([]*runtimeskills.Package, error) {
	if h.tools.runtimeSkillsCfg == nil || !h.tools.runtimeSkillsCfg.Enabled || h.tools.skillIndex == nil || !h.tools.skillIndex.Ready() || len(h.tools.skillPackagesByID) == 0 {
		return nil, nil
	}
	k := h.tools.runtimeSkillsCfg.MaxSkillsPerTurn
	ids, err := skillindex.SearchSkillIDs(ctx, h.memory.embedder, h.tools.skillIndex.Store(), userText, k)
	if err != nil {
		return nil, err
	}
	var out []*runtimeskills.Package
	for _, id := range ids {
		if p, ok := h.tools.skillPackagesByID[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (h *conversationHandler) completionOptionsMergedCatalogNative(ids []string) (*llm.CompletionOptions, error) {
	if !h.llm.firstProviderSupportsTools {
		return nil, nil
	}
	toolDefsForLLM, err := toolcatalog.BuildToolDefs(h.tools.catalog, ids)
	if err != nil {
		return nil, err
	}
	if len(toolDefsForLLM) == 0 && h.tools.nativeRegistry == nil {
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
	if h.tools.nativeRegistry == nil || h.tools.catalog == nil {
		return nil
	}
	names := h.tools.nativeRegistry.List()
	sort.Strings(names)
	var out []llm.ToolDef
	for _, name := range names {
		if _, inCat := h.tools.catalog.Tools[name]; inCat {
			continue
		}
		nt, ok := h.tools.nativeRegistry.Get(name)
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
	if h.tools.catalog != nil {
		if _, ok := h.tools.catalog.Tools[toolID]; ok {
			return h.executeCatalogToolCall(ctx, toolID, argsJSON)
		}
	}
	if h.tools.nativeRegistry != nil {
		if nt, ok := h.tools.nativeRegistry.Get(toolID); ok {
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
	// error-masking: safe — best-effort log enrichment only; omit remote_command on bad JSON
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return ""
	}
	c, _ := m["command"].(string)
	return strings.TrimSpace(c)
}

func (h *conversationHandler) executeCatalogToolCall(ctx context.Context, toolID, argsJSON string) (stdout string, err error) {
	tool, args, err := toolcatalog.ValidateToolCall(h.tools.catalog, toolID, argsJSON)
	if err != nil {
		return "", err
	}
	command, err := toolcatalog.Substitute(tool.Template, args)
	if err != nil {
		return "", fmt.Errorf("tool %q: %w", toolID, err)
	}
	if err := cmdsafe.ValidateRemoteCommand(command); err != nil {
		if h.llm.logger != nil {
			h.llm.logger.InfoContext(ctx, "catalog tool remote command rejected", "tool_id", toolID, "node_id", tool.NodeID, "remote_command", h.redactLogString(command), "error", err)
		}
		return "", fmt.Errorf("tool %q: %w", toolID, err)
	}
	if h.tools.nodeRunner == nil {
		return "", fmt.Errorf("tool %q: no node runner configured", toolID)
	}
	return h.tools.nodeRunner.RunOnNode(ctx, tool.NodeID, command)
}
