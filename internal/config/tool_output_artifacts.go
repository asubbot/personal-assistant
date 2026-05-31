package config

import (
	"fmt"
	"strings"
)

var allowedToolOutputArtifactsKeys = []string{
	"enabled",
	"directory",
	"tool_result_prompt_bytes",
	"max_artifact_bytes",
	"omission_marker",
	"preview_min_tail_bytes",
	"max_stderr_bytes_in_prompt",
	"max_reads_per_turn",
	"max_read_bytes_per_turn",
	"max_bytes_per_read",
	"retention_max_total_bytes",
	"retention_max_files",
}

// ArtifactDirectory returns the resolved artifact storage directory when enabled; empty otherwise.
func ArtifactDirectory(cfg *Config) string {
	if cfg == nil || cfg.Tools == nil || cfg.Tools.ToolOutputArtifacts == nil {
		return ""
	}
	a := cfg.Tools.ToolOutputArtifacts
	if !a.Enabled {
		return ""
	}
	return strings.TrimSpace(a.Directory)
}

func validateToolOutputArtifacts(a *ToolOutputArtifactsConfig) error {
	if a == nil {
		return nil
	}
	const prefix = "tools.tool_output_artifacts"
	if a.Enabled && strings.TrimSpace(a.Directory) == "" {
		return fmt.Errorf("config: %s.directory is required when enabled is true", prefix)
	}
	if strings.TrimSpace(a.OmissionMarker) == "" {
		return fmt.Errorf("config: %s.omission_marker is required", prefix)
	}
	if a.PreviewMinTailBytes < 0 {
		return fmt.Errorf("config: %s.preview_min_tail_bytes must be >= 0", prefix)
	}
	for _, field := range []struct {
		name string
		v    int
	}{
		{"tool_result_prompt_bytes", a.ToolResultPromptBytes},
		{"max_artifact_bytes", a.MaxArtifactBytes},
		{"max_stderr_bytes_in_prompt", a.MaxStderrBytesInPrompt},
		{"max_reads_per_turn", a.MaxReadsPerTurn},
		{"max_read_bytes_per_turn", a.MaxReadBytesPerTurn},
		{"max_bytes_per_read", a.MaxBytesPerRead},
		{"retention_max_total_bytes", a.RetentionMaxTotalBytes},
		{"retention_max_files", a.RetentionMaxFiles},
	} {
		if field.v < 1 {
			return fmt.Errorf("config: %s.%s must be >= 1", prefix, field.name)
		}
	}
	return nil
}
