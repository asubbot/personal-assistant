package escalationpolicy

import (
	"pa/internal/core/toolfailure"
	"strings"
)

// WrapCatalogValidateError maps errors from toolcatalog.ValidateToolCall to typed tool failures.
// Unrecognized errors fail closed (NoEscalate) so arbitrary errors never qualify for escalation (REQ-06.017).
func WrapCatalogValidateError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "tool catalog: unknown tool") {
		return toolfailure.NoEscalate(err)
	}
	if catalogValidateErrorQualifies(msg) {
		return toolfailure.MayEscalate(err)
	}
	return toolfailure.NoEscalate(err)
}

// catalogValidateErrorQualifies returns true only for message shapes produced by
// toolcatalog.ValidateToolCall / validateArgs (stable substrings; replace with sentinels when available).
func catalogValidateErrorQualifies(msg string) bool {
	if !strings.Contains(msg, `tool "`) {
		return false
	}
	markers := []string{
		"invalid arguments JSON",
		"missing required argument",
		"must be one of",
		"does not match pattern",
		"pattern:",
		"must be >=",
		"must be <=",
		"must be number for min/max",
		"must be string",
		"must be integer",
		"must be number",
		"must be boolean",
	}
	for _, m := range markers {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}
