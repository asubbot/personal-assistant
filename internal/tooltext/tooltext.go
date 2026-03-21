// Package tooltext implements Hermes-style text-based tool calls for models without Tool-calling API (REQ-04.027).
package tooltext

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"pa/internal/llm"
	"pa/internal/toolcatalog"
	"strings"
)

// FormatDescription documents the expected model output (for operators and prompts).
const FormatDescription = `To invoke a tool, output one or more blocks exactly in this form:
<tool_call>
{"name": "<tool_id>", "arguments": { ... }}
</tool_call>
Use the tool id from the list below. Arguments must be a JSON object matching the tool schema. No text is required outside the blocks when calling tools; you may add a short reply after the blocks if needed.`

// InstructionsForTools appends tool descriptions and the Hermes format to the system prompt.
func InstructionsForTools(defs []llm.ToolDef) string {
	if len(defs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n---\nAvailable tools (invoke via text format, not API):\n")
	for _, d := range defs {
		b.WriteString("- ")
		b.WriteString(d.Name)
		b.WriteString(": ")
		b.WriteString(d.Description)
		if d.Parameters != "" {
			b.WriteString("\n  Parameters (JSON schema): ")
			b.WriteString(d.Parameters)
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(FormatDescription)
	b.WriteByte('\n')
	return b.String()
}

// InstructionsForCatalogTools builds the Hermes tool block using hermes_prompt per tool (fallback index_text).
func InstructionsForCatalogTools(cat *toolcatalog.Catalog, ids []string) string {
	if cat == nil || len(ids) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n---\nAvailable tools (invoke via text format, not API):\n")
	for _, id := range ids {
		t := cat.Tools[id]
		if t == nil {
			continue
		}
		b.WriteString("- ")
		b.WriteString(id)
		b.WriteString(": ")
		b.WriteString(t.HermesBody())
		if params := toolcatalog.ParametersJSONForTool(t); params != "" {
			b.WriteString("\n  Parameters (JSON schema): ")
			b.WriteString(params)
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n")
	b.WriteString(FormatDescription)
	b.WriteByte('\n')
	return b.String()
}

// ParseHermesToolCalls extracts tool calls from assistant content. Returns empty slice if no
// <tool_call> blocks. Returns an error if a <tool_call> block is malformed (unclosed, invalid JSON, missing name).
func ParseHermesToolCalls(content string) ([]llm.ToolCall, error) {
	const openTag = "<tool_call>"
	const closeTag = "</tool_call>"
	var out []llm.ToolCall
	i := 0
	for {
		idx := strings.Index(content[i:], openTag)
		if idx == -1 {
			break
		}
		start := i + idx + len(openTag)
		endRel := strings.Index(content[start:], closeTag)
		if endRel == -1 {
			return nil, fmt.Errorf("unclosed %s", openTag)
		}
		inner := strings.TrimSpace(content[start : start+endRel])
		var payload struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(inner), &payload); err != nil {
			return nil, fmt.Errorf("tool_call JSON: %w", err)
		}
		if strings.TrimSpace(payload.Name) == "" {
			return nil, fmt.Errorf("tool_call missing name")
		}
		args := string(payload.Arguments)
		if args == "" || args == "null" {
			args = "{}"
		}
		out = append(out, llm.ToolCall{
			ID:        syntheticID(),
			Name:      strings.TrimSpace(payload.Name),
			Arguments: args,
		})
		i = start + endRel + len(closeTag)
	}
	return out, nil
}

const (
	hermesOpenSuffix = "tool_call>"
	hermesCloseTag   = "</tool_call>"
)

// SuspectedBrokenHermesMarkup reports whether content looks like Hermes-style tool markup
// that ParseHermesToolCalls did not parse (no exact "<tool_call>" blocks). Intended for use
// only when ParseHermesToolCalls returned (nil error, zero calls). Does not inspect or repair JSON.
func SuspectedBrokenHermesMarkup(content string) bool {
	s := strings.TrimSpace(content)
	if s == "" || strings.Contains(s, "<tool_call>") {
		return false
	}
	if strings.Count(s, hermesOpenSuffix) < 1 || strings.Count(s, hermesCloseTag) < 1 {
		return false
	}
	i := strings.Index(s, hermesOpenSuffix)
	j := strings.Index(s, hermesCloseTag)
	return i >= 0 && j >= 0 && i < j
}

func syntheticID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "call_fallback"
	}
	return "call_" + hex.EncodeToString(b)
}
