package config

// ConfigFileName is the name of the main config file inside the config directory (PA_CONFIG_DIR).
const ConfigFileName = "config.json"

// Config holds application configuration loaded from JSON.
type Config struct {
	Version      int                `json:"version"`
	Telegram     Telegram           `json:"telegram"`
	LLMProviders []LLMProvider      `json:"llm_providers"`
	Embedding    *EmbeddingProvider `json:"embedding"` // optional; dedicated provider for vector memory embeddings
	Paths        Paths              `json:"paths"`
	Nodes        map[string]Node    `json:"nodes"`
	LogRedaction *LogRedaction      `json:"log_redaction"` // optional; additional redaction patterns (built-in are always applied)
	PATimezone   string             `json:"pa_timezone"`   // optional; IANA timezone for assistant's day (e.g. Europe/Moscow); used for summarization date; empty = UTC
}

// LogRedaction holds optional additional redaction patterns (REQ-028). Built-in patterns cannot be overridden (REQ-027).
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
}

// Telegram holds Telegram bot configuration.
type Telegram struct {
	TokenPath        string `json:"token_path"`
	UsersPath        string `json:"users_path"`
	MaxMessageLength int    `json:"max_message_length"` // max message length in runes; 0 = no limit; over-length messages rejected
	NotifyChatID     int64  `json:"notify_chat_id"`     // optional; chat ID for scheduler "notify" action (e.g. first admin); 0 = use first allowed user
}

// LLMProvider holds one LLM provider configuration (order = priority).
type LLMProvider struct {
	Type       string `json:"type"`
	Endpoint   string `json:"endpoint"`
	APIKeyPath string `json:"api_key_path"`
	Model      string `json:"model"`
}

// Paths holds paths for memory, logs, vector index, and scheduled tasks.
type Paths struct {
	MemoryDir           string `json:"memory_dir"`
	LogPath             string `json:"log_path"`
	VectorIndexPath     string `json:"vector_index_path"`
	LLMLogDir           string `json:"llm_log_dir"`
	LLMLogRetentionDays int    `json:"llm_log_retention_days"` // Required. Delete llm-YYYY-MM-DD.jsonl older than N days (UTC). Must be >= 1; validated at load (fail fast).
	ScheduledTasksPath  string `json:"scheduled_tasks_path"`
	SSHKnownHostsPath   string `json:"ssh_known_hosts_path"` // Required when nodes are configured. OpenSSH known_hosts file for host key verification.
}

// Node holds SSH node configuration.
type Node struct {
	Host                 string   `json:"host"`
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
