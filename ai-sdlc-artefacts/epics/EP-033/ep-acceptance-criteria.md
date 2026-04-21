# EP-033 — Memory Summarization Retry — Acceptance criteria

This document defines acceptance criteria for [ep-scope.md](ep-scope.md), traced to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC | REQ (trace) | Summary |
|----|-------------|---------|
| [AC-33.001](#ac-33-001) | [REQ-33.001](ep-requirements.md#retry-policy-scope) | `catchup_day` uses retry policy |
| [AC-33.002](#ac-33-002) | [REQ-33.002](ep-requirements.md#retry-policy-scope) | `summarize_yesterday` uses retry policy |
| [AC-33.003](#ac-33-003) | [REQ-33.003](ep-requirements.md#retry-policy-scope) | Month/year flows unchanged |
| [AC-33.004](#ac-33-004) | [REQ-33.004](ep-requirements.md#retry-scheduling-behavior) | Retry schedules with bounded backoff |
| [AC-33.005](#ac-33-005) | [REQ-33.005](ep-requirements.md#retry-scheduling-behavior) | Retry exhaustion stops automatic retries |
| [AC-33.006](#ac-33-006) | [REQ-33.006](ep-requirements.md#retry-scheduling-behavior) | Duplicate retry chains are prevented |
| [AC-33.007](#ac-33-007) | [REQ-33.007](ep-requirements.md#queue-semantics) | User-turn deferral remains active |
| [AC-33.008](#ac-33-008) | [REQ-33.008](ep-requirements.md#queue-semantics) | Retry runs in existing memoryjob loop |
| [AC-33.009](#ac-33-009) | [REQ-33.009](ep-requirements.md#observability) | Retry scheduling and exhaustion are logged |
| [AC-33.010](#ac-33-010) | [REQ-33.010](ep-requirements.md#existing-behavior-preservation) | Normal day-success flow unchanged |
| [AC-33.011](#ac-33-011) | [REQ-33.011](ep-requirements.md#verification) | Retry timing policy is deterministic |
| [AC-33.012](#ac-33-012) | [REQ-33.012](ep-requirements.md#verification) | Automated retry tests are present |
| [AC-33.013](#ac-33-013) | [REQ-33.013](ep-requirements.md#verification) | `make check` passes |
| [AC-33.014](#ac-33-014) | [REQ-33.014](ep-requirements.md#verification) | `./bin/validate EP-033` passes |

---

## Acceptance criteria

### AC-33.001

Given startup enqueue creates `catchup_day` job  
When `catchup_day` returns a retryable error  
Then `memoryjob` SHALL schedule bounded retry attempts for that day target.

### AC-33.002

Given scheduled enqueue creates `summarize_yesterday` job  
When `summarize_yesterday` returns a retryable error  
Then `memoryjob` SHALL schedule bounded retry attempts for that day target.

### AC-33.003

Given EP-033 changes are applied  
When `catchup_month`, `catchup_year`, or scheduled month/year rollups run  
Then behavior SHALL match pre-EP-033 behavior without new retry chains.

### AC-33.004

Given a failed day job and retry policy delays  
When retry is scheduled  
Then retry execution SHALL not occur before configured backoff delay.

### AC-33.005

Given repeated day-job failures for one day target  
When max retry attempts are reached  
Then no additional automatic retry SHALL be enqueued for that target.

### AC-33.006

Given a retry for day target `D` is already queued  
When another enqueue request for retry of `D` occurs  
Then queue SHALL keep one retry chain for `D` and reject duplicate insertion.

### AC-33.007

Given user turn is active  
When a retryable day job is pending  
Then existing deferral behavior SHALL postpone execution until user turn is inactive.

### AC-33.008

Given retry attempts are scheduled  
When retries execute  
Then execution SHALL occur within existing `memoryjob.Runner` queue and drain loop.

### AC-33.009

Given retry scheduling or retry exhaustion event occurs  
When logs are emitted  
Then structured log entries SHALL include job name, day target key, attempt index, and delay or exhaustion marker.

### AC-33.010

Given day summarization succeeds on first attempt  
When summary write and vector index run  
Then behavior and outcomes SHALL match pre-EP-033 success path.

### AC-33.011

Given fixed runner clock and deterministic failures  
When the same test scenario runs repeatedly  
Then retry schedule timestamps and attempt sequence SHALL be identical.

### AC-33.012

Given EP-033 code and tests  
When test suite for `internal/memoryjob` runs  
Then it SHALL include tests for retry timing, exhaustion, and duplicate prevention.

### AC-33.013

Given EP-033 change set on clean working tree  
When `make check` runs  
Then command SHALL exit with status `0`.

### AC-33.014

Given EP-033 change set on clean working tree  
When `./bin/validate EP-033` runs  
Then command SHALL exit with status `0`.
