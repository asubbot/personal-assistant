package jobs

import (
	"context"
	"fmt"
	"pa/internal/tools"
	"strings"
)

const creationPathNativeToolExplicit = "native_tool_explicit"

// CreateScheduledJobTool is the native tool for creating a daily scheduled job with explicit parameters.
type CreateScheduledJobTool struct {
	manager       *Manager
	managerLookup func() *Manager
}

func NewCreateScheduledJobTool(manager *Manager) *CreateScheduledJobTool {
	return &CreateScheduledJobTool{manager: manager}
}

func NewCreateScheduledJobToolWithLookup(lookup func() *Manager) *CreateScheduledJobTool {
	return &CreateScheduledJobTool{managerLookup: lookup}
}

func (t *CreateScheduledJobTool) Name() string { return "create_scheduled_job" }

func (t *CreateScheduledJobTool) Description() string {
	return "Create one daily scheduled agent job. Requires explicit instruction text and local clock hour (0-23) and minute (0-59). Schedule is minute hour * * * (daily). Optional timezone overrides the server default. Do not use for one-off messages with no recurring schedule."
}

func (t *CreateScheduledJobTool) ParamsSchema() []tools.ParamSpec {
	return []tools.ParamSpec{
		{Name: "instruction", Required: true, Type: "string"},
		{Name: "hour", Required: true, Type: "number"},
		{Name: "minute", Required: true, Type: "number"},
		{Name: "timezone", Required: false, Type: "string"},
		{Name: "actor_user_id", Required: false, Type: "number"},
		{Name: "delivery_chat_id", Required: false, Type: "number"},
	}
}

func (t *CreateScheduledJobTool) Run(ctx context.Context, params map[string]any) (string, error) {
	manager := t.resolveManager()
	if t == nil || manager == nil {
		return "", fmt.Errorf("jobs: create_scheduled_job: not configured")
	}
	if err := tools.ValidateParams(t.ParamsSchema(), params); err != nil {
		return "", err
	}
	args, soft, err := parseCreateScheduledJobArgs(ctx, params)
	if err != nil {
		return "", err
	}
	if soft != "" {
		return soft, nil
	}
	reply, _, err := manager.CreateScheduledJobFromSpec(
		ctx,
		args.actorUserID,
		args.deliveryChatID,
		args.instruction,
		args.hour,
		args.minute,
		args.timezone,
		creationPathNativeToolExplicit,
	)
	return reply, err
}

type createScheduledJobArgs struct {
	instruction    string
	hour, minute   int
	timezone       string
	actorUserID    int64
	deliveryChatID int64
}

// parseCreateScheduledJobArgs returns soft validation text (non-empty, err nil) or a hard error (e.g. missing actor).
func parseCreateScheduledJobArgs(ctx context.Context, params map[string]any) (createScheduledJobArgs, string, error) {
	instruction, _ := params["instruction"].(string)
	instruction = strings.TrimSpace(instruction)
	hour, hourOK := intFromParam(params["hour"])
	minute, minuteOK := intFromParam(params["minute"])
	timezone, _ := params["timezone"].(string)
	timezone = strings.TrimSpace(timezone)
	actorUserID := int64FromAny(params["actor_user_id"])
	deliveryChatID := int64FromAny(params["delivery_chat_id"])
	if actorUserID == 0 {
		actorUserID = CreateContextActorUserID(ctx)
	}
	if deliveryChatID == 0 {
		deliveryChatID = CreateContextDeliveryChatID(ctx)
	}
	if actorUserID == 0 {
		return createScheduledJobArgs{}, "", fmt.Errorf("jobs: create_scheduled_job: actor_user_id is required")
	}
	if instruction == "" {
		return createScheduledJobArgs{}, "Instruction must be non-empty.", nil
	}
	if !hourOK || !minuteOK {
		return createScheduledJobArgs{}, "hour and minute must be integers in range (hour 0-23, minute 0-59).", nil
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return createScheduledJobArgs{}, "Invalid schedule: hour must be 0-23 and minute must be 0-59.", nil
	}
	return createScheduledJobArgs{
		instruction:    instruction,
		hour:           hour,
		minute:         minute,
		timezone:       timezone,
		actorUserID:    actorUserID,
		deliveryChatID: deliveryChatID,
	}, "", nil
}

func (t *CreateScheduledJobTool) resolveManager() *Manager {
	if t == nil {
		return nil
	}
	if t.manager != nil {
		return t.manager
	}
	if t.managerLookup == nil {
		return nil
	}
	return t.managerLookup()
}

func intFromParam(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	default:
		return 0, false
	}
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
