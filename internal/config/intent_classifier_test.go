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
				HTTPTimeout:      "30s",
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
				HTTPTimeout:      "30s",
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

// Covers AC-22.003, AC-22.007, AC-22.008: intent_classifier.model_stage.http_timeout
// must be a required, parseable, strictly positive Go duration when model_stage.enabled,
// mirroring the fail-fast contract the rest of EP-022 applies to llm/embedding/web_tools.
// Regression guard for the startup crash where buildIntentClassifier produced a
// config.LLMProvider with an empty HTTPTimeout.
func TestValidateIntentClassifier_ModelStage_HTTPTimeout_requiredAndPositive(t *testing.T) {
	base := func() *Config {
		return &Config{
			IntentClassifier: &IntentClassifierConfig{
				Enabled: true,
				ModelStage: &ClassificationModelConfig{
					Enabled:          true,
					Type:             "openai-compatible",
					Endpoint:         "http://localhost:11434/v1",
					Model:            "gemma3:1b",
					DefaultMaxTokens: 10,
				},
			},
		}
	}

	t.Run("empty is rejected", func(t *testing.T) {
		cfg := base()
		cfg.IntentClassifier.ModelStage.HTTPTimeout = ""
		err := validateIntentClassifier(cfg)
		if err == nil || !strings.Contains(err.Error(), "intent_classifier.model_stage.http_timeout") {
			t.Fatalf("empty http_timeout: want field-qualified error, got %v", err)
		}
	})
	t.Run("invalid duration is rejected", func(t *testing.T) {
		cfg := base()
		cfg.IntentClassifier.ModelStage.HTTPTimeout = "not-a-duration"
		err := validateIntentClassifier(cfg)
		if err == nil || !strings.Contains(err.Error(), "intent_classifier.model_stage.http_timeout") {
			t.Fatalf("invalid http_timeout: want field-qualified error, got %v", err)
		}
	})
	t.Run("zero is rejected", func(t *testing.T) {
		cfg := base()
		cfg.IntentClassifier.ModelStage.HTTPTimeout = "0s"
		err := validateIntentClassifier(cfg)
		if err == nil || !strings.Contains(err.Error(), "intent_classifier.model_stage.http_timeout") {
			t.Fatalf("zero http_timeout: want field-qualified error, got %v", err)
		}
	})
	t.Run("negative is rejected", func(t *testing.T) {
		cfg := base()
		cfg.IntentClassifier.ModelStage.HTTPTimeout = "-1s"
		err := validateIntentClassifier(cfg)
		if err == nil || !strings.Contains(err.Error(), "intent_classifier.model_stage.http_timeout") {
			t.Fatalf("negative http_timeout: want field-qualified error, got %v", err)
		}
	})
	t.Run("valid duration accepted", func(t *testing.T) {
		cfg := base()
		cfg.IntentClassifier.ModelStage.HTTPTimeout = "30s"
		if err := validateIntentClassifier(cfg); err != nil {
			t.Fatalf("valid http_timeout: unexpected error %v", err)
		}
	})
	t.Run("independent of application timeout", func(t *testing.T) {
		cfg := base()
		cfg.IntentClassifier.ModelStage.HTTPTimeout = "30s"
		cfg.IntentClassifier.ModelStage.Timeout = "" // application-level timeout may remain unset
		if err := validateIntentClassifier(cfg); err != nil {
			t.Fatalf("valid http_timeout with empty app timeout: unexpected error %v", err)
		}
	})
	t.Run("not enforced when model_stage disabled", func(t *testing.T) {
		cfg := base()
		cfg.IntentClassifier.ModelStage.Enabled = false
		cfg.IntentClassifier.ModelStage.HTTPTimeout = ""
		if err := validateIntentClassifier(cfg); err != nil {
			t.Fatalf("disabled model_stage: unexpected error %v", err)
		}
	})
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
