package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-36.013
func TestLoad_RejectRemovedIntentClassifier_model_stage(t *testing.T) {
	_, err := Load(loadConfigFixtureRaw(t, "intent_classifier_model_stage_rejected"))
	if err == nil {
		t.Fatal("Load: expected error for model_stage, got nil")
	}
	if !strings.Contains(err.Error(), "model_stage") {
		t.Fatalf("Load: error = %v, want model_stage mention", err)
	}
}

// Covers AC-36.014
func TestLoad_RejectRemovedIntentClassifier_full_lite_patterns(t *testing.T) {
	_, err := Load(loadConfigFixtureRaw(t, "intent_classifier_full_lite_patterns_rejected"))
	if err == nil {
		t.Fatal("Load: expected error for full_lite_patterns, got nil")
	}
	if !strings.Contains(err.Error(), "full_lite_patterns") {
		t.Fatalf("Load: error = %v, want full_lite_patterns mention", err)
	}
}

// Covers AC-36.015, AC-36.016
func TestLoad_IntentClassifier_enabledHeuristicOnly(t *testing.T) {
	cfg := loadConfigFixture(t, "intent_classifier_enabled_heuristic_only")
	if cfg.IntentClassifier == nil || !cfg.IntentClassifier.Enabled {
		t.Fatal("Load: expected enabled intent_classifier")
	}
	if cfg.IntentClassifier.Heuristic == nil {
		t.Fatal("Load: expected heuristic block")
	}
	if len(cfg.IntentClassifier.Heuristic.SimplePatterns) == 0 || len(cfg.IntentClassifier.Heuristic.FullPatterns) == 0 {
		t.Fatal("Load: expected simple and full patterns")
	}
	if cfg.IntentClassifier.Heuristic.MaxSimpleLen < 1 {
		t.Fatal("Load: max_simple_len must be >= 1")
	}
}

// Covers AC-36.015
func TestLoad_IntentClassifier_invalidRegexRejected(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFixtureToDir(t, dir, "intent_classifier_invalid_regex", nil)
	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected invalid regex error")
	}
}

// Covers AC-36.015
func TestLoad_IntentClassifier_maxSimpleLenBelowOneRejected(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFixtureToDir(t, dir, "intent_classifier_max_simple_len_zero", nil)
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "max_simple_len") {
		t.Fatalf("Load: want max_simple_len error, got %v", err)
	}
}

// Covers AC-36.016
func TestLoad_validWithToolCatalog_intentClassifierNull(t *testing.T) {
	cfg := loadConfigFixture(t, "valid_with_tool_catalog")
	if cfg.IntentClassifier != nil {
		t.Fatal("intent_classifier must be null")
	}
}

// Covers AC-36.022
func TestConfigExample_intentClassifierNull_noRemovedKeys(t *testing.T) {
	root := findRepoRootFromConfigPackage(t)
	data, err := os.ReadFile(filepath.Join(root, "config.examples", "config.example.json"))
	if err != nil {
		t.Fatalf("read config.example.json: %v", err)
	}
	if err := rejectRemovedUnsupportedConfigKeys(data); err != nil {
		t.Fatalf("config.example.json: %v", err)
	}
	var rootObj map[string]json.RawMessage
	if err := json.Unmarshal(data, &rootObj); err != nil {
		t.Fatalf("parse: %v", err)
	}
	rawIC, ok := rootObj["intent_classifier"]
	if !ok {
		t.Fatal("config.example.json: intent_classifier key must be present")
	}
	if string(rawIC) != "null" {
		t.Fatalf("config.example.json: intent_classifier = %s, want null", rawIC)
	}
}

// Covers AC-36.022
func TestLoad_IntentClassifierEnabledHeuristicOnly_testdata(t *testing.T) {
	TestLoad_IntentClassifier_enabledHeuristicOnly(t)
}

// Covers AC-17.015
func TestValidateIntentClassifier_PatternsConfigurable(t *testing.T) {
	cfg := &Config{
		IntentClassifier: &IntentClassifierConfig{
			Enabled: true,
			Heuristic: &HeuristicConfig{
				SimplePatterns: []string{`^hello$`, `^bye$`},
				FullPatterns:   []string{`(search|find)`},
				MaxSimpleLen:   50,
			},
		},
	}
	if err := validateIntentClassifier(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Covers AC-17.015
func TestValidateIntentClassifier_InvalidRegexFails(t *testing.T) {
	cfg := &Config{
		IntentClassifier: &IntentClassifierConfig{
			Enabled: true,
			Heuristic: &HeuristicConfig{
				SimplePatterns: []string{`[invalid`},
				MaxSimpleLen:   50,
			},
		},
	}
	if err := validateIntentClassifier(cfg); err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

// Covers AC-17.014
func TestValidateIntentClassifier_Disabled(t *testing.T) {
	cfg := &Config{
		IntentClassifier: &IntentClassifierConfig{
			Enabled: false,
		},
	}
	if err := validateIntentClassifier(cfg); err != nil {
		t.Fatalf("unexpected error for disabled classifier: %v", err)
	}
}

// Covers AC-17.014
func TestValidateIntentClassifier_Nil(t *testing.T) {
	cfg := &Config{}
	if err := validateIntentClassifier(cfg); err != nil {
		t.Fatalf("unexpected error for nil classifier: %v", err)
	}
}

// Covers AC-17.018
func TestDocs_configuration_mentionsIntentClassifier(t *testing.T) {
	repoRoot := findRepoRootFromConfigPackage(t)
	p := filepath.Join(repoRoot, "docs", "configuration.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	s := string(b)
	if !strings.Contains(s, "intent_classifier") {
		t.Errorf("%s must document intent_classifier", p)
	}
}
