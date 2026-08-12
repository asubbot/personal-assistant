package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"pa/internal/logredact"
	"pa/internal/toolcatalog"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
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
	if err := rejectLegacyScheduledTasksPath(data); err != nil {
		return nil, err
	}
	if err := rejectLegacyEP039Shapes(data); err != nil {
		return nil, err
	}

	var raw Config
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := prepareConfig(&raw, path, data); err != nil {
		return nil, err
	}
	return &raw, nil
}

func prepareConfig(raw *Config, path string, rootJSON []byte) error {
	if err := rejectRemovedUnsupportedConfigKeys(rootJSON); err != nil {
		return err
	}
	if err := validateConfigRootObjectKeys(rootJSON); err != nil {
		return err
	}
	if err := validate(raw); err != nil {
		return err
	}
	if err := compileCreateToolSecretPatterns(raw); err != nil {
		return err
	}
	raw.PATimezone = strings.TrimSpace(raw.PATimezone)

	ResolvePaths(raw, path)

	if err := validateAfterResolvePaths(raw); err != nil {
		return err
	}

	// Validate users file if set (path is now resolved).
	if raw.Telegram.UsersPath != "" {
		if _, err := LoadTelegramUsers(raw.Telegram.UsersPath); err != nil {
			return fmt.Errorf("telegram users file %s: %w", raw.Telegram.UsersPath, err)
		}
	}

	// Load tool catalog (path is required); fail fast on parse or schema error.
	cat, err := toolcatalog.Load(raw.Paths.ToolCatalogPath)
	if err != nil {
		return err
	}
	raw.ToolCatalog = cat

	if err := validateToolsAlwaysInclude(raw); err != nil {
		return err
	}
	if err := validateToolsSelectionAlwaysIncludeFloor(raw); err != nil {
		return err
	}
	if err := finalizeRuntimeSkills(raw); err != nil {
		return err
	}
	return nil
}

func rejectRemovedUnsupportedConfigKeys(data []byte) error {
	if !json.Valid(data) {
		return nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("config: parse: %w", err)
	}
	if rawTools, ok := root["tools"]; ok {
		if err := rejectRemovedToolsConfigKeys(rawTools); err != nil {
			return err
		}
	}
	if _, has := root["tool_pre_selection"]; has {
		return errors.New("config: tool_pre_selection is not supported; use tools.selection (EP-037)")
	}
	if rawIC, ok := root["intent_classifier"]; ok {
		if err := rejectRemovedIntentClassifierKeys(rawIC); err != nil {
			return err
		}
	}
	return rejectUnsupportedLLMProviderFields(data)
}

func rejectRemovedIntentClassifierKeys(rawIC json.RawMessage) error {
	if len(rawIC) == 0 || string(rawIC) == "null" {
		return nil
	}
	var ic map[string]json.RawMessage
	if err := json.Unmarshal(rawIC, &ic); err != nil {
		return fmt.Errorf("config: intent_classifier: %w", err)
	}
	if _, has := ic["model_stage"]; has {
		return errors.New("config: intent_classifier.model_stage is not supported; intent classification is heuristic-only (EP-036)")
	}
	rawHeuristic, ok := ic["heuristic"]
	if !ok {
		return nil
	}
	var heuristic map[string]json.RawMessage
	if err := json.Unmarshal(rawHeuristic, &heuristic); err != nil {
		return fmt.Errorf("config: intent_classifier.heuristic: %w", err)
	}
	if _, has := heuristic["full_lite_patterns"]; has {
		return errors.New("config: intent_classifier.heuristic.full_lite_patterns is not supported; use full_patterns or rely on default full tier (EP-036)")
	}
	return nil
}

var allowedToolsKeys = []string{
	"always_include",
	"selection",
	"vector_search_tools",
	"create_tool_secret_patterns",
	"tool_output_artifacts",
}

func rejectRemovedToolsConfigKeys(rawTools json.RawMessage) error {
	if len(rawTools) == 0 || string(rawTools) == "null" {
		return nil
	}
	var tools map[string]json.RawMessage
	if err := json.Unmarshal(rawTools, &tools); err != nil {
		return fmt.Errorf("config: tools: %w", err)
	}
	if _, has := tools["dynamic_selection"]; has {
		return errors.New("config: tools.dynamic_selection is not supported; use tools.selection (EP-037)")
	}
	if _, has := tools["text_based_enabled"]; has {
		return errors.New("config: tools.text_based_enabled is not supported; use llm_providers[].supports_tools true for native tool calling")
	}
	if _, has := tools["llm_escalation"]; has {
		return errors.New("config: tools.llm_escalation is not supported; tool-path LLM escalation was removed (EP-034)")
	}
	if err := validateToolsObjectKeys(tools); err != nil {
		return err
	}
	if raw, ok := tools["tool_output_artifacts"]; ok {
		return validateToolOutputArtifactsObjectKeys(raw)
	}
	return nil
}

func validateToolOutputArtifactsObjectKeys(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fmt.Errorf("config: tools.tool_output_artifacts: %w", err)
	}
	for got := range obj {
		if !slices.Contains(allowedToolOutputArtifactsKeys, got) {
			return fmt.Errorf("config: unknown tools.tool_output_artifacts key %q", got)
		}
	}
	return nil
}

func validateToolsObjectKeys(tools map[string]json.RawMessage) error {
	for got := range tools {
		if !slices.Contains(allowedToolsKeys, got) {
			return fmt.Errorf("config: unknown tools key %q", got)
		}
	}
	return nil
}

func rejectUnsupportedLLMProviderFields(data []byte) error {
	var stub struct {
		LLMProviders []map[string]json.RawMessage `json:"llm_providers"`
	}
	if err := json.Unmarshal(data, &stub); err != nil {
		return fmt.Errorf("config: parse (llm_providers): %w", err)
	}
	for i, p := range stub.LLMProviders {
		if p == nil {
			continue
		}
		if _, has := p["supports_json_mode"]; has {
			return fmt.Errorf("config: llm_providers[%d].supports_json_mode is not supported; remove this field (response format is text-only)", i)
		}
	}
	return nil
}

func rejectLegacyEP039Shapes(data []byte) error {
	if !json.Valid(data) {
		return nil
	}
	if err := rejectLegacyVectorSearchToolsShape(data); err != nil {
		return err
	}
	return rejectLegacySQLiteReliabilityShape(data)
}

func parseJSONObject(raw []byte) (map[string]json.RawMessage, bool) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return nil, false
	}
	return obj, true
}

func rejectLegacyVectorSearchToolsShape(data []byte) error {
	root, ok := parseJSONObject(data)
	if !ok {
		return nil
	}
	rawTools, ok := root["tools"]
	if !ok || len(rawTools) == 0 || string(rawTools) == "null" {
		return nil
	}
	return rejectLegacyVectorSearchToolsInToolsBlock(rawTools)
}

func rejectLegacyVectorSearchToolsInToolsBlock(rawTools json.RawMessage) error {
	tools, ok := parseJSONObject(rawTools)
	if !ok {
		return nil
	}
	rawVST, ok := tools["vector_search_tools"]
	if !ok || len(rawVST) == 0 || string(rawVST) == "null" {
		return nil
	}
	return rejectLegacyVectorSearchToolsBlock(rawVST)
}

func rejectLegacyVectorSearchToolsBlock(rawVST json.RawMessage) error {
	vst, ok := parseJSONObject(rawVST)
	if !ok {
		return nil
	}
	_, hasDefaults := vst["defaults"]
	for _, key := range []string{"search_vector_memory", "search_vector_tool", "search_vector_skill"} {
		rawTool, ok := vst[key]
		if !ok {
			continue
		}
		tool, ok := parseJSONObject(rawTool)
		if !ok {
			continue
		}
		if _, has := tool["default_top_k"]; has && !hasDefaults {
			return fmt.Errorf("config: tools.vector_search_tools: legacy per-tool shape without defaults is not supported (EP-039)")
		}
	}
	return nil
}

func rejectLegacySQLiteReliabilityShape(data []byte) error {
	root, ok := parseJSONObject(data)
	if !ok {
		return nil
	}
	if _, hasDefaults := root["sqlite_store_defaults"]; hasDefaults {
		return nil
	}
	rawVector, okVector := root["vector_store_reliability"]
	rawJobs, okJobs := root["jobs_store_reliability"]
	if !okVector || !okJobs {
		return nil
	}
	vector, okVector := parseJSONObject(rawVector)
	jobs, okJobs := parseJSONObject(rawJobs)
	if !okVector || !okJobs {
		return nil
	}
	if _, hasVector := vector["journal_mode"]; hasVector {
		if _, hasJobs := jobs["journal_mode"]; hasJobs {
			return fmt.Errorf("config: vector_store_reliability and jobs_store_reliability: legacy full duplicate PRAGMA blocks without sqlite_store_defaults are not supported (EP-039); add sqlite_store_defaults and shrink store blocks to foreign_keys only")
		}
	}
	return nil
}

func rejectLegacyScheduledTasksPath(data []byte) error {
	if !json.Valid(data) {
		// Parsing error will be returned by the main unmarshal path.
		return nil
	}
	var root map[string]json.RawMessage
	_ = json.Unmarshal(data, &root)
	pathsRaw, ok := root["paths"]
	if !ok {
		return nil
	}
	if !json.Valid(pathsRaw) {
		return nil
	}
	var paths map[string]json.RawMessage
	_ = json.Unmarshal(pathsRaw, &paths)
	if _, hasLegacy := paths["scheduled_tasks_path"]; hasLegacy {
		return errors.New("config: paths.scheduled_tasks_path is not supported; use paths.jobs_db_path")
	}
	return nil
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
	if err := c.ValidateVectorStoreReliability(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := c.ValidateJobsStoreReliability(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}

func validateMandatoryJSONSections(c *Config) error {
	if err := validateMandatoryJSONSectionsCore(c); err != nil {
		return err
	}
	return validateIntentClassifier(c)
}

func validateMandatoryJSONSectionsCore(c *Config) error {
	if err := validateTools(c); err != nil {
		return err
	}
	if err := validateReadMemory(c); err != nil {
		return err
	}
	if err := validateWriteMemory(c); err != nil {
		return err
	}
	if err := validateRuntimeSkillsNumericFields(c); err != nil {
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
	if err := validateConversationSession(c); err != nil {
		return err
	}
	if err := validateToolsSelectionBounds(c); err != nil {
		return err
	}
	if err := validateWebTools(c); err != nil {
		return err
	}
	return validateObservabilityHTTP(c)
}

func validateObservabilityHTTP(c *Config) error {
	if c == nil || c.ObservabilityHTTP == nil {
		return nil
	}
	o := c.ObservabilityHTTP
	if strings.TrimSpace(o.ListenAddress) == "" {
		return errors.New("config: observability_http.listen_address is required when observability_http is set")
	}
	if err := validateObservabilityPath("observability_http.health_path", o.HealthPath); err != nil {
		return err
	}
	if err := validateObservabilityPath("observability_http.readiness_path", o.ReadinessPath); err != nil {
		return err
	}
	if strings.TrimSpace(o.HealthPath) == strings.TrimSpace(o.ReadinessPath) {
		return errors.New("config: observability_http.health_path and observability_http.readiness_path must differ")
	}
	return nil
}

func validateObservabilityPath(field, p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return fmt.Errorf("config: %s is required when observability_http is set", field)
	}
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("config: %s must start with /", field)
	}
	return nil
}

func validateReadMemory(c *Config) error {
	if c == nil || c.ReadMemory == nil {
		return errors.New("config: read_memory is required")
	}
	rm := c.ReadMemory
	if rm.MaxSpanDays < 1 || rm.MaxSpanDays > 3660 {
		return errors.New("config: read_memory.max_span_days must be in 1..3660")
	}
	if rm.MaxOutputBytes < 1024 || rm.MaxOutputBytes > 50*1024*1024 {
		return errors.New("config: read_memory.max_output_bytes must be in 1024..52428800")
	}
	return nil
}

func validateWriteMemory(c *Config) error {
	if c == nil || c.WriteMemory == nil {
		return errors.New("config: write_memory is required")
	}
	wm := c.WriteMemory
	if wm.MaxAppendBytes < 256 || wm.MaxAppendBytes > 1024*1024 {
		return errors.New("config: write_memory.max_append_bytes must be in 256..1048576")
	}
	if wm.MaxFileBytes < wm.MaxAppendBytes || wm.MaxFileBytes > 50*1024*1024 {
		return errors.New("config: write_memory.max_file_bytes must be >= max_append_bytes and at most 52428800")
	}
	return nil
}

// validateRuntimeSkillsNumericFields requires explicit caps when runtime_skills is present (no implicit defaults).
func validateRuntimeSkillsNumericFields(c *Config) error {
	if c == nil || c.RuntimeSkills == nil {
		return nil
	}
	rs := c.RuntimeSkills
	if rs.MaxSkillsPerTurn < 1 {
		return errors.New("config: runtime_skills.max_skills_per_turn must be >= 1")
	}
	if rs.ToolVectorTopKCap < 1 {
		return errors.New("config: runtime_skills.tool_vector_top_k_cap must be >= 1")
	}
	return nil
}

func validateTools(c *Config) error {
	if c.Tools == nil {
		return errors.New("config: tools is required (use {\"tools\": {}} minimum)")
	}
	if c.Tools.VectorSearchTools != nil {
		if err := validateVectorSearchTools(c.Tools.VectorSearchTools); err != nil {
			return err
		}
	}
	return validateToolOutputArtifacts(c.Tools.ToolOutputArtifacts)
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

func validateToolsSelectionBounds(c *Config) error {
	if c == nil || c.Tools == nil || c.Tools.Selection == nil {
		return errors.New("config: tools.selection is required")
	}
	s := c.Tools.Selection
	if s.ToolSearchTopK < 1 {
		return errors.New("config: tools.selection.tool_search_top_k must be >= 1")
	}
	if s.ToolSearchTopK > maxToolSearchTopK {
		return fmt.Errorf("config: tools.selection.tool_search_top_k must be <= %d", maxToolSearchTopK)
	}
	if s.ToolMinCount < 1 {
		return errors.New("config: tools.selection.tool_min_count must be >= 1")
	}
	if s.ToolMinCount > maxToolMinCount {
		return fmt.Errorf("config: tools.selection.tool_min_count must be <= %d", maxToolMinCount)
	}
	if s.ToolFallbackCap < 1 {
		return errors.New("config: tools.selection.tool_fallback_cap must be >= 1")
	}
	if s.ToolFallbackCap > maxToolFallbackCap {
		return fmt.Errorf("config: tools.selection.tool_fallback_cap must be <= %d", maxToolFallbackCap)
	}
	if s.Enabled && s.MaxToolsForLLMRequest < 1 {
		return errors.New("config: tools.selection.max_tools_for_llm_request must be >= 1 when selection.enabled is true")
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
	mv := cc.MemoryVector
	if mv.NotesTopK < 0 || mv.SummariesTopK < 0 || mv.TurnsTopK < 0 {
		return errors.New("config: conversation_context.memory_vector top_k fields must be >= 0")
	}
	if mv.NotesTopK > maxVectorSearchTopK {
		return fmt.Errorf("config: conversation_context.memory_vector.notes_top_k must be <= %d", maxVectorSearchTopK)
	}
	if mv.SummariesTopK > maxVectorSearchTopK {
		return fmt.Errorf("config: conversation_context.memory_vector.summaries_top_k must be <= %d", maxVectorSearchTopK)
	}
	if mv.TurnsTopK > maxVectorSearchTopK {
		return fmt.Errorf("config: conversation_context.memory_vector.turns_top_k must be <= %d", maxVectorSearchTopK)
	}
	return nil
}

func validateConversationSession(c *Config) error {
	if c == nil || c.ConversationSession == nil || !c.ConversationSession.Enabled {
		return nil
	}
	if c.ConversationSession.MaxSessionExchanges < 1 {
		return errors.New("config: conversation_session.max_session_exchanges must be >= 1 when conversation_session.enabled is true")
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
	if err := validateHTTPTimeout("embedding.http_timeout", e.HTTPTimeout); err != nil {
		return err
	}
	return nil
}

// validateHTTPTimeout enforces explicit, positive Go-duration outbound HTTP timeouts (EP-022, REQ-22.003/22.004).
func validateHTTPTimeout(field, raw string) error {
	s := strings.TrimSpace(raw)
	if s == "" {
		return fmt.Errorf("config: %s is required (Go duration, e.g. \"60s\")", field)
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("config: %s invalid duration %q: %w", field, raw, err)
	}
	if d <= 0 {
		return fmt.Errorf("config: %s must be > 0, got %s", field, d)
	}
	return nil
}

// PALocation returns the IANA location for cfg.pa_timezone. Use after successful Load.
func PALocation(c *Config) (*time.Location, error) {
	if c == nil {
		return time.UTC, nil
	}
	return time.LoadLocation(strings.TrimSpace(c.PATimezone))
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
	for i := range c.LLMProviders {
		if err := validateOneLLMProvider(i, &c.LLMProviders[i]); err != nil {
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
		return fmt.Errorf("config: llm_providers[%d].default_response_format is required (must be \"text\")", idx)
	}
	if rf != "text" {
		return fmt.Errorf("config: llm_providers[%d].default_response_format must be \"text\", got %q", idx, rf)
	}
	if err := validateHTTPTimeout(fmt.Sprintf("llm_providers[%d].http_timeout", idx), p.HTTPTimeout); err != nil {
		return err
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
	if strings.TrimSpace(c.Paths.JobsDBPath) == "" {
		return errors.New("config: paths.jobs_db_path is required")
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

type nodePrivateKeyRow struct {
	id   string
	path string
}

// validateAfterResolvePaths runs checks that need resolved paths (node keys, known_hosts when nodes exist).
func validateAfterResolvePaths(c *Config) error {
	if err := validateDistinctNodePrivateKeys(c); err != nil {
		return err
	}
	if len(c.Nodes) == 0 {
		return nil
	}
	if _, err := os.Stat(c.Paths.SSHKnownHostsPath); err != nil {
		return fmt.Errorf("paths.ssh_known_hosts_path %s: %w", c.Paths.SSHKnownHostsPath, err)
	}
	return nil
}

func buildNodePrivateKeyRows(c *Config) []nodePrivateKeyRow {
	rows := make([]nodePrivateKeyRow, 0, len(c.Nodes))
	for id, n := range c.Nodes {
		p := filepath.Clean(strings.TrimSpace(n.Auth.PrivateKeyPath))
		if p == "." || p == "" {
			continue
		}
		rows = append(rows, nodePrivateKeyRow{id: id, path: p})
	}
	return rows
}

func errDuplicatePrivateKeyCleanPath(path string, ids []string) error {
	sort.Strings(ids)
	return fmt.Errorf("config: nodes %q use the same SSH private_key_path (%s)", strings.Join(ids, ", "), path)
}

func findDuplicatePrivateKeyCleanPaths(rows []nodePrivateKeyRow) error {
	byPath := make(map[string][]string)
	for _, r := range rows {
		byPath[r.path] = append(byPath[r.path], r.id)
	}
	for p, ids := range byPath {
		if len(ids) < 2 {
			continue
		}
		return errDuplicatePrivateKeyCleanPath(p, ids)
	}
	return nil
}

func findSameFilePrivateKeyPairs(rows []nodePrivateKeyRow) error {
	for i := 0; i < len(rows); i++ {
		statI, errI := os.Stat(rows[i].path)
		if errI != nil {
			continue
		}
		for j := i + 1; j < len(rows); j++ {
			if rows[i].path == rows[j].path {
				continue
			}
			statJ, errJ := os.Stat(rows[j].path)
			if errJ != nil {
				continue
			}
			if !os.SameFile(statI, statJ) {
				continue
			}
			a, b := rows[i].id, rows[j].id
			if a > b {
				a, b = b, a
			}
			return fmt.Errorf("config: nodes %q and %q resolve to the same SSH private key file (%s and %s)", a, b, rows[i].path, rows[j].path)
		}
	}
	return nil
}

func validateIntentClassifier(c *Config) error {
	ic := c.IntentClassifier
	if ic == nil || !ic.Enabled {
		return nil
	}
	return validateICHeuristic(ic.Heuristic)
}

func validateICHeuristic(h *HeuristicConfig) error {
	if h == nil {
		return nil
	}
	if h.MaxSimpleLen < 1 {
		return errors.New("config: intent_classifier.heuristic.max_simple_len must be >= 1")
	}
	for i, p := range h.SimplePatterns {
		if _, err := regexp.Compile("(?i)" + p); err != nil {
			return fmt.Errorf("config: intent_classifier.heuristic.simple_patterns[%d]: %w", i, err)
		}
	}
	for i, p := range h.FullPatterns {
		if _, err := regexp.Compile("(?i)" + p); err != nil {
			return fmt.Errorf("config: intent_classifier.heuristic.full_patterns[%d]: %w", i, err)
		}
	}
	return nil
}

func countValidAlwaysIncludeTools(c *Config) int {
	if c == nil || c.Tools == nil || c.ToolCatalog == nil {
		return 0
	}
	seen := make(map[string]struct{})
	for _, id := range c.Tools.AlwaysInclude {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := c.ToolCatalog.Tools[id]; ok {
			seen[id] = struct{}{}
			continue
		}
		if NativeToolAllowed(c, id) {
			seen[id] = struct{}{}
		}
	}
	return len(seen)
}

func validateToolsSelectionAlwaysIncludeFloor(c *Config) error {
	if c == nil || c.Tools == nil || c.Tools.Selection == nil {
		return nil
	}
	s := c.Tools.Selection
	if !s.Enabled {
		return nil
	}
	n := countValidAlwaysIncludeTools(c)
	if n > 0 && s.MaxToolsForLLMRequest < n {
		return fmt.Errorf("config: tools.selection.max_tools_for_llm_request (%d) must be >= valid always_include count (%d)", s.MaxToolsForLLMRequest, n)
	}
	return nil
}

// validateDistinctNodePrivateKeys fails when two or more nodes share the same private key file
// after path resolution (including symlink / hardlink via os.SameFile). Missing key files are
// ignored for SameFile pairing so SSH can still surface a clearer error later.
func validateDistinctNodePrivateKeys(c *Config) error {
	if c == nil || len(c.Nodes) < 2 {
		return nil
	}
	rows := buildNodePrivateKeyRows(c)
	if len(rows) < 2 {
		return nil
	}
	if err := findDuplicatePrivateKeyCleanPaths(rows); err != nil {
		return err
	}
	return findSameFilePrivateKeyPairs(rows)
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
