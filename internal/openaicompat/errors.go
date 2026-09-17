package openaicompat

import (
	"encoding/json"
	"net/http"
	"strings"
)

// APIError is the structured OpenAI-compatible error object (machine codes + message).
type APIError struct {
	Message string
	Code    string
	Type    string
}

// DecodeError decodes error.message, error.code, and error.type from resp.
// The caller retains ownership of resp.Body. Missing JSON fields stay empty;
// null error.code is treated as empty. Message is the JSON message only (not resp.Status).
func DecodeError(resp *http.Response) APIError {
	var errBody struct {
		Error struct {
			Message string          `json:"message"`
			Type    string          `json:"type"`
			Code    json.RawMessage `json:"code"`
		} `json:"error"`
	}
	if resp != nil && resp.Body != nil {
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
	}
	return APIError{
		Message: strings.TrimSpace(errBody.Error.Message),
		Type:    strings.TrimSpace(errBody.Error.Type),
		Code:    decodeJSONString(errBody.Error.Code),
	}
}

func decodeJSONString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

// DecodeErrorMessage decodes an OpenAI-compatible error message from resp.
// The caller retains ownership of resp.Body.
func DecodeErrorMessage(resp *http.Response) string {
	msg := DecodeError(resp).Message
	if msg != "" {
		return msg
	}
	if resp != nil {
		return resp.Status
	}
	return ""
}
