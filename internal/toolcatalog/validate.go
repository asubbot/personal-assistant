// ValidateToolCall looks up the tool in the catalog and validates arguments (REQ-04.007, REQ-04.008).
// Returns the tool and validated args map, or a deterministic error. No command must be executed on error.
package toolcatalog

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValidateToolCall returns the catalog tool and validated arguments for the given tool id and JSON arguments.
// If tool id is unknown or arguments fail validation (required, type, allowed_values, pattern, min/max), returns an error.
func ValidateToolCall(catalog *Catalog, toolID string, argsJSON string) (*Tool, map[string]any, error) {
	if catalog == nil {
		return nil, nil, fmt.Errorf("tool catalog: unknown tool %q", toolID)
	}
	tool, ok := catalog.Tools[toolID]
	if !ok {
		return nil, nil, fmt.Errorf("tool catalog: unknown tool %q", toolID)
	}
	var args map[string]any
	if strings.TrimSpace(argsJSON) != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return nil, nil, fmt.Errorf("tool %q: invalid arguments JSON: %w", toolID, err)
		}
	}
	if args == nil {
		args = make(map[string]any)
	}
	if err := validateArgs(toolID, tool.Arguments, args); err != nil {
		return nil, nil, err
	}
	return tool, args, nil
}

func validateArgs(toolID string, rules []ArgumentRule, args map[string]any) error {
	for _, r := range rules {
		if err := validateOneArg(toolID, r, args); err != nil {
			return err
		}
	}
	return nil
}

func validateOneArg(toolID string, r ArgumentRule, args map[string]any) error {
	v, ok := args[r.Name]
	if !ok || v == nil {
		if r.Required {
			return fmt.Errorf("tool %q: missing required argument %q", toolID, r.Name)
		}
		return nil
	}
	if err := validateArgType(toolID, r.Name, r.Type, v); err != nil {
		return err
	}
	if err := validateAllowedValues(toolID, r.Name, r.AllowedValues, v); err != nil {
		return err
	}
	if err := validatePattern(toolID, r.Name, r.Pattern, v); err != nil {
		return err
	}
	return validateMinMax(toolID, r.Name, r.Min, r.Max, v)
}

func validateAllowedValues(toolID, name string, allowed []string, v any) error {
	if len(allowed) == 0 {
		return nil
	}
	s := argToString(v)
	for _, a := range allowed {
		if s == a {
			return nil
		}
	}
	return fmt.Errorf("tool %q: argument %q must be one of %v", toolID, name, allowed)
}

func validatePattern(toolID, name, pattern string, v any) error {
	if pattern == "" {
		return nil
	}
	s := argToString(v)
	matched, err := regexp.MatchString(pattern, s)
	if err != nil {
		return fmt.Errorf("tool %q: argument %q pattern: %w", toolID, name, err)
	}
	if !matched {
		return fmt.Errorf("tool %q: argument %q does not match pattern %q", toolID, name, pattern)
	}
	return nil
}

func validateMinMax(toolID, name string, min, max *int, v any) error {
	if min == nil && max == nil {
		return nil
	}
	n, err := argToNumber(v)
	if err != nil {
		return fmt.Errorf("tool %q: argument %q must be number for min/max: %w", toolID, name, err)
	}
	if min != nil && n < *min {
		return fmt.Errorf("tool %q: argument %q must be >= %d", toolID, name, *min)
	}
	if max != nil && n > *max {
		return fmt.Errorf("tool %q: argument %q must be <= %d", toolID, name, *max)
	}
	return nil
}

func validateArgType(toolID, name, typ string, v any) error {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "string":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("tool %q: argument %q must be string", toolID, name)
		}
	case "integer", "int":
		switch val := v.(type) {
		case float64:
			if val != float64(int64(val)) {
				return fmt.Errorf("tool %q: argument %q must be integer", toolID, name)
			}
		case int, int64:
		default:
			return fmt.Errorf("tool %q: argument %q must be integer", toolID, name)
		}
	case "number":
		switch v.(type) {
		case float64, int, int64:
		default:
			return fmt.Errorf("tool %q: argument %q must be number", toolID, name)
		}
	case "boolean", "bool":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("tool %q: argument %q must be boolean", toolID, name)
		}
	default:
		if _, ok := v.(string); !ok {
			return fmt.Errorf("tool %q: argument %q must be string", toolID, name)
		}
	}
	return nil
}

func argToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(v)
	}
}

func argToNumber(v any) (int, error) {
	switch val := v.(type) {
	case float64:
		return int(val), nil
	case int:
		return val, nil
	case int64:
		return int(val), nil
	default:
		return 0, fmt.Errorf("not a number: %T", v)
	}
}
