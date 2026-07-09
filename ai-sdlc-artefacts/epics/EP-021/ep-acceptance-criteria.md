# EP-021 — Acceptance criteria

Introduction: Testable conditions for scheduler routing without a Telegram-only gate and explicit-parameter `create_scheduled_job`. The example runtime skill is **optional** (REQ-21.007); AC-21.007 only checks the template loads and validates — **not** that production chat depends on skills. Traceability to [ep-requirements.md](ep-requirements.md).

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-21.001](#ac-21001) | [REQ-21.001](ep-requirements.md#requirements), [REQ-21.011](ep-requirements.md#req-21-011--keep-jobs-list-and-management-behaviour) | `/jobs list` works when runtime is ready |
| [AC-21.002](#ac-21002) | [REQ-21.002](ep-requirements.md#requirements) | Plain chat delegates to base once |
| [AC-21.003](#ac-21003) | [REQ-21.002](ep-requirements.md#requirements), [REQ-21.003](ep-requirements.md#requirements) | Schedule-shaped free text delegates to base once (no wrapper fallback) |
| [AC-21.004](#ac-21004) | [REQ-21.004](ep-requirements.md#requirements), [REQ-21.005](ep-requirements.md#requirements), [REQ-21.009](ep-requirements.md#requirements), [REQ-21.012](ep-requirements.md#req-21-012--automated-tests-trace-acceptance-criteria) | Tool creates job with confirmation and audit path |
| [AC-21.005](#ac-21005) | [REQ-21.010](ep-requirements.md#requirements), [REQ-21.012](ep-requirements.md#req-21-012--automated-tests-trace-acceptance-criteria) | Invalid clock fields yield message and nil tool error |
| [AC-21.006](#ac-21006) | [REQ-21.006](ep-requirements.md#requirements) | Static system prompt unchanged (**Deferred** — manual diff review) |
| [AC-21.007](#ac-21007) | [REQ-21.007](ep-requirements.md#requirements), [REQ-21.008](ep-requirements.md#requirements) | **Optional** example skill template loads; tool ref validates when jobs path set |

## Acceptance criteria

### AC-21.001

**AC-21.001** (Trace: REQ-21.001, REQ-21.011)

Given a `jobsCommandHandler` with a ready `jobsRuntimeState` backed by an empty job store  
When the user sends `/jobs list`  
Then the reply contains `No scheduled jobs configured` or `Scheduled jobs:` and the handler returns without error.

### AC-21.002

**AC-21.002** (Trace: REQ-21.002)

Given the same handler with a mock base that records incoming text  
When the user sends `hello` (no `/jobs` prefix)  
Then the base handler runs exactly once with user text `hello` and the mock reply is returned.

### AC-21.003

**AC-21.003** (Trace: REQ-21.002, REQ-21.003)

Given the handler with a ready manager and a mock base  
When the user sends `send me AI news digest at 08:15 every day`  
Then the base handler runs exactly once with that full string as the user message and no job is persisted by the wrapper path alone.

### AC-21.004

**AC-21.004** (Trace: REQ-21.004, REQ-21.005, REQ-21.009)

Given a `CreateScheduledJobTool` wired to a test `Manager` and store  
When `Run` is called with `instruction`, `hour`, `minute`, and context actor/delivery  
Then the reply contains `Scheduled job created`, the store lists one job with the expected cron fields, and logs contain `creation_path=native_tool_explicit`.

### AC-21.005

**AC-21.005** (Trace: REQ-21.010)

Given the same tool and manager  
When `Run` is called with `hour` 99 and valid `instruction` and `minute`  
Then the result string describes invalid time and the returned Go error is nil.

### AC-21.006

**AC-21.006** (Trace: REQ-21.006) — **Deferred** — **MANUAL ONLY**

Given the agreed baseline commit before EP-021 implementation  
When an operator compares `systemStaticHead` and static personality strings in `internal/core/handler.go`  
Then there is no change to TrustPolicy, MarkerSupplement, date line format, or base personality prose introduced by EP-021.

### AC-21.007

**AC-21.007** (Trace: REQ-21.007, REQ-21.008)

Given the repository `config.examples/skills/scheduled-jobs/SKILL.md` template  
When the package is loaded in isolation and `ValidateToolRefs` runs with a native allowlist that includes `create_scheduled_job`  
Then the `scheduled-jobs` package loads without error and its declared tool id is accepted.  
**Note:** This AC does **not** assert that operators must enable `runtime_skills` for NL create; it only guards the template and config validation path.
