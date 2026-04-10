package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePaths updates path fields in cfg by joining relative paths with the appropriate base.
// Three bases (Variant A): config dir from configFilePath, PA_DATA_DIR, PA_SECRETS_DIR.
// - Config dir = filepath.Dir(configFilePath): command_allowlist_path, scheduled_tasks_path.
// - PA_DATA_DIR (env; default "."): memory_dir, log_path, vector_index_path, llm_log_dir.
// - PA_SECRETS_DIR (env; default "."): token_path, users_path, api_key_path (LLM and embedding), private_key_path.
// Paths that are absolute (filepath.IsAbs) are left unchanged.
func ResolvePaths(cfg *Config, configFilePath string) {
	configDir := filepath.Dir(configFilePath)
	dataDir := envDefault("PA_DATA_DIR", ".")
	secretsDir := envDefault("PA_SECRETS_DIR", ".")

	cfg.Telegram.TokenPath = resolve(secretsDir, cfg.Telegram.TokenPath)
	cfg.Telegram.UsersPath = resolve(secretsDir, cfg.Telegram.UsersPath)

	cfg.Paths.MemoryDir = resolve(dataDir, cfg.Paths.MemoryDir)
	cfg.Paths.LogPath = resolve(dataDir, cfg.Paths.LogPath)
	cfg.Paths.VectorIndexPath = resolve(dataDir, cfg.Paths.VectorIndexPath)
	cfg.Paths.LLMLogDir = resolve(dataDir, cfg.Paths.LLMLogDir)
	cfg.Paths.ScheduledTasksPath = resolve(configDir, cfg.Paths.ScheduledTasksPath)
	cfg.Paths.SSHKnownHostsPath = resolve(configDir, cfg.Paths.SSHKnownHostsPath)
	cfg.Paths.ToolCatalogPath = resolve(configDir, cfg.Paths.ToolCatalogPath)
	cfg.Paths.SkillsDir = resolve(configDir, cfg.Paths.SkillsDir)

	for i := range cfg.LLMProviders {
		cfg.LLMProviders[i].APIKeyPath = resolve(secretsDir, cfg.LLMProviders[i].APIKeyPath)
	}

	if cfg.Embedding != nil {
		cfg.Embedding.APIKeyPath = resolve(secretsDir, cfg.Embedding.APIKeyPath)
	}

	for id := range cfg.Nodes {
		n := cfg.Nodes[id]
		n.Auth.PrivateKeyPath = resolve(secretsDir, n.Auth.PrivateKeyPath)
		n.CommandAllowlistPath = resolve(configDir, n.CommandAllowlistPath)
		cfg.Nodes[id] = n
	}

	if cfg.WebTools != nil && cfg.WebTools.Enabled {
		cfg.WebTools.Search.BraveAPIKeyPath = resolve(secretsDir, cfg.WebTools.Search.BraveAPIKeyPath)
	}
}

func envDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func resolve(base, path string) string {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}
