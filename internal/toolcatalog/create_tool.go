package toolcatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Whitelisted template prefixes for create_tool (REQ-09.009).
const (
	createToolPrefixBridge = "docker run --rm --network bridge"
	createToolPrefixNone   = "docker run --rm --network none"
)

// ValidateCreateToolTemplatePrefix returns an error if template does not start with an allowed docker run prefix.
func ValidateCreateToolTemplatePrefix(template string) error {
	s := strings.TrimSpace(template)
	if s == "" {
		return errors.New("toolcatalog: template is empty")
	}
	if strings.HasPrefix(s, createToolPrefixBridge) || strings.HasPrefix(s, createToolPrefixNone) {
		return nil
	}
	return fmt.Errorf("toolcatalog: template must start with %q or %q", createToolPrefixBridge, createToolPrefixNone)
}

// ValidateSandboxResourceSubstrings enforces a 30s timeout bound in the template string (REQ-09.004).
// Memory and CPU limits (--memory=256m, --cpus=0.5) are not substring-validated here; operators SHOULD add them in templates for production sandboxes.
func ValidateSandboxResourceSubstrings(template string) error {
	if !strings.Contains(template, "timeout 30s") && !strings.Contains(template, "timeout 30 ") {
		return errors.New("toolcatalog: template must include a 30s timeout (e.g. timeout 30s before docker or in shell)")
	}
	return nil
}

// AppendToolToCatalogFile reads YAML at absPath, appends one tool, and replaces the file atomically (same dir temp + rename).
func AppendToolToCatalogFile(absPath string, tool *Tool) error {
	if absPath == "" {
		return errors.New("toolcatalog: catalog path is empty")
	}
	if tool == nil {
		return errors.New("toolcatalog: tool is nil")
	}
	if err := validateTool(tool, 0); err != nil {
		return err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("toolcatalog: read catalog: %w", err)
	}
	var raw rawCatalog
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("toolcatalog: parse catalog: %w", err)
	}
	raw.Tools = append(raw.Tools, *tool)

	out, err := yaml.Marshal(&raw)
	if err != nil {
		return fmt.Errorf("toolcatalog: marshal catalog: %w", err)
	}
	dir := filepath.Dir(absPath)
	tmp, err := os.CreateTemp(dir, ".pa-tool-catalog-*.yaml")
	if err != nil {
		return fmt.Errorf("toolcatalog: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_, werr := tmp.Write(out)
	if cerr := tmp.Close(); cerr != nil && werr == nil {
		werr = cerr
	}
	if werr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("toolcatalog: write temp: %w", werr)
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("toolcatalog: atomic replace: %w", err)
	}
	return nil
}

// ParseArgumentRulesFromCreateToolParams decodes the optional "arguments" value from create_tool params (JSON array or JSON string of array).
func ParseArgumentRulesFromCreateToolParams(v any) ([]ArgumentRule, error) {
	if v == nil {
		return nil, nil
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil, nil
		}
		var rules []ArgumentRule
		if err := json.Unmarshal([]byte(s), &rules); err != nil {
			return nil, fmt.Errorf("toolcatalog: arguments string: %w", err)
		}
		return rules, nil
	case []any:
		b, err := json.Marshal(x)
		if err != nil {
			return nil, err
		}
		var rules []ArgumentRule
		if err := json.Unmarshal(b, &rules); err != nil {
			return nil, fmt.Errorf("toolcatalog: arguments array: %w", err)
		}
		return rules, nil
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return nil, err
		}
		var rules []ArgumentRule
		if err := json.Unmarshal(b, &rules); err != nil {
			return nil, fmt.Errorf("toolcatalog: arguments: %w", err)
		}
		return rules, nil
	}
}
