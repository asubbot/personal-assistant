package escalationpolicy

import (
	"errors"
	"fmt"
	"pa/internal/core/toolfailure"
	"testing"
)

// Covers AC-06.014 (REQ-06.017): policy mapping lives in this package and is unit-tested without handler.
// Supporting REQ-06.005 (unknown tool → no escalation).
func TestWrapCatalogValidateError_unknownTool_noEscalate(t *testing.T) {
	err := fmt.Errorf(`tool catalog: unknown tool %q`, "x")
	w := WrapCatalogValidateError(err)
	if toolfailure.QualifiesForEscalation(w) {
		t.Fatal("unknown tool should not qualify")
	}
}

// Supporting AC-06.014 (REQ-06.017).
func TestWrapCatalogValidateError_invalidJSON_mayEscalate(t *testing.T) {
	inner := errors.New("syntax")
	err := fmt.Errorf(`tool %q: invalid arguments JSON: %w`, "t", inner)
	w := WrapCatalogValidateError(err)
	if !toolfailure.QualifiesForEscalation(w) {
		t.Fatal("invalid JSON from catalog should qualify")
	}
}

// Supporting AC-06.014 (REQ-06.017).
func TestWrapCatalogValidateError_missingArg_mayEscalate(t *testing.T) {
	err := fmt.Errorf(`tool %q: missing required argument %q`, "t", "node")
	w := WrapCatalogValidateError(err)
	if !toolfailure.QualifiesForEscalation(w) {
		t.Fatal("missing required argument should qualify")
	}
}

// Covers AC-06.014: fail closed on arbitrary errors (not from ValidateToolCall).
func TestWrapCatalogValidateError_unrecognized_failClosed(t *testing.T) {
	err := errors.New(`tool "t": something unexpected from future code`)
	w := WrapCatalogValidateError(err)
	if toolfailure.QualifiesForEscalation(w) {
		t.Fatal("unrecognized validate-shaped error must not qualify (fail closed)")
	}
}

// Supporting AC-06.014 (REQ-06.017).
func TestWrapCatalogValidateError_nil(t *testing.T) {
	if WrapCatalogValidateError(nil) != nil {
		t.Fatal("want nil")
	}
}
