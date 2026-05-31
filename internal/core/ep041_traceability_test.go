package core

import (
	"context"
	"log/slog"
	"pa/internal/llm"
	"reflect"
	"testing"
)

// Covers AC-41.002 (REQ-41.006): five pipeline steps in documented fixed order.
func TestEP041_fullTierStepOrderConstants(t *testing.T) {
	want := []int{
		fullTierStepSelectSkills,
		fullTierStepMergeTools,
		fullTierStepApplyDynamicCap,
		fullTierStepFitTailBudget,
		fullTierStepBuildCompletionOptions,
	}
	for i, step := range want {
		if step != i+1 {
			t.Fatalf("step index %d = %d, want %d", i, step, i+1)
		}
	}
}

// Covers AC-41.001 (REQ-41.003): mergeTailMergedToolsAndOptions remains a thin delegate.
func TestEP041_mergeTailMergedToolsAndOptions_delegate(t *testing.T) {
	ctx := context.Background()
	h := testHandlerDeps{logger: slog.New(slog.DiscardHandler)}.handler()
	sysHead := "system-head"
	msgs := []llm.Message{{Role: "system", Content: sysHead}, {Role: "user", Content: "hello"}}
	if _, err := h.mergeTailMergedToolsAndOptions(ctx, "hello", sysHead, nil, nil, msgs); err != nil {
		t.Fatalf("mergeTailMergedToolsAndOptions: %v", err)
	}
}

// Covers AC-41.001 (REQ-41.001): fullTierAssembler type holds handler and turn inputs.
func TestEP041_fullTierAssemblerType(t *testing.T) {
	rt := reflect.TypeOf(fullTierAssembler{})
	if rt.Kind() != reflect.Struct {
		t.Fatalf("fullTierAssembler kind = %v, want struct", rt.Kind())
	}
	for _, name := range []string{"h", "ctx", "userText", "sysHead", "chunks", "messages"} {
		if _, ok := rt.FieldByName(name); !ok {
			t.Fatalf("fullTierAssembler missing field %q", name)
		}
	}
}
