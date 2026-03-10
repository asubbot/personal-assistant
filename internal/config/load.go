package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const supportedVersion = 1

// Load reads and validates config from path. On validation failure returns a clear error.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var raw Config
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := validate(&raw); err != nil {
		return nil, err
	}

	// All paths in config are relative to project root (CWD at startup). Validate users file if set.
	if raw.Telegram.UsersPath != "" {
		if _, err := LoadTelegramUsers(raw.Telegram.UsersPath); err != nil {
			return nil, fmt.Errorf("telegram users file %s: %w", raw.Telegram.UsersPath, err)
		}
	}

	return &raw, nil
}

func validate(c *Config) error {
	if err := validateVersion(c); err != nil {
		return err
	}
	if err := validateTelegram(c); err != nil {
		return err
	}
	if err := validateLLMProviders(c); err != nil {
		return err
	}
	if err := validatePaths(c); err != nil {
		return err
	}
	if err := validateEmbedding(c); err != nil {
		return err
	}
	if err := validateNodes(c); err != nil {
		return err
	}
	return nil
}

func validateEmbedding(c *Config) error {
	if c.Embedding == nil {
		return errors.New("config: embedding is required for vector memory (assistant requires it for good UX)")
	}
	e := c.Embedding
	if strings.TrimSpace(e.Type) == "" {
		return errors.New("config: embedding.type is required when embedding is set")
	}
	if strings.TrimSpace(e.Endpoint) == "" {
		return errors.New("config: embedding.endpoint is required when embedding is set")
	}
	if strings.TrimSpace(e.Model) == "" {
		return errors.New("config: embedding.model is required when embedding is set")
	}
	if e.Dimensions <= 0 {
		return errors.New("config: embedding.dimensions must be positive when embedding is set")
	}
	if strings.TrimSpace(e.APIKeyPath) == "" && (e.Type == "openai" || e.Type == "openai-compatible") {
		return errors.New("config: embedding.api_key_path is required for type openai/openai-compatible")
	}
	return nil
}

func validateVersion(c *Config) error {
	if c.Version != supportedVersion {
		return fmt.Errorf("config: version must be %d (got %d)", supportedVersion, c.Version)
	}
	return nil
}

func validateTelegram(c *Config) error {
	if strings.TrimSpace(c.Telegram.TokenPath) == "" {
		return errors.New("config: telegram.token_path is required")
	}
	// telegram.users_path is optional; if missing, behaviour is allow-none (defined at adapter level)
	return nil
}

func validateLLMProviders(c *Config) error {
	if len(c.LLMProviders) == 0 {
		return errors.New("config: at least one llm_providers entry is required")
	}
	for i, p := range c.LLMProviders {
		if strings.TrimSpace(p.Type) == "" {
			return fmt.Errorf("config: llm_providers[%d].type is required", i)
		}
		if strings.TrimSpace(p.Endpoint) == "" {
			return fmt.Errorf("config: llm_providers[%d].endpoint is required", i)
		}
		if strings.TrimSpace(p.APIKeyPath) == "" && (p.Type == "openai" || p.Type == "openai-compatible") {
			return fmt.Errorf("config: llm_providers[%d].api_key_path is required for type %q", i, p.Type)
		}
	}
	return nil
}

func validatePaths(c *Config) error {
	if strings.TrimSpace(c.Paths.MemoryDir) == "" {
		return errors.New("config: paths.memory_dir is required")
	}
	if strings.TrimSpace(c.Paths.LogPath) == "" {
		return errors.New("config: paths.log_path is required")
	}
	if strings.TrimSpace(c.Paths.VectorIndexPath) == "" {
		return errors.New("config: paths.vector_index_path is required")
	}
	if strings.TrimSpace(c.Paths.LLMLogDir) == "" {
		return errors.New("config: paths.llm_log_dir is required")
	}
	return nil
}

func validateNodes(c *Config) error {
	for id, n := range c.Nodes {
		if strings.TrimSpace(n.Host) == "" {
			return fmt.Errorf("config: nodes.%s.host is required", id)
		}
		if strings.TrimSpace(n.DedicatedUser) == "" {
			return fmt.Errorf("config: nodes.%s.dedicated_user is required", id)
		}
		if strings.TrimSpace(n.Auth.PrivateKeyPath) == "" {
			return fmt.Errorf("config: nodes.%s.auth.private_key_path is required", id)
		}
		if strings.TrimSpace(n.CommandAllowlistPath) == "" {
			return fmt.Errorf("config: nodes.%s.command_allowlist_path is required", id)
		}
	}
	return nil
}

// LoadTelegramUsers reads and validates the Telegram users JSON file. Returns list of users or error.
func LoadTelegramUsers(path string) ([]TelegramUser, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var users []TelegramUser
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("parse users: %w", err)
	}
	for i, u := range users {
		if u.UserID <= 0 {
			return nil, fmt.Errorf("users[%d]: user_id must be positive", i)
		}
		role := strings.TrimSpace(strings.ToLower(u.Role))
		if role != "user" && role != "admin" {
			return nil, fmt.Errorf("users[%d]: role must be %q or %q (got %q)", i, "user", "admin", u.Role)
		}
	}
	return users, nil
}
