package tools

import (
	"context"
	"testing"
)

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

func TestRegistry_RegisterEmptyName_panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register(empty name) did not panic")
		}
	}()
	r := NewRegistry()
	r.Register(&mockTool{name: ""})
}

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

func TestValidateParams_missingRequired(t *testing.T) {
	spec := []ParamSpec{{Name: "a", Required: true, Type: "string"}}
	if err := ValidateParams(spec, map[string]any{}); err == nil {
		t.Error("ValidateParams(missing required) want error")
	}
	if err := ValidateParams(spec, nil); err == nil {
		t.Error("ValidateParams(nil params) want error")
	}
}

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
