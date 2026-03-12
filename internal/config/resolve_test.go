package config

import (
	"os"
	"path/filepath"
	"testing"
)

// No AC: path resolution (PA_DATA_DIR, PA_SECRETS_DIR) — relative paths joined with config dir and env bases.
func TestResolvePaths_relativePaths_joinedWithBases(t *testing.T) {
	configPath := filepath.Join("/etc", "pa", "config.json")
	cfg := &Config{
		Telegram: Telegram{
			TokenPath: "telegram_bot_token",
			UsersPath: "telegram_users.json",
		},
		Paths: Paths{
			MemoryDir:          "memory",
			LogPath:            "pa.log",
			VectorIndexPath:    "pa_vectors.sqlite",
			LLMLogDir:          "llm_logs",
			ScheduledTasksPath: "scheduled_tasks.json",
		},
		LLMProviders: []LLMProvider{
			{Type: "openai", Endpoint: "https://api.openai.com", APIKeyPath: "openai_key", Model: "gpt-4"},
		},
		Embedding: &EmbeddingProvider{APIKeyPath: "openai_key", Type: "openai", Endpoint: "x", Model: "m", Dimensions: 768},
		Nodes: map[string]Node{
			"n1": {
				Host: "h", DedicatedUser: "u",
				Auth:                 NodeAuth{PrivateKeyPath: "node_key"},
				CommandAllowlistPath: "allowlist.txt",
			},
		},
	}

	_ = os.Setenv("PA_DATA_DIR", "/data")
	_ = os.Setenv("PA_SECRETS_DIR", "/run/secrets")
	t.Cleanup(func() {
		_ = os.Unsetenv("PA_DATA_DIR")
		_ = os.Unsetenv("PA_SECRETS_DIR")
	})

	ResolvePaths(cfg, configPath)

	configDir := "/etc/pa"
	if got := cfg.Telegram.TokenPath; got != filepath.Join("/run/secrets", "telegram_bot_token") {
		t.Errorf("TokenPath = %q", got)
	}
	if got := cfg.Telegram.UsersPath; got != filepath.Join("/run/secrets", "telegram_users.json") {
		t.Errorf("UsersPath = %q", got)
	}
	if got := cfg.Paths.MemoryDir; got != filepath.Join("/data", "memory") {
		t.Errorf("MemoryDir = %q", got)
	}
	if got := cfg.Paths.LogPath; got != filepath.Join("/data", "pa.log") {
		t.Errorf("LogPath = %q", got)
	}
	if got := cfg.Paths.VectorIndexPath; got != filepath.Join("/data", "pa_vectors.sqlite") {
		t.Errorf("VectorIndexPath = %q", got)
	}
	if got := cfg.Paths.LLMLogDir; got != filepath.Join("/data", "llm_logs") {
		t.Errorf("LLMLogDir = %q", got)
	}
	if got := cfg.Paths.ScheduledTasksPath; got != filepath.Join(configDir, "scheduled_tasks.json") {
		t.Errorf("ScheduledTasksPath = %q", got)
	}
	if got := cfg.LLMProviders[0].APIKeyPath; got != filepath.Join("/run/secrets", "openai_key") {
		t.Errorf("LLM APIKeyPath = %q", got)
	}
	if got := cfg.Embedding.APIKeyPath; got != filepath.Join("/run/secrets", "openai_key") {
		t.Errorf("Embedding APIKeyPath = %q", got)
	}
	if got := cfg.Nodes["n1"].Auth.PrivateKeyPath; got != filepath.Join("/run/secrets", "node_key") {
		t.Errorf("PrivateKeyPath = %q", got)
	}
	if got := cfg.Nodes["n1"].CommandAllowlistPath; got != filepath.Join(configDir, "allowlist.txt") {
		t.Errorf("CommandAllowlistPath = %q", got)
	}
}

// No AC: path resolution — absolute paths are left unchanged.
func TestResolvePaths_absolutePaths_unchanged(t *testing.T) {
	cfg := &Config{
		Telegram:     Telegram{TokenPath: "/abs/secrets/token", UsersPath: "/abs/users.json"},
		Paths:        Paths{MemoryDir: "/var/data/memory", LogPath: "/var/log/pa.log", VectorIndexPath: "/var/pa.sqlite", LLMLogDir: "/var/llm"},
		LLMProviders: []LLMProvider{{APIKeyPath: "/abs/key"}},
		Embedding:    &EmbeddingProvider{APIKeyPath: "/abs/emb", Type: "x", Endpoint: "x", Model: "m", Dimensions: 1},
		Nodes: map[string]Node{
			"n": {Auth: NodeAuth{PrivateKeyPath: "/abs/ssh"}, CommandAllowlistPath: "/abs/allowlist"},
		},
	}
	_ = os.Setenv("PA_DATA_DIR", "/data")
	_ = os.Setenv("PA_SECRETS_DIR", "/secrets")
	t.Cleanup(func() {
		_ = os.Unsetenv("PA_DATA_DIR")
		_ = os.Unsetenv("PA_SECRETS_DIR")
	})

	ResolvePaths(cfg, "/etc/config.json")

	if cfg.Telegram.TokenPath != "/abs/secrets/token" {
		t.Errorf("TokenPath = %q", cfg.Telegram.TokenPath)
	}
	if cfg.Telegram.UsersPath != "/abs/users.json" {
		t.Errorf("UsersPath = %q", cfg.Telegram.UsersPath)
	}
	if cfg.Paths.MemoryDir != "/var/data/memory" {
		t.Errorf("MemoryDir = %q", cfg.Paths.MemoryDir)
	}
	if cfg.LLMProviders[0].APIKeyPath != "/abs/key" {
		t.Errorf("LLM APIKeyPath = %q", cfg.LLMProviders[0].APIKeyPath)
	}
	if cfg.Nodes["n"].Auth.PrivateKeyPath != "/abs/ssh" {
		t.Errorf("PrivateKeyPath = %q", cfg.Nodes["n"].Auth.PrivateKeyPath)
	}
}

// No AC: path resolution — empty paths remain empty.
func TestResolvePaths_emptyPath_unchanged(t *testing.T) {
	cfg := &Config{
		Telegram:  Telegram{TokenPath: "/t", UsersPath: ""},
		Paths:     Paths{MemoryDir: "/d", LogPath: "/d", VectorIndexPath: "/d", LLMLogDir: "/d", ScheduledTasksPath: ""},
		Embedding: &EmbeddingProvider{Type: "ollama", Endpoint: "x", Model: "m", Dimensions: 1},
		Nodes:     map[string]Node{},
	}
	ResolvePaths(cfg, "/etc/pa/config.json")
	if cfg.Telegram.UsersPath != "" {
		t.Errorf("UsersPath = %q, want empty", cfg.Telegram.UsersPath)
	}
	if cfg.Paths.ScheduledTasksPath != "" {
		t.Errorf("ScheduledTasksPath = %q, want empty", cfg.Paths.ScheduledTasksPath)
	}
}

// No AC: path resolution — when PA_DATA_DIR/PA_SECRETS_DIR unset, base is ".".
func TestResolvePaths_envUnset_usesDot(t *testing.T) {
	_ = os.Unsetenv("PA_DATA_DIR")
	_ = os.Unsetenv("PA_SECRETS_DIR")
	cfg := &Config{
		Telegram:  Telegram{TokenPath: "token"},
		Paths:     Paths{MemoryDir: "data/memory", LogPath: "pa.log", VectorIndexPath: "v.sqlite", LLMLogDir: "llm"},
		Embedding: &EmbeddingProvider{APIKeyPath: "key", Type: "x", Endpoint: "x", Model: "m", Dimensions: 1},
		Nodes:     map[string]Node{},
	}
	ResolvePaths(cfg, filepath.Join("config", "config.json"))
	wantToken := filepath.Join(".", "token")
	if cfg.Telegram.TokenPath != wantToken {
		t.Errorf("TokenPath = %q, want %q", cfg.Telegram.TokenPath, wantToken)
	}
	wantMemory := filepath.Join(".", "data", "memory")
	if cfg.Paths.MemoryDir != wantMemory {
		t.Errorf("MemoryDir = %q, want %q", cfg.Paths.MemoryDir, wantMemory)
	}
}
