package config

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// configRootJSONKeys is the exhaustive set of top-level keys allowed in config.json.
// Every key must appear exactly once; optional product blocks use JSON null when disabled.
// Keep sorted for stable diffs and error messages.
var configRootJSONKeys = []string{
	"conversation_context",
	"conversation_session",
	"embedding",
	"intent_classifier",
	"jobs_store_reliability",
	"llm_providers",
	"log_redaction",
	"nodes",
	"observability_http",
	"pa_timezone",
	"paths",
	"read_memory",
	"runtime_skills",
	"sqlite_store_defaults",
	"telegram",
	"tools",
	"vector_store_reliability",
	"version",
	"web_tools",
	"write_memory",
}

func validateConfigRootObjectKeys(data []byte) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("config: root must be a JSON object: %w", err)
	}
	for _, want := range configRootJSONKeys {
		if _, ok := root[want]; !ok {
			return fmt.Errorf("config: missing required top-level key %q (every product root key must appear in config.json; use null for disabled optional blocks)", want)
		}
	}
	for got := range root {
		if !slices.Contains(configRootJSONKeys, got) {
			return fmt.Errorf("config: unknown top-level key %q (only documented root keys are allowed)", got)
		}
	}
	return nil
}

// ConfigRootJSONKeys returns a copy of the required root key list (for tests and tooling).
func ConfigRootJSONKeys() []string {
	return slices.Clone(configRootJSONKeys)
}

// ExplainConfigRootKeysForDocs returns a comma-separated list for operator documentation.
func ExplainConfigRootKeysForDocs() string {
	return strings.Join(configRootJSONKeys, ", ")
}
