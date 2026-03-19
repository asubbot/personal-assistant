package escalationpolicy

import (
	"errors"
	"pa/internal/core/toolfailure"
	"pa/internal/toolcatalog"
)

// WrapCatalogValidateError maps errors from toolcatalog.ValidateToolCall to typed tool failures.
// Only *toolcatalog.ValidateError is mapped; other errors fail closed (NoEscalate) (REQ-06.017).
func WrapCatalogValidateError(err error) error {
	if err == nil {
		return nil
	}
	var ve *toolcatalog.ValidateError
	if errors.As(err, &ve) {
		if ve == nil || ve.Kind == 0 {
			return toolfailure.NoEscalate(err)
		}
		if ve.Kind == toolcatalog.ValidateKindUnknownTool {
			return toolfailure.NoEscalate(err)
		}
		return toolfailure.MayEscalate(err)
	}
	return toolfailure.NoEscalate(err)
}
