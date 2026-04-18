package config

import (
	"strings"
	"testing"
)

// Covers AC-11.001 — web_tools validation when enabled (REQ-11.019, REQ-11.020).
func TestValidateWebTools_disabled_NoError(t *testing.T) {
	c := &Config{WebTools: nil}
	if err := validateWebTools(c); err != nil {
		t.Fatalf("validateWebTools(nil): %v", err)
	}
	c2 := &Config{WebTools: &WebToolsConfig{Enabled: false}}
	if err := validateWebTools(c2); err != nil {
		t.Fatalf("validateWebTools(disabled): %v", err)
	}
}

// Covers AC-11.001 — invalid numeric bounds rejected at validation.
func TestValidateWebTools_enabled_InvalidBounds(t *testing.T) {
	c := &Config{WebTools: &WebToolsConfig{
		Enabled: true,
		Search: WebSearchConfig{
			Provider:        "duckduckgo",
			TimeoutSeconds:  0,
			CacheTTLSeconds: 60,
			CacheMaxEntries: 10,
		},
		Fetch:       WebFetchConfig{TimeoutSeconds: 30, MaxBodyBytes: 1024, MaxRedirects: 3},
		HTTPTimeout: "30s",
	}}
	if err := validateWebTools(c); err == nil || !strings.Contains(err.Error(), "timeout_seconds") {
		t.Fatalf("want timeout_seconds error, got %v", err)
	}
}

// Covers AC-11.001 — brave without key path fails.
func TestValidateWebTools_braveMissingKeyPath(t *testing.T) {
	c := &Config{WebTools: &WebToolsConfig{
		Enabled: true,
		Search: WebSearchConfig{
			Provider:        "brave",
			BraveAPIKeyPath: "",
			TimeoutSeconds:  10, CacheTTLSeconds: 60, CacheMaxEntries: 10,
		},
		Fetch:       WebFetchConfig{TimeoutSeconds: 30, MaxBodyBytes: 1024, MaxRedirects: 3},
		HTTPTimeout: "30s",
	}}
	if err := validateWebTools(c); err == nil || !strings.Contains(err.Error(), "brave_api_key_path") {
		t.Fatalf("want brave_api_key_path error, got %v", err)
	}
}

// Covers AC-11.001 — duckduckgo valid minimal.
func TestValidateWebTools_duckduckgo_OK(t *testing.T) {
	c := &Config{WebTools: &WebToolsConfig{
		Enabled: true,
		Search: WebSearchConfig{
			Provider:       "duckduckgo",
			TimeoutSeconds: 10, CacheTTLSeconds: 60, CacheMaxEntries: 10,
		},
		Fetch:       WebFetchConfig{TimeoutSeconds: 30, MaxBodyBytes: 1024, MaxRedirects: 3},
		HTTPTimeout: "30s",
	}}
	if err := validateWebTools(c); err != nil {
		t.Fatal(err)
	}
}

// Covers AC-11.016 — fallback must differ from primary.
func TestValidateWebTools_fallbackSameAsPrimary(t *testing.T) {
	c := &Config{WebTools: &WebToolsConfig{
		Enabled: true,
		Search: WebSearchConfig{
			Provider:         "duckduckgo",
			FallbackProvider: "duckduckgo",
			TimeoutSeconds:   10, CacheTTLSeconds: 60, CacheMaxEntries: 10,
		},
		Fetch:       WebFetchConfig{TimeoutSeconds: 30, MaxBodyBytes: 1024, MaxRedirects: 3},
		HTTPTimeout: "30s",
	}}
	if err := validateWebTools(c); err == nil || !strings.Contains(err.Error(), "fallback_provider") {
		t.Fatalf("want fallback_provider error, got %v", err)
	}
}

// Covers AC-11.016 — brave as fallback requires API key path.
func TestValidateWebTools_ddgPrimaryBraveFallback_missingKeyPath(t *testing.T) {
	c := &Config{WebTools: &WebToolsConfig{
		Enabled: true,
		Search: WebSearchConfig{
			Provider:         "duckduckgo",
			FallbackProvider: "brave",
			BraveAPIKeyPath:  "",
			TimeoutSeconds:   10, CacheTTLSeconds: 60, CacheMaxEntries: 10,
		},
		Fetch:       WebFetchConfig{TimeoutSeconds: 30, MaxBodyBytes: 1024, MaxRedirects: 3},
		HTTPTimeout: "30s",
	}}
	if err := validateWebTools(c); err == nil || !strings.Contains(err.Error(), "brave_api_key_path") {
		t.Fatalf("want brave_api_key_path error, got %v", err)
	}
}

// Covers AC-22.006, AC-22.008 (EP-022): web_tools.http_timeout must be present and a positive Go duration.
func TestValidateWebTools_httpTimeout_requiredAndPositive(t *testing.T) {
	base := &Config{WebTools: &WebToolsConfig{
		Enabled: true,
		Search: WebSearchConfig{
			Provider:       "duckduckgo",
			TimeoutSeconds: 10, CacheTTLSeconds: 60, CacheMaxEntries: 10,
		},
		Fetch: WebFetchConfig{TimeoutSeconds: 30, MaxBodyBytes: 1024, MaxRedirects: 3},
	}}

	base.WebTools.HTTPTimeout = ""
	if err := validateWebTools(base); err == nil || !strings.Contains(err.Error(), "http_timeout") {
		t.Fatalf("empty http_timeout: want http_timeout error, got %v", err)
	}
	base.WebTools.HTTPTimeout = "not-a-duration"
	if err := validateWebTools(base); err == nil || !strings.Contains(err.Error(), "http_timeout") {
		t.Fatalf("invalid http_timeout: want http_timeout error, got %v", err)
	}
	base.WebTools.HTTPTimeout = "0s"
	if err := validateWebTools(base); err == nil || !strings.Contains(err.Error(), "http_timeout") {
		t.Fatalf("zero http_timeout: want http_timeout error, got %v", err)
	}
	base.WebTools.HTTPTimeout = "30s"
	if err := validateWebTools(base); err != nil {
		t.Fatalf("valid http_timeout: unexpected error %v", err)
	}
}
