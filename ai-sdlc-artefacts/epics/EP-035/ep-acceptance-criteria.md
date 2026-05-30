---
artefact: ep-acceptance-criteria
epic_id: EP-035
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# EP-035 — Consolidate small internal packages — Acceptance criteria

## Introduction

Testable conditions for removing stub `internal/logging`, relocating the EP-022 concurrent-write reliability test out of `internal/reliability`, and merging `internal/promptmarkers` with `internal/systemprompt` into `internal/prompt` without changing product behaviour, explicit JSON configuration, or EP-013 security-sensitive prompt contracts. Criteria trace to [ep-requirements.md](ep-requirements.md) and [ep-scope.md](ep-scope.md). Test levels follow [strategy.md](../../strategy.md) §2 (Unit / Integration / E2E / Manual).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Test level | Summary |
|-------|-------------|------------|---------|
| [AC-35.001](#ac-35-001) | [REQ-35.001](ep-requirements.md#req-35-001--delete-logging-stub-package) | Manual | `internal/logging` directory removed |
| [AC-35.002](#ac-35-002) | [REQ-35.002](ep-requirements.md#req-35-002--no-logging-package-imports) | Manual (build/grep) | No Go imports of `pa/internal/logging` |
| [AC-35.003](#ac-35-003) | [REQ-35.003](ep-requirements.md#req-35-003--delete-reliability-test-package) | Manual | `internal/reliability` directory removed |
| [AC-35.004](#ac-35-004) | [REQ-35.004](ep-requirements.md#req-35-004--relocate-concurrent-write-test) | Integration | `TestConcurrentWrites_NoBusyErrors` lives under `tests/integration` |
| [AC-35.005](#ac-35-005) | [REQ-35.005](ep-requirements.md#req-35-005--preserve-ac-22-010-race-test-intent) | Integration | Relocated test passes under `go test -race` (EP-022 race-test intent) |
| [AC-35.006](#ac-35-006) | [REQ-35.006](ep-requirements.md#req-35-006--update-reliability-test-documentation-path) | Manual | `docs/configuration.md` cites new test path |
| [AC-35.007](#ac-35-007) | [REQ-35.007](ep-requirements.md#req-35-007--provide-merged-internalprompt-package) | Unit | `internal/prompt` exports merged marker, trust, and wrap API |
| [AC-35.008](#ac-35-008) | [REQ-35.008](ep-requirements.md#req-35-008--byte-identical-trust-policy) | Unit | `TrustPolicy` byte-identical to pre-EP-035 `systemprompt` |
| [AC-35.009](#ac-35-009) | [REQ-35.009](ep-requirements.md#req-35-009--byte-identical-marker-constants) | Unit | Six canonical marker lines byte-identical to pre-EP-035 `promptmarkers` |
| [AC-35.010](#ac-35-010) | [REQ-35.010](ep-requirements.md#req-35-010--equivalent-forbidden-marker-validation) | Unit | Forbidden-marker validation matches pre-EP-035 behaviour |
| [AC-35.011](#ac-35-011) | [REQ-35.011](ep-requirements.md#req-35-011--equivalent-block-wrap-helpers) | Unit | Wrap helpers match pre-EP-035 `systemprompt` output |
| [AC-35.012](#ac-35-012) | [REQ-35.012](ep-requirements.md#req-35-012--delete-legacy-prompt-packages) | Manual | Legacy `promptmarkers` and `systemprompt` dirs removed |
| [AC-35.013](#ac-35-013) | [REQ-35.013](ep-requirements.md#req-35-013--no-legacy-prompt-package-imports) | Manual (build/grep) | No Go imports of removed prompt packages |
| [AC-35.014](#ac-35-014) | [REQ-35.014](ep-requirements.md#req-35-014--update-prompt-package-importers) | Manual (build/grep) | Listed importers build using `internal/prompt` only |
| [AC-35.015](#ac-35-015) | [REQ-35.015](ep-requirements.md#req-35-015--no-configjson-changes) | Manual | EP-035 diff leaves `config.json` and load validation unchanged |
| [AC-35.016](#ac-35-016) | [REQ-35.016](ep-requirements.md#req-35-016--quality-gate-passes) | Manual (make check) | `make check` exits zero |
| [AC-35.017](#ac-35-017) | [REQ-35.017](ep-requirements.md#req-35-017--preserve-system-prompt-assembly) | Integration | Core handler system-message assembly unchanged vs baseline |
| [AC-35.018](#ac-35-018) | [REQ-35.018](ep-requirements.md#req-35-018--preserve-runtime-skills-marker-rejection) | Integration | Runtime skills still fail startup on forbidden marker lines |
| [AC-35.019](#ac-35-019) | [REQ-35.019](ep-requirements.md#req-35-019--preserve-memory-indexing-marker-rejection) | Unit | `write_memory` / handler still reject forbidden marker lines in chunks |
| [AC-35.020](#ac-35-020) | [REQ-35.020](ep-requirements.md#req-35-020--ep-013-prompt-tests-retain-intent) | Integration | EP-013 marker, wrap, handler, and runtime-skills tests pass |

---

## Scenarios

### AC-35.002 No logging package imports (Trace: REQ-35.002)

Given the EP-035 change set is applied  
When searching `*.go` under `cmd/`, `internal/`, and `tests/` for `pa/internal/logging`  
Then the search SHALL return zero matches.

### AC-35.005 Relocated race test (Trace: REQ-35.005)

Given temporary vector and jobs SQLite stores under `sqlitepragma.RecommendedPolicy`  
When `go test -race` runs `TestConcurrentWrites_NoBusyErrors` at its new integration path  
Then the test SHALL complete without busy/locked errors or data races.

### AC-35.008 Byte-identical trust policy (Trace: REQ-35.008)

Given a frozen pre-EP-035 `TrustPolicy` reference  
When compared to `internal/prompt.TrustPolicy`  
Then the strings SHALL be byte-identical.

### AC-35.010 Forbidden marker validation (Trace: REQ-35.010)

Given inputs that exercised `internal/promptmarkers` before EP-035  
When run through `internal/prompt` validation helpers  
Then results SHALL match the prior behaviour.

### AC-35.013 No legacy prompt imports (Trace: REQ-35.013)

Given the EP-035 change set is applied  
When searching `*.go` for `pa/internal/promptmarkers` or `pa/internal/systemprompt`  
Then the search SHALL return zero matches.

### AC-35.016 Quality gate (Trace: REQ-35.016)

Given EP-035 implementation is complete  
When `make check` runs from the repository root  
Then it SHALL exit with status zero.

### AC-35.020 EP-013 tests retain intent (Trace: REQ-35.020)

Given EP-013 marker, wrap, handler, and runtime-skills tests on the epic branch  
When the test suite runs  
Then all such tests SHALL pass.

---

## Acceptance criteria

<a id="ac-35-001"></a>

### AC-35.001

**Trace:** [REQ-35.001](ep-requirements.md#req-35-001--delete-logging-stub-package)  
**Test level:** Manual  
**Status:** AC-35.001 MANUAL ONLY — verified by repository tree inspection (directory removed) and `make check`; no unit test applies.

Given the EP-035 change set is applied on the epic branch  
When the repository tree is inspected  
Then the path `internal/logging/` SHALL NOT exist.

---

<a id="ac-35-002"></a>

### AC-35.002

**Trace:** [REQ-35.002](ep-requirements.md#req-35-002--no-logging-package-imports)  
**Test level:** Manual (build/grep)  
**Status:** AC-35.002 MANUAL ONLY — verified by repository grep for `pa/internal/logging` (zero matches) and `make check`; no unit test applies.

Given the EP-035 change set is applied  
When searching all `*.go` files under `cmd/`, `internal/`, and `tests/` for the import path `pa/internal/logging`  
Then the search SHALL return zero matches.

---

<a id="ac-35-003"></a>

### AC-35.003

**Trace:** [REQ-35.003](ep-requirements.md#req-35-003--delete-reliability-test-package)  
**Test level:** Manual  
**Status:** AC-35.003 MANUAL ONLY — verified by repository tree inspection (directory removed) and `make check`; no unit test applies.

Given the EP-035 change set is applied  
When the repository tree is inspected  
Then the path `internal/reliability/` SHALL NOT exist.

---

<a id="ac-35-004"></a>

### AC-35.004

**Trace:** [REQ-35.004](ep-requirements.md#req-35-004--relocate-concurrent-write-test)  
**Test level:** Integration

Given the EP-035 change set is applied  
When the `tests/integration` Go package sources are inspected  
Then a test function named `TestConcurrentWrites_NoBusyErrors` SHALL be present in that package (or another existing integration test package allowed by [ep-scope.md](ep-scope.md))  
And that test SHALL import `internal/jobs`, `internal/vector/sqlite`, and `internal/sqlitepragma`.

---

<a id="ac-35-005"></a>

### AC-35.005

**Trace:** [REQ-35.005](ep-requirements.md#req-35-005--preserve-ac-22-010-race-test-intent)  
**Test level:** Integration

Given temporary vector and jobs SQLite stores opened with `sqlitepragma.RecommendedPolicy`  
When `go test -race` runs `TestConcurrentWrites_NoBusyErrors` at its relocated package path for the full per-writer iteration budget  
Then the test SHALL complete without `SQLITE_BUSY`, `database is locked`, or data-race failures  
And the concurrent writers SHALL exercise both stores, preserving the EP-022 concurrent-write reliability criterion ([EP-022 ac-22010](../EP-022/ep-acceptance-criteria.md#ac-22010)) intent. The relocated test retains its EP-022 coverage trace so that epic's own validation still recognises it.

---

<a id="ac-35-006"></a>

### AC-35.006

**Trace:** [REQ-35.006](ep-requirements.md#req-35-006--update-reliability-test-documentation-path)  
**Test level:** Manual  
**Status:** AC-35.006 MANUAL ONLY — verified by reading `docs/configuration.md` (cites `tests/integration`, no `internal/reliability`); no unit test applies.

Given `docs/configuration.md` on the epic branch  
When the section describing concurrent-writer reliability under `-race` is read  
Then it SHALL cite the relocated test path (for example `tests/integration`)  
And it SHALL NOT cite `internal/reliability` as the test location.

---

<a id="ac-35-007"></a>

### AC-35.007

**Trace:** [REQ-35.007](ep-requirements.md#req-35-007--provide-merged-internalprompt-package)  
**Test level:** Unit

Given the EP-035 change set is applied  
When the `internal/prompt` package API is inspected  
Then it SHALL export the six canonical block marker line constants, `TrustPolicy`, `TextContainsForbiddenMarkerLine`, `ForbiddenMarkerLines`, and the `WrapRetrievedContext`, `WrapToolInstructions`, and `WrapRuntimeSkills` functions previously split across `internal/promptmarkers` and `internal/systemprompt`.

---

<a id="ac-35-008"></a>

### AC-35.008

**Trace:** [REQ-35.008](ep-requirements.md#req-35-008--byte-identical-trust-policy)  
**Test level:** Unit

Given the `TrustPolicy` string constant from `internal/systemprompt` immediately before EP-035 implementation (captured in a unit test golden, VCS parent snapshot, or equivalent frozen reference)  
When comparing it to `prompt.TrustPolicy` in `internal/prompt`  
Then the two values SHALL be byte-identical.

---

<a id="ac-35-009"></a>

### AC-35.009

**Trace:** [REQ-35.009](ep-requirements.md#req-35-009--byte-identical-marker-constants)  
**Test level:** Unit

Given the six canonical marker line constants from `internal/promptmarkers` immediately before EP-035 implementation (frozen reference as for AC-35.008)  
When comparing each constant to the corresponding export in `internal/prompt`  
Then every pair SHALL be byte-identical for `<<<PA_BEGIN_CONTEXT>>>`, `<<<PA_END_CONTEXT>>>`, `<<<PA_BEGIN_TOOLS>>>`, `<<<PA_END_TOOLS>>>`, `<<<PA_BEGIN_SKILLS>>>`, and `<<<PA_END_SKILLS>>>`.

---

<a id="ac-35-010"></a>

### AC-35.010

**Trace:** [REQ-35.010](ep-requirements.md#req-35-010--equivalent-forbidden-marker-validation)  
**Test level:** Unit

Given a table of inputs covering empty text, text without marker lines, text with marker lines at line boundaries, and text with marker-like substrings not on their own trimmed lines  
When `prompt.TextContainsForbiddenMarkerLine` and `prompt.ForbiddenMarkerLines` are exercised  
Then results SHALL match the pre-EP-035 `internal/promptmarkers` behaviour for the same inputs.

---

<a id="ac-35-011"></a>

### AC-35.011

**Trace:** [REQ-35.011](ep-requirements.md#req-35-011--equivalent-block-wrap-helpers)  
**Test level:** Unit

Given representative non-empty and empty inner strings for retrieved context, tool instructions, and runtime skills bodies  
When `prompt.WrapRetrievedContext`, `prompt.WrapToolInstructions`, and `prompt.WrapRuntimeSkills` are called  
Then each wrapped result SHALL equal the output of the corresponding pre-EP-035 `internal/systemprompt` function for the same inner input.

---

<a id="ac-35-012"></a>

### AC-35.012

**Trace:** [REQ-35.012](ep-requirements.md#req-35-012--delete-legacy-prompt-packages)  
**Test level:** Manual  
**Status:** AC-35.012 MANUAL ONLY — verified by repository tree inspection (directories removed) and `make check`; no unit test applies.

Given the EP-035 change set is applied  
When the repository tree is inspected  
Then the paths `internal/promptmarkers/` and `internal/systemprompt/` SHALL NOT exist.

---

<a id="ac-35-013"></a>

### AC-35.013

**Trace:** [REQ-35.013](ep-requirements.md#req-35-013--no-legacy-prompt-package-imports)  
**Test level:** Manual (build/grep)  
**Status:** AC-35.013 MANUAL ONLY — verified by repository grep for `pa/internal/promptmarkers` and `pa/internal/systemprompt` (zero matches) and `make check`; no unit test applies.

Given the EP-035 change set is applied  
When searching all `*.go` files under `cmd/`, `internal/`, and `tests/` for `pa/internal/promptmarkers` or `pa/internal/systemprompt`  
Then the search SHALL return zero matches.

---

<a id="ac-35-014"></a>

### AC-35.014

**Trace:** [REQ-35.014](ep-requirements.md#req-35-014--update-prompt-package-importers)  
**Test level:** Manual (build/grep)  
**Status:** AC-35.014 MANUAL ONLY — verified by `make check` (build/vet of all listed importers) and grep confirming `pa/internal/prompt`-only imports; behaviour is covered by the same packages' existing tests.

Given the EP-035 change set is applied  
When building packages that previously imported `internal/promptmarkers` or `internal/systemprompt` — namely `internal/core` (`handler.go`, `system_tail.go`, and related tests), `internal/tools` (`write_memory.go`), `internal/runtimeskills`, `tests/integration/runtime_skills_handler_test.go`, and `tests/integration/runtime_skills_config_test.go`  
Then each SHALL compile with imports of `pa/internal/prompt` only  
And SHALL NOT reference the removed package paths.

---

<a id="ac-35-015"></a>

### AC-35.015

**Trace:** [REQ-35.015](ep-requirements.md#req-35-015--no-configjson-changes)  
**Test level:** Manual  
**Status:** AC-35.015 MANUAL ONLY — verified by inspecting the EP-035 branch diff (no `config.json`, `config.examples/`, or `internal/config` validation changes); no unit test applies.

Given the EP-035 branch diff against its merge base  
When inspecting product configuration artefacts (`config.json`, `config.examples/`, and config load validation in `internal/config`)  
Then the diff SHALL introduce no new, removed, or altered top-level configuration keys and no changed validation rules attributable to EP-035.

---

<a id="ac-35-016"></a>

### AC-35.016

**Trace:** [REQ-35.016](ep-requirements.md#req-35-016--quality-gate-passes)  
**Test level:** Manual (make check)  
**Status:** AC-35.016 MANUAL ONLY — verified by running `make check` from the repository root (exit 0); this is a process gate, not a unit test.

Given the EP-035 implementation is complete on the epic branch  
When `make check` runs from the repository root  
Then it SHALL exit with status zero.

---

<a id="ac-35-017"></a>

### AC-35.017

**Trace:** [REQ-35.017](ep-requirements.md#req-35-017--preserve-system-prompt-assembly)  
**Test level:** Integration

Given handler tests that assert merged system-message structure (trust policy prefix, non-empty block wrapping, and dynamic block ordering) from EP-013  
When those tests run after EP-035 on the epic branch  
Then they SHALL pass without changing expected trust placement, marker wrapping, or block order relative to the pre-EP-035 baseline.

---

<a id="ac-35-018"></a>

### AC-35.018

**Trace:** [REQ-35.018](ep-requirements.md#req-35-018--preserve-runtime-skills-marker-rejection)  
**Test level:** Integration

Given a runtime skill package whose `SKILL.md` body or frontmatter contains a line equal to a canonical marker line after trim  
When configuration loads with `runtime_skills.enabled` true  
Then startup SHALL fail with an error that identifies the skill directory  
As covered by existing runtime-skills integration or package tests on the epic branch.

---

<a id="ac-35-019"></a>

### AC-35.019

**Trace:** [REQ-35.019](ep-requirements.md#req-35-019--preserve-memory-indexing-marker-rejection)  
**Test level:** Unit

Given conversation chunk text prepared for vector indexing that contains a trimmed line equal to a canonical marker constant  
When indexing or memory-write paths validate the chunk  
Then that chunk SHALL be refused for that indexing attempt  
As asserted by existing unit tests for `internal/core` and `internal/tools/write_memory` on the epic branch.

---

<a id="ac-35-020"></a>

### AC-35.020

**Trace:** [REQ-35.020](ep-requirements.md#req-35-020--ep-013-prompt-tests-retain-intent)  
**Test level:** Integration

Given the automated tests that cover EP-013 marker collision, wrap layout, handler prompt structure, and runtime-skills integration (whether retained under `internal/prompt` or relocated with unchanged intent)  
When the EP-035 test suite runs on the epic branch  
Then all such tests SHALL pass.
