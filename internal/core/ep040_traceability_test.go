package core

import (
	"reflect"
	"testing"
)

// Covers AC-40.001
func TestEP040_conversationHandlerGroupedDeps(t *testing.T) {
	rt := reflect.TypeOf(conversationHandler{})
	if rt.NumField() != 5 {
		t.Fatalf("conversationHandler field count = %d, want 5 (four groups + toolResultPromptBytes)", rt.NumField())
	}
	ep040AssertGroupedField(t, rt, 0, "tools", reflect.TypeOf(handlerToolDeps{}))
	ep040AssertGroupedField(t, rt, 1, "memory", reflect.TypeOf(handlerMemoryDeps{}))
	ep040AssertGroupedField(t, rt, 2, "session", reflect.TypeOf(handlerSessionDeps{}))
	ep040AssertGroupedField(t, rt, 3, "llm", reflect.TypeOf(handlerLLMDeps{}))
	f, ok := rt.FieldByName("toolResultPromptBytes")
	if !ok {
		t.Fatal("conversationHandler missing field toolResultPromptBytes")
	}
	if f.Type.Kind() != reflect.Int {
		t.Fatalf("toolResultPromptBytes kind = %v, want int", f.Type.Kind())
	}
}

func ep040AssertGroupedField(t *testing.T, rt reflect.Type, idx int, name string, want reflect.Type) {
	t.Helper()
	f := rt.Field(idx)
	if f.Name != name {
		t.Fatalf("field %d name = %q, want %q", idx, f.Name, name)
	}
	if f.Type != want {
		t.Fatalf("%s field type = %v, want %v", name, f.Type, want)
	}
}
