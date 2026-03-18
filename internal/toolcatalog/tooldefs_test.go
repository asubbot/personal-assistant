package toolcatalog

import (
	"encoding/json"
	"testing"
)

// Covers AC-04.003: tool list for request built from catalog for selected ids; parameters as JSON schema.
func TestBuildToolDefs_ReturnsToolDefsForIdsInCatalog(t *testing.T) {
	catalog := &Catalog{
		Tools: map[string]*Tool{
			"run_uptime": {
				ID:        "run_uptime",
				IndexText: "Run uptime on the node",
				Template:  "uptime",
				NodeID:    "nas",
				Arguments: []ArgumentRule{{Name: "node_id", Type: "string", Required: true}},
			},
			"other": {
				ID:        "other",
				IndexText: "Other tool",
				Template:  "cmd",
				NodeID:    "nas",
				Arguments: nil,
			},
		},
	}
	ids := []string{"run_uptime", "missing_id", "other"}

	defs, err := BuildToolDefs(catalog, ids)
	if err != nil {
		t.Fatalf("BuildToolDefs: %v", err)
	}
	// missing_id is skipped; order preserved for run_uptime, other.
	if len(defs) != 2 {
		t.Fatalf("BuildToolDefs: len = %d, want 2 (missing_id skipped)", len(defs))
	}
	if defs[0].Name != "run_uptime" || defs[0].Description != "Run uptime on the node" {
		t.Errorf("BuildToolDefs: first tool = %+v", defs[0])
	}
	if defs[1].Name != "other" {
		t.Errorf("BuildToolDefs: second tool Name = %q, want other", defs[1].Name)
	}
	// First tool has parameters schema (object with node_id).
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(defs[0].Parameters), &schema); err != nil {
		t.Fatalf("BuildToolDefs: parameters not valid JSON: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("BuildToolDefs: parameters type = %v, want object", schema["type"])
	}
	props, _ := schema["properties"].(map[string]interface{})
	if props == nil || props["node_id"] == nil {
		t.Errorf("BuildToolDefs: parameters.properties must contain node_id: %v", schema)
	}
}

func TestBuildToolDefs_NilOrEmptyCatalog_ReturnsNil(t *testing.T) {
	defs, err := BuildToolDefs(nil, []string{"x"})
	if err != nil {
		t.Fatalf("BuildToolDefs(nil catalog): %v", err)
	}
	if defs != nil {
		t.Errorf("BuildToolDefs(nil catalog): got %v, want nil", defs)
	}

	defs2, err := BuildToolDefs(&Catalog{Tools: map[string]*Tool{}}, []string{})
	if err != nil {
		t.Fatalf("BuildToolDefs(empty ids): %v", err)
	}
	if defs2 != nil {
		t.Errorf("BuildToolDefs(empty ids): got %v, want nil", defs2)
	}
}

func TestBuildToolDefs_ArgumentsSchema_RequiredAndEnum(t *testing.T) {
	catalog := &Catalog{
		Tools: map[string]*Tool{
			"t": {
				ID:        "t",
				IndexText: "Desc",
				Template:  "cmd",
				NodeID:    "n",
				Arguments: []ArgumentRule{
					{Name: "level", Type: "string", Required: true, AllowedValues: []string{"low", "high"}},
				},
			},
		},
	}
	defs, err := BuildToolDefs(catalog, []string{"t"})
	if err != nil {
		t.Fatalf("BuildToolDefs: %v", err)
	}
	if len(defs) != 1 {
		t.Fatal("BuildToolDefs: want one tool")
	}
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(defs[0].Parameters), &schema); err != nil {
		t.Fatalf("parameters JSON: %v", err)
	}
	required, _ := schema["required"].([]interface{})
	if len(required) != 1 || required[0] != "level" {
		t.Errorf("parameters.required = %v, want [level]", required)
	}
	props, _ := schema["properties"].(map[string]interface{})
	levelProp, _ := props["level"].(map[string]interface{})
	enum, _ := levelProp["enum"].([]interface{})
	if len(enum) != 2 {
		t.Errorf("parameters.properties.level.enum = %v, want [low, high]", enum)
	}
}
