package tools

import (
	"encoding/json"
	"pa/internal/llm"
)

// LLMDefFromTool builds an llm.ToolDef from a native Tool (JSON parameters schema from ParamSpec).
func LLMDefFromTool(t Tool) llm.ToolDef {
	if t == nil {
		return llm.ToolDef{}
	}
	specs := t.ParamsSchema()
	props := make(map[string]any)
	var required []string
	for _, p := range specs {
		prop := map[string]any{"type": jsonSchemaParamType(p.Type)}
		props[p.Name] = prop
		if p.Required {
			required = append(required, p.Name)
		}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	b, _ := json.Marshal(schema)
	return llm.ToolDef{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  string(b),
	}
}

func jsonSchemaParamType(t string) string {
	switch t {
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}
