package jobs

import (
	"context"
	"fmt"
	"pa/internal/tools"
	"regexp"
	"strconv"
	"strings"
)

// CreateScheduledJobTool is a native fallback tool for explicit schedule-intent free-form create requests.
type CreateScheduledJobTool struct {
	manager *Manager
}

func NewCreateScheduledJobTool(manager *Manager) *CreateScheduledJobTool {
	return &CreateScheduledJobTool{manager: manager}
}

func (t *CreateScheduledJobTool) Name() string { return "create_scheduled_job" }

func (t *CreateScheduledJobTool) Description() string {
	return "Create a scheduled job from explicit schedule-intent text when strict NL parser does not match."
}

func (t *CreateScheduledJobTool) ParamsSchema() []tools.ParamSpec {
	return []tools.ParamSpec{
		{Name: "text", Required: true, Type: "string"},
		{Name: "actor_user_id", Required: true, Type: "number"},
		{Name: "delivery_chat_id", Required: false, Type: "number"},
		{Name: "timezone", Required: false, Type: "string"},
	}
}

func (t *CreateScheduledJobTool) Run(ctx context.Context, params map[string]any) (string, error) {
	if t == nil || t.manager == nil {
		return "", fmt.Errorf("jobs: create_scheduled_job: not configured")
	}
	if err := tools.ValidateParams(t.ParamsSchema(), params); err != nil {
		return "", err
	}

	text, _ := params["text"].(string)
	text = strings.TrimSpace(text)
	actorUserID := int64FromAny(params["actor_user_id"])
	deliveryChatID := int64FromAny(params["delivery_chat_id"])
	timezone, _ := params["timezone"].(string)

	req, ok := parseFallbackNaturalLanguageCreateRequest(text)
	if !ok {
		t.manager.audit(actorUserID, "", "create_nl", "invalid_syntax", "creation_path", "native_tool_fallback")
		return "Invalid schedule format. Use: <instruction> and send it at HH:MM every day", nil
	}
	reply, _, err := t.manager.CreateScheduledJobFromSpec(
		ctx,
		actorUserID,
		deliveryChatID,
		req.Instruction,
		req.Hour,
		req.Minute,
		timezone,
		"native_tool_fallback",
	)
	return reply, err
}

var (
	// Example: "collect AI digest and send me it at 09:00 every day"
	fallbackCreateRegexInstructionFirst = regexp.MustCompile(`(?i)^\s*(.+?)\s+(?:and\s+)?send(?:\s+it|\s+me|\s+me\s+it)?\s+at\s+([01]?\d|2[0-3]):([0-5]\d)(?:\s+(?:every\s+day|daily))?\s*$`)
	// Example: "send me AI digest at 09:00 every day"
	fallbackCreateRegexSendFirst = regexp.MustCompile(`(?i)^\s*send(?:\s+it|\s+me|\s+me\s+it)?\s+(.+?)\s+at\s+([01]?\d|2[0-3]):([0-5]\d)(?:\s+(?:every\s+day|daily))?\s*$`)
	explicitScheduleIntentRegex  = regexp.MustCompile(`(?i)\bsend\b.*\bat\b`)
)

func LooksLikeNaturalLanguageCreateRequest(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	return explicitScheduleIntentRegex.MatchString(trimmed)
}

func parseFallbackNaturalLanguageCreateRequest(text string) (naturalLanguageCreateRequest, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !LooksLikeNaturalLanguageCreateRequest(trimmed) {
		return naturalLanguageCreateRequest{}, false
	}
	if req, ok := parseCreateByPattern(trimmed, fallbackCreateRegexInstructionFirst); ok {
		return req, true
	}
	if req, ok := parseCreateByPattern(trimmed, fallbackCreateRegexSendFirst); ok {
		return req, true
	}
	return naturalLanguageCreateRequest{}, false
}

func parseCreateByPattern(text string, pattern *regexp.Regexp) (naturalLanguageCreateRequest, bool) {
	m := pattern.FindStringSubmatch(text)
	if len(m) != 4 {
		return naturalLanguageCreateRequest{}, false
	}
	hour, _ := strconv.Atoi(m[2])
	minute, _ := strconv.Atoi(m[3])
	return naturalLanguageCreateRequest{
		Instruction: normalizeFallbackInstruction(m[1]),
		Hour:        hour,
		Minute:      minute,
	}, true
}

func normalizeFallbackInstruction(instruction string) string {
	v := strings.TrimSpace(instruction)
	v = strings.Trim(v, " .,:;!?")
	return v
}

func int64FromAny(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	default:
		return 0
	}
}
