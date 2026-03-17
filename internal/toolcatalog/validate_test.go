package toolcatalog

import (
	"strings"
	"testing"
)

// Covers AC-04.005, AC-04.006: validation checks types, allowed_values, pattern, min/max; unknown tool or invalid args return error.
func TestValidateToolCall_UnknownTool_ReturnsError(t *testing.T) {
	catalog := &Catalog{Tools: map[string]*Tool{}}
	_, _, err := ValidateToolCall(catalog, "no_such_tool", `{}`)
	if err == nil {
		t.Fatal("ValidateToolCall(unknown tool): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("ValidateToolCall(unknown tool): error = %v, want 'unknown tool'", err)
	}
}

func TestValidateToolCall_MissingRequired_ReturnsError(t *testing.T) {
	catalog := &Catalog{
		Tools: map[string]*Tool{
			"t": {
				ID: "t", ShortDescription: "x", Template: "cmd", NodeID: "n",
				Arguments: []ArgumentRule{{Name: "req", Type: "string", Required: true}},
			},
		},
	}
	_, _, err := ValidateToolCall(catalog, "t", `{}`)
	if err == nil {
		t.Fatal("ValidateToolCall(missing required): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing required") || !strings.Contains(err.Error(), "req") {
		t.Errorf("ValidateToolCall(missing required): error = %v", err)
	}
}

func TestValidateToolCall_WrongType_ReturnsError(t *testing.T) {
	catalog := &Catalog{
		Tools: map[string]*Tool{
			"t": {
				ID: "t", ShortDescription: "x", Template: "cmd", NodeID: "n",
				Arguments: []ArgumentRule{{Name: "level", Type: "integer", Required: true}},
			},
		},
	}
	_, _, err := ValidateToolCall(catalog, "t", `{"level": "not_a_number"}`)
	if err == nil {
		t.Fatal("ValidateToolCall(wrong type): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "must be integer") {
		t.Errorf("ValidateToolCall(wrong type): error = %v", err)
	}
}

func TestValidateToolCall_AllowedValues_ReturnsError(t *testing.T) {
	catalog := &Catalog{
		Tools: map[string]*Tool{
			"t": {
				ID: "t", ShortDescription: "x", Template: "cmd", NodeID: "n",
				Arguments: []ArgumentRule{{Name: "x", Type: "string", Required: true, AllowedValues: []string{"a", "b"}}},
			},
		},
	}
	_, _, err := ValidateToolCall(catalog, "t", `{"x": "c"}`)
	if err == nil {
		t.Fatal("ValidateToolCall(not in allowed_values): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "must be one of") {
		t.Errorf("ValidateToolCall(allowed_values): error = %v", err)
	}
}

func TestValidateToolCall_Pattern_ReturnsError(t *testing.T) {
	catalog := &Catalog{
		Tools: map[string]*Tool{
			"t": {
				ID: "t", ShortDescription: "x", Template: "cmd", NodeID: "n",
				Arguments: []ArgumentRule{{Name: "id", Type: "string", Required: true, Pattern: `^[a-z]+$`}},
			},
		},
	}
	_, _, err := ValidateToolCall(catalog, "t", `{"id": "Bad123"}`)
	if err == nil {
		t.Fatal("ValidateToolCall(pattern mismatch): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "does not match pattern") {
		t.Errorf("ValidateToolCall(pattern): error = %v", err)
	}
}

func TestValidateToolCall_MinMax_ReturnsError(t *testing.T) {
	min, max := 1, 10
	catalog := &Catalog{
		Tools: map[string]*Tool{
			"t": {
				ID: "t", ShortDescription: "x", Template: "cmd", NodeID: "n",
				Arguments: []ArgumentRule{{Name: "n", Type: "integer", Required: true, Min: &min, Max: &max}},
			},
		},
	}
	_, _, err := ValidateToolCall(catalog, "t", `{"n": 0}`)
	if err == nil {
		t.Fatal("ValidateToolCall(n < min): expected error, got nil")
	}
	_, _, err = ValidateToolCall(catalog, "t", `{"n": 11}`)
	if err == nil {
		t.Fatal("ValidateToolCall(n > max): expected error, got nil")
	}
}

func TestValidateToolCall_ValidArgs_ReturnsToolAndArgs(t *testing.T) {
	catalog := &Catalog{
		Tools: map[string]*Tool{
			"run_uptime": {
				ID: "run_uptime", ShortDescription: "Uptime", Template: "uptime", NodeID: "nas",
				Arguments: []ArgumentRule{{Name: "node_id", Type: "string", Required: true}},
			},
		},
	}
	tool, args, err := ValidateToolCall(catalog, "run_uptime", `{"node_id": "nas"}`)
	if err != nil {
		t.Fatalf("ValidateToolCall(valid): %v", err)
	}
	if tool == nil || tool.ID != "run_uptime" {
		t.Errorf("ValidateToolCall: tool = %v", tool)
	}
	if args["node_id"] != "nas" {
		t.Errorf("ValidateToolCall: args[node_id] = %v, want nas", args["node_id"])
	}
}

func TestValidateToolCall_InvalidJSON_ReturnsError(t *testing.T) {
	catalog := &Catalog{Tools: map[string]*Tool{"t": {ID: "t", ShortDescription: "x", Template: "c", NodeID: "n"}}}
	_, _, err := ValidateToolCall(catalog, "t", `{invalid}`)
	if err == nil {
		t.Fatal("ValidateToolCall(invalid JSON): expected error, got nil")
	}
}
