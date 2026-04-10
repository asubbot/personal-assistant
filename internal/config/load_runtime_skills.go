package config

import (
	"fmt"
	"os"
	"pa/internal/runtimeskills"
	"strings"
)

//nolint:gocyclo // validation pipeline for runtime skills paths, packages, and tool refs
func finalizeRuntimeSkills(c *Config) error {
	if c == nil || c.RuntimeSkills == nil || !c.RuntimeSkills.Enabled {
		return nil
	}
	rs := c.RuntimeSkills
	applyRuntimeSkillsDefaults(rs)
	if err := validateRuntimeSkillsNumbers(rs); err != nil {
		return err
	}
	dir := strings.TrimSpace(c.Paths.SkillsDir)
	if dir == "" {
		return fmt.Errorf("runtime_skills.enabled requires paths.skills_dir")
	}
	st, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("paths.skills_dir %s: %w", dir, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("paths.skills_dir %s: not a directory", dir)
	}
	pkgs, err := runtimeskills.LoadDir(dir)
	if err != nil {
		return fmt.Errorf("runtime skills: %w", err)
	}
	if err := runtimeskills.ValidateToolRefs(pkgs, c.ToolCatalog, AllowedNativeToolIDs(c)); err != nil {
		return err
	}
	for _, id := range rs.AlwaysInclude {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := c.ToolCatalog.Tools[id]; ok {
			continue
		}
		if NativeToolAllowed(c, id) {
			continue
		}
		return fmt.Errorf("runtime_skills.always_include: unknown tool id %q", id)
	}
	c.RuntimeSkillPackages = pkgs
	return nil
}

func applyRuntimeSkillsDefaults(rs *RuntimeSkillsConfig) {
	if rs.MaxSkillsPerTurn < 1 {
		rs.MaxSkillsPerTurn = 2
	}
	if rs.ToolVectorTopKCap < 1 {
		rs.ToolVectorTopKCap = 20
	}
	if rs.MaxSkillRunesPerTurn < 1 {
		rs.MaxSkillRunesPerTurn = 32000
	}
	if rs.MaxToolInstructionRunesPerTurn < 1 {
		rs.MaxToolInstructionRunesPerTurn = 200000
	}
}

func validateRuntimeSkillsNumbers(rs *RuntimeSkillsConfig) error {
	if rs.MaxSkillsPerTurn > 50 {
		return fmt.Errorf("runtime_skills.max_skills_per_turn must be <= 50")
	}
	if rs.ToolVectorTopKCap > 500 {
		return fmt.Errorf("runtime_skills.tool_vector_top_k_cap must be <= 500")
	}
	if rs.MaxSkillRunesPerTurn > 10_000_000 {
		return fmt.Errorf("runtime_skills.max_skill_runes_per_turn too large")
	}
	if rs.MaxToolInstructionRunesPerTurn > 10_000_000 {
		return fmt.Errorf("runtime_skills.max_tool_instruction_runes_per_turn too large")
	}
	return nil
}
