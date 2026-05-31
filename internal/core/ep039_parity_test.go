package core

import (
	"fmt"
	"pa/internal/config"
	"strings"
	"testing"
)

// Covers AC-39.004
// Covers AC-39.019
func TestEP039_Parity_ToolResultPromptBytes(t *testing.T) {
	tests := []struct {
		name      string
		artifacts *config.ToolOutputArtifactsConfig
		wantLimit int
		omitTail  int
	}{
		{
			name:      "historical_hardcoded_default",
			artifacts: nil,
			wantLimit: maxToolResultPromptBytes,
			omitTail:  73,
		},
		{
			name: "operator_default_8192",
			artifacts: &config.ToolOutputArtifactsConfig{
				ToolResultPromptBytes: 8192,
			},
			wantLimit: 8192,
			omitTail:  41,
		},
		{
			name: "custom_limit_4096",
			artifacts: &config.ToolOutputArtifactsConfig{
				ToolResultPromptBytes: 4096,
			},
			wantLimit: 4096,
			omitTail:  37,
		},
		{
			name: "custom_limit_16384",
			artifacts: &config.ToolOutputArtifactsConfig{
				ToolResultPromptBytes: 16384,
			},
			wantLimit: 16384,
			omitTail:  512,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := minimalConfigForRun()
			if tc.artifacts != nil {
				cfg.Tools.ToolOutputArtifacts = tc.artifacts
			}
			gotLimit := toolResultPromptBytesFromConfig(cfg)
			if gotLimit != tc.wantLimit {
				t.Fatalf("toolResultPromptBytesFromConfig = %d, want %d", gotLimit, tc.wantLimit)
			}
			h := testHandlerDeps{toolResultPromptBytes: gotLimit}.handler()
			large := strings.Repeat("x", gotLimit+tc.omitTail)
			truncated := h.truncateToolResultForPrompt(large)
			marker := fmt.Sprintf("[tool output truncated: %d bytes omitted]", tc.omitTail)
			if !strings.Contains(truncated, marker) {
				t.Fatalf("truncation marker %q missing in output tail: %q", marker, truncated[len(truncated)-120:])
			}
			if truncated == large {
				t.Fatal("expected large content to be truncated")
			}
		})
	}
}
