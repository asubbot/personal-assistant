package config

import (
	"pa/internal/toolcatalog"
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
	LogRedaction        *LogRedaction              `json:"log_redaction"`        // optional; additional redaction patterns (built-in are always applied)
	PATimezone          string                     `json:"pa_timezone"`          // optional; IANA timezone for assistant's day (e.g. Europe/Moscow); used for summarization date; empty = UTC
	ConversationContext *ConversationContextConfig `json:"conversation_context"` // optional; injected context limits; zero = use defaults
	// ToolCatalog is the parsed tool catalog when paths.tool_catalog_path is set; nil otherwise. Populated at config load (fail fast on parse/schema error).
	ToolCatalog *toolcatalog.Catalog `json:"-"`
	// ToolPreSelection is optional; when present, values are validated (>= 0). Zero values mean use code defaults (topK=10, minTools=1, fallbackCap=50).
	ToolPreSelection *ToolPreSelection `json:"tool_pre_selection"`
	// Tools is optional; text_based_enabled defaults to false when omitted or section absent.
	Tools *ToolsConfig `json:"tools"`
	// LLMEscalation is optional; when enabled, core uses per-message provider escalation on qualifying tool failures (EP-006).
	LLMEscalation *LLMEscalationConfig `json:"llm_escalation"`
}

// LLMEscalationConfig enables tool-driven escalation along llm_providers order (REQ-06.002).
type LLMEscalationConfig struct {
	Enabled           bool `json:"enabled"`
	MaxPerUserMessage int  `json:"max_per_user_message"`
	BaselineIndex     int  `json:"baseline_index"`
}

// ToolsConfig holds optional tool-invocation settings (REQ-04.030).
type ToolsConfig struct {
	TextBasedEnabled bool `json:"text_based_enabled"`
}

// ToolPreSelection holds parameters for tool pre-selection (REQ-04.019, REQ-04.020). All zero = use code defaults.
type ToolPreSelection struct {
	ToolSearchTopK  int `json:"tool_search_top_k"` // top-k from vector search; 0 = 10
	ToolMinCount    int `json:"tool_min_count"`    // minimum tools; if result has fewer, use fallback; 0 = 1
	ToolFallbackCap int `json:"tool_fallback_cap"` // max tools when using fallback (sorted catalog ids); 0 = 50
}

// ConversationContextConfig holds parameters for context injected into the LLM (vector search results).
// Optional; zero values mean use defaults (4000 chars, top 10).
type ConversationContextConfig struct {
	InjectedContextMaxChars int `json:"injected_context_max_chars"` // max chars for vector+memory block injected into LLM; 0 = 4000
	VectorSearchTopK        int `json:"vector_search_top_k"`        // number of vector search results to inject; 0 = 10
}

// LogRedaction holds optional additional redaction patterns (REQ-01.028). Built-in patterns cannot be overridden (REQ-01.027).
type LogRedaction struct {
	AdditionalPatterns []RedactionPattern `json:"additional_patterns"`
}

// RedactionPattern is one pattern (id, regex, replacement) for log redaction.
type RedactionPattern struct {
	ID          string `json:"id"`
	Regex       string `json:"regex"`
	Replacement string `json:"replacement"`
}

// EmbeddingProvider is the dedicated provider for vector store embeddings (separate from chat LLM).
type EmbeddingProvider struct {
	Type       string `json:"type"`     // e.g. "openai", "openai-compatible", "ollama"
	Endpoint   string `json:"endpoint"` // base URL for embeddings API
	APIKeyPath string `json:"api_key_path"`
	Model      string `json:"model"`      // embedding model name
	Dimensions int    `json:"dimensions"` // embedding vector size; must match model output
	BatchSize  int    `json:"batch_size"` // required; max texts per batch for tool index embedding (REQ-04.021); must be 1–1000
}

// Telegram holds Telegram bot configuration.
type Telegram struct {
	TokenPath        string `json:"token_path"`
	UsersPath        string `json:"users_path"`
	MaxMessageLength int    `json:"max_message_length"` // max message length in runes; 0 = no limit; over-length messages rejected
	NotifyChatID     int64  `json:"notify_chat_id"`     // optional; chat ID for scheduler "notify" action (e.g. first admin); 0 = use first allowed user
}

// LLMProvider holds one LLM provider configuration (order = priority).
// SupportsTools is required (fail fast if missing); when false, HTTP requests omit tools (REQ-04.026).
type LLMProvider struct {
	Type          string `json:"type"`
	Endpoint      string `json:"endpoint"`
	APIKeyPath    string `json:"api_key_path"`
	Model         string `json:"model"`
	SupportsTools *bool  `json:"supports_tools"`
}

// Paths holds paths for memory, logs, vector index, scheduled tasks, and optional tool catalog.
type Paths struct {
	MemoryDir           string `json:"memory_dir"`
	LogPath             string `json:"log_path"`
	VectorIndexPath     string `json:"vector_index_path"`
	LLMLogDir           string `json:"llm_log_dir"`
	LLMLogRetentionDays int    `json:"llm_log_retention_days"` // Required. Delete llm-YYYY-MM-DD.jsonl older than N days (UTC). Must be >= 1; validated at load (fail fast).
	ScheduledTasksPath  string `json:"scheduled_tasks_path"`
	SSHKnownHostsPath   string `json:"ssh_known_hosts_path"` // Required when nodes are configured. OpenSSH known_hosts file for host key verification.
	ToolCatalogPath     string `json:"tool_catalog_path"`    // Required. Path to tool catalog YAML file; catalog is loaded at startup (fail fast on parse/schema error).
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
