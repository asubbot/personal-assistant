package config

import (
	"fmt"
	"os"
	"pa/internal/runtimeskills"
	"strings"
)

func finalizeRuntimeSkills(c *Config) error {
	if c == nil || c.RuntimeSkills == nil || !c.RuntimeSkills.Enabled {
		return nil
	}
	rs := c.RuntimeSkills
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
	c.RuntimeSkillPackages = pkgs
	return nil
}

func validateRuntimeSkillsNumbers(rs *RuntimeSkillsConfig) error {
	if rs.MaxSkillsPerTurn > 50 {
		return fmt.Errorf("runtime_skills.max_skills_per_turn must be <= 50")
	}
	if rs.ToolVectorTopKCap > 500 {
		return fmt.Errorf("runtime_skills.tool_vector_top_k_cap must be <= 500")
	}
	return nil
}
