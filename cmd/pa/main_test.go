package main

import (
	"os"
	"pa/internal/config"
	"path/filepath"
	"testing"
)

// No AC: config path derived from PA_CONFIG_DIR (implementation detail).
func TestConfigFilePath_PAConfigDirSet(t *testing.T) {
	dir := "/etc/pa"
	_ = os.Setenv("PA_CONFIG_DIR", dir)
	t.Cleanup(func() { _ = os.Unsetenv("PA_CONFIG_DIR") })

	got := configFilePath()
	want := filepath.Join(dir, config.ConfigFileName)
	if got != want {
		t.Errorf("configFilePath() = %q, want %q", got, want)
	}
}

// No AC: config path derived from PA_CONFIG_DIR (implementation detail).
func TestConfigFilePath_PAConfigDirUnsetOrEmpty(t *testing.T) {
	_ = os.Unsetenv("PA_CONFIG_DIR")
	t.Cleanup(func() { _ = os.Unsetenv("PA_CONFIG_DIR") })

	got := configFilePath()
	want := filepath.Join("./config", config.ConfigFileName)
	if got != want {
		t.Errorf("configFilePath() = %q, want %q", got, want)
	}
}
