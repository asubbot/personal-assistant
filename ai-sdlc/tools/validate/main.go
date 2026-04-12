package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type ACCode string

// CoverageRef is one test function reference with optional manual-only classification.
type CoverageRef struct {
	Ref    string `json:"ref"`
	Manual bool   `json:"manual"`
}

type Report struct {
	Epic                string                   `json:"epic"`
	TotalACs            int                      `json:"total_acs"`
	DeferredACs         int                      `json:"deferred_acs"`
	InScopeACs          int                      `json:"in_scope_acs"`
	AutomatedCoveredACs int                      `json:"automated_covered_acs"`
	ManualOnlyTracedACs int                      `json:"manual_only_traced_acs"`
	TraceabilityRatio   float64                  `json:"traceability_ratio"`
	AutomatedRatio      float64                  `json:"automated_ratio"`
	TestFuncsWithSkip   int                      `json:"test_funcs_with_skip"`
	TestsMissingACTrace []string                 `json:"tests_missing_ac_trace,omitempty"`
	Gaps                []ACGap                  `json:"gaps"`
	Coverage            map[string][]CoverageRef `json:"ac_to_tests"`
}

type ACGap struct {
	Code      string `json:"code"`
	Criterion string `json:"criterion"`
	Status    string `json:"status"` // "not_covered" | "deferred"
	Reason    string `json:"reason,omitempty"`
}

type EpicSummary struct {
	Epic                string  `json:"epic"`
	TotalACs            int     `json:"total_acs"`
	DeferredACs         int     `json:"deferred_acs"`
	InScopeACs          int     `json:"in_scope_acs"`
	AutomatedCoveredACs int     `json:"automated_covered_acs"`
	ManualOnlyTracedACs int     `json:"manual_only_traced_acs"`
	TraceabilityRatio   float64 `json:"traceability_ratio"`
	AutomatedRatio      float64 `json:"automated_ratio"`
}

// ProjectNotCoveredAC is one AC that has no test traceability comment (project-wide run).
type ProjectNotCoveredAC struct {
	Epic      string `json:"epic"`
	Code      string `json:"code"`
	Criterion string `json:"criterion,omitempty"`
}

type AllEpicsReport struct {
	Epics               []EpicSummary         `json:"epics"`
	NotCoveredACs       []ProjectNotCoveredAC `json:"not_covered_acs"`
	NotCoveredCount     int                   `json:"not_covered_count"`
	TotalACs            int                   `json:"total_acs"`
	DeferredACs         int                   `json:"deferred_acs"`
	InScopeACs          int                   `json:"in_scope_acs"`
	AutomatedCoveredACs int                   `json:"automated_covered_acs"`
	ManualOnlyTracedACs int                   `json:"manual_only_traced_acs"`
	TraceabilityRatio   float64               `json:"traceability_ratio"`
	AutomatedRatio      float64               `json:"automated_ratio"`
	TestFuncsWithSkip   int                   `json:"test_funcs_with_skip"`
	TestsMissingACTrace []string              `json:"tests_missing_ac_trace,omitempty"`
	HasGaps             bool                  `json:"has_gaps"`
}

// parseACsFromFile extracts all AC-EE.NNN codes from an acceptance criteria markdown file.
func parseACsFromFile(path string) (map[ACCode]string, map[ACCode]bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	acs := make(map[ACCode]string)
	deferred := make(map[ACCode]bool)
	lines := strings.Split(string(content), "\n")

	// Capture AC code in common formats:
	// - **AC-09.001**
	// - ### AC-09.001
	// - [AC-09.001](...)
	acCodePattern := regexp.MustCompile(`AC-(\d{2})\.(\d{3})`)

	for i, line := range lines {
		matches := acCodePattern.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}

		for _, match := range matches {
			code := fmt.Sprintf("AC-%s.%s", match[1], match[2])
			ac := ACCode(code)

			if _, exists := acs[ac]; !exists {
				criterion := strings.TrimSpace(acCodePattern.ReplaceAllString(line, ""))
				criterion = strings.Trim(criterion, "*[]()#:- \t")

				if criterion == "" && i+1 < len(lines) {
					criterion = strings.TrimSpace(lines[i+1])
				}
				acs[ac] = criterion
			}

			if isDeferredAC(lines, i, code) {
				deferred[ac] = true
			}
		}
	}

	return acs, deferred, nil
}

// normalizeCriterionPreview drops table/index lines from ep-acceptance-criteria.md that are not useful as a one-line hint.
func normalizeCriterionPreview(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return ""
	}
	// Markdown table rows (e.g. index) — not a criterion sentence
	if strings.HasPrefix(c, "|") {
		return ""
	}
	if strings.Count(c, "|") >= 3 {
		return ""
	}
	if len(c) > 160 {
		return c[:157] + "..."
	}
	return c
}

func isDeferredAC(lines []string, idx int, code string) bool {
	targetUpper := strings.ToUpper(code)
	start := idx - 3
	if start < 0 {
		start = 0
	}
	end := idx + 6
	if end >= len(lines) {
		end = len(lines) - 1
	}

	for i := start; i <= end; i++ {
		lineUpper := strings.ToUpper(lines[i])
		if strings.Contains(lineUpper, targetUpper) &&
			(strings.Contains(lineUpper, "DEFERRED") || strings.Contains(lineUpper, "MANUAL ONLY")) {
			return true
		}
		if strings.Contains(lineUpper, "STATUS:") && strings.Contains(lineUpper, "DEFERRED") {
			return true
		}
	}
	return false
}

// Regexes for common traceability comment shapes (see project test style).
var (
	reEpicACLine = regexp.MustCompile(`EP-\d+\s+AC-\d{2}\.\d{3}`)
	reACLabel    = regexp.MustCompile(`\bAC-\d{2}\.\d{3}\s*:`)
	reACSlashReq = regexp.MustCompile(`\bAC-\d{2}\.\d{3}\s*/\s*REQ-`)
	reACCode     = regexp.MustCompile(`AC-\d{2}\.\d{3}`)
	reManualWord = regexp.MustCompile(`(?i)\bmanual\b`)
)

// lineDeclaresManualTrace is true when the traceability line explicitly marks manual-only intent.
func lineDeclaresManualTrace(line string) bool {
	return reManualWord.MatchString(line)
}

// lineDeclaresACCoverage returns true if a line in a *_test.go file should be scanned for AC codes.
// Accepts: "Covers AC-…" / "Supporting AC-…" (case-insensitive), EP-NNN AC-…, AC-EE.NNN:, AC / REQ, REQ + AC on same line.
func lineDeclaresACCoverage(line string) bool {
	lineLower := strings.ToLower(line)
	if strings.Contains(lineLower, "covers") || strings.Contains(lineLower, "supporting") {
		return true
	}
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "//") {
		return false
	}
	if reEpicACLine.MatchString(line) {
		return true
	}
	if reACLabel.MatchString(line) {
		return true
	}
	if reACSlashReq.MatchString(line) {
		return true
	}
	// e.g. (AC-06.005, AC-06.006, … AC-06.010 / REQ-06.013)
	if strings.Contains(line, "REQ-") && reACCode.MatchString(line) {
		return true
	}
	return false
}

// findCoverageInCodebase searches for AC traceability comments in test files.
// The second return value is the number of Test* functions whose body contains t.Skip (project-wide).
func findCoverageInCodebase(codebasePath string) (map[ACCode][]CoverageRef, int, error) {
	coverage := make(map[ACCode][]CoverageRef)
	testFuncsWithSkip := 0

	root, err := os.OpenRoot(codebasePath)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = root.Close() }()

	fsys := root.FS()

	for _, dir := range []string{"tests", "internal", "cmd"} {
		n, werr := appendCoverageFromTestDir(root, fsys, dir, coverage)
		if werr != nil {
			return nil, 0, werr
		}
		testFuncsWithSkip += n
	}

	return coverage, testFuncsWithSkip, nil
}

// appendCoverageFromTestDir walks dir under root (tests, internal, or cmd) and merges AC coverage into coverage.
func appendCoverageFromTestDir(root *os.Root, fsys fs.FS, dir string, coverage map[ACCode][]CoverageRef) (int, error) {
	if _, err := root.Stat(dir); os.IsNotExist(err) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	var testFuncsWithSkip int
	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip entries we cannot stat; continue the walk.
			return nil //nolint:nilerr // skip path-level walk errors; continue scanning other files
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := root.ReadFile(path)
		if err != nil {
			return nil //nolint:nilerr // skip unreadable test file and continue walk
		}

		skipMap, errParse := parseTestFuncsWithTSkip(content, path)
		if errParse != nil {
			skipMap = map[string]bool{}
		} else {
			testFuncsWithSkip += countTestFuncsWithSkip(skipMap)
		}

		fileContent := string(content)
		// path from fs.WalkDir is relative to codebasePath with '/' separators.
		relPath := path

		lines := strings.Split(fileContent, "\n")

		for i, line := range lines {
			if !lineDeclaresACCoverage(line) {
				continue
			}
			testName := testFuncForTraceLine(lines, i)
			manual := lineDeclaresManualTrace(line) || skipMap[testName]
			ref := CoverageRef{
				Ref:    fmt.Sprintf("%s::%s", relPath, testName),
				Manual: manual,
			}

			// Extract ACs using more sophisticated parsing
			// Handles: AC-09.001, AC-09.001–003, AC-09.001, AC-09.002
			acs := extractACsFromLine(line)
			for _, ac := range acs {
				coverage[ac] = append(coverage[ac], ref)
			}
		}

		return nil
	})
	return testFuncsWithSkip, err
}

func filterCoverageForEpicNum(coverage map[ACCode][]CoverageRef, epicNum string) map[ACCode][]CoverageRef {
	out := make(map[ACCode][]CoverageRef)
	for ac, tests := range coverage {
		if strings.Contains(string(ac), "-"+epicNum+".") {
			out[ac] = tests
		}
	}
	return out
}

var funcTestPattern = regexp.MustCompile(`func (Test\w+)`)

// testFuncForTraceLine resolves which Test* function a traceability line belongs to:
// prefer the next Test function after the line (comment-above style); else the most recent Test at or before the line (comment inside body).
func testFuncForTraceLine(lines []string, lineIdx int) string {
	for j := lineIdx + 1; j < len(lines); j++ {
		if m := funcTestPattern.FindStringSubmatch(lines[j]); len(m) > 1 {
			return m[1]
		}
	}
	for j := lineIdx; j >= 0; j-- {
		if m := funcTestPattern.FindStringSubmatch(lines[j]); len(m) > 1 {
			return m[1]
		}
	}
	return "unknown"
}

// extractACsFromLine extracts all AC codes from a traceability line
// Handles formats like: AC-09.001, AC-09.001–003, AC-09.001, AC-09.002
func extractACsFromLine(line string) []ACCode {
	var result []ACCode
	singlePattern := regexp.MustCompile(`AC-(\d{2})\.(\d{3})`)

	// Find all AC patterns
	matches := singlePattern.FindAllStringSubmatch(line, -1)
	acMap := make(map[ACCode]bool) // deduplicate

	for _, match := range matches {
		epic := match[1]
		num := match[2]
		code := fmt.Sprintf("AC-%s.%s", epic, num)
		acMap[ACCode(code)] = true
	}

	// Check for ranges like AC-09.008–013
	// First find the base AC, then check if there's a range
	rangePattern := regexp.MustCompile(`AC-(\d{2})\.(\d{3})[–-](\d{3})`)
	rangeMatches := rangePattern.FindAllStringSubmatch(line, -1)

	for _, match := range rangeMatches {
		epic := match[1]
		startStr := match[2]
		endStr := match[3]

		start, err1 := strconv.Atoi(startStr)
		end, err2 := strconv.Atoi(endStr)
		if err1 != nil || err2 != nil {
			continue
		}

		for i := start; i <= end; i++ {
			code := fmt.Sprintf("AC-%s.%03d", epic, i)
			acMap[ACCode(code)] = true
		}
	}

	// Convert to sorted slice
	for ac := range acMap {
		result = append(result, ac)
	}
	sort.Slice(result, func(i, j int) bool {
		return string(result[i]) < string(result[j])
	})

	return result
}

// generateReport creates the AC coverage report.
func generateReport(epic string, acs map[ACCode]string, deferred map[ACCode]bool, coverage map[ACCode][]CoverageRef) *Report {
	r := &Report{
		Epic:     epic,
		TotalACs: len(acs),
		Coverage: make(map[string][]CoverageRef),
	}

	for ac, testRefs := range coverage {
		r.Coverage[string(ac)] = append([]CoverageRef(nil), testRefs...)
	}

	for ac := range acs {
		if deferred[ac] {
			r.Gaps = append(r.Gaps, ACGap{
				Code:   string(ac),
				Status: "deferred",
				Reason: "Deferred in ep-acceptance-criteria.md",
			})
			r.DeferredACs++
			continue
		}
		refs := coverage[ac]
		if len(refs) == 0 {
			r.Gaps = append(r.Gaps, ACGap{
				Code:   string(ac),
				Status: "not_covered",
			})
			continue
		}
		hasAuto := false
		for _, ref := range refs {
			if !ref.Manual {
				hasAuto = true
				break
			}
		}
		if hasAuto {
			r.AutomatedCoveredACs++
		} else {
			r.ManualOnlyTracedACs++
		}
	}

	r.InScopeACs = r.TotalACs - r.DeferredACs
	if r.InScopeACs > 0 {
		traced := r.AutomatedCoveredACs + r.ManualOnlyTracedACs
		r.TraceabilityRatio = float64(traced) / float64(r.InScopeACs)
		r.AutomatedRatio = float64(r.AutomatedCoveredACs) / float64(r.InScopeACs)
	}

	// Sort gaps for consistent output
	sort.Slice(r.Gaps, func(i, j int) bool {
		return r.Gaps[i].Code < r.Gaps[j].Code
	})

	return r
}

func hasBlockingGaps(r *Report) bool {
	for _, gap := range r.Gaps {
		if gap.Status == "not_covered" {
			return true
		}
	}
	return false
}

func acCoverageCell(refs []CoverageRef, deferred bool) (status, testStr string) {
	if deferred {
		return "↷", "DEFERRED"
	}
	if len(refs) == 0 {
		return "✗", "NOT COVERED"
	}
	hasAuto := false
	for _, ref := range refs {
		if !ref.Manual {
			hasAuto = true
			break
		}
	}
	if !hasAuto {
		if len(refs) == 1 {
			return "✎", "MANUAL " + refs[0].Ref
		}
		return "✎", fmt.Sprintf("%d MANUAL", len(refs))
	}
	if len(refs) == 1 {
		return "✓", refs[0].Ref
	}
	return "✓", fmt.Sprintf("%d tests", len(refs))
}

// printTable prints a formatted table report. testFuncsWithSkip is project-wide Test* count with t.Skip.
// testsMissingACTrace is project-wide (same list as full-repo validate), not filtered to the current epic.
func printTable(r *Report, acs map[ACCode]string, deferred map[ACCode]bool, testFuncsWithSkip int, testsMissingACTrace []string) {
	writeStdout("\n📋 AC Coverage Report for %s\n\n", r.Epic)
	writeStdout("%-15s %-50s %-30s\n", "AC Code", "Criterion", "Coverage")
	writelnStdout(strings.Repeat("─", 95))

	keys := make([]ACCode, 0, len(acs))
	for ac := range acs {
		keys = append(keys, ac)
	}
	sort.Slice(keys, func(i, j int) bool {
		return string(keys[i]) < string(keys[j])
	})

	for _, ac := range keys {
		criterion := acs[ac]
		if len(criterion) > 47 {
			criterion = criterion[:44] + "..."
		}

		refs := r.Coverage[string(ac)]
		status, testStr := acCoverageCell(refs, deferred[ac])

		if len(testStr) > 27 {
			testStr = testStr[:24] + "..."
		}

		writeStdout("%s %-15s %-50s %-30s\n", status, string(ac), criterion, testStr)
	}

	writelnStdout(strings.Repeat("─", 95))

	coverageStr := "❌"
	if r.TraceabilityRatio == 1.0 {
		coverageStr = "✅"
	} else if r.TraceabilityRatio >= 0.9 {
		coverageStr = "⚠️"
	}

	writeStdout("\n%s RESULT: in-scope %d/%d traced (%.1f%%), automated %d (%.1f%%), manual-only %d | deferred %d | total ACs %d\n",
		coverageStr,
		r.AutomatedCoveredACs+r.ManualOnlyTracedACs, r.InScopeACs, r.TraceabilityRatio*100,
		r.AutomatedCoveredACs, r.AutomatedRatio*100,
		r.ManualOnlyTracedACs,
		r.DeferredACs, r.TotalACs,
	)
	writeStdout("   Project-wide: Test functions with t.Skip: %d\n", testFuncsWithSkip)

	if hasBlockingGaps(r) {
		writeStdout("\n❌ Missing coverage for:\n")
		for _, gap := range r.Gaps {
			if gap.Status == "not_covered" {
				writeStdout("  • %s\n", gap.Code)
			}
		}
		writeStdout("\nAction: Add tests for missing ACs or defer them in ep-acceptance-criteria.md\n")
	} else {
		writeStdout("\n✅ All ACs covered — epic is ready for audit\n")
	}

	if len(testsMissingACTrace) > 0 {
		writeStdout("\n❌ Test functions without AC trace comment (project-wide): %d\n", len(testsMissingACTrace))
		for _, ref := range testsMissingACTrace {
			writeStdout("  • %s\n", ref)
		}
		writeStdout("\nAction: Add a trace line (e.g. // Covers AC-EE.NNN) bound to each Test* per VALIDATION.md\n")
	}
	writelnStdout("")
}

func scanEpicsAgainstCoverage(cwd string, epics []string, globalCoverage map[ACCode][]CoverageRef) ([]EpicSummary, []ProjectNotCoveredAC, bool) {
	var results []EpicSummary
	var projectNotCovered []ProjectNotCoveredAC
	hasGaps := false

	for _, epic := range epics {
		epicNum := getEpicNumber(epic)
		acPath := filepath.Join(cwd, "ai-sdlc-artefacts", "epics", epic, "ep-acceptance-criteria.md")
		if _, err := os.Stat(acPath); os.IsNotExist(err) {
			continue
		}
		acs, deferred, err := parseACsFromFile(acPath)
		if err != nil {
			continue
		}
		epicCoverage := filterCoverageForEpicNum(globalCoverage, epicNum)
		r := generateReport(epic, acs, deferred, epicCoverage)
		results = append(results, EpicSummary{
			Epic:                epic,
			TotalACs:            r.TotalACs,
			DeferredACs:         r.DeferredACs,
			InScopeACs:          r.InScopeACs,
			AutomatedCoveredACs: r.AutomatedCoveredACs,
			ManualOnlyTracedACs: r.ManualOnlyTracedACs,
			TraceabilityRatio:   r.TraceabilityRatio,
			AutomatedRatio:      r.AutomatedRatio,
		})
		if hasBlockingGaps(r) {
			hasGaps = true
		}
		for _, gap := range r.Gaps {
			if gap.Status != "not_covered" {
				continue
			}
			crit := gap.Criterion
			if crit == "" {
				crit = acs[ACCode(gap.Code)]
			}
			crit = normalizeCriterionPreview(crit)
			projectNotCovered = append(projectNotCovered, ProjectNotCoveredAC{
				Epic:      epic,
				Code:      gap.Code,
				Criterion: crit,
			})
		}
	}
	return results, projectNotCovered, hasGaps
}

// printProjectNotCoveredHuman prints the AC-not-covered block (may be zero items).
func printProjectNotCoveredHuman(projectNotCovered []ProjectNotCoveredAC) {
	writelnStdout("")
	writeStdout("❌ AC not covered by tests (project-wide): %d\n", len(projectNotCovered))
	if len(projectNotCovered) == 0 {
		return
	}
	currentEpic := ""
	for _, item := range projectNotCovered {
		if item.Epic != currentEpic {
			currentEpic = item.Epic
			writeStdout("\n%s\n", currentEpic)
		}
		line := fmt.Sprintf("  • %s", item.Code)
		if c := strings.TrimSpace(item.Criterion); c != "" {
			if len(c) > 72 {
				c = c[:69] + "..."
			}
			line += " — " + c
		}
		writelnStdout(line)
	}
	writelnStdout("")
}

// printTestsMissingACTraceHuman prints the Test*-without-trace block.
func printTestsMissingACTraceHuman(testsMissingACTrace []string) {
	writelnStdout("")
	writeStdout("❌ Test functions without AC trace comment (project-wide): %d\n", len(testsMissingACTrace))
	for _, ref := range testsMissingACTrace {
		writeStdout("  • %s\n", ref)
	}
	writelnStdout("")
	writeStdout("Action: Add a trace line (e.g. // Covers AC-EE.NNN) bound to each Test* per VALIDATION.md\n")
}

func printAllEpicsHuman(
	results []EpicSummary,
	projectNotCovered []ProjectNotCoveredAC,
	totalACs, totalDeferred, totalInScope, totalAuto, totalManual int,
	traceRatio, autoRatio float64,
	testFuncsWithSkip int,
	hasGaps bool,
	testsMissingACTrace []string,
) {
	writelnStdout("📋 Epic Validation Summary")
	writelnStdout("")
	writeStdout("%-10s %-12s %-12s\n", "Epic", "Trace%", "Status")
	writelnStdout(strings.Repeat("─", 36))

	for _, res := range results {
		status := "✓"
		if res.AutomatedCoveredACs+res.ManualOnlyTracedACs < res.InScopeACs {
			status = "✗"
		}
		pct := int(res.TraceabilityRatio * 100)
		writeStdout("%s %-12s %3d%% %s\n", status, res.Epic, pct, "")
	}

	writelnStdout(strings.Repeat("─", 36))

	hasTestTraceGaps := len(testsMissingACTrace) > 0
	overallFail := hasGaps || hasTestTraceGaps

	statusEmoji := "✅"
	if overallFail {
		statusEmoji = "❌"
	}

	writeStdout("\n%s OVERALL: in-scope %d/%d traced (%.1f%%), automated %d (%.1f%%), manual-only %d | deferred %d | total ACs %d\n",
		statusEmoji, totalAuto+totalManual, totalInScope, traceRatio*100,
		totalAuto, autoRatio*100, totalManual, totalDeferred, totalACs)
	writeStdout("   Project-wide: Test functions with t.Skip: %d\n", testFuncsWithSkip)

	if !overallFail {
		return
	}

	if hasGaps {
		printProjectNotCoveredHuman(projectNotCovered)
	}
	if hasTestTraceGaps {
		printTestsMissingACTraceHuman(testsMissingACTrace)
	}

	writelnStdout("Tip: run `./bin/validate EP-XXX` for per-AC detail and test refs.")
	os.Exit(1)
}

func aggregateEpicTotals(results []EpicSummary) (totalACs, totalDeferred, totalInScope, totalAuto, totalManual int, traceRatio, autoRatio float64) {
	for _, res := range results {
		totalACs += res.TotalACs
		totalDeferred += res.DeferredACs
		totalInScope += res.InScopeACs
		totalAuto += res.AutomatedCoveredACs
		totalManual += res.ManualOnlyTracedACs
	}
	if totalInScope > 0 {
		traceRatio = float64(totalAuto+totalManual) / float64(totalInScope)
		autoRatio = float64(totalAuto) / float64(totalInScope)
	}
	return totalACs, totalDeferred, totalInScope, totalAuto, totalManual, traceRatio, autoRatio
}

func writeAllEpicsJSON(allReport AllEpicsReport, hasGaps bool) {
	data, err := json.MarshalIndent(allReport, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	writelnStdout(string(data))
	if hasGaps {
		os.Exit(1)
	}
}

// findCoverageAndTestTrace runs both codebase scans used for validation.
func findCoverageAndTestTrace(cwd string) (map[ACCode][]CoverageRef, int, []string, error) {
	globalCoverage, testFuncsWithSkip, err := findCoverageInCodebase(cwd)
	if err != nil {
		return nil, 0, nil, err
	}
	testsMissingACTrace, err := findTestsMissingACTrace(cwd)
	if err != nil {
		return nil, 0, nil, err
	}
	return globalCoverage, testFuncsWithSkip, testsMissingACTrace, nil
}

// validateAllEpics finds and validates all epics in ai-sdlc-artefacts/epics/
func validateAllEpics(jsonOutput bool) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	epicsPath := filepath.Join(cwd, "ai-sdlc-artefacts", "epics")
	entries, err := os.ReadDir(epicsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading epics directory: %v\n", err)
		os.Exit(1)
	}

	var epics []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "EP-") {
			epics = append(epics, entry.Name())
		}
	}

	if len(epics) == 0 {
		fmt.Fprintf(os.Stderr, "No epics found in %s\n", epicsPath)
		os.Exit(1)
	}

	sort.Strings(epics)

	if !jsonOutput {
		writeStdout("🔍 Validating AC coverage for all %d epics...\n\n", len(epics))
	}

	globalCoverage, testFuncsWithSkip, testsMissingACTrace, err := findCoverageAndTestTrace(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning codebase: %v\n", err)
		os.Exit(1)
	}

	results, projectNotCovered, hasGaps := scanEpicsAgainstCoverage(cwd, epics, globalCoverage)
	if len(testsMissingACTrace) > 0 {
		hasGaps = true
	}

	sort.Slice(projectNotCovered, func(i, j int) bool {
		if projectNotCovered[i].Epic != projectNotCovered[j].Epic {
			return projectNotCovered[i].Epic < projectNotCovered[j].Epic
		}
		return projectNotCovered[i].Code < projectNotCovered[j].Code
	})

	totalACs, totalDeferred, totalInScope, totalAuto, totalManual, traceRatio, autoRatio := aggregateEpicTotals(results)

	allReport := AllEpicsReport{
		Epics:               results,
		NotCoveredACs:       projectNotCovered,
		NotCoveredCount:     len(projectNotCovered),
		TotalACs:            totalACs,
		DeferredACs:         totalDeferred,
		InScopeACs:          totalInScope,
		AutomatedCoveredACs: totalAuto,
		ManualOnlyTracedACs: totalManual,
		TraceabilityRatio:   traceRatio,
		AutomatedRatio:      autoRatio,
		TestFuncsWithSkip:   testFuncsWithSkip,
		TestsMissingACTrace: testsMissingACTrace,
		HasGaps:             hasGaps,
	}

	if jsonOutput {
		writeAllEpicsJSON(allReport, hasGaps)
		return
	}

	printAllEpicsHuman(results, projectNotCovered, totalACs, totalDeferred, totalInScope, totalAuto, totalManual, traceRatio, autoRatio, testFuncsWithSkip, hasGaps, testsMissingACTrace)
}

// getEpicNumber extracts the numeric part from "EP-009" → "09" (2 digits, not 3)
func getEpicNumber(epic string) string {
	num := epic
	if strings.HasPrefix(epic, "EP-") {
		num = strings.TrimPrefix(epic, "EP-")
	}
	// Remove leading zero if 3 digits (e.g., "009" → "09")
	if len(num) == 3 && strings.HasPrefix(num, "0") {
		num = num[1:]
	}
	return num
}

// jsonOutputRequested returns true if JSON output is requested. Go's flag package stops
// at the first non-flag argument, so `./bin/validate EP-009 --json` does not set
// -json; argvTail should be os.Args[1:] so --json works in any position.
func jsonOutputRequested(flagVal bool, argvTail []string) bool {
	if flagVal {
		return true
	}
	for _, a := range argvTail {
		if a == "--json" {
			return true
		}
	}
	return false
}

func validateSingleEpic(epic string, jsonOut bool) {
	epicNum, err := normalizeEpicNumber(epic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	if !jsonOut {
		writeStdout("🔍 Validating AC coverage for %s...\n\n", epic)
	}

	acPath := filepath.Join(cwd, "ai-sdlc-artefacts", "epics", epic, "ep-acceptance-criteria.md")
	if _, err := os.Stat(acPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: %s not found\n", acPath)
		fmt.Fprintf(os.Stderr, "\nUsage:\n")
		fmt.Fprintf(os.Stderr, "  validate          - Validate all epics (default)\n")
		fmt.Fprintf(os.Stderr, "  validate EP-009   - Validate single epic\n")
		fmt.Fprintf(os.Stderr, "  validate --json   - JSON output\n")
		os.Exit(1)
	}

	acs, deferred, err := parseACsFromFile(acPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing AC file: %v\n", err)
		os.Exit(1)
	}

	coverage, testFuncsWithSkip, testsMissingACTrace, err := findCoverageAndTestTrace(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning codebase: %v\n", err)
		os.Exit(1)
	}

	epicCoverage := filterCoverageForEpicNum(coverage, epicNum)
	r := generateReport(epic, acs, deferred, epicCoverage)
	r.TestFuncsWithSkip = testFuncsWithSkip
	r.TestsMissingACTrace = testsMissingACTrace

	if jsonOut {
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		writelnStdout(string(data))
	} else {
		printTable(r, acs, deferred, testFuncsWithSkip, testsMissingACTrace)
	}

	if hasBlockingGaps(r) || len(testsMissingACTrace) > 0 {
		os.Exit(1)
	}
}

func main() {
	jsonFlag := flag.Bool("json", false, "Output report as JSON instead of table")
	flag.Parse()
	jsonOut := jsonOutputRequested(*jsonFlag, os.Args[1:])

	args := flag.Args()
	epic := "all"
	if len(args) > 0 {
		epic = args[0]
	}

	if epic == "all" {
		validateAllEpics(jsonOut)
		return
	}

	validateSingleEpic(epic, jsonOut)
}

func normalizeEpicNumber(epic string) (string, error) {
	num := getEpicNumber(epic)
	if num == "" {
		return "", fmt.Errorf("invalid epic id: %s", epic)
	}
	if len(num) == 2 {
		return num, nil
	}
	n, err := strconv.Atoi(num)
	if err != nil {
		return "", fmt.Errorf("invalid epic id: %s", epic)
	}
	return fmt.Sprintf("%02d", n), nil
}
