package openaicompat

import (
	"encoding/json"
	"net/http"
)

// DecodeErrorMessage decodes an OpenAI-compatible error message from resp.
// The caller retains ownership of resp.Body.
func DecodeErrorMessage(resp *http.Response) string {
	var errBody struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody.Error.Message != "" {
		return errBody.Error.Message
	}
	return resp.Status
}
