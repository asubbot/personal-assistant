---
artefact: ep-acceptance-criteria
epic_id: EP-043
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-043 — Test suite organization — Acceptance criteria

**Git branch:** `epic/EP-043-test-suite-organization`

## Acceptance criteria index

| AC ID | REQ | Test level | Summary |
|-------|-----|------------|---------|
| [AC-43.001](#ac-43-001) | REQ-43.001, REQ-43.002 | Manual | handler_test.go split; ≤600 LOC |
| [AC-43.002](#ac-43-002) | REQ-43.001 | Unit | All handler tests pass unchanged |
| [AC-43.003](#ac-43-003) | REQ-43.004, REQ-43.005 | Unit | Fixture helper + ≥10 migrations |
| [AC-43.004](#ac-43-004) | REQ-43.006, REQ-43.007 | Unit/Manual | architecture_guards_test.go; AC comments preserved |
| [AC-43.005](#ac-43-005) | REQ-43.008 | Manual (make check) | Coverage within 0.5% of baseline |
| [AC-43.006](#ac-43-006) | REQ-43.009 | Manual (make check) | make check passes |
| [AC-43.007](#ac-43-007) | REQ-43.010 | Manual (validate) | make validate passes |

## Acceptance criteria

<a id="ac-43-001"></a>

### AC-43.001

**Trace:** REQ-43.001, REQ-43.002  
**Test level:** Manual

Given the epic branch  
When counting lines in `internal/core/handler_test.go`  
Then the file SHALL be at most 600 lines and domain-specific handler test files SHALL exist.

<a id="ac-43-002"></a>

### AC-43.002

**Trace:** REQ-43.001  
**Test level:** Unit

When `go test ./internal/core/...` runs  
Then all tests SHALL pass with zero failures versus pre-epic baseline.

<a id="ac-43-003"></a>

### AC-43.003

**Trace:** REQ-43.004, REQ-43.005  
**Test level:** Unit

Given `config_test_helpers.go`  
When counting tests migrated from inline JSON  
Then at least ten test functions SHALL use `loadConfigFixture` or equivalent.

<a id="ac-43-004"></a>

### AC-43.004

**Trace:** REQ-43.006, REQ-43.007  
**Test level:** Manual

Given `architecture_guards_test.go`  
When `./bin/validate EP-034` and `./bin/validate EP-038` run after epic registration  
Then in-scope AC traceability SHALL remain satisfied.

<a id="ac-43-006"></a>

### AC-43.006

**Trace:** REQ-43.009  
**Test level:** Manual (make check)

When `make check` runs  
Then exit code SHALL be zero.

<a id="ac-43-007"></a>

### AC-43.007

**Trace:** REQ-43.010  
**Test level:** Manual (validate)

When `make validate` runs  
Then exit code SHALL be zero.
