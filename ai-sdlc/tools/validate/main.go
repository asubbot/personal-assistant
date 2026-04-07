package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type (
	ACCode  string
	TestRef string
)

type Report struct {
	Epic          string              `json:"epic"`
	TotalACs      int                 `json:"total_acs"`
	CoveredACs    int                 `json:"covered_acs"`
	DeferredACs   int                 `json:"deferred_acs"`
	CoverageRatio float64             `json:"coverage_ratio"`
	Gaps          []ACGap             `json:"gaps"`
	Coverage      map[string][]string `json:"ac_to_tests"`
}

type ACGap struct {
	Code      string `json:"code"`
	Criterion string `json:"criterion"`
	Status    string `json:"status"` // "not_covered" | "deferred"
	Reason    string `json:"reason,omitempty"`
}

type EpicSummary struct {
	Epic          string  `json:"epic"`
	TotalACs      int     `json:"total_acs"`
	CoveredACs    int     `json:"covered_acs"`
	DeferredACs   int     `json:"deferred_acs"`
	CoverageRatio float64 `json:"coverage_ratio"`
}

// ProjectNotCoveredAC is one AC that has no test traceability comment (project-wide run).
type ProjectNotCoveredAC struct {
	Epic      string `json:"epic"`
	Code      string `json:"code"`
	Criterion string `json:"criterion,omitempty"`
}

type AllEpicsReport struct {
	Epics           []EpicSummary         `json:"epics"`
	NotCoveredACs   []ProjectNotCoveredAC `json:"not_covered_acs"`
	NotCoveredCount int                   `json:"not_covered_count"`
	TotalACs        int                   `json:"total_acs"`
	CoveredACs      int                   `json:"covered_acs"`
	DeferredACs     int                   `json:"deferred_acs"`
	CoverageRatio   float64               `json:"coverage_ratio"`
	HasGaps         bool                  `json:"has_gaps"`
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
)

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
func findCoverageInCodebase(codebasePath string) (map[ACCode][]TestRef, error) {
	coverage := make(map[ACCode][]TestRef)

	// Walk tests/, internal/, and cmd/ (main packages may hold *_test.go with AC comments).
	for _, dir := range []string{"tests", "internal", "cmd"} {
		dirPath := filepath.Join(codebasePath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Skip entries we cannot stat; continue the walk.
				return nil //nolint:nilerr // skip path-level walk errors; continue scanning other files
			}

			if !strings.HasSuffix(path, "_test.go") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil //nolint:nilerr // skip unreadable test file and continue walk
			}

			fileContent := string(content)
			relPath := strings.TrimPrefix(path, codebasePath)
			relPath = strings.TrimPrefix(relPath, "/")

			lines := strings.Split(fileContent, "\n")
			funcPattern := regexp.MustCompile(`func (Test\w+)`)
			currentTestFunc := "unknown"

			for _, line := range lines {
				// Update current test function if we see one
				if matches := funcPattern.FindStringSubmatch(line); len(matches) > 1 {
					currentTestFunc = matches[1]
				}

				if lineDeclaresACCoverage(line) {
					testRef := TestRef(fmt.Sprintf("%s::%s", relPath, currentTestFunc))

					// Extract ACs using more sophisticated parsing
					// Handles: AC-09.001, AC-09.001–003, AC-09.001, AC-09.002
					acs := extractACsFromLine(line)
					for _, ac := range acs {
						coverage[ac] = append(coverage[ac], testRef)
					}
				}
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return coverage, nil
}

func filterCoverageForEpicNum(coverage map[ACCode][]TestRef, epicNum string) map[ACCode][]TestRef {
	out := make(map[ACCode][]TestRef)
	for ac, tests := range coverage {
		if strings.Contains(string(ac), "-"+epicNum+".") {
			out[ac] = tests
		}
	}
	return out
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

// generateReport creates the AC coverage report
func generateReport(epic string, acs map[ACCode]string, deferred map[ACCode]bool, coverage map[ACCode][]TestRef) *Report {
	r := &Report{
		Epic:     epic,
		TotalACs: len(acs),
		Coverage: make(map[string][]string),
	}

	for ac, testRefs := range coverage {
		tests := make([]string, len(testRefs))
		for i, t := range testRefs {
			tests[i] = string(t)
		}
		r.Coverage[string(ac)] = tests
	}

	// Find gaps
	for ac := range acs {
		if len(coverage[ac]) == 0 {
			if deferred[ac] {
				r.Gaps = append(r.Gaps, ACGap{
					Code:   string(ac),
					Status: "deferred",
					Reason: "Deferred in ep-acceptance-criteria.md",
				})
				r.DeferredACs++
				continue
			}
			r.Gaps = append(r.Gaps, ACGap{
				Code:   string(ac),
				Status: "not_covered",
			})
		} else {
			r.CoveredACs++
		}
	}

	if r.TotalACs > 0 {
		r.CoverageRatio = float64(r.CoveredACs+r.DeferredACs) / float64(r.TotalACs)
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

func acCoverageCell(tests []string, deferred bool) (status, testStr string) {
	switch {
	case len(tests) == 0 && deferred:
		return "↷", "DEFERRED"
	case len(tests) == 0:
		return "✗", "NOT COVERED"
	case len(tests) == 1:
		return "✓", tests[0]
	default:
		return "✓", fmt.Sprintf("%d tests", len(tests))
	}
}

// printTable prints a formatted table report
func printTable(r *Report, acs map[ACCode]string, deferred map[ACCode]bool) {
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

		tests := r.Coverage[string(ac)]
		status, testStr := acCoverageCell(tests, deferred[ac])

		if len(testStr) > 27 {
			testStr = testStr[:24] + "..."
		}

		writeStdout("%s %-15s %-50s %-30s\n", status, string(ac), criterion, testStr)
	}

	writelnStdout(strings.Repeat("─", 95))

	coverageStr := "❌"
	if r.CoverageRatio == 1.0 {
		coverageStr = "✅"
	} else if r.CoverageRatio >= 0.9 {
		coverageStr = "⚠️"
	}

	writeStdout("\n%s RESULT: %d covered, %d deferred, %d total (%.1f%%)\n", coverageStr, r.CoveredACs, r.DeferredACs, r.TotalACs, r.CoverageRatio*100)

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
	writelnStdout("")
}

func scanEpicsAgainstCoverage(cwd string, epics []string, globalCoverage map[ACCode][]TestRef) ([]EpicSummary, []ProjectNotCoveredAC, bool) {
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
			Epic:          epic,
			TotalACs:      r.TotalACs,
			CoveredACs:    r.CoveredACs,
			DeferredACs:   r.DeferredACs,
			CoverageRatio: r.CoverageRatio,
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

func printAllEpicsHuman(
	results []EpicSummary,
	projectNotCovered []ProjectNotCoveredAC,
	totalCovered, totalDeferred, totalACs int,
	overallRatio float64,
	hasGaps bool,
) {
	writelnStdout("📋 Epic Validation Summary")
	writelnStdout("")
	writeStdout("%-10s %-12s %-12s\n", "Epic", "Coverage", "Status")
	writelnStdout(strings.Repeat("─", 36))

	for _, res := range results {
		status := "✓"
		if res.CoveredACs+res.DeferredACs < res.TotalACs {
			status = "✗"
		}
		pct := int(res.CoverageRatio * 100)
		writeStdout("%s %-12s %3d%% %s\n", status, res.Epic, pct, "")
	}

	writelnStdout(strings.Repeat("─", 36))

	statusEmoji := "✅"
	if hasGaps {
		statusEmoji = "❌"
	}

	writeStdout("\n%s OVERALL: %d covered, %d deferred, %d total (%.1f%%)\n", statusEmoji, totalCovered, totalDeferred, totalACs, overallRatio*100)

	if !hasGaps {
		return
	}

	writelnStdout("")
	writeStdout("❌ AC not covered by tests (project-wide): %d\n", len(projectNotCovered))
	if len(projectNotCovered) > 0 {
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
	writelnStdout("Tip: run `./bin/validate EP-XXX` for per-AC detail and test refs.")
	os.Exit(1)
}

func aggregateEpicTotals(results []EpicSummary) (totalACs, totalCovered, totalDeferred int, overallRatio float64) {
	for _, res := range results {
		totalACs += res.TotalACs
		totalCovered += res.CoveredACs
		totalDeferred += res.DeferredACs
	}
	if totalACs > 0 {
		overallRatio = float64(totalCovered+totalDeferred) / float64(totalACs)
	}
	return totalACs, totalCovered, totalDeferred, overallRatio
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

	globalCoverage, err := findCoverageInCodebase(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning codebase: %v\n", err)
		os.Exit(1)
	}

	results, projectNotCovered, hasGaps := scanEpicsAgainstCoverage(cwd, epics, globalCoverage)

	sort.Slice(projectNotCovered, func(i, j int) bool {
		if projectNotCovered[i].Epic != projectNotCovered[j].Epic {
			return projectNotCovered[i].Epic < projectNotCovered[j].Epic
		}
		return projectNotCovered[i].Code < projectNotCovered[j].Code
	})

	totalACs, totalCovered, totalDeferred, overallRatio := aggregateEpicTotals(results)

	allReport := AllEpicsReport{
		Epics:           results,
		NotCoveredACs:   projectNotCovered,
		NotCoveredCount: len(projectNotCovered),
		TotalACs:        totalACs,
		CoveredACs:      totalCovered,
		DeferredACs:     totalDeferred,
		CoverageRatio:   overallRatio,
		HasGaps:         hasGaps,
	}

	if jsonOutput {
		writeAllEpicsJSON(allReport, hasGaps)
		return
	}

	printAllEpicsHuman(results, projectNotCovered, totalCovered, totalDeferred, totalACs, overallRatio, hasGaps)
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

func main() {
	jsonFlag := flag.Bool("json", false, "Output report as JSON instead of table")
	flag.Parse()
	jsonOut := jsonOutputRequested(*jsonFlag, os.Args[1:])

	args := flag.Args()

	// Default to "all" if no argument provided
	epic := "all"
	if len(args) > 0 {
		epic = args[0]
	}

	// Handle "all" to validate all epics
	if epic == "all" {
		validateAllEpics(jsonOut)
		return
	}

	epicNum, err := normalizeEpicNumber(epic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Find repo root
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Print what we're validating (human mode only; JSON must be stdout-clean for CI)
	if !jsonOut {
		writeStdout("🔍 Validating AC coverage for %s...\n\n", epic)
	}

	// Locate ep-acceptance-criteria.md
	acPath := filepath.Join(cwd, "ai-sdlc-artefacts", "epics", epic, "ep-acceptance-criteria.md")
	if _, err := os.Stat(acPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: %s not found\n", acPath)
		fmt.Fprintf(os.Stderr, "\nUsage:\n")
		fmt.Fprintf(os.Stderr, "  validate          - Validate all epics (default)\n")
		fmt.Fprintf(os.Stderr, "  validate EP-009   - Validate single epic\n")
		fmt.Fprintf(os.Stderr, "  validate --json   - JSON output\n")
		os.Exit(1)
	}

	// Parse ACs from file
	acs, deferred, err := parseACsFromFile(acPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing AC file: %v\n", err)
		os.Exit(1)
	}

	// Find coverage in codebase
	coverage, err := findCoverageInCodebase(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning codebase: %v\n", err)
		os.Exit(1)
	}

	epicCoverage := filterCoverageForEpicNum(coverage, epicNum)

	// Generate report
	r := generateReport(epic, acs, deferred, epicCoverage)

	if jsonOut {
		// Output JSON
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		writelnStdout(string(data))
	} else {
		// Output table
		printTable(r, acs, deferred)
	}

	// Exit with error if coverage is incomplete
	if hasBlockingGaps(r) {
		os.Exit(1)
	}
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
