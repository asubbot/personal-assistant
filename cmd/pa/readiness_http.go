package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func writeJSON(logger *slog.Logger, w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil && logger != nil {
		logger.Error("observability json encode", "error", err)
	}
}
