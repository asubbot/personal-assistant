package main

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

const testMainName = "TestMain"

// findTestsMissingACTrace walks tests/, internal/, and cmd/ like findCoverageInCodebase and
// returns sorted "rel/path/to/file_test.go::TestName" entries for top-level Test functions that
// have no AC trace comment bound to them (same rules as coverage: lineDeclaresACCoverage,
// extractACsFromLine non-empty, testFuncForTraceLine). TestMain is excluded.
func findTestsMissingACTrace(codebasePath string) ([]string, error) {
	root, err := os.OpenRoot(codebasePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	fsys := root.FS()

	var out []string
	for _, dir := range []string{"tests", "internal", "cmd"} {
		if _, err := root.Stat(dir); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		err = fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, errWalk error) error {
			if errWalk != nil {
				// Skip entries we cannot stat; continue the walk.
				return nil //nolint:nilerr // skip path-level walk errors; continue scanning other files
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			content, errRead := root.ReadFile(path)
			if errRead != nil {
				return nil //nolint:nilerr // skip unreadable test file and continue walk
			}
			out = append(out, testFuncsMissingACTraceInFile(path, string(content))...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

// testFuncsMissingACTraceInFile returns refs relPath::TestName for Test* functions without an AC trace.
func testFuncsMissingACTraceInFile(relPath, fileContent string) []string {
	lines := strings.Split(fileContent, "\n")
	testNames := topLevelTestFuncNames(lines)
	if len(testNames) == 0 {
		return nil
	}
	traced := make(map[string]struct{})
	for i, line := range lines {
		if !lineDeclaresACCoverage(line) || len(extractACsFromLine(line)) == 0 {
			continue
		}
		name := testFuncForTraceLine(lines, i)
		if name != "" && name != "unknown" {
			traced[name] = struct{}{}
		}
	}
	var out []string
	for _, name := range testNames {
		if name == testMainName {
			continue
		}
		if _, ok := traced[name]; ok {
			continue
		}
		out = append(out, fmt.Sprintf("%s::%s", relPath, name))
	}
	return out
}

func topLevelTestFuncNames(lines []string) []string {
	var names []string
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if m := funcTestPattern.FindStringSubmatch(s); len(m) > 1 {
			names = append(names, m[1])
		}
	}
	return names
}
