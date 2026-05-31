package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadConfigFixture loads testdata/<name>.json and fails the test on error.
func loadConfigFixture(t *testing.T, name string) *Config {
	t.Helper()
	path := loadConfigFixtureRaw(t, name)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}
	return cfg
}

// loadConfigFixtureRaw returns the path to testdata/<name>.json without loading.
func loadConfigFixtureRaw(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name+".json")
}

// writeConfigFixtureToDir copies testdata/<fixtureName>.json into dir/config.json,
// applying repl on the file body (e.g. "__CFG_DIR__" -> absolute temp dir).
func writeConfigFixtureToDir(t *testing.T, dir, fixtureName string, repl map[string]string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", fixtureName+".json"))
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixtureName, err)
	}
	s := string(src)
	for old, new := range repl {
		s = strings.ReplaceAll(s, old, new)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(s), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
