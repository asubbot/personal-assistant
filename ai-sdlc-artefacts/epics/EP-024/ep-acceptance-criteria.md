# EP-024 — Acceptance criteria

## Introduction

This document lists testable acceptance criteria for operator documentation and safe logging defaults, traceable to [ep-requirements.md](ep-requirements.md). IDs use **AC-24.NNN** (epic 24).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|-------------|---------|
| [AC-24.001](#ac-24-001) | [REQ-24.001](ep-requirements.md#operator-documentation) | Operator doc states zero-based ordered pool |
| [AC-24.002](#ac-24-002) | [REQ-24.002](ep-requirements.md#operator-documentation) | Operator doc describes baseline index and fallback |
| [AC-24.003](#ac-24-003) | [REQ-24.003](ep-requirements.md#operator-documentation) | Operator doc describes summarization adapter |
| [AC-24.004](#ac-24-004) | [REQ-24.004](ep-requirements.md#operator-documentation) | Operator doc states classifier is not pool-indexed |
| [AC-24.005](#ac-24-005) | [REQ-24.005](ep-requirements.md#operator-documentation) | Operator doc includes three sketches |
| [AC-24.006](#ac-24-006) | [REQ-24.009](ep-requirements.md#operator-documentation) | Operator doc documents `PA_ENV=development` |
| [AC-24.007](#ac-24-007) | [REQ-24.006](ep-requirements.md#docker-defaults) | Dockerfile sets `PA_LOG_LEVEL=info` |
| [AC-24.008](#ac-24-008) | [REQ-24.007](ep-requirements.md#docker-defaults) | Compose sets `PA_LOG_LEVEL=info` |
| [AC-24.009](#ac-24-009) | [REQ-24.008](ep-requirements.md#startup-policy) | Startup emits WARN on debug without dev env |
| [AC-24.010](#ac-24-010) | [REQ-24.010](ep-requirements.md#verification) | `make check` passes |

---

## Acceptance criteria

### AC-24.001

**Trace:** [REQ-24.001](ep-requirements.md#operator-documentation)

Given the operator opens the EP-024 provider roles documentation file at `docs/llm-provider-roles-and-logging.md` in the product repository (also linked from `docs/configuration.md`)  
When they read the section that defines the provider pool  
Then the text states that `llm_providers` is ordered, zero-based, and identifies the pool as the authoritative list for router-backed roles.

### AC-24.002

**Trace:** [REQ-24.002](ep-requirements.md#operator-documentation)

Given the same documentation file  
When the operator reads the main conversation section  
Then the text names `tools.llm_escalation.baseline_index`, explains the starting index for a new turn when escalation is enabled versus disabled, and mentions that transport fallback can advance along the pool on qualifying errors.

### AC-24.003

**Trace:** [REQ-24.003](ep-requirements.md#operator-documentation)

Given the same documentation file  
When the operator reads the summarization section  
Then the text states summarization uses the same `llm_providers` entries and describes baseline selection consistent with `SummarizeRouterConfig`.

### AC-24.004

**Trace:** [REQ-24.004](ep-requirements.md#operator-documentation)

Given the same documentation file  
When the operator reads the intent classifier section  
Then the text states the model stage is configured under `intent_classifier.model_stage` and is not selected by an index into `llm_providers`.

### AC-24.005

**Trace:** [REQ-24.005](ep-requirements.md#operator-documentation)

Given the same documentation file  
When the operator scans the examples subsection  
Then the file contains three labelled sketches: single-provider, escalation-enabled multi-provider, and classifier-enabled reference.

### AC-24.006

**Trace:** [REQ-24.009](ep-requirements.md#operator-documentation)

Given the same documentation file  
When the operator reads the diagnostic logging guidance  
Then the text documents setting `PA_ENV` to `development` (case-insensitive) to acknowledge intentional diagnostic sessions.

### AC-24.007

**Trace:** [REQ-24.006](ep-requirements.md#docker-defaults)

Given the repository root `Dockerfile`  
When an automated test reads the runtime stage instructions  
Then the file contains `ENV PA_LOG_LEVEL=info` and does not set `PA_LOG_LEVEL` to `debug`.

### AC-24.008

**Trace:** [REQ-24.007](ep-requirements.md#docker-defaults)

Given `docker-compose.yml` at the repository root  
When an automated test inspects the `pa` service `environment` list  
Then the list declares `PA_LOG_LEVEL` with a default of `info` (for example `PA_LOG_LEVEL=${PA_LOG_LEVEL:-info}`) and does not set a literal `PA_LOG_LEVEL=debug` entry.

### AC-24.009

**Trace:** [REQ-24.008](ep-requirements.md#startup-policy)

Given `PA_LOG_LEVEL=debug` and (`PA_ENV` is unset OR `PA_ENV` is set to a value that is not `development` under ASCII case-folding, for example `staging` or `production`)  
When the sensitive-logging startup check runs with the effective `slog` level `debug`  
Then exactly one `WARN` record is emitted that states full LLM payloads may be logged.

Given `PA_LOG_LEVEL=debug` and `PA_ENV` is set to `development` or `DEVELOPMENT`  
When the sensitive-logging startup check runs  
Then no `WARN` record is emitted for this policy.

Given `PA_LOG_LEVEL=info`  
When the sensitive-logging startup check runs  
Then no `WARN` record is emitted for this policy.

### AC-24.010

**Trace:** [REQ-24.010](ep-requirements.md#verification)

Given the EP-024 change set is merged on the working branch  
When the operator runs `make check` from the repository root  
Then the command exits with status zero.
