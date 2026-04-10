//go:build integration

package integration_test

import (
	"os"
	"pa/internal/config"
	"pa/internal/promptmarkers"
	"path/filepath"
	"strings"
	"testing"
)

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyRuntimeSkillsMinimalFixture(t *testing.T, destDir string) {
	t.Helper()
	base := filepath.Join("testdata", "runtime_skills", "minimal_ok")
	if err := os.MkdirAll(filepath.Join(destDir, "skills", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, filepath.Join(base, "config.json"), filepath.Join(destDir, "config.json"))
	copyFile(t, filepath.Join(base, "tools.yaml"), filepath.Join(destDir, "tools.yaml"))
	copyFile(t, filepath.Join(base, "skills", "demo", "SKILL.md"), filepath.Join(destDir, "skills", "demo", "SKILL.md"))
}

// Covers runtime_skills load path: config.Load populates RuntimeSkillPackages from paths.skills_dir.
func TestRuntimeSkills_configLoad_populatesPackages(t *testing.T) {
	t.Parallel()
	cfgPath := filepath.Join("testdata", "runtime_skills", "minimal_ok", "config.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RuntimeSkills == nil || !cfg.RuntimeSkills.Enabled {
		t.Fatal("expected runtime_skills.enabled")
	}
	pkgs := cfg.RuntimeSkillPackages
	if len(pkgs) != 1 {
		t.Fatalf("RuntimeSkillPackages: len=%d, want 1", len(pkgs))
	}
	if pkgs[0].ID != "demo" {
		t.Fatalf("package ID = %q, want demo", pkgs[0].ID)
	}
	body := pkgs[0].Body
	if !strings.Contains(body, "DEMO_SKILL_LOAD_OK") {
		t.Fatalf("unexpected skill body: %q", body)
	}
}

// SKILL.md references a tool id missing from the catalog: Load must fail (ValidateToolRefs).
func TestRuntimeSkills_configLoad_rejectsUnknownToolInSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	copyRuntimeSkillsMinimalFixture(t, dir)
	skillDir := filepath.Join(dir, "skills", "bad")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	badSkill := `---
name: Bad Skill
description: skill with invalid tool reference
tools:
  - definitely_not_in_catalog_xyz
---

Body.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(badSkill), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(filepath.Join(dir, "config.json"))
	if err == nil {
		t.Fatal("expected error for unknown tool in skill")
	}
	if !strings.Contains(err.Error(), "unknown tool id") {
		t.Fatalf("error = %v", err)
	}
}

// Forbidden PA marker line in SKILL.md must fail at package load.
func TestRuntimeSkills_configLoad_rejectsForbiddenMarkerInSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	copyRuntimeSkillsMinimalFixture(t, dir)
	skillDir := filepath.Join(dir, "skills", "marked")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	badSkill := "---\nname: X\ndescription: y\ntools: []\n---\n\nline before\n" + promptmarkers.BeginRetrievedContext + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(badSkill), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(filepath.Join(dir, "config.json"))
	if err == nil {
		t.Fatal("expected error for forbidden marker in SKILL.md")
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("error = %v", err)
	}
}

// runtime_skills.enabled without paths.skills_dir fails fast.
func TestRuntimeSkills_configLoad_requiresSkillsDirWhenEnabled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	base := filepath.Join("testdata", "runtime_skills", "minimal_ok")
	copyFile(t, filepath.Join(base, "config.json"), filepath.Join(dir, "config.json"))
	copyFile(t, filepath.Join(base, "tools.yaml"), filepath.Join(dir, "tools.yaml"))
	// Remove skills_dir from JSON: read and patch
	raw, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(raw), `"skills_dir": "skills",`, "", 1)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = config.Load(filepath.Join(dir, "config.json"))
	if err == nil {
		t.Fatal("expected error when skills_dir missing")
	}
	if !strings.Contains(err.Error(), "skills_dir") {
		t.Fatalf("error = %v", err)
	}
}
