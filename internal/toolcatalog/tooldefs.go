// BuildToolDefs builds LLM provider tool definitions from the catalog for the given tool ids (REQ-04.004).
// Used to send a bounded tool list in the completion request. Ids not in the catalog are skipped.
package toolcatalog

import (
	"encoding/json"
)

// ToolDefForLLM is one tool definition for the LLM request (name, description, parameters JSON schema).
// Core converts to llm.ToolDef to avoid import cycle.
type ToolDefForLLM struct {
	Name        string
	Description string
	Parameters  string // JSON schema (type object, properties, required)
}

// BuildToolDefs returns one ToolDefForLLM per id that exists in the catalog (order preserved).
// Parameters are a JSON schema derived from the tool's Arguments.
func BuildToolDefs(catalog *Catalog, ids []string) ([]ToolDefForLLM, error) {
	if catalog == nil || len(ids) == 0 {
		return nil, nil
	}
	out := make([]ToolDefForLLM, 0, len(ids))
	for _, id := range ids {
		tool, ok := catalog.Tools[id]
		if !ok {
			continue
		}
		params := buildParamsSchema(tool.Arguments)
		out = append(out, ToolDefForLLM{
			Name:        tool.ID,
			Description: tool.ShortDescription,
			Parameters:  params,
		})
	}
	return out, nil
}

// buildParamsSchema returns a JSON string for OpenAI-style parameters (type object, properties, required).
func buildParamsSchema(args []ArgumentRule) string {
	if len(args) == 0 {
		return ""
	}
	props := make(map[string]interface{})
	var required []string
	for _, a := range args {
		prop := map[string]interface{}{"type": jsonSchemaType(a.Type)}
		if len(a.AllowedValues) > 0 {
			prop["enum"] = a.AllowedValues
		}
		props[a.Name] = prop
		if a.Required {
			required = append(required, a.Name)
		}
	}
	schema := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	b, _ := json.Marshal(schema)
	return string(b)
}

func jsonSchemaType(catalogType string) string {
	switch catalogType {
	case "integer", "int":
		return "integer"
	case "number":
		return "number"
	case "boolean", "bool":
		return "boolean"
	default:
		return "string"
	}
}
