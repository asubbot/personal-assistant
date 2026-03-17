package tools

import (
	"context"
	"fmt"
)

// ParamSpec describes one parameter for tool input validation (AC-01.023).
type ParamSpec struct {
	Name     string
	Required bool
	Type     string // "string", "number", "boolean"
}

// Tool is the extensible tool contract (REQ-01.010): name, description, params schema, Run (AC-01.022, AC-01.023).
type Tool interface {
	Name() string
	Description() string
	ParamsSchema() []ParamSpec
	Run(ctx context.Context, params map[string]any) (result string, err error)
}

// ValidateParams checks params against spec: required keys present, types match. Returns error if invalid.
func ValidateParams(spec []ParamSpec, params map[string]any) error {
	if params == nil {
		params = make(map[string]any)
	}
	for _, p := range spec {
		v, ok := params[p.Name]
		if !ok || v == nil {
			if p.Required {
				return fmt.Errorf("tools: missing required param %q", p.Name)
			}
			continue
		}
		switch p.Type {
		case "string":
			if _, ok := v.(string); !ok {
				return fmt.Errorf("tools: param %q must be string", p.Name)
			}
		case "number":
			switch v.(type) {
			case float64, int, int64:
			default:
				return fmt.Errorf("tools: param %q must be number", p.Name)
			}
		case "boolean":
			if _, ok := v.(bool); !ok {
				return fmt.Errorf("tools: param %q must be boolean", p.Name)
			}
		default:
			return fmt.Errorf("tools: unknown param type %q for %q", p.Type, p.Name)
		}
	}
	return nil
}

// Registry holds tools by name; built at startup, read-only after init (REQ-01.010, REQ-01.011).
type Registry struct {
	m map[string]Tool
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{m: make(map[string]Tool)}
}

// Register adds a tool by its name. Panics if name is empty or already registered.
func (r *Registry) Register(t Tool) {
	name := t.Name()
	if name == "" {
		panic("tools: tool name is empty")
	}
	if _, ok := r.m[name]; ok {
		panic("tools: tool already registered: " + name)
	}
	r.m[name] = t
}

// Get returns the tool by name and true, or nil and false if not found.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.m[name]
	return t, ok
}

// List returns all registered tool names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.m))
	for name := range r.m {
		names = append(names, name)
	}
	return names
}
