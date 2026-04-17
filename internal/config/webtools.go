package config

import (
	"errors"
	"fmt"
	"strings"
)

// WebToolsConfig holds optional native web_search / web_fetch settings (EP-011, REQ-11.019).
// When nil or Enabled is false, native web tools are not registered.
type WebToolsConfig struct {
	Enabled bool            `json:"enabled"`
	Search  WebSearchConfig `json:"search"`
	Fetch   WebFetchConfig  `json:"fetch"`
	// HTTPTimeout is the per-request total timeout for the shared outbound
	// *http.Client used by web_search upstream calls and web_fetch (EP-022,
	// REQ-22.008). Required Go duration literal, e.g. "30s". Fail-fast at
	// config.Load when web_tools.enabled is true.
	HTTPTimeout string `json:"http_timeout"`
}

// WebSearchConfig configures web_search (EP-011).
type WebSearchConfig struct {
	Provider         string `json:"provider"`                    // primary: "brave" or "duckduckgo"
	FallbackProvider string `json:"fallback_provider,omitempty"` // optional second provider, must differ from primary
	BraveAPIKeyPath  string `json:"brave_api_key_path"`          // required when primary or fallback is brave
	TimeoutSeconds   int    `json:"timeout_seconds"`             // upstream HTTP call (each attempt)
	CacheTTLSeconds  int    `json:"cache_ttl_seconds"`           // TTL for in-memory cache
	CacheMaxEntries  int    `json:"cache_max_entries"`           // LRU cap
}

// WebFetchConfig configures web_fetch (EP-011).
type WebFetchConfig struct {
	TimeoutSeconds int   `json:"timeout_seconds"` // full operation including redirects
	MaxBodyBytes   int64 `json:"max_body_bytes"`  // max bytes read from body
	MaxRedirects   int   `json:"max_redirects"`   // max redirect hops
}

func parseWebSearchProvider(raw, field string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "brave", "duckduckgo":
		return s, nil
	case "":
		return "", fmt.Errorf("config: web_tools.search.%s is empty", field)
	default:
		return "", fmt.Errorf("config: web_tools.search.%s must be brave or duckduckgo, got %q", field, raw)
	}
}

func validateWebTools(c *Config) error {
	if c == nil || c.WebTools == nil || !c.WebTools.Enabled {
		return nil
	}
	w := c.WebTools
	if err := validateWebToolsNumericBounds(w); err != nil {
		return err
	}
	if err := validateHTTPTimeout("web_tools.http_timeout", w.HTTPTimeout); err != nil {
		return err
	}
	return validateWebSearchProviders(w)
}

func validateWebSearchProviders(w *WebToolsConfig) error {
	primary, err := parseWebSearchProvider(w.Search.Provider, "provider")
	if err != nil {
		return err
	}
	var fallback string
	if strings.TrimSpace(w.Search.FallbackProvider) != "" {
		fallback, err = parseWebSearchProvider(w.Search.FallbackProvider, "fallback_provider")
		if err != nil {
			return err
		}
		if fallback == primary {
			return errors.New("config: web_tools.search.fallback_provider must differ from provider")
		}
	}
	if primary == "brave" || fallback == "brave" {
		if strings.TrimSpace(w.Search.BraveAPIKeyPath) == "" {
			return errors.New("config: web_tools.search.brave_api_key_path is required when brave is primary or fallback search provider")
		}
	}
	return nil
}

func validateWebToolsNumericBounds(w *WebToolsConfig) error {
	switch {
	case w.Search.TimeoutSeconds < 1:
		return errors.New("config: web_tools.search.timeout_seconds must be >= 1 when web_tools.enabled")
	case w.Search.CacheTTLSeconds < 1:
		return errors.New("config: web_tools.search.cache_ttl_seconds must be >= 1 when web_tools.enabled")
	case w.Search.CacheMaxEntries < 1:
		return errors.New("config: web_tools.search.cache_max_entries must be >= 1 when web_tools.enabled")
	case w.Fetch.TimeoutSeconds < 1:
		return errors.New("config: web_tools.fetch.timeout_seconds must be >= 1 when web_tools.enabled")
	case w.Fetch.MaxBodyBytes < 1:
		return errors.New("config: web_tools.fetch.max_body_bytes must be >= 1 when web_tools.enabled")
	case w.Fetch.MaxRedirects < 0:
		return errors.New("config: web_tools.fetch.max_redirects must be >= 0 when web_tools.enabled")
	default:
		return nil
	}
}
