package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"pa/internal/config"
	"pa/internal/embedding"
	"pa/internal/toolcatalog"
	"pa/internal/toolindex"
	"regexp"
	"strings"
	"sync"
)

// CreateToolTool implements the native create_tool for EP-009 (REQ-09.008–013, REQ-09.017) and EP-023 catalog durability (REQ-23.005–007).
type CreateToolTool struct {
	mu          sync.Locker
	catalog     *toolcatalog.Catalog
	catalogPath string
	cfg         *config.Config
	secretRX    []*regexp.Regexp
	embedder    embedding.Embedder
	toolIndex   *toolindex.Index
	logger      *slog.Logger
}

// NewCreateTool builds create_tool. catalogPath must be absolute; cfg must be non-nil; mu must be non-nil.
func NewCreateTool(mu sync.Locker, catalog *toolcatalog.Catalog, catalogPath string, cfg *config.Config, embedder embedding.Embedder, toolIndex *toolindex.Index, logger *slog.Logger) *CreateToolTool {
	if mu == nil {
		mu = &sync.Mutex{} // should not happen in production
	}
	return &CreateToolTool{
		mu:          mu,
		catalog:     catalog,
		catalogPath: catalogPath,
		cfg:         cfg,
		secretRX:    cfg.CreateToolSecretRegex,
		embedder:    embedder,
		toolIndex:   toolIndex,
		logger:      logger,
	}
}

// Name implements Tool.
func (c *CreateToolTool) Name() string { return "create_tool" }

// createToolLLMDescription is the OpenAI-style tool description: rules + one canonical JSON example (quick-win for model adherence).
const createToolLLMDescription = `Create a new catalog tool that runs a Docker sandbox command on a node (SSH). Persists to tools.yaml and updates the runtime catalog.

TEMPLATE (must be ONE physical line — no newline U+000A or carriage return U+000D inside the template string; multi-line Python/shell scripts will be rejected):
- Prefix: docker run --rm --network bridge  OR  docker run --rm --network none
- Include a 30s bound (e.g. timeout 30s before docker, or inside the container command per validation)
- Reference an image on the node (e.g. pa-sandbox:base)
- For non-trivial logic: use a one-liner (semicolons in python -c), or curl/rss in a single line, or a small script file already on the node plus an allowlisted exec line — do not embed multiline source in the template.

Remote command policy: letters, digits, space, and limited ASCII punctuation (including " for quoting); no tab; no ; & | $() backtick; no newlines. ? is not allowed — avoid raw URLs with query strings in the template (use path-only endpoints or placeholders if the node allowlist permits).

ARGUMENTS (optional): omit the key or use "" if there are no {{placeholder}} segments in the template. If there are placeholders, pass arguments as a JSON-encoded string of an array of objects. Each object may include: name, type (string or integer), required, allowed_values, pattern, min, max.

Canonical example — pass a single JSON object like this for the function arguments (replace id, index_text, template, node_id as needed):
{
  "id": "weather_wttr",
  "index_text": "Weather for a city via wttr.in",
  "template": "docker run --rm --network bridge pa-sandbox:base timeout 30s curl -fsS wttr.in/{{city}}",
  "node_id": "nas",
  "arguments": "[{\"name\":\"city\",\"type\":\"string\",\"required\":true}]",
  "system_prompt": ""
}
If there are no placeholders, omit "arguments" or set "arguments": "".
`

// Description implements Tool.
func (c *CreateToolTool) Description() string {
	return createToolLLMDescription
}

// ParamsSchema implements Tool.
func (c *CreateToolTool) ParamsSchema() []ParamSpec {
	return []ParamSpec{
		{Name: "id", Required: true, Type: "string"},
		{Name: "index_text", Required: true, Type: "string"},
		{Name: "template", Required: true, Type: "string"},
		{Name: "node_id", Required: true, Type: "string"},
		{Name: "arguments", Required: false, Type: "string"},
		{Name: "system_prompt", Required: false, Type: "string"},
	}
}

// Run implements Tool.
func (c *CreateToolTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if c == nil || c.catalog == nil || c.cfg == nil {
		return "", fmt.Errorf("tools: create_tool: not configured")
	}
	if params == nil {
		params = map[string]any{}
	}
	id, indexText, template, nodeID, sysPrompt, argRules, err := c.parseCreateParams(params)
	if err != nil {
		return "", err
	}
	if _, ok := c.cfg.Nodes[nodeID]; !ok {
		return "", fmt.Errorf("tools: create_tool: unknown node_id %q", nodeID)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lockedCreate(ctx, id, indexText, template, nodeID, sysPrompt, argRules)
}

func (c *CreateToolTool) parseCreateParams(params map[string]any) (id, indexText, template, nodeID, sysPrompt string, argRules []toolcatalog.ArgumentRule, err error) {
	id = stringFromParam(params, "id")
	indexText = stringFromParam(params, "index_text")
	template = stringFromParam(params, "template")
	nodeID = stringFromParam(params, "node_id")
	if id == "" || indexText == "" || template == "" || nodeID == "" {
		return "", "", "", "", "", nil, fmt.Errorf("tools: create_tool: id, index_text, template, and node_id are required")
	}
	sysPrompt = stringFromParam(params, "system_prompt")
	if v, has := params["arguments"]; has && v != nil {
		argRules, err = toolcatalog.ParseArgumentRulesFromCreateToolParams(v)
		if err != nil {
			return "", "", "", "", "", nil, err
		}
	}
	return id, indexText, template, nodeID, sysPrompt, argRules, nil
}

func (c *CreateToolTool) lockedCreate(ctx context.Context, id, indexText, template, nodeID, sysPrompt string, argRules []toolcatalog.ArgumentRule) (string, error) {
	if err := toolcatalog.ValidateCreateToolTemplatePrefix(template); err != nil {
		return "", err
	}
	if err := toolcatalog.ValidateSandboxResourceSubstrings(template); err != nil {
		return "", err
	}
	if _, exists := c.catalog.Tools[id]; exists {
		return "", fmt.Errorf("tools: create_tool: duplicate tool id %q", id)
	}
	payload := strings.Join([]string{id, indexText, template, sysPrompt}, "\n")
	if len(argRules) > 0 {
		b, _ := json.Marshal(argRules)
		payload += "\n" + string(b)
	}
	if err := c.matchSecrets(payload); err != nil {
		return "", err
	}
	newTool := &toolcatalog.Tool{
		ID:           id,
		IndexText:    indexText,
		Template:     template,
		NodeID:       nodeID,
		Arguments:    argRules,
		SystemPrompt: sysPrompt,
	}
	snapshot, err := os.ReadFile(c.catalogPath)
	if err != nil {
		return "", fmt.Errorf("tools: create_tool: read catalog: %w", err)
	}
	if err := toolcatalog.AppendToolToCatalogFile(c.catalogPath, newTool); err != nil {
		return "", fmt.Errorf("tools: create_tool: persist catalog: %w", err)
	}
	c.catalog.Tools[id] = newTool
	if c.embedder != nil && c.toolIndex != nil {
		if err := toolindex.UpsertToolEmbedding(ctx, c.toolIndex, c.embedder, newTool); err != nil {
			delete(c.catalog.Tools, id)
			if rerr := toolcatalog.RestoreCatalogFile(c.catalogPath, snapshot); rerr != nil {
				return "", fmt.Errorf("tools: create_tool: index: %w (catalog rollback: %w)", err, rerr)
			}
			return "", fmt.Errorf("tools: create_tool: index: %w", err)
		}
	}
	return fmt.Sprintf("Created tool %q successfully.", id), nil
}

func stringFromParam(params map[string]any, key string) string {
	v, ok := params[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case json.Number:
		return strings.TrimSpace(x.String())
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%g", x))
	case bool:
		return fmt.Sprintf("%v", x)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
}

func (c *CreateToolTool) matchSecrets(payload string) error {
	if len(c.secretRX) == 0 {
		return nil
	}
	for _, re := range c.secretRX {
		if re != nil && re.MatchString(payload) {
			return errors.New("tools: create_tool: tool definition rejected (possible secret content)")
		}
	}
	return nil
}
