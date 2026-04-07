# Validation Tool

Multi-purpose validation tool for SDLC pipeline. Currently validates Acceptance Criteria (AC) coverage.

## AC Validation

The `validate` tool automatically validates that all Acceptance Criteria (AC) from an epic's `ep-acceptance-criteria.md` are covered by tests.

### Purpose

Before completing an epic's audit, use this tool to ensure:
- ✅ Every AC-EE.NNN code has at least one test covering it
- ✅ Tests explicitly declare coverage with `// Covers AC-EE.NNN` comments
- ✅ No AC is silently missed without test coverage

### Usage

#### Build

```bash
make build
```

#### Validate All Epics (Default)

```bash
./bin/validate
```

#### Validate Single Epic

```bash
./bin/validate EP-009
```

Output:
```
🔍 Validating AC coverage for all 9 epics...

📋 Epic Validation Summary

Epic       Coverage     Status
────────────────────────────────────
✓ EP-001        95%
✓ EP-004        88%
✗ EP-006        82%
✗ EP-009        61%
────────────────────────────────────

❌ OVERALL: 84 covered, 2 deferred, 113 total (76.1%)

❌ AC not covered by tests (project-wide): 27

EP-009
  • AC-09.001
  • AC-09.005
  ...

Tip: run `./bin/validate EP-XXX` for per-AC detail and test refs.
```

When a one-line criterion is parsed from `ep-acceptance-criteria.md` (not a markdown table row), it may appear after `—` on each bullet.

Use this for:
- Project health check
- Identifying which epics need work
- Seeing the full list of AC ids still missing `Covers AC-…` traceability
- Tracking overall AC coverage trend
- CI/CD dashboard reporting

#### Single Epic (Detailed View)

Output (table format):
```
🔍 Validating AC coverage for EP-009...

📋 AC Coverage Report for EP-009

AC Code         Criterion                                          Coverage
───────────────────────────────────────────────────────────────────────────────────────────────
✓ AC-09.008                                                        5 tests
✓ AC-09.009                                                        3 tests
✗ AC-09.001                                                        NOT COVERED
✗ AC-09.005                                                        NOT COVERED
...

❌ RESULT: 11/18 AC covered (61.1%)

❌ Missing coverage for:
  • AC-09.001
  • AC-09.005
  • AC-09.006
  ...

Action: Add tests for missing ACs or defer them in ep-acceptance-criteria.md
```

#### JSON output: Parse results programmatically

```bash
./bin/validate --json EP-009
./bin/validate --json
```

**Single epic** output:
```json
{
  "epic": "EP-009",
  "total_acs": 18,
  "covered_acs": 11,
  "coverage_ratio": 0.6111,
  "gaps": [
    {"code": "AC-09.001", "status": "not_covered"},
    ...
  ],
  "ac_to_tests": {
    "AC-09.008": ["internal/tools/create_tool_test.go::TestFunc1", ...],
    ...
  }
}
```

**All epics** (`./bin/validate --json`) additionally includes `not_covered_acs` (flat list with `epic`, `code`, optional `criterion`) and `not_covered_count`.

## Test Coverage Declaration

### Format: Comment before test function

Mark which ACs your test covers with a comment directly above (or before) the test function:

```go
// Covers AC-09.008: create_tool accepts required parameters
func TestCreateToolTool_Run_success(t *testing.T) {
    // ...test code...
}
```

### Supported formats

Single AC:
```go
// Covers AC-09.001
func TestSomething(t *testing.T) { }
```

Multiple ACs (comma-separated):
```go
// Covers AC-09.001, AC-09.002
func TestMultipleACs(t *testing.T) { }
```

Range of ACs (using en-dash or hyphen):
```go
// Covers AC-09.008–013
func TestRangeOfACs(t *testing.T) { }
```

Supporting test (non-primary coverage):
```go
// Supporting AC-09.001: helper test
func TestHelper(t *testing.T) { }
```

Mixed:
```go
// Covers AC-09.001, AC-09.003–005, AC-09.010
func TestMixed(t *testing.T) { }
```

## Integration Points

### Before Git Commit (Optional)

Add to `settings.json` (hooks) to validate before every commit:
```json
{
  "hooks": {
    "before_git_commit": "epic=$(git rev-parse --abbrev-ref HEAD | grep -o 'EP-[0-9]*'); if [ -n \"$epic\" ]; then ./bin/validate \"$epic\"; fi"
  }
}
```

Or validate manually before committing:
```bash
./bin/validate EP-009
git commit -m "feat(EP-009): implement create_tool..."
```

### In CI/CD

Example GitHub Actions:
```yaml
- name: Validate AC coverage
  run: |
    ./bin/validate --json EP-009 > /tmp/report.json
    coverage=$(jq '.coverage_ratio' /tmp/report.json)
    if (( $(echo "$coverage < 1.0" | bc -l) )); then
      echo "❌ Not all ACs covered"
      exit 1
    fi
```

### In SDLC Pipeline

See [Stage 8 (Task Execution)](../specification/skills/08-task-execution.skill.md) and [Stage 9 (Audit)](../specification/skills/09-audit.skill.md).

## Architecture

**Location:** `ai-sdlc/tools/validate/` (multi-purpose validation tool)

**Binary:** `ai-sdlc/tools/validate/main.go` (~350 lines)

**Core functions:**
- `parseACsFromFile()` — Extract AC-EE.NNN codes from markdown
- `findCoverageInCodebase()` — Scan tests/ and internal/ for "Covers AC-" comments
- `extractACsFromLine()` — Parse AC codes including ranges (AC-09.001–005)
- `generateReport()` — Build coverage report
- `validateAllEpics()` — Validate all epics in project
- `printTable()` — Format human-readable output

**Tests:** `ai-sdlc/tools/validate/main_test.go` (~190 lines)
- Parses AC markdown
- Detects coverage in temp test files
- Validates report generation
- Tests range handling (AC-09.008–013)

## Building

```bash
# Build all binaries (pa + validate)
make build

# Run tests
go test ./ai-sdlc/tools/validate/...

# Quick rebuild if only validate changed (Make tracks dependencies)
make validate
```

## Exit Codes

- **0** — All ACs covered ✅
- **1** — Some ACs not covered ❌

## Common Workflows

### 1. Project Health Check

```bash
# Quick overview of all epics
./bin/validate

# Output shows which epics need work
```

### 2. During Development (Task Execution)

```bash
# Add test, mark it with "// Covers AC-09.001"
# Build and check coverage for specific epic
make build
./bin/validate EP-009

# If incomplete: add more tests or defer ACs
# If complete: ready for audit
```

### 3. Before Audit

```bash
# Last check before epic completion
./bin/validate EP-009

# If gaps exist: list them for manual review/deferral
# If all covered: proceed to stage 9 audit
```

### 4. Deferring ACs

If an AC cannot be reasonably tested (e.g., "Docker image available"), document in `ep-acceptance-criteria.md`:

```markdown
## AC-09.005 (DEFERRED)
Python 3.14 pa-sandbox image available
**Status:** Deferred to operations team; verified by deployment only.
```

Then update the test to acknowledge the deferral:
```go
// Deferred AC-09.005: sandbox image availability (ops manual verification)
func TestSandboxSetup(t *testing.T) {
    // ...basic setup test without full image verification...
}
```

## Troubleshooting

### No coverage found even though tests exist

1. Check test file is under `tests/`, `internal/`, or `cmd/` (only `*_test.go` files are scanned).
2. Use a traceability comment the tool recognizes, for example:
   - `// Covers AC-XX.YYY` or `// Supporting AC-…` (case-insensitive `covers` / `supporting` is OK)
   - `// EP-008 AC-08.001 / REQ-08.001: …`
   - `// AC-04.025: …` (label form)
   - `// … (AC-06.005, … AC-06.010 / REQ-06.013)` when `REQ-` appears on the same line
3. Ensure AC codes are comma/range-separated as documented; ranges like `AC-09.008–013` are supported.
4. Run debug: `grep -rE 'Covers AC-|EP-[0-9]+ AC-|AC-[0-9]{2}\\.[0-9]{3}:' tests/ internal/ cmd/`

### AC code not found in markdown

Ensure format in `ep-acceptance-criteria.md` is:
```markdown
**AC-09.001** (Trace: [REQ-09.001])
Description of criterion...
```

The parser looks for `**AC-XX.YYY**` pattern.

### JSON output is invalid or mixed with log lines

In JSON mode, **stdout** is only the JSON document (no banner lines). Use `--json` before or after the epic id:
```bash
./bin/validate --json EP-009
./bin/validate EP-009 --json
```
Diagnostics and usage errors go to **stderr**.

## Future Enhancements

- [ ] REQ code traceability (REQ-EE.NNN coverage)
- [ ] Design spec consistency checks
- [ ] Coverage trend reporting (epic-to-epic)
- [ ] HTML report generation
- [ ] Slack integration for automated reports
