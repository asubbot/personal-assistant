package logredact

import (
	"testing"
)

// Covers AC-038 (US-16): built-in redaction patterns applied to output.
func TestRedact_builtInPatterns(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"openai_key", "key is sk-abc123def456ghi789jkl012", "key is [REDACTED]"},
		{"telegram_token", "bot 1234567890:AAHxYz-abcdefghijklmnopqrstuvwxyz12", "bot [REDACTED]"},
		{"bearer_token", "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", "Authorization: [REDACTED]"},
		{"bearer_lower", "authorization: bearer some-secret-token", "authorization: [REDACTED]"},
		{"secret_path", "config at /etc/app/token file", "config at [REDACTED] file"},
		{"no_match", "hello world", "hello world"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Redact(tt.in, nil)
			if got != tt.want {
				t.Errorf("Redact() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Covers AC-040 (US-16): additional patterns applied when configured.
func TestRedact_additionalPatterns(t *testing.T) {
	additional := []Pattern{
		{ID: "custom", Regex: `\bsecret-\d+\b`, Replacement: "[CUSTOM]"},
	}
	in := "user secret-42 and secret-99"
	got := Redact(in, additional)
	want := "user [CUSTOM] and [CUSTOM]"
	if got != want {
		t.Errorf("Redact() with additional = %q, want %q", got, want)
	}
}

// Covers AC-040 (US-16): built-in and additional patterns both applied; no duplicate built-in id.
func TestRedact_builtInAndAdditional(t *testing.T) {
	additional := []Pattern{
		{ID: "custom", Regex: `\bMYKEY-\w+\b`, Replacement: "[MYKEY]"},
	}
	// Built-in sk- pattern requires 20+ chars; additional redacts MYKEY-xyz
	in := "sk-abc123def456ghi789jkl012 and MYKEY-xyz"
	got := Redact(in, additional)
	if got != "[REDACTED] and [MYKEY]" {
		t.Errorf("Redact() = %q", got)
	}
}

// Covers AC-041 (US-16): additional pattern id equals built-in → refuse start, clear error.
func TestValidateConfig_reservedID(t *testing.T) {
	builtIn := BuiltInIDs()
	additional := []Pattern{{ID: BuiltInIDOpenAIKey, Regex: `x`, Replacement: ""}}
	err := ValidateConfig(builtIn, additional)
	if err == nil {
		t.Fatal("ValidateConfig(reserved id): want error")
	}
	if err.Error() != `log_redaction: reserved pattern id "api_key_openai"` {
		t.Errorf("ValidateConfig error = %q", err.Error())
	}
}

// Covers AC-041 (US-16): invalid regex in additional pattern → refuse start, clear error.
func TestValidateConfig_invalidRegex(t *testing.T) {
	additional := []Pattern{{ID: "foo", Regex: `[invalid`, Replacement: ""}}
	err := ValidateConfig(BuiltInIDs(), additional)
	if err == nil {
		t.Fatal("ValidateConfig(invalid regex): want error")
	}
	if err.Error() == "" {
		t.Error("ValidateConfig: error message should mention pattern id and invalid regex")
	}
}

// Covers AC-039, AC-040 (US-16): valid additional_patterns config accepted; patterns applied.
func TestValidateConfig_valid(t *testing.T) {
	additional := []Pattern{
		{ID: "custom", Regex: `\d+`, Replacement: "#"},
	}
	err := ValidateConfig(BuiltInIDs(), additional)
	if err != nil {
		t.Errorf("ValidateConfig(valid) = %v", err)
	}
}

// Covers AC-039 (US-16): no log_redaction or empty additional_patterns → only built-in used, start succeeds.
func TestValidateConfig_emptyAdditional(t *testing.T) {
	err := ValidateConfig(BuiltInIDs(), nil)
	if err != nil {
		t.Errorf("ValidateConfig(nil) = %v", err)
	}
	err = ValidateConfig(BuiltInIDs(), []Pattern{})
	if err != nil {
		t.Errorf("ValidateConfig(empty) = %v", err)
	}
}

// Supporting AC-038 (US-16): NewRedactor applies patterns to written output.
func TestNewRedactor(t *testing.T) {
	redactor := NewRedactor([]Pattern{{ID: "x", Regex: `sk-[\w]+`, Replacement: "[X]"}})
	got := redactor("key sk-abc123")
	if got != "key [X]" {
		t.Errorf("NewRedactor() = %q", got)
	}
}

// Covers AC-038 (US-16): built-in pattern identifiers defined in code.
func TestBuiltInIDs(t *testing.T) {
	ids := BuiltInIDs()
	if len(ids) < 4 {
		t.Errorf("BuiltInIDs() has %d elements, want at least 4", len(ids))
	}
	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("BuiltInIDs() duplicate %q", id)
		}
		seen[id] = true
	}
	for _, want := range []string{BuiltInIDOpenAIKey, BuiltInIDTelegramToken, BuiltInIDBearerToken, BuiltInIDSecretPath} {
		if !seen[want] {
			t.Errorf("BuiltInIDs() missing %q", want)
		}
	}
}
