package logredact

import (
	"fmt"
	"regexp"
	"sync"
)

const replacement = "[REDACTED]"

// Built-in pattern identifiers. Additional config patterns must not use these ids (REQ-01.028, REQ-01.029).
const (
	BuiltInIDOpenAIKey     = "api_key_openai"
	BuiltInIDTelegramToken = "telegram_bot_token"
	BuiltInIDBearerToken   = "bearer_token"
	BuiltInIDSecretPath    = "generic_secret_path"
)

// builtInPatterns are fixed patterns applied in code; config cannot override or disable them (REQ-01.027).
var builtInPatterns = []struct {
	id    string
	regex *regexp.Regexp
}{
	{BuiltInIDOpenAIKey, regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`)},
	{BuiltInIDTelegramToken, regexp.MustCompile(`\d{8,10}:[a-zA-Z0-9_-]{35}`)},
	{BuiltInIDBearerToken, regexp.MustCompile(`(?i)Bearer\s+[^\s]+`)},
	{BuiltInIDSecretPath, regexp.MustCompile(`/[\w/.-]*(?:token|secret|key|credential|password)(?:s)?[\w/.-]*`)},
}

// BuiltInIDs returns the list of built-in pattern identifiers for validation (REQ-01.029).
func BuiltInIDs() []string {
	ids := make([]string, len(builtInPatterns))
	for i, p := range builtInPatterns {
		ids[i] = p.id
	}
	return ids
}

// Pattern holds one additional redaction pattern (id, regex, replacement).
type Pattern struct {
	ID          string
	Regex       string
	Replacement string
}

// Redact applies all built-in patterns and then additional patterns to s and returns the redacted string (REQ-01.026).
func Redact(s string, additional []Pattern) string {
	out := s
	for _, p := range builtInPatterns {
		out = p.regex.ReplaceAllString(out, replacement)
	}
	for _, p := range additional {
		repl := p.Replacement
		if repl == "" {
			repl = replacement
		}
		if re := compiledAdditional(p.Regex); re != nil {
			out = re.ReplaceAllString(out, repl)
		}
	}
	return out
}

var (
	additionalCache   = make(map[string]*regexp.Regexp)
	additionalCacheMu sync.RWMutex
)

// compiledAdditional compiles and caches regex for additional patterns (used at redact time after ValidateConfig has run).
func compiledAdditional(expr string) *regexp.Regexp {
	additionalCacheMu.RLock()
	re := additionalCache[expr]
	additionalCacheMu.RUnlock()
	if re != nil {
		return re
	}
	additionalCacheMu.Lock()
	defer additionalCacheMu.Unlock()
	if re = additionalCache[expr]; re != nil {
		return re
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil
	}
	additionalCache[expr] = re
	return re
}

// ValidateConfig checks that no additional pattern id is reserved and all regexes compile (REQ-01.029).
// Returns an error with a clear message such as "log_redaction: reserved pattern id 'api_key_openai'"
// or "log_redaction: invalid regex in pattern 'foo': ...".
func ValidateConfig(builtInIDs []string, additional []Pattern) error {
	builtInSet := make(map[string]bool)
	for _, id := range builtInIDs {
		builtInSet[id] = true
	}
	for _, p := range additional {
		if builtInSet[p.ID] {
			return fmt.Errorf("log_redaction: reserved pattern id %q", p.ID)
		}
		if _, err := regexp.Compile(p.Regex); err != nil {
			return fmt.Errorf("log_redaction: invalid regex in pattern %q: %w", p.ID, err)
		}
	}
	return nil
}

// NewRedactor returns a function that redacts a string using built-in and the given additional patterns.
func NewRedactor(additional []Pattern) func(string) string {
	return func(s string) string {
		return Redact(s, additional)
	}
}
