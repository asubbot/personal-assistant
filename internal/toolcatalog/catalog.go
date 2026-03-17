// Package toolcatalog implements loading and validation of the tool catalog (single source of truth for invocable tools). REQ-04.001, REQ-04.002, REQ-04.003.
package toolcatalog

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Catalog holds the parsed tool catalog: tools by id. Parsed at startup; used for LLM payload and for validation/execution.
type Catalog struct {
	Tools map[string]*Tool // key = tool id
}

// Tool is one invocable tool: id, template, node_id, argument rules, optional triggers.
type Tool struct {
	ID               string         `yaml:"id"`
	ShortDescription string         `yaml:"short_description"`
	Template         string         `yaml:"template"`
	NodeID           string         `yaml:"node_id"`
	Arguments        []ArgumentRule `yaml:"arguments"`
	Triggers         []string       `yaml:"triggers"` // optional; example phrases for pre-selection
}

// ArgumentRule defines one argument: name, type, required, allowed_values, pattern, min, max.
type ArgumentRule struct {
	Name          string   `yaml:"name"`
	Type          string   `yaml:"type"` // e.g. string, integer
	Required      bool     `yaml:"required"`
	AllowedValues []string `yaml:"allowed_values"`
	Pattern       string   `yaml:"pattern"`
	Min           *int     `yaml:"min"`
	Max           *int     `yaml:"max"`
}

// rawCatalog is the YAML shape: list of tools.
type rawCatalog struct {
	Tools []Tool `yaml:"tools"`
}

// Load reads and validates the tool catalog from path (YAML). Returns error on parse or schema failure (fail fast).
func Load(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tool catalog %s: %w", path, err)
	}

	var raw rawCatalog
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("tool catalog %s: parse: %w", path, err)
	}

	c := &Catalog{Tools: make(map[string]*Tool)}
	for i := range raw.Tools {
		t := &raw.Tools[i]
		if err := validateTool(t, i); err != nil {
			return nil, fmt.Errorf("tool catalog %s: %w", path, err)
		}
		if _, exists := c.Tools[t.ID]; exists {
			return nil, fmt.Errorf("tool catalog %s: duplicate tool id %q", path, t.ID)
		}
		c.Tools[t.ID] = t
	}
	return c, nil
}

func validateTool(t *Tool, index int) error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("tools[%d]: id is required", index)
	}
	if strings.TrimSpace(t.ShortDescription) == "" {
		return fmt.Errorf("tools[%d]: short_description is required", index)
	}
	if strings.TrimSpace(t.Template) == "" {
		return fmt.Errorf("tools[%d]: template is required", index)
	}
	if strings.TrimSpace(t.NodeID) == "" {
		return fmt.Errorf("tools[%d]: node_id is required", index)
	}
	return nil
}
