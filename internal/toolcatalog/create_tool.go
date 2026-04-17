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

// test hooks (non-nil only from tests in this package)
var (
	testRenameHook      func(oldpath, newpath string) error
	testPostMarshalHook func([]byte) []byte
	testOnTempDataSync  func()
	testOnDirSync       func()
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

func renameCatalog(oldpath, newpath string) error {
	if testRenameHook != nil {
		return testRenameHook(oldpath, newpath)
	}
	return os.Rename(oldpath, newpath)
}

func syncParentDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("toolcatalog: open dir for sync: %w", err)
	}
	if testOnDirSync != nil {
		testOnDirSync()
	}
	syncErr := f.Sync()
	closeErr := f.Close()
	if syncErr != nil {
		return fmt.Errorf("toolcatalog: sync dir: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("toolcatalog: close dir: %w", closeErr)
	}
	return nil
}

// atomicReplaceContent writes content to absPath using same-directory temp file, data sync, rename, then parent dir sync.
func atomicReplaceContent(absPath string, content []byte) error {
	dir := filepath.Dir(absPath)
	tmp, err := os.CreateTemp(dir, ".pa-tool-catalog-*.yaml")
	if err != nil {
		return fmt.Errorf("toolcatalog: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(content); err != nil {
		cleanup()
		_ = tmp.Close()
		return fmt.Errorf("toolcatalog: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		_ = tmp.Close()
		return fmt.Errorf("toolcatalog: sync temp data: %w", err)
	}
	if testOnTempDataSync != nil {
		testOnTempDataSync()
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("toolcatalog: close temp: %w", err)
	}
	if err := renameCatalog(tmpPath, absPath); err != nil {
		cleanup()
		return fmt.Errorf("toolcatalog: atomic replace: %w", err)
	}
	if err := syncParentDir(dir); err != nil {
		return err
	}
	return nil
}

// RestoreCatalogFile writes snapshot bytes to absPath using the same atomic replace path as create_tool catalog updates.
func RestoreCatalogFile(absPath string, snapshot []byte) error {
	if absPath == "" {
		return errors.New("toolcatalog: catalog path is empty")
	}
	return atomicReplaceContent(absPath, snapshot)
}

// AppendToolToCatalogFile reads YAML at absPath, appends one tool, replaces the file atomically with sync,
// validates with Load, and restores the pre-call bytes if validation fails.
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
	snapshot, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("toolcatalog: read catalog: %w", err)
	}
	var raw rawCatalog
	if err := yaml.Unmarshal(snapshot, &raw); err != nil {
		return fmt.Errorf("toolcatalog: parse catalog: %w", err)
	}
	raw.Tools = append(raw.Tools, *tool)

	out, err := yaml.Marshal(&raw)
	if err != nil {
		return fmt.Errorf("toolcatalog: marshal catalog: %w", err)
	}
	if testPostMarshalHook != nil {
		out = testPostMarshalHook(out)
	}
	if err := atomicReplaceContent(absPath, out); err != nil {
		return err
	}
	if _, err := Load(absPath); err != nil {
		if rerr := atomicReplaceContent(absPath, snapshot); rerr != nil {
			return fmt.Errorf("toolcatalog: post-write validate: %w (restore: %w)", err, rerr)
		}
		return fmt.Errorf("toolcatalog: post-write validate: %w", err)
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
