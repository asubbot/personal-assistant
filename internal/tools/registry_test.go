package tools

import (
	"context"
	"testing"
)

// TestRegistry_RegisterGetList (AC-022): registry Get/List and single contract (tool by name).
func TestRegistry_RegisterGetList(t *testing.T) {
	r := NewRegistry()
	if got := r.List(); len(got) != 0 {
		t.Errorf("List() = %v, want empty", got)
	}
	tool := &mockTool{name: "foo"}
	r.Register(tool)
	if got, ok := r.Get("foo"); !ok || got != tool {
		t.Errorf("Get(\"foo\") = %v, %v, want tool, true", got, ok)
	}
	if got, ok := r.Get("bar"); ok || got != nil {
		t.Errorf("Get(\"bar\") = %v, %v, want nil, false", got, ok)
	}
	list := r.List()
	if len(list) != 1 || list[0] != "foo" {
		t.Errorf("List() = %v, want [foo]", list)
	}
}

// Covers AC-022 (US-12): tool registration with empty name is rejected (fail fast).
func TestRegistry_RegisterEmptyName_panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register(empty name) did not panic")
		}
	}()
	r := NewRegistry()
	r.Register(&mockTool{name: ""})
}

// Covers AC-022 (US-12): tool registration with duplicate name is rejected (fail fast).
func TestRegistry_RegisterDuplicate_panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register(duplicate) did not panic")
		}
	}()
	r := NewRegistry()
	r.Register(&mockTool{name: "x"})
	r.Register(&mockTool{name: "x"})
}

// TestValidateParams_valid (AC-022): params conforming to schema pass validation.
func TestValidateParams_valid(t *testing.T) {
	spec := []ParamSpec{
		{Name: "a", Required: true, Type: "string"},
		{Name: "b", Required: false, Type: "string"},
	}
	if err := ValidateParams(spec, map[string]any{"a": "v"}); err != nil {
		t.Errorf("ValidateParams = %v", err)
	}
	if err := ValidateParams(spec, map[string]any{"a": "v", "b": "w"}); err != nil {
		t.Errorf("ValidateParams = %v", err)
	}
}

// TestValidateParams_missingRequired (AC-023): missing required param is rejected.
func TestValidateParams_missingRequired(t *testing.T) {
	spec := []ParamSpec{{Name: "a", Required: true, Type: "string"}}
	if err := ValidateParams(spec, map[string]any{}); err == nil {
		t.Error("ValidateParams(missing required) want error")
	}
	if err := ValidateParams(spec, nil); err == nil {
		t.Error("ValidateParams(nil params) want error")
	}
}

// TestValidateParams_wrongType (AC-023): wrong param type is rejected.
func TestValidateParams_wrongType(t *testing.T) {
	spec := []ParamSpec{{Name: "a", Required: true, Type: "string"}}
	if err := ValidateParams(spec, map[string]any{"a": 123}); err == nil {
		t.Error("ValidateParams(wrong type) want error")
	}
}

type mockTool struct {
	name string
}

func (m *mockTool) Name() string              { return m.name }
func (m *mockTool) Description() string       { return "mock" }
func (m *mockTool) ParamsSchema() []ParamSpec { return nil }
func (m *mockTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if v, ok := params["fail"]; ok && v.(bool) {
		return "", context.DeadlineExceeded
	}
	return "ok", nil
}
