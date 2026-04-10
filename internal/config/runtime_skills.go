package config

import (
	"slices"
)

// RuntimeSkillsConfig enables runtime SKILL.md packages and tool-union policy (EP-013).
type RuntimeSkillsConfig struct {
	Enabled                        bool     `json:"enabled"`
	MaxSkillsPerTurn               int      `json:"max_skills_per_turn"`
	ToolVectorTopKCap              int      `json:"tool_vector_top_k_cap"`
	MaxSkillRunesPerTurn           int      `json:"max_skill_runes_per_turn"`
	MaxToolInstructionRunesPerTurn int      `json:"max_tool_instruction_runes_per_turn"`
	AlwaysInclude                  []string `json:"always_include"`
}

// AllowedNativeToolIDs returns tool ids that may appear in skills or always_include without a catalog row.
func AllowedNativeToolIDs(c *Config) []string {
	if c == nil {
		return []string{"run_on_node", "create_tool"}
	}
	out := []string{"run_on_node", "create_tool"}
	if c.WebTools != nil && c.WebTools.Enabled {
		out = append(out, "web_search", "web_fetch")
	}
	return out
}

// NativeToolAllowed reports whether id is an allowed native tool for skill references.
func NativeToolAllowed(c *Config, id string) bool {
	return slices.Contains(AllowedNativeToolIDs(c), id)
}
