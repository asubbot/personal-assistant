# Validation Tool

Multi-purpose validation tool for SDLC pipeline. Currently validates Acceptance Criteria (AC) coverage.

## AC Validation

The `validate` tool automatically validates that all Acceptance Criteria (AC) from an epic's `ep-acceptance-criteria.md` are covered by tests.

### Purpose

Before completing an epic's audit, use this tool to ensure:
- ✅ Every non-deferred AC-EE.NNN has traceability from a test comment (see [Test coverage declaration](#test-coverage-declaration)) or is explicitly **deferred** in `ep-acceptance-criteria.md`
- ✅ AC codes in tests are found via the supported comment shapes (`covers` / `supporting`, EP-N AC-, label form, REQ+AC on the same line, etc.)
- ✅ No AC is silently missed without traceability or documented deferral
- ✅ Every top-level `func Test…` under `tests/`, `internal/`, and `cmd/` has at least one **AC trace line** bound to it (see [Test functions must declare AC trace](#test-functions-must-declare-ac-trace-reverse-check))

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

Epic       Trace%       Status
────────────────────────────────────
✓ EP-001        93%
✗ EP-004        82%
...

────────────────────────────────────

❌ OVERALL: in-scope 96/111 traced (86.5%), automated 96 (86.5%), manual-only 0 | deferred 2 | total ACs 113
   Project-wide: Test functions with t.Skip: 0

❌ AC not covered by tests (project-wide): 15

EP-009
  • AC-09.001
  ...

Tip: run `./bin/validate EP-XXX` for per-AC detail and test refs.

❌ Test functions without AC trace comment (project-wide): N
  • internal/foo/bar_test.go::TestBaz
  ...

Action: Add a trace line (e.g. `// Covers AC-EE.NNN`) bound to each `Test*` per this document.
```

**Trace%** is traceability **in scope** (non-deferred ACs only): `(automated + manual-only) / in_scope`. Deferred ACs are **not** counted in the numerator; they reduce `in_scope` instead of inflating the percentage.

When a one-line criterion is parsed from `ep-acceptance-criteria.md` (not a markdown table row), it may appear after `—` on each bullet.

Use this for:
- Project health check
- Identifying which epics need work
- Seeing the full list of AC ids still missing `Covers AC-…` traceability
- Tracking overall AC coverage trend
- CI/CD dashboard reporting

#### Single Epic (Detailed View)

Output (table format; human mode prints the banner — JSON mode prints JSON only):
```
🔍 Validating AC coverage for EP-009...

📋 AC Coverage Report for EP-009

AC Code         Criterion                                          Coverage
───────────────────────────────────────────────────────────────────────────────────────────────
✓ AC-09.008                                                        5 tests
✓ AC-09.009                                                        3 tests
✗ AC-09.001                                                        NOT COVERED
↷ AC-09.005                                                        DEFERRED
✎ AC-09.007                                                        MANUAL …
...

⚠️ RESULT: in-scope 15/16 traced (93.8%), automated 14 (87.5%), manual-only 1 | deferred 1 | total ACs 18
   Project-wide: Test functions with t.Skip: 3

❌ Missing coverage for:
  • AC-09.001
  • AC-09.006
  ...

Action: Add tests for missing ACs or defer them in ep-acceptance-criteria.md

❌ Test functions without AC trace comment (project-wide): …
  • …
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
  "deferred_acs": 1,
  "in_scope_acs": 17,
  "automated_covered_acs": 14,
  "manual_only_traced_acs": 1,
  "traceability_ratio": 0.8824,
  "automated_ratio": 0.8235,
  "test_funcs_with_skip": 3,
  "tests_missing_ac_trace": ["internal/foo/bar_test.go::TestBaz"],
  "gaps": [
    {"code": "AC-09.001", "criterion": "", "status": "not_covered"},
    {"code": "AC-09.005", "criterion": "", "status": "deferred", "reason": "Deferred in ep-acceptance-criteria.md"},
    ...
  ],
  "ac_to_tests": {
    "AC-09.008": [
      {"ref": "internal/tools/create_tool_test.go::TestFunc1", "manual": false}
    ]
  }
}
```

**All epics** (`./bin/validate --json`) includes the same aggregate fields (`in_scope_acs`, `traceability_ratio`, `automated_ratio`, `test_funcs_with_skip`, …), plus `not_covered_acs` (flat list with `epic`, `code`, optional `criterion`) and `not_covered_count`, and **`tests_missing_ac_trace`**: a sorted list of `path/to/file_test.go::TestName` for top-level tests missing an AC trace (same scan roots as coverage).

For **`./bin/validate EP-XXX --json`**, `tests_missing_ac_trace` is still the **full repository** list (not limited to ACs belonging to that epic), so CI and local runs can fix every stray `Test*` in one pass.

## Metrics

For each epic (and for the all-epics JSON aggregate):

| Field | Meaning |
|-------|---------|
| `in_scope_acs` | `total_acs - deferred_acs` — ACs that still require test traceability (deferred ACs are excluded from this count). |
| `automated_covered_acs` | In-scope ACs with at least one **non-manual** test reference. |
| `manual_only_traced_acs` | In-scope ACs where **only** manual references exist (see below). |
| `traceability_ratio` | `(automated_covered_acs + manual_only_traced_acs) / in_scope_acs` — deferred are **not** in the numerator. |
| `automated_ratio` | `automated_covered_acs / in_scope_acs`. |
| `test_funcs_with_skip` | Project-wide count of `Test*` functions whose body contains `t.Skip` (direct call on `t`); scanned under `tests/`, `internal/`, `cmd/`. |
| `tests_missing_ac_trace` | (JSON only, when non-empty) Sorted `rel/path_test.go::TestName` entries for top-level `Test*` functions without a bound AC trace line (see below). |

## Test functions must declare AC trace (reverse check)

In addition to **AC → tests**, the tool enforces **test → AC**: every top-level `func Test\w+` in scanned `*_test.go` files must have at least one qualifying trace line **bound** to that function (same attribution as coverage: `testFuncForTraceLine`). **`TestMain` is excluded.** `Benchmark*`, `Example*`, and `Fuzz*` are not checked.

A line counts as an **AC trace** only if **both** hold:

1. `lineDeclaresACCoverage(line)` is true (same rules as the coverage scanner — e.g. `covers` / `supporting`, `// AC-EE.NNN:`, EP+AC lines, `REQ-` + AC on the same line).
2. The line contains at least one parseable **`AC-EE.NNN`** (`extractACsFromLine` is non-empty).

So a comment like `// Covers integration` **without** an `AC-EE.NNN` code does **not** satisfy the reverse check, even though it contains the word “covers”.

## Test Coverage Declaration

### Automatic vs manual traceability

A test reference is **manual** if either:

- The traceability line contains the whole word `manual` (case-insensitive), e.g. `// manual Covers AC-01.004`, or
- The `Test*` function that owns the trace line contains a direct **`t.Skip(...)`** call (parsed via Go AST). If both apply, manual wins for that reference.

If an AC has **only** manual references, it counts toward `manual_only_traced_acs` and `traceability_ratio`, but **not** toward `automated_ratio`. If an AC has **any** non-manual reference, it counts as **automated** for `automated_ratio`.

Trace lines are attributed to a `Test*` function by resolving the **next** `func Test…` after the line (comment-above style), or else the **most recent** `func Test…` at or before the line (comment inside the function body).

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

### Epic-prefixed manual test files

Operator scenarios live in `ai-sdlc-artefacts/epics/EP-XXX/ep-manual-tests.md` (or `ep-manual-test-scenarios.md` for EP-001). To anchor those ACs in code **without** mixing with automated tests, use dedicated files under `tests/integration/`:

- [`ep001_manual_test.go`](../../../tests/integration/ep001_manual_test.go), [`ep004_manual_test.go`](../../../tests/integration/ep004_manual_test.go), [`ep009_manual_test.go`](../../../tests/integration/ep009_manual_test.go) (EP-002 and EP-006 use automated-only traces; no dedicated `ep002` / `ep006` manual files)

Conventions: `//go:build integration`, `package integration_test`, `// manual Covers AC-…` on the trace line, `t.Skip("manual: …")` with a pointer to the epic manual doc (and optional anchor). `./bin/validate` reads these files like any other `*_test.go` under `tests/`.

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
    coverage=$(jq '.traceability_ratio' /tmp/report.json)
    if (( $(echo "$coverage < 1.0" | bc -l) )); then
      echo "❌ Not all ACs covered"
      exit 1
    fi
```

### In SDLC Pipeline

See [Stage 9 (Task Execution)](../specification/skills/09-task-execution.skill.md), [Stage 10 (Code review)](../specification/skills/10-code-review.skill.md), and [Stage 11 (Audit)](../specification/skills/11-audit.skill.md).

## Architecture

**Location:** `ai-sdlc/tools/validate/` (multi-purpose validation tool)

**Sources:** `main.go` (CLI, parsing, reports), `ast_skip.go` (`parseTestFuncsWithTSkip`, `t.Skip` detection), `test_ac_trace.go` (`findTestsMissingACTrace`, reverse check), `policy_nolint_gocyclo.go` (`findNolintGocycloViolations`, human output for policy failures), `output.go` (stdout helpers for `fmt.Fprintf` / forbidigo), `main_test.go`, `policy_nolint_gocyclo_test.go`.

**Core functions:**
- `findNolintGocycloViolations()` — Walk the same `tests/`, `internal/`, `cmd/` trees; flag lines where `nolint:gocyclo` appears outside double-quoted strings (AGENTS.md); merge into `has_gaps` / JSON `nolint_gocyclo_violations`
- `parseACsFromFile()` — Extract AC-EE.NNN from markdown (multiple heading/link shapes)
- `findCoverageInCodebase()` — Walk `tests/`, `internal/`, and `cmd/` for `*_test.go`; use `lineDeclaresACCoverage()` + `extractACsFromLine()` + `lineDeclaresManualTrace()` + `testFuncForTraceLine()` + `parseTestFuncsWithTSkip()`
- `findTestsMissingACTrace()` — Second pass over the same trees: top-level `Test*` without a bound AC trace line (exit 1 + stdout list when non-empty; JSON field `tests_missing_ac_trace`)
- `filterCoverageForEpicNum()` — Filter global map per epic
- `generateReport()` — Build coverage report (deferred gaps; metrics use `in_scope_acs` without inflating deferred into the traceability numerator)
- `validateAllEpics()` / `scanEpicsAgainstCoverage()` — one codebase scan for all epics
- `printTable()` / `printAllEpicsHuman()` — Human-readable output

**Tests:** `main_test.go` — parses AC markdown, coverage detection, report shape, `lineDeclaresACCoverage`, range extraction, etc.

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

- **0** — All checks passed: every in-scope AC is traced, every scanned `Test*` has an AC trace line, and the AGENTS.md gocyclo-suppression policy scan is clean ✅
- **1** — Failure: at least one in-scope AC has no test trace, **or** at least one top-level `Test*` is missing a bound AC trace comment, **or** `findNolintGocycloViolations` reported one or more paths ❌

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
# If complete: ready for code review and audit (stages 10–11)
```

### 3. Before Audit

```bash
# Last check before epic completion
./bin/validate EP-009

# If gaps exist: list them for manual review/deferral
# If all covered: proceed to stages 10–11 (code review, then audit)
```

### 4. Deferring ACs

If an AC cannot be reasonably tested (e.g., "Docker image available"), document in `ep-acceptance-criteria.md` near the AC (e.g. `DEFERRED`, `MANUAL ONLY`, or `**Status:** … Deferred …`). The tool marks that AC as **deferred** in the report; it does **not** require a `Covers AC-…` line in tests for that AC.

Optional: add a normal test comment if you still want traceability for partial automation, e.g. `// Covers AC-09.005` only if the test actually contributes — the validator does **not** treat `// Deferred AC-…` as a coverage marker (use markdown deferral for the defer).

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

The parser matches `AC-EE.NNN` in many line shapes, including `**AC-09.001**`, `### AC-09.001`, and `[AC-09.001](...)`. Prefer the same bold form as in your epic template for consistency.

### JSON output is invalid or mixed with log lines

In JSON mode, **stdout** is only the JSON document (no banner lines). Use `--json` before or after the epic id:
```bash
./bin/validate --json EP-009
./bin/validate EP-009 --json
```
Diagnostics and usage errors go to **stderr**.

