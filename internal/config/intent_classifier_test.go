package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// Covers AC-17.009
func TestValidateIntentClassifier_ModelStageConfig(t *testing.T) {
	cfg := &Config{
		IntentClassifier: &IntentClassifierConfig{
			Enabled: true,
			Heuristic: &HeuristicConfig{
				MaxSimpleLen: 40,
			},
			ModelStage: &ClassificationModelConfig{
				Enabled:          true,
				Type:             "openai-compatible",
				Endpoint:         "http://localhost:11434/v1",
				Model:            "gemma3:1b",
				DefaultMaxTokens: 10,
				Timeout:          "5s",
			},
		},
	}
	if err := validateIntentClassifier(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Supporting AC-17.009
func TestValidateIntentClassifier_ModelStage_BadTimeout(t *testing.T) {
	cfg := &Config{
		IntentClassifier: &IntentClassifierConfig{
			Enabled: true,
			ModelStage: &ClassificationModelConfig{
				Enabled:          true,
				Endpoint:         "http://localhost:11434/v1",
				Model:            "test",
				DefaultMaxTokens: 10,
				Timeout:          "not-a-duration",
			},
		},
	}
	if err := validateIntentClassifier(cfg); err == nil {
		t.Fatal("expected error for bad timeout")
	}
}

// Supporting AC-17.009
func TestValidateIntentClassifier_ModelStage_MissingEndpoint(t *testing.T) {
	cfg := &Config{
		IntentClassifier: &IntentClassifierConfig{
			Enabled: true,
			ModelStage: &ClassificationModelConfig{
				Enabled:          true,
				Model:            "test",
				DefaultMaxTokens: 10,
			},
		},
	}
	if err := validateIntentClassifier(cfg); err == nil {
		t.Fatal("expected error for missing endpoint")
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
