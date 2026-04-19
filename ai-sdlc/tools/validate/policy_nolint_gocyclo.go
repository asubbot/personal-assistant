package main

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

const forbiddenNolintGocycloSubstring = "nolint:gocyclo"

// matchOutsideStrings reports whether the first occurrence of needle in line is not inside
// a double-quoted Go string (so literals that mention the forbidden directive substring are ignored).
func matchOutsideStrings(line, needle string) bool {
	idx := strings.Index(line, needle)
	if idx < 0 {
		return false
	}
	inString := false
	escaped := false
	for i := 0; i < idx; i++ {
		c := line[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
		}
	}
	return !inString
}

// findNolintGocycloViolations scans *.go under tests/, internal/, and cmd/ for the
// forbidden golangci-lint directive (AGENTS.md).
func findNolintGocycloViolations(codebasePath string) ([]string, error) {
	root, err := os.OpenRoot(codebasePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	fsys := root.FS()

	var out []string
	for _, dir := range []string{"tests", "internal", "cmd"} {
		if _, statErr := root.Stat(dir); os.IsNotExist(statErr) {
			continue
		} else if statErr != nil {
			return nil, statErr
		}
		walkErr := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, errWalk error) error {
			if errWalk != nil {
				// Skip entries we cannot stat; continue the walk.
				return nil //nolint:nilerr // skip path-level walk errors; continue scanning other files
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			content, errRead := root.ReadFile(path)
			if errRead != nil {
				return nil //nolint:nilerr // skip unreadable file and continue walk
			}
			lines := strings.Split(string(content), "\n")
			for i, line := range lines {
				if matchOutsideStrings(line, forbiddenNolintGocycloSubstring) {
					out = append(out, fmt.Sprintf("%s:%d", path, i+1))
				}
			}
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	sort.Strings(out)
	return out, nil
}

func printNolintGocycloViolationsHuman(violations []string) {
	if len(violations) == 0 {
		return
	}
	writelnStdout("")
	writeStdout("❌ Forbidden nolint:gocyclo in product Go code (project-wide): %d\n", len(violations))
	for _, v := range violations {
		writeStdout("  • %s\n", v)
	}
	writelnStdout("")
	writeStdout("Action: Remove directives and refactor per AGENTS.md (gocyclo min-complexity applies).\n")
}
