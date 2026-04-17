package config

import (
	"fmt"
	"pa/internal/runtimeskills"
	"pa/internal/sqlitepragma"
	"pa/internal/toolcatalog"
	"regexp"
	"strings"
	"time"
)

// ConfigFileName is the name of the main config file inside the config directory (PA_CONFIG_DIR).
const ConfigFileName = "config.json"

// Config holds application configuration loaded from JSON.
type Config struct {
	Version             int                        `json:"version"`
	Telegram            Telegram                   `json:"telegram"`
	LLMProviders        []LLMProvider              `json:"llm_providers"`
	Embedding           *EmbeddingProvider         `json:"embedding"` // optional; dedicated provider for vector memory embeddings
	Paths               Paths                      `json:"paths"`
	Nodes               map[string]Node            `json:"nodes"`
	LogRedaction        *LogRedaction              `json:"log_redaction"`        // required; additional_patterns may be empty (built-in patterns always applied)
	PATimezone          string                     `json:"pa_timezone"`          // required; IANA name for assistant's day (e.g. UTC, Europe/Moscow); used for summarization
	ConversationContext *ConversationContextConfig `json:"conversation_context"` // required; injected context limits (all fields must be >= 1)
	// ToolCatalog is the parsed tool catalog when paths.tool_catalog_path is set; nil otherwise. Populated at config load (fail fast on parse/schema error).
	ToolCatalog *toolcatalog.Catalog `json:"-"`
	// ToolPreSelection is required; all numeric fields must be >= 1 (no runtime defaults).
	ToolPreSelection *ToolPreSelection `json:"tool_pre_selection"`
	// Tools is required; use {"tools":{}} minimum. Prefer explicit text_based_enabled in JSON for clarity.
	Tools *ToolsConfig `json:"tools"`
	// CreateToolSecretRegex is compiled at Load from tools.create_tool_secret_patterns (REQ-09.017). Nil when absent or empty.
	CreateToolSecretRegex []*regexp.Regexp `json:"-"`
	// WebTools is optional; when non-nil and enabled, registers web_search and web_fetch (EP-011, REQ-11.001, REQ-11.002).
	WebTools *WebToolsConfig `json:"web_tools,omitempty"`
	// RuntimeSkills is optional; when enabled, loads skill packages from paths.skills_dir (EP-013).
	RuntimeSkills *RuntimeSkillsConfig `json:"runtime_skills,omitempty"`
	// RuntimeSkillPackages is populated at Load when runtime_skills.enabled (json "-").
	RuntimeSkillPackages []*runtimeskills.Package `json:"-"`
	// ConversationSession is optional; when enabled, keeps a sliding window of exchanges per session key (EP-014).
	ConversationSession *ConversationSessionConfig `json:"conversation_session,omitempty"`
	// ReadMemory is required; limits for native read_memory (EP-002). No load-time defaults.
	ReadMemory *ReadMemoryConfig `json:"read_memory"`
	// WriteMemory is required; limits for native write_memory (EP-016). No load-time defaults.
	WriteMemory *WriteMemoryConfig `json:"write_memory"`
	// IntentClassifier is optional; when enabled, classifies messages into complexity tiers before prompt assembly (EP-017).
	IntentClassifier *IntentClassifierConfig `json:"intent_classifier,omitempty"`
	// VectorStoreReliability is required; configures SQLite PRAGMA policy for the vector store (EP-022, REQ-22.001).
	VectorStoreReliability *SQLiteStoreReliabilityConfig `json:"vector_store_reliability"`
	// JobsStoreReliability is required; configures SQLite PRAGMA policy for the scheduled-jobs store (EP-022, REQ-22.002).
	JobsStoreReliability *SQLiteStoreReliabilityConfig `json:"jobs_store_reliability"`
}

// SQLiteStoreReliabilityConfig is the explicit per-store PRAGMA policy for Local SQLite Stores (EP-022).
// Every field must be set in JSON; the loader fails fast on missing or invalid values (no hidden defaults).
type SQLiteStoreReliabilityConfig struct {
	JournalMode string `json:"journal_mode"` // e.g. "WAL"; required
	BusyTimeout string `json:"busy_timeout"` // Go duration literal, e.g. "5s"; > 0 required
	Synchronous string `json:"synchronous"`  // e.g. "NORMAL"; required
	ForeignKeys *bool  `json:"foreign_keys"` // required; vector store must be false, jobs store must be true
}

// ToPolicy returns the sqlitepragma.Policy represented by this config block.
// Callers must only invoke this after config.Load has succeeded: Load enforces
// required fields, duration parseability, and non-nil foreign_keys. This
// method therefore panics on any invariant violation — the panic documents an
// unreachable code path when the config is obtained through the public Load
// entry point.
//
// Design deviation from ep-system-design.md: the config struct keeps
// `busy_timeout` and `http_timeout` as JSON strings (validated Go durations)
// rather than typed `time.Duration` fields. Reason: keeping the JSON
// boundary string-typed avoids a custom UnmarshalJSON on every duration
// field and preserves round-trip readability of `config.json` dumps. The
// parsed duration is re-derived at the few consumer sites through this
// method (stores) or `parseHTTPTimeout` helpers (llm/embedding).
func (c *SQLiteStoreReliabilityConfig) ToPolicy() sqlitepragma.Policy {
	if c == nil {
		panic("config: SQLiteStoreReliabilityConfig is nil (Load should have rejected this)")
	}
	if c.ForeignKeys == nil {
		panic("config: SQLiteStoreReliabilityConfig.foreign_keys is nil (Load should have rejected this)")
	}
	d, err := time.ParseDuration(strings.TrimSpace(c.BusyTimeout))
	if err != nil {
		panic(fmt.Sprintf("config: SQLiteStoreReliabilityConfig.busy_timeout %q not parseable (Load should have rejected this): %v", c.BusyTimeout, err))
	}
	return sqlitepragma.Policy{
		JournalMode: c.JournalMode,
		BusyTimeout: d,
		Synchronous: c.Synchronous,
		ForeignKeys: *c.ForeignKeys,
	}
}

// validateSQLiteStoreReliability validates a per-store reliability block. storeName is used in error messages.
// wantForeignKeys is the required value for foreign_keys (jobs store: true; vector store: false).
func validateSQLiteStoreReliability(store string, c *SQLiteStoreReliabilityConfig, wantForeignKeys bool) error {
	if c == nil {
		return fmt.Errorf("%s_store_reliability: block is required", store)
	}
	if strings.TrimSpace(c.JournalMode) == "" {
		return fmt.Errorf("%s_store_reliability.journal_mode: required", store)
	}
	if strings.TrimSpace(c.BusyTimeout) == "" {
		return fmt.Errorf("%s_store_reliability.busy_timeout: required", store)
	}
	d, err := time.ParseDuration(strings.TrimSpace(c.BusyTimeout))
	if err != nil {
		return fmt.Errorf("%s_store_reliability.busy_timeout: invalid duration %q: %w", store, c.BusyTimeout, err)
	}
	if d <= 0 {
		return fmt.Errorf("%s_store_reliability.busy_timeout: must be > 0, got %s", store, d)
	}
	if strings.TrimSpace(c.Synchronous) == "" {
		return fmt.Errorf("%s_store_reliability.synchronous: required", store)
	}
	if c.ForeignKeys == nil {
		return fmt.Errorf("%s_store_reliability.foreign_keys: required (true or false)", store)
	}
	if *c.ForeignKeys != wantForeignKeys {
		return fmt.Errorf("%s_store_reliability.foreign_keys: must be %t for %s store", store, wantForeignKeys, store)
	}
	return nil
}

// ValidateVectorStoreReliability validates the vector store reliability block (foreign_keys must be false).
func (c *Config) ValidateVectorStoreReliability() error {
	return validateSQLiteStoreReliability("vector", c.VectorStoreReliability, false)
}

// ValidateJobsStoreReliability validates the jobs store reliability block (foreign_keys must be true).
func (c *Config) ValidateJobsStoreReliability() error {
	return validateSQLiteStoreReliability("jobs", c.JobsStoreReliability, true)
}

// ReadMemoryConfig limits the native read_memory tool (EP-002). The tool is always registered when memory is configured.
type ReadMemoryConfig struct {
	MaxSpanDays    int `json:"max_span_days"`
	MaxOutputBytes int `json:"max_output_bytes"`
}

// WriteMemoryConfig limits the native write_memory tool (EP-016). All fields must be set in JSON.
type WriteMemoryConfig struct {
	MaxAppendBytes int `json:"max_append_bytes"`
	MaxFileBytes   int `json:"max_file_bytes"`
}

// ConversationSessionConfig enables in-process sliding session memory (EP-014, REQ-14.001).
type ConversationSessionConfig struct {
	Enabled             bool `json:"enabled"`
	MaxSessionExchanges int  `json:"max_session_exchanges"`
}

// LLMEscalationConfig enables tool-driven escalation along llm_providers order (REQ-06.002). JSON: tools.llm_escalation.
// When Enabled is true, MaxPerUserMessage must be >= 1 at load (zero would disable policy escalation and is rejected as a misconfiguration).
type LLMEscalationConfig struct {
	Enabled           bool `json:"enabled"`
	MaxPerUserMessage int  `json:"max_per_user_message"`
	BaselineIndex     int  `json:"baseline_index"`
}

// ToolsConfig holds optional tool-invocation settings (REQ-04.030) and tools.llm_escalation (EP-006).
type ToolsConfig struct {
	TextBasedEnabled bool `json:"text_based_enabled"`
	// AlwaysInclude lists catalog or allowed-native tool ids merged into every turn’s tool set (EP-013, REQ-13.011).
	AlwaysInclude []string `json:"always_include,omitempty"`
	// DynamicSelection is optional (EP-018). When present and enabled, max_tools_for_llm_request must be set in JSON (>= 1).
	DynamicSelection *ToolDynamicSelection `json:"dynamic_selection,omitempty"`
	// CreateToolSecretPatterns is optional; each entry is a Go regexp (RE2). Invalid regex fails config load (REQ-09.017).
	CreateToolSecretPatterns []string             `json:"create_tool_secret_patterns,omitempty"`
	LLMEscalation            *LLMEscalationConfig `json:"llm_escalation,omitempty"`
}

// ToolDynamicSelection configures EP-018 dynamic narrowing of the main LLM tool list.
// When Enabled is true, TierFull applies the cap after merge; TierFullLite applies it only if
// tools.text_based_enabled is also true (Hermes path).
type ToolDynamicSelection struct {
	Enabled               bool `json:"enabled"`
	MaxToolsForLLMRequest int  `json:"max_tools_for_llm_request"`
}

// ToolsLLMEscalation returns tools.llm_escalation for EP-006 (nil if tools section or escalation block absent).
func (c *Config) ToolsLLMEscalation() *LLMEscalationConfig {
	if c == nil || c.Tools == nil {
		return nil
	}
	return c.Tools.LLMEscalation
}

// ToolPreSelection holds parameters for tool pre-selection (REQ-04.019, REQ-04.020). Validated at load; no implicit defaults.
type ToolPreSelection struct {
	ToolSearchTopK  int `json:"tool_search_top_k"` // top-k from vector search (>= 1)
	ToolMinCount    int `json:"tool_min_count"`    // minimum tools from search before accepting; else fallback (>= 1)
	ToolFallbackCap int `json:"tool_fallback_cap"` // max tools when using fallback (sorted catalog ids) (>= 1)
}

// ConversationContextConfig holds parameters for context injected into the LLM (vector search results). All fields >= 1 at load.
type ConversationContextConfig struct {
	// MaxDynamicSystemRunes caps the dynamic tail of the system message (after trust/marker/personality): tool instructions, Hermes, retrieved memory, runtime skills (UTF-8 runes).
	MaxDynamicSystemRunes int `json:"max_dynamic_system_runes"`
	VectorSearchTopK      int `json:"vector_search_top_k"` // number of vector search results to consider (whole chunks; tail fit may drop some)
}

// LogRedaction holds additional redaction patterns (REQ-01.028). Built-in patterns cannot be overridden (REQ-01.027).
type LogRedaction struct {
	AdditionalPatterns []RedactionPattern `json:"additional_patterns"`
}

// RedactionPattern is one pattern (id, regex, replacement) for log redaction.
type RedactionPattern struct {
	ID          string `json:"id"`
	Regex       string `json:"regex"`
	Replacement string `json:"replacement"`
}

// IntentClassifierConfig holds EP-017 intent classification settings.
type IntentClassifierConfig struct {
	Enabled    bool                       `json:"enabled"`
	Heuristic  *HeuristicConfig           `json:"heuristic,omitempty"`
	ModelStage *ClassificationModelConfig `json:"model_stage,omitempty"`
}

// HeuristicConfig defines patterns for the heuristic classification stage (EP-017, EP-018).
type HeuristicConfig struct {
	SimplePatterns   []string `json:"simple_patterns"`
	FullPatterns     []string `json:"full_patterns"`
	FullLitePatterns []string `json:"full_lite_patterns,omitempty"`
	MaxSimpleLen     int      `json:"max_simple_len"`
}

// ClassificationModelConfig holds the cheap-model classification stage settings (EP-017).
type ClassificationModelConfig struct {
	Enabled            bool    `json:"enabled"`
	Type               string  `json:"type"`
	Endpoint           string  `json:"endpoint"`
	APIKeyPath         string  `json:"api_key_path"`
	Model              string  `json:"model"`
	DefaultTemperature float64 `json:"default_temperature"`
	DefaultMaxTokens   int     `json:"default_max_tokens"`
	Timeout            string  `json:"timeout,omitempty"`
}

// EmbeddingProvider is the dedicated provider for vector store embeddings (separate from chat LLM).
type EmbeddingProvider struct {
	Type       string `json:"type"`     // e.g. "openai", "openai-compatible", "ollama"
	Endpoint   string `json:"endpoint"` // base URL for embeddings API
	APIKeyPath string `json:"api_key_path"`
	Model      string `json:"model"`      // embedding model name
	Dimensions int    `json:"dimensions"` // embedding vector size; must match model output
	BatchSize  int    `json:"batch_size"` // required; max texts per batch for tool index embedding (REQ-04.021); must be 1–1000
	// HTTPTimeout is the total per-request timeout for outbound embedding calls (EP-022, REQ-22.004).
	// Required in config.json (Go duration literal, e.g. "60s"). When the struct is constructed directly
	// in tests without going through config.Load, an empty value falls back to a documented reference.
	HTTPTimeout string `json:"http_timeout"`
}

// Telegram holds Telegram bot configuration.
type Telegram struct {
	TokenPath        string `json:"token_path"`
	UsersPath        string `json:"users_path"`
	MaxMessageLength int    `json:"max_message_length"` // max message length in runes; 0 = no limit; over-length messages rejected
	NotifyChatID     int64  `json:"notify_chat_id"`     // optional; default target chat for operator-facing notifications
}

// LLMProvider holds one LLM provider configuration (order = priority).
// Model, endpoint, type, supports_tools, default_temperature, default_max_tokens (>= 1),
// supports_json_mode, and default_response_format are required at load (fail fast; no runtime defaults for those).
// api_key_path is required for openai / openai-compatible; optional for ollama.
// SupportsTools: when false, HTTP requests omit tools (REQ-04.026).
type LLMProvider struct {
	Type                  string  `json:"type"`
	Endpoint              string  `json:"endpoint"`
	APIKeyPath            string  `json:"api_key_path"`
	Model                 string  `json:"model"`
	SupportsTools         *bool   `json:"supports_tools"`
	DefaultTemperature    float64 `json:"default_temperature"`     // required; provider default for completion requests
	DefaultMaxTokens      int     `json:"default_max_tokens"`      // required; provider default for completion requests (>= 1)
	SupportsJSONMode      bool    `json:"supports_json_mode"`      // required; when true, provider supports response_format: json_object
	DefaultResponseFormat string  `json:"default_response_format"` // required; "text" or "json_object"
	// HTTPTimeout is the total per-request timeout for outbound LLM calls (EP-022, REQ-22.003).
	// Required in config.json (Go duration literal, e.g. "120s"). When the struct is constructed
	// directly in tests, an empty value falls back to a documented reference.
	HTTPTimeout string `json:"http_timeout"`
}

// Paths holds paths for memory, logs, vector index, scheduled jobs, and optional tool catalog.
type Paths struct {
	MemoryDir           string `json:"memory_dir"`
	LogPath             string `json:"log_path"`
	VectorIndexPath     string `json:"vector_index_path"`
	LLMLogDir           string `json:"llm_log_dir"`
	LLMLogRetentionDays int    `json:"llm_log_retention_days"` // Required. Delete llm-YYYY-MM-DD.jsonl older than N days (UTC). Must be >= 1; validated at load (fail fast).
	JobsDBPath          string `json:"jobs_db_path"`           // Required. SQLite path for scheduled jobs runtime (EP-019).
	SSHKnownHostsPath   string `json:"ssh_known_hosts_path"`   // Required when nodes are configured. OpenSSH known_hosts file for host key verification.
	ToolCatalogPath     string `json:"tool_catalog_path"`      // Required. Path to tool catalog YAML file; catalog is loaded at startup (fail fast on parse/schema error).
	SkillsDir           string `json:"skills_dir,omitempty"`   // Optional. Runtime skill packages root (subdirectory per skill); required when runtime_skills.enabled.
}

// Node holds SSH node configuration.
type Node struct {
	Host                 string   `json:"host"`
	Port                 int      `json:"port"` // 0 = default 22
	DedicatedUser        string   `json:"dedicated_user"`
	Auth                 NodeAuth `json:"auth"`
	CommandAllowlistPath string   `json:"command_allowlist_path"`
}

// NodeAuth holds node authentication (private key path).
type NodeAuth struct {
	PrivateKeyPath string `json:"private_key_path"`
}

// TelegramUser holds one entry from the Telegram users file (user_id, role, name).
type TelegramUser struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
	Name   string `json:"name"`
}
