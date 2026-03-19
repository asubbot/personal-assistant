package escalationpolicy

import (
	"errors"
	"pa/internal/core/toolfailure"
	"pa/internal/toolcatalog"
	"testing"
)

// Covers AC-06.014 (REQ-06.017): policy mapping lives in this package and is unit-tested without handler.
// Supporting REQ-06.005 (unknown tool → no escalation).
func TestWrapCatalogValidateError_unknownTool_noEscalate(t *testing.T) {
	_, _, err := toolcatalog.ValidateToolCall(nil, "x", `{}`)
	if err == nil {
		t.Fatal("expected validate error")
	}
	w := WrapCatalogValidateError(err)
	if toolfailure.QualifiesForEscalation(w) {
		t.Fatal("unknown tool should not qualify")
	}
}

// Covers AC-06.015 (REQ-06.018): catalog path uses ValidateKind via errors.As, not substring matching.
func TestWrapCatalogValidateError_invalidJSON_mayEscalate(t *testing.T) {
	catalog := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{"t": {ID: "t", IndexText: "x", Template: "c", NodeID: "n"}}}
	_, _, err := toolcatalog.ValidateToolCall(catalog, "t", `{bad}`)
	if err == nil {
		t.Fatal("expected validate error")
	}
	w := WrapCatalogValidateError(err)
	if !toolfailure.QualifiesForEscalation(w) {
		t.Fatal("invalid JSON from catalog should qualify")
	}
}

// Supporting AC-06.014 (REQ-06.017).
func TestWrapCatalogValidateError_missingArg_mayEscalate(t *testing.T) {
	catalog := &toolcatalog.Catalog{
		Tools: map[string]*toolcatalog.Tool{
			"t": {
				ID: "t", IndexText: "x", Template: "cmd", NodeID: "n",
				Arguments: []toolcatalog.ArgumentRule{{Name: "node", Type: "string", Required: true}},
			},
		},
	}
	_, _, err := toolcatalog.ValidateToolCall(catalog, "t", `{}`)
	if err == nil {
		t.Fatal("expected validate error")
	}
	w := WrapCatalogValidateError(err)
	if !toolfailure.QualifiesForEscalation(w) {
		t.Fatal("missing required argument should qualify")
	}
}

// Covers AC-06.014: fail closed on arbitrary errors (not *toolcatalog.ValidateError).
func TestWrapCatalogValidateError_unrecognized_failClosed(t *testing.T) {
	err := errors.New(`tool "t": something unexpected from future code`)
	w := WrapCatalogValidateError(err)
	if toolfailure.QualifiesForEscalation(w) {
		t.Fatal("unrecognized error must not qualify (fail closed)")
	}
}

// Supporting AC-06.014 (REQ-06.017).
func TestWrapCatalogValidateError_nil(t *testing.T) {
	if WrapCatalogValidateError(nil) != nil {
		t.Fatal("want nil")
	}
}
