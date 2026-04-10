package core

import (
	"pa/internal/config"
	"pa/internal/runtimeskills"
	"pa/internal/toolcatalog"
	"strings"
	"testing"
)

func TestTrimSkillPackagesByBudget_dropsLowerRank(t *testing.T) {
	// Covers AC-13.014
	a := &runtimeskills.Package{ID: "a", Name: "A", Description: "d", Body: "aaaa"}
	b := &runtimeskills.Package{ID: "b", Name: "B", Description: "d", Body: "bbbbbbbb"}
	// PlaybookText is "# Name\n\n" + body; budget must fit first skill but not both.
	out := trimSkillPackagesByBudget([]*runtimeskills.Package{a, b}, 16)
	if len(out) != 1 || out[0].ID != "a" {
		t.Fatalf("got %+v", out)
	}
}

func TestTrimSkillPackagesByBudget_singleOversizedYieldsNone(t *testing.T) {
	// Covers AC-13.014, REQ-13.012 (single skill over cap)
	p := &runtimeskills.Package{ID: "x", Name: "X", Description: "d", Body: "toobig"}
	out := trimSkillPackagesByBudget([]*runtimeskills.Package{p}, 3)
	if len(out) != 0 {
		t.Fatalf("got %+v, want empty", out)
	}
}

func TestMergeToolIDs_orderAlwaysSkillThenVector(t *testing.T) {
	rs := &config.RuntimeSkillsConfig{
		Enabled:       true,
		AlwaysInclude: []string{"  always_t  ", ""},
	}
	skills := []*runtimeskills.Package{
		{ID: "s1", Tools: []string{"skill_t"}},
	}
	vec := []string{"vec_t"}
	ordered, src := mergeToolIDs(rs, skills, vec)
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
	ordered, src := mergeToolIDs(rs, skills, vec)
	if len(ordered) != 1 || ordered[0] != "dup" {
		t.Fatalf("got ordered %v", ordered)
	}
	wantOrig := originSkill | originVector
	if src["dup"] != wantOrig {
		t.Fatalf("origin = %v, want %v", src["dup"], wantOrig)
	}
}

func TestMergeToolIDs_runtimeDisabled_onlyVector(t *testing.T) {
	rs := &config.RuntimeSkillsConfig{Enabled: false, AlwaysInclude: []string{"x"}}
	skills := []*runtimeskills.Package{{ID: "s", Tools: []string{"skill_t"}}}
	vec := []string{"vec_t"}
	ordered, src := mergeToolIDs(rs, skills, vec)
	if len(ordered) != 1 || ordered[0] != "vec_t" {
		t.Fatalf("got %v", ordered)
	}
	if src["vec_t"] != originVector {
		t.Fatalf("origin = %v", src["vec_t"])
	}
}

func TestMergeToolIDs_nilConfig_onlyVector(t *testing.T) {
	ordered, src := mergeToolIDs(nil, []*runtimeskills.Package{{Tools: []string{"t"}}}, []string{"v"})
	if len(ordered) != 1 || ordered[0] != "v" {
		t.Fatalf("got %v", ordered)
	}
	if src["v"] != originVector {
		t.Fatalf("origin = %v", src["v"])
	}
}

func TestTrimToolIDsForInstructionBudget_dropsVectorOnlyFromEnd(t *testing.T) {
	long := strings.Repeat("p", 120)
	short := strings.Repeat("v", 120)
	cat := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"pin": {
				ID: "pin", IndexText: "P", SystemPrompt: long,
				Template: "echo x", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
			"vec": {
				ID: "vec", IndexText: "V", SystemPrompt: short,
				Template: "echo y", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	ids := []string{"pin", "vec"}
	sources := map[string]toolOrigin{
		"pin": originAlways,
		"vec": originVector,
	}
	// Copy sources: trim mutates map and slice.
	srcCopy := cloneToolOrigins(sources)
	out := trimToolIDsForInstructionBudget(cat, append([]string(nil), ids...), srcCopy, 200)
	if len(out) != 1 || out[0] != "pin" {
		t.Fatalf("got %v, want [pin]", out)
	}
}

func TestTrimToolIDsForInstructionBudget_stopsWhenNoVectorLeft(t *testing.T) {
	huge := strings.Repeat("x", 500)
	cat := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"pin": {
				ID: "pin", IndexText: "P", SystemPrompt: huge,
				Template: "echo x", NodeID: "nas",
				Arguments: []toolcatalog.ArgumentRule{{Name: "msg", Type: "string", Required: true}},
			},
		},
	}
	sources := map[string]toolOrigin{"pin": originAlways}
	out := trimToolIDsForInstructionBudget(cat, []string{"pin"}, cloneToolOrigins(sources), 50)
	if len(out) != 1 || out[0] != "pin" {
		t.Fatalf("got %v, want [pin]", out)
	}
}

func cloneToolOrigins(m map[string]toolOrigin) map[string]toolOrigin {
	o := make(map[string]toolOrigin, len(m))
	for k, v := range m {
		o[k] = v
	}
	return o
}
