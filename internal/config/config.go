package config

// Config holds application configuration loaded from JSON.
type Config struct {
	Version      int             `json:"version"`
	Telegram     Telegram        `json:"telegram"`
	LLMProviders []LLMProvider   `json:"llm_providers"`
	Paths        Paths           `json:"paths"`
	Nodes        map[string]Node `json:"nodes"`
}

// Telegram holds Telegram bot configuration.
type Telegram struct {
	TokenPath string `json:"token_path"`
	UsersPath string `json:"users_path"`
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
	MemoryDir          string `json:"memory_dir"`
	LogPath            string `json:"log_path"`
	VectorIndexPath    string `json:"vector_index_path"`
	LLMLogDir          string `json:"llm_log_dir"`
	ScheduledTasksPath string `json:"scheduled_tasks_path"`
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
