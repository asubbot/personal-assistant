package toolcatalog

import (
	"os"
	"path/filepath"
	"testing"
)

// Covers AC-09.009: allowed docker run prefixes pass validation.
func TestValidateCreateToolTemplatePrefix_acceptsBridgeAndNone(t *testing.T) {
	t.Parallel()
	if err := ValidateCreateToolTemplatePrefix(`docker run --rm --network bridge img`); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCreateToolTemplatePrefix(`docker run --rm --network none img`); err != nil {
		t.Fatal(err)
	}
}

// Covers AC-09.009: invalid prefix rejected.
func TestValidateCreateToolTemplatePrefix_rejectsBadPrefix(t *testing.T) {
	t.Parallel()
	if err := ValidateCreateToolTemplatePrefix("docker run alpine"); err == nil {
		t.Fatal("expected error")
	}
}

// Covers AC-09.002–004: resource substring validation.
func TestValidateSandboxResourceSubstrings(t *testing.T) {
	t.Parallel()
	good := `docker run --rm --network bridge pa-sandbox:base timeout 30s echo ok`
	if err := ValidateSandboxResourceSubstrings(good); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSandboxResourceSubstrings(`docker run --rm --network bridge`); err == nil {
		t.Fatal("expected error for missing 30s timeout")
	}
	// Alternative accepted form: "timeout 30 " (space after 30).
	if err := ValidateSandboxResourceSubstrings(`docker run --rm --network bridge pa-sandbox:base timeout 30 curl -fsS example.com`); err != nil {
		t.Fatal(err)
	}
}

// Covers AC-09.011, AC-23.001, AC-23.003: append twice yields valid YAML with two tools; durable replace path.
func TestAppendToolToCatalogFile_twice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.yaml")
	initial := []byte("tools: []\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}
	t1 := &Tool{ID: "a", IndexText: "A", Template: "echo a", NodeID: "n"}
	t2 := &Tool{ID: "b", IndexText: "B", Template: "echo b", NodeID: "n"}
	if err := AppendToolToCatalogFile(path, t1); err != nil {
		t.Fatal(err)
	}
	if err := AppendToolToCatalogFile(path, t2); err != nil {
		t.Fatal(err)
	}
	cat, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Tools) != 2 {
		t.Fatalf("tools = %d", len(cat.Tools))
	}
}

// Covers AC-09.008: parse arguments from JSON array in params.
func TestParseArgumentRulesFromCreateToolParams_array(t *testing.T) {
	t.Parallel()
	raw := []any{map[string]any{"name": "x", "type": "string", "required": true}}
	rules, err := ParseArgumentRulesFromCreateToolParams(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Name != "x" {
		t.Fatalf("%+v", rules)
	}
}

// Covers AC-09.008: parse arguments from JSON string.
func TestParseArgumentRulesFromCreateToolParams_string(t *testing.T) {
	t.Parallel()
	s := `[{"name":"k","type":"string","required":false}]`
	rules, err := ParseArgumentRulesFromCreateToolParams(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Name != "k" {
		t.Fatalf("%+v", rules)
	}
}
