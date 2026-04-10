package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"pa/internal/logredact"
	"pa/internal/toolcatalog"
	"regexp"
	"strings"
	"time"
)

const supportedVersion = 1

// Upper bounds for tool pre-selection and conversation context (catch typos; values must be explicit in config).
const (
	maxToolSearchTopK        = 500
	maxToolMinCount          = 500
	maxToolFallbackCap       = 1000
	maxVectorSearchTopK      = 500
	maxMaxDynamicSystemRunes = 10_000_000
)

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
	if err := compileCreateToolSecretPatterns(&raw); err != nil {
		return nil, err
	}
	raw.PATimezone = strings.TrimSpace(raw.PATimezone)

	ResolvePaths(&raw, path)

	if len(raw.Nodes) > 0 {
		if _, err := os.Stat(raw.Paths.SSHKnownHostsPath); err != nil {
			return nil, fmt.Errorf("paths.ssh_known_hosts_path %s: %w", raw.Paths.SSHKnownHostsPath, err)
		}
	}

	// Validate users file if set (path is now resolved).
	if raw.Telegram.UsersPath != "" {
		if _, err := LoadTelegramUsers(raw.Telegram.UsersPath); err != nil {
			return nil, fmt.Errorf("telegram users file %s: %w", raw.Telegram.UsersPath, err)
		}
	}

	// Load tool catalog (path is required); fail fast on parse or schema error.
	cat, err := toolcatalog.Load(raw.Paths.ToolCatalogPath)
	if err != nil {
		return nil, err
	}
	raw.ToolCatalog = cat

	if err := validateToolsAlwaysInclude(&raw); err != nil {
		return nil, err
	}
	if err := finalizeRuntimeSkills(&raw); err != nil {
		return nil, err
	}

	return &raw, nil
}

func validate(c *Config) error {
	if err := validateCore(c); err != nil {
		return err
	}
	return validateMandatoryJSONSections(c)
}

func validateCore(c *Config) error {
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

func validateMandatoryJSONSections(c *Config) error {
	if err := validateTools(c); err != nil {
		return err
	}
	if err := validateLogRedaction(c); err != nil {
		return err
	}
	if err := validatePATimezone(c); err != nil {
		return err
	}
	if err := validateConversationContext(c); err != nil {
		return err
	}
	if err := validateToolPreSelection(c); err != nil {
		return err
	}
	if err := validateLLMEscalation(c); err != nil {
		return err
	}
	if err := validateWebTools(c); err != nil {
		return err
	}
	return nil
}

func validateTools(c *Config) error {
	if c.Tools == nil {
		return errors.New("config: tools is required (use {\"tools\": {}} with explicit text_based_enabled if needed)")
	}
	return nil
}

// compileCreateToolSecretPatterns compiles tools.create_tool_secret_patterns; invalid regex fails load (REQ-09.017).
func compileCreateToolSecretPatterns(c *Config) error {
	if c == nil || c.Tools == nil || len(c.Tools.CreateToolSecretPatterns) == 0 {
		return nil
	}
	out := make([]*regexp.Regexp, 0, len(c.Tools.CreateToolSecretPatterns))
	for i, s := range c.Tools.CreateToolSecretPatterns {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("config: tools.create_tool_secret_patterns[%d] is empty", i)
		}
		re, err := regexp.Compile(s)
		if err != nil {
			return fmt.Errorf("config: tools.create_tool_secret_patterns[%d]: %w", i, err)
		}
		out = append(out, re)
	}
	c.CreateToolSecretRegex = out
	return nil
}

func validateLLMEscalation(c *Config) error {
	e := c.ToolsLLMEscalation()
	if e == nil || !e.Enabled {
		return nil
	}
	n := len(c.LLMProviders)
	if n < 2 {
		return errors.New("config: tools.llm_escalation.enabled requires at least two llm_providers")
	}
	if e.BaselineIndex < 0 || e.BaselineIndex >= n {
		return fmt.Errorf("config: tools.llm_escalation.baseline_index must be in [0, %d] (0-based index into llm_providers, count=%d)", n-1, n)
	}
	if e.MaxPerUserMessage < 1 {
		return errors.New("config: tools.llm_escalation.max_per_user_message must be >= 1 when enabled")
	}
	return nil
}

func validateToolPreSelection(c *Config) error {
	if c.ToolPreSelection == nil {
		return errors.New("config: tool_pre_selection is required")
	}
	t := c.ToolPreSelection
	if t.ToolSearchTopK < 1 {
		return errors.New("config: tool_pre_selection.tool_search_top_k must be >= 1")
	}
	if t.ToolSearchTopK > maxToolSearchTopK {
		return fmt.Errorf("config: tool_pre_selection.tool_search_top_k must be <= %d", maxToolSearchTopK)
	}
	if t.ToolMinCount < 1 {
		return errors.New("config: tool_pre_selection.tool_min_count must be >= 1")
	}
	if t.ToolMinCount > maxToolMinCount {
		return fmt.Errorf("config: tool_pre_selection.tool_min_count must be <= %d", maxToolMinCount)
	}
	if t.ToolFallbackCap < 1 {
		return errors.New("config: tool_pre_selection.tool_fallback_cap must be >= 1")
	}
	if t.ToolFallbackCap > maxToolFallbackCap {
		return fmt.Errorf("config: tool_pre_selection.tool_fallback_cap must be <= %d", maxToolFallbackCap)
	}
	return nil
}

func validateConversationContext(c *Config) error {
	if c.ConversationContext == nil {
		return errors.New("config: conversation_context is required")
	}
	cc := c.ConversationContext
	if cc.MaxDynamicSystemRunes < 1 {
		return errors.New("config: conversation_context.max_dynamic_system_runes must be >= 1")
	}
	if cc.MaxDynamicSystemRunes > maxMaxDynamicSystemRunes {
		return fmt.Errorf("config: conversation_context.max_dynamic_system_runes must be <= %d", maxMaxDynamicSystemRunes)
	}
	if cc.VectorSearchTopK < 1 {
		return errors.New("config: conversation_context.vector_search_top_k must be >= 1")
	}
	if cc.VectorSearchTopK > maxVectorSearchTopK {
		return fmt.Errorf("config: conversation_context.vector_search_top_k must be <= %d", maxVectorSearchTopK)
	}
	return nil
}

func validatePATimezone(c *Config) error {
	tz := strings.TrimSpace(c.PATimezone)
	if tz == "" {
		return errors.New("config: pa_timezone is required (e.g. \"UTC\" or an IANA timezone name)")
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("config: invalid pa_timezone %q: %w", tz, err)
	}
	return nil
}

func validateLogRedaction(c *Config) error {
	if c.LogRedaction == nil {
		return errors.New("config: log_redaction is required (use {\"log_redaction\": {\"additional_patterns\": []}} if none)")
	}
	additional := make([]logredact.Pattern, 0, len(c.LogRedaction.AdditionalPatterns))
	for _, p := range c.LogRedaction.AdditionalPatterns {
		additional = append(additional, logredact.Pattern{ID: p.ID, Regex: p.Regex, Replacement: p.Replacement})
	}
	return logredact.ValidateConfig(logredact.BuiltInIDs(), additional)
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
	if e.BatchSize < 1 || e.BatchSize > 1000 {
		return errors.New("config: embedding.batch_size is required and must be between 1 and 1000")
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
		if err := validateOneLLMProvider(i, &p); err != nil {
			return err
		}
	}
	return nil
}

func validateOneLLMProvider(idx int, p *LLMProvider) error {
	if err := validateLLMProviderCore(idx, p); err != nil {
		return err
	}
	return validateLLMProviderDefaults(idx, p)
}

func validateLLMProviderCore(idx int, p *LLMProvider) error {
	if p.SupportsTools == nil {
		return fmt.Errorf("config: llm_providers[%d].supports_tools is required (boolean)", idx)
	}
	if strings.TrimSpace(p.Type) == "" {
		return fmt.Errorf("config: llm_providers[%d].type is required", idx)
	}
	if strings.TrimSpace(p.Endpoint) == "" {
		return fmt.Errorf("config: llm_providers[%d].endpoint is required", idx)
	}
	if strings.TrimSpace(p.APIKeyPath) == "" && (p.Type == "openai" || p.Type == "openai-compatible") {
		return fmt.Errorf("config: llm_providers[%d].api_key_path is required for type %q", idx, p.Type)
	}
	if strings.TrimSpace(p.Model) == "" {
		return fmt.Errorf("config: llm_providers[%d].model is required", idx)
	}
	return nil
}

func validateLLMProviderDefaults(idx int, p *LLMProvider) error {
	if p.DefaultTemperature < 0 || p.DefaultTemperature > 2 {
		return fmt.Errorf("config: llm_providers[%d].default_temperature must be in [0, 2]", idx)
	}
	if p.DefaultMaxTokens < 1 {
		return fmt.Errorf("config: llm_providers[%d].default_max_tokens must be >= 1", idx)
	}
	rf := strings.TrimSpace(p.DefaultResponseFormat)
	if rf == "" {
		return fmt.Errorf("config: llm_providers[%d].default_response_format is required (\"text\" or \"json_object\")", idx)
	}
	if rf != "text" && rf != "json_object" {
		return fmt.Errorf("config: llm_providers[%d].default_response_format must be \"text\" or \"json_object\", got %q", idx, rf)
	}
	if rf == "json_object" && !p.SupportsJSONMode {
		return fmt.Errorf("config: llm_providers[%d].default_response_format=\"json_object\" requires supports_json_mode=true", idx)
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
	if c.Paths.LLMLogRetentionDays < 1 {
		return errors.New("config: paths.llm_log_retention_days must be >= 1")
	}
	if len(c.Nodes) > 0 && strings.TrimSpace(c.Paths.SSHKnownHostsPath) == "" {
		return errors.New("config: paths.ssh_known_hosts_path is required when nodes are configured")
	}
	if strings.TrimSpace(c.Paths.ToolCatalogPath) == "" {
		return errors.New("config: paths.tool_catalog_path is required")
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

// validateToolsAlwaysInclude rejects unknown tool ids in tools.always_include (REQ-13.003).
func validateToolsAlwaysInclude(c *Config) error {
	if c == nil || c.Tools == nil || c.ToolCatalog == nil {
		return nil
	}
	for _, id := range c.Tools.AlwaysInclude {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := c.ToolCatalog.Tools[id]; ok {
			continue
		}
		if NativeToolAllowed(c, id) {
			continue
		}
		return fmt.Errorf("tools.always_include: unknown tool id %q", id)
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
