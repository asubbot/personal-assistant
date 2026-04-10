package core

import (
	"pa/internal/config"
	"pa/internal/runtimeskills"
	"testing"
)

func TestMergeToolIDs_orderAlwaysSkillThenVector(t *testing.T) {
	tc := &config.ToolsConfig{AlwaysInclude: []string{"  always_t  ", ""}}
	rs := &config.RuntimeSkillsConfig{Enabled: true}
	skills := []*runtimeskills.Package{
		{ID: "s1", Tools: []string{"skill_t"}},
	}
	vec := []string{"vec_t"}
	ordered, src := mergeToolIDs(tc, rs, skills, vec)
	want := []string{"always_t", "skill_t", "vec_t"}
	if len(ordered) != len(want) {
		t.Fatalf("got %v", ordered)
	}
	for i, id := range want {
		if ordered[i] != id {
			t.Fatalf("ordered[%d] = %q, want %q (full %v)", i, ordered[i], id, ordered)
		}
	}
	if src["always_t"] != originAlways {
		t.Fatalf("always_t origin = %v", src["always_t"])
	}
	if src["skill_t"] != originSkill {
		t.Fatalf("skill_t origin = %v", src["skill_t"])
	}
	if src["vec_t"] != originVector {
		t.Fatalf("vec_t origin = %v", src["vec_t"])
	}
}

func TestMergeToolIDs_skillAndVectorSameID_unionsOrigin(t *testing.T) {
	rs := &config.RuntimeSkillsConfig{Enabled: true}
	skills := []*runtimeskills.Package{{ID: "s", Tools: []string{"dup"}}}
	vec := []string{"dup"}
	ordered, src := mergeToolIDs(nil, rs, skills, vec)
	if len(ordered) != 1 || ordered[0] != "dup" {
		t.Fatalf("got ordered %v", ordered)
	}
	wantOrig := originSkill | originVector
	if src["dup"] != wantOrig {
		t.Fatalf("origin = %v, want %v", src["dup"], wantOrig)
	}
}

func TestMergeToolIDs_runtimeDisabled_onlyVector(t *testing.T) {
	rs := &config.RuntimeSkillsConfig{Enabled: false}
	skills := []*runtimeskills.Package{{ID: "s", Tools: []string{"skill_t"}}}
	vec := []string{"vec_t"}
	ordered, src := mergeToolIDs(nil, rs, skills, vec)
	if len(ordered) != 1 || ordered[0] != "vec_t" {
		t.Fatalf("got %v", ordered)
	}
	if src["vec_t"] != originVector {
		t.Fatalf("origin = %v", src["vec_t"])
	}
}

func TestMergeToolIDs_toolsAlwaysIncludeWhenRuntimeDisabled(t *testing.T) {
	tc := &config.ToolsConfig{AlwaysInclude: []string{"pinned"}}
	rs := &config.RuntimeSkillsConfig{Enabled: false}
	ordered, src := mergeToolIDs(tc, rs, nil, []string{"vec_t"})
	if len(ordered) != 2 || ordered[0] != "pinned" || ordered[1] != "vec_t" {
		t.Fatalf("got %v", ordered)
	}
	if src["pinned"] != originAlways || src["vec_t"] != originVector {
		t.Fatalf("origins: %+v", src)
	}
}

func TestMergeToolIDs_nilConfig_onlyVector(t *testing.T) {
	ordered, src := mergeToolIDs(nil, nil, []*runtimeskills.Package{{Tools: []string{"t"}}}, []string{"v"})
	if len(ordered) != 1 || ordered[0] != "v" {
		t.Fatalf("got %v", ordered)
	}
	if src["v"] != originVector {
		t.Fatalf("origin = %v", src["v"])
	}
}

func TestTryRemoveToolStep4_prefersVectorFromEnd(t *testing.T) {
	st := &tailFitState{
		merged:  []string{"pin", "vec"},
		sources: map[string]toolOrigin{"pin": originAlways, "vec": originVector},
	}
	if !tryRemoveToolStep4(st) {
		t.Fatal("expected removal")
	}
	if len(st.merged) != 1 || st.merged[0] != "pin" {
		t.Fatalf("got %v", st.merged)
	}
}

func TestTryRemoveToolStep4_skillLinkedNotAlwaysBeforeAlwaysInclude(t *testing.T) {
	st := &tailFitState{
		merged: []string{"always_id", "skill_id"},
		sources: map[string]toolOrigin{
			"always_id": originAlways,
			"skill_id":  originSkill,
		},
	}
	if !tryRemoveToolStep4(st) {
		t.Fatal("expected removal")
	}
	if len(st.merged) != 1 || st.merged[0] != "always_id" {
		t.Fatalf("got %v", st.merged)
	}
}

func TestTryRemoveToolStep4_alwaysIncludeLast(t *testing.T) {
	st := &tailFitState{
		merged: []string{"skill_only", "always_id"},
		sources: map[string]toolOrigin{
			"always_id":  originAlways,
			"skill_only": originSkill,
		},
	}
	if !tryRemoveToolStep4(st) {
		t.Fatal("expected removal of skill_only first")
	}
	if len(st.merged) != 1 || st.merged[0] != "always_id" {
		t.Fatalf("got %v", st.merged)
	}
	if !tryRemoveToolStep4(st) {
		t.Fatal("expected removal of always")
	}
	if len(st.merged) != 0 {
		t.Fatalf("got %v", st.merged)
	}
}
