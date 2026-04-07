package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseACsFromFile(t *testing.T) {
	// Create temporary markdown file
	tmpFile, err := os.CreateTemp("", "ac-test-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `# EP-009 Acceptance Criteria

## Acceptance Criteria

### AC-09.001 First criterion
Trigger: something happens

### AC-09.002 Second criterion
Expected: something works

**AC-09.003** (Trace: REQ-09.003)
Some requirement
`

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	_ = tmpFile.Close()

	acs, deferred, err := parseACsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseACsFromFile failed: %v", err)
	}

	// Check that we found the ACs
	if len(acs) == 0 {
		t.Fatal("Expected to find ACs, but got none")
	}

	// Check specific ACs
	if _, ok := acs[ACCode("AC-09.003")]; !ok {
		t.Error("Expected AC-09.003 to be found")
	}
	if len(deferred) != 0 {
		t.Errorf("Expected no deferred ACs, got %d", len(deferred))
	}

	t.Logf("Found %d ACs: %v", len(acs), acs)
}

func TestGetEpicNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"EP-009", "09"},  // 3-digit → 2-digit
		{"EP-001", "01"},  // 3-digit → 2-digit
		{"EP-99", "99"},   // 2-digit stays 2-digit
		{"EP-100", "100"}, // 3-digit but not leading zero
		{"009", "09"},     // Just number, convert
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getEpicNumber(tt.input)
			if result != tt.expected {
				t.Errorf("getEpicNumber(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateReport(t *testing.T) {
	acs := map[ACCode]string{
		"AC-09.001": "First criterion",
		"AC-09.002": "Second criterion",
		"AC-09.003": "Third criterion",
	}
	deferred := map[ACCode]bool{}

	coverage := map[ACCode][]TestRef{
		"AC-09.001": {TestRef("tests/test.go::TestFunc1")},
		"AC-09.002": {TestRef("tests/test.go::TestFunc2")},
		// AC-09.003 has no coverage
	}

	r := generateReport("EP-009", acs, deferred, coverage)

	if r.Epic != "EP-009" {
		t.Errorf("Epic = %s, want EP-009", r.Epic)
	}

	if r.TotalACs != 3 {
		t.Errorf("TotalACs = %d, want 3", r.TotalACs)
	}

	if r.CoveredACs != 2 {
		t.Errorf("CoveredACs = %d, want 2", r.CoveredACs)
	}

	if len(r.Gaps) != 1 {
		t.Errorf("Gaps = %d, want 1", len(r.Gaps))
	}

	if r.Gaps[0].Code != "AC-09.003" {
		t.Errorf("First gap = %s, want AC-09.003", r.Gaps[0].Code)
	}

	expectedRatio := 2.0 / 3.0
	if r.CoverageRatio != expectedRatio {
		t.Errorf("CoverageRatio = %f, want %f", r.CoverageRatio, expectedRatio)
	}
}

func TestGenerateReport_FullCoverage(t *testing.T) {
	acs := map[ACCode]string{
		"AC-09.001": "First criterion",
		"AC-09.002": "Second criterion",
	}
	deferred := map[ACCode]bool{}

	coverage := map[ACCode][]TestRef{
		"AC-09.001": {TestRef("tests/test.go::TestFunc1")},
		"AC-09.002": {TestRef("tests/test.go::TestFunc2")},
	}

	r := generateReport("EP-009", acs, deferred, coverage)

	if r.CoveredACs != r.TotalACs {
		t.Errorf("CoveredACs = %d, want %d (full coverage)", r.CoveredACs, r.TotalACs)
	}

	if r.CoverageRatio != 1.0 {
		t.Errorf("CoverageRatio = %f, want 1.0", r.CoverageRatio)
	}

	if len(r.Gaps) != 0 {
		t.Errorf("Gaps = %d, want 0", len(r.Gaps))
	}
}

func TestGenerateReport_DeferredAC(t *testing.T) {
	acs := map[ACCode]string{
		"AC-09.001": "First criterion",
		"AC-09.002": "Second criterion",
		"AC-09.003": "Third criterion",
	}
	deferred := map[ACCode]bool{
		"AC-09.003": true,
	}

	coverage := map[ACCode][]TestRef{
		"AC-09.001": {TestRef("tests/test.go::TestFunc1")},
		"AC-09.002": {TestRef("tests/test.go::TestFunc2")},
	}

	r := generateReport("EP-009", acs, deferred, coverage)

	if r.DeferredACs != 1 {
		t.Errorf("DeferredACs = %d, want 1", r.DeferredACs)
	}
	if r.CoverageRatio != 1.0 {
		t.Errorf("CoverageRatio = %f, want 1.0", r.CoverageRatio)
	}
	if hasBlockingGaps(r) {
		t.Error("Expected no blocking gaps when uncovered AC is deferred")
	}
}

func TestFindCoverageInCodebase(t *testing.T) {
	// Create temporary test directory structure
	tmpDir, err := os.MkdirTemp("", "coverage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create tests subdirectory
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.Mkdir(testsDir, 0755); err != nil {
		t.Fatalf("Failed to create tests dir: %v", err)
	}

	// Create test file with Covers comments
	testFile := filepath.Join(testsDir, "example_test.go")
	testContent := `package tests

// Covers AC-09.001: test for criterion 1
func TestFunc1(t *testing.T) {
	// test code
}

// Covers AC-09.002, AC-09.003: test for criteria 2 and 3
func TestFunc2(t *testing.T) {
	// test code
}
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Find coverage
	coverage, err := findCoverageInCodebase(tmpDir)
	if err != nil {
		t.Fatalf("findCoverageInCodebase failed: %v", err)
	}

	// Check that we found coverage for AC-09.001
	if tests, ok := coverage[ACCode("AC-09.001")]; !ok {
		t.Error("Expected AC-09.001 to be found in coverage")
	} else if len(tests) == 0 {
		t.Error("Expected AC-09.001 to have at least one test")
	}

	t.Logf("Coverage: %v", coverage)
}

func TestParseACsFromFile_Deferred(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "ac-deferred-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `# EP-009 Acceptance Criteria

**AC-09.005** (Trace: [REQ-09.005](ep-requirements.md#docker-sandbox-execution))

Given something
When something
Then something
**Status:** Deferred to operations team.
`

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	_ = tmpFile.Close()

	acs, deferred, err := parseACsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseACsFromFile failed: %v", err)
	}

	if len(acs) != 1 {
		t.Fatalf("Expected 1 AC, got %d", len(acs))
	}
	if !deferred[ACCode("AC-09.005")] {
		t.Fatal("Expected AC-09.005 to be marked deferred")
	}
}

func TestNormalizeEpicNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		ok       bool
	}{
		{"EP-009", "09", true},
		{"EP-9", "09", true},
		{"9", "09", true},
		{"EP-100", "100", true},
		{"bad", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeEpicNumber(tt.input)
			if tt.ok && err != nil {
				t.Fatalf("normalizeEpicNumber returned unexpected error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("normalizeEpicNumber expected error, got nil")
			}
			if tt.ok && got != tt.expected {
				t.Fatalf("normalizeEpicNumber(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLineDeclaresACCoverage(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"// Covers AC-09.001", true},
		{"// covers AC-01.029, AC-01.030", true},
		{"// Supporting AC-06.002", true},
		{"// supporting AC-06.002", true},
		{"// EP-008 AC-08.001 / REQ-08.001: default_temperature", true},
		{"// AC-04.025: tools.text_based_enabled", true},
		{"// each … (AC-06.005, AC-06.010 / REQ-06.013).", true},
		{"// (AC-06.005, AC-06.006, AC-06.010 / REQ-06.013)", true},
		{"// no keyword and no pattern", false},
		{"func foo() {}", false},
	}
	for _, tt := range tests {
		if got := lineDeclaresACCoverage(tt.line); got != tt.want {
			t.Errorf("lineDeclaresACCoverage(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestFindCoverageInCodebase_CmdAndFormats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "coverage-cmd-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	for _, sub := range []string{"tests", "internal", "cmd"} {
		if err := os.Mkdir(filepath.Join(tmpDir, sub), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// cmd/pa — EP-008 style
	cmdTest := filepath.Join(tmpDir, "cmd", "pa", "x_test.go")
	if err := os.MkdirAll(filepath.Join(tmpDir, "cmd", "pa"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cmdTest, []byte(`package pa

// EP-008 AC-08.001 / REQ-08.001: body
func TestA(t *testing.T) {}
`), 0644); err != nil {
		t.Fatal(err)
	}

	// internal — lowercase covers
	intTest := filepath.Join(tmpDir, "internal", "pkg", "y_test.go")
	if err := os.MkdirAll(filepath.Join(tmpDir, "internal", "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(intTest, []byte(`package pkg

// TestFoo covers AC-01.029
func TestFoo(t *testing.T) {}
`), 0644); err != nil {
		t.Fatal(err)
	}

	cov, err := findCoverageInCodebase(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cov[ACCode("AC-08.001")]; !ok {
		t.Error("expected AC-08.001 from cmd/ EP-008 style comment")
	}
	if _, ok := cov[ACCode("AC-01.029")]; !ok {
		t.Error("expected AC-01.029 from lowercase covers comment")
	}
}

func TestNormalizeCriterionPreview(t *testing.T) {
	if got := normalizeCriterionPreview("| col | col |"); got != "" {
		t.Errorf("table row should be empty, got %q", got)
	}
	if got := normalizeCriterionPreview("Given a user\nWhen x"); got == "" {
		t.Error("Gherkin line should be kept")
	}
}

func TestJsonOutputRequested(t *testing.T) {
	if !jsonOutputRequested(true, []string{"EP-009"}) {
		t.Error("flag true should request JSON")
	}
	if !jsonOutputRequested(false, []string{"EP-009", "--json"}) {
		t.Error("tail --json should request JSON (Go flag stops at first non-flag)")
	}
	if jsonOutputRequested(false, []string{"EP-009"}) {
		t.Error("no --json should not request JSON")
	}
}
