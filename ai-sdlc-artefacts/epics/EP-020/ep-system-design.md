# EP-020 - System Design

## Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Module boundaries](#module-boundaries)
- [Components and interfaces](#components-and-interfaces)
- [Data models](#data-models)
- [Error handling](#error-handling)
- [Testing strategy](#testing-strategy)
- [Requirement traceability](#requirement-traceability)
- [Risks and trade-offs](#risks-and-trade-offs)

## Overview

EP-020 extends the EP-019 scheduler path by adding a hybrid natural-language creation route in Telegram chat flow. The design applies deterministic parsing first and native-tool fallback second for explicit schedule-intent messages, without changing runtime execution semantics.

## Architecture

<p align="center"><img src="diagrams/c4-container.png" alt="C4 C2 - EP-020" /></p>

**Source:** [c4-container.puml](diagrams/c4-container.puml). Regenerate: `plantuml -tpng diagrams/c4-container.puml`.

High-level behavior:

1. Telegram adapter authorizes user and forwards message.
2. Jobs command handler checks management commands first (`/jobs ...`).
3. If message matches strict NL-create syntax, manager creates a persisted job.
4. If strict syntax does not match but explicit schedule-intent is detected, manager invokes native fallback tool to extract instruction/time and create a persisted job.
5. Existing runtime loop computes due runs and delivers results.

## Module boundaries

| Module | Responsibility | Allowed dependencies | Forbidden dependencies |
|--------|----------------|----------------------|------------------------|
| `cmd/pa` | Message routing and runtime wiring | `internal/core`, `internal/jobs`, `internal/telegram` | Direct DB access |
| `internal/jobs/manager` | Hybrid NL create orchestration, validation, response, audit | `internal/jobs/store`, `internal/jobs/runtime` APIs, `internal/tools` contracts | Telegram transport-specific code |
| `internal/jobs/store` | Persistence | SQLite driver only | Core/chat orchestration |
| `internal/jobs/runtime` | Scheduling and execution lifecycle | `store`, `runner` contract | Parser or Telegram command parsing logic |
| `internal/telegram` | Adapter allowlist and transport | `internal/core` handler contract | Business parser/persistence logic |

## Components and interfaces

| Component | Responsibility | Key interface/contract |
|-----------|----------------|------------------------|
| `cmd/pa/jobsCommandHandler` | Route `/jobs` and NL-create messages before default conversation path | `HandleMessage(ctx, userID, sessionKey, text)` |
| `internal/jobs/Manager` | Try strict parser first, execute native-tool fallback second, validate, persist, respond deterministically, audit outcomes | `HandleNaturalLanguageCreate(...)`, `CreateScheduledJobFromSpec(...)`, `HandleCommand(...)` |
| `internal/jobs/CreateScheduledJobTool` | Extract and normalize instruction + HH:MM for explicit schedule-intent free-form messages | `Run(ctx, params)` |
| `internal/jobs/Store` | Persist created jobs and runs | `CreateJob`, `ListJobs`, `GetJob`, `RecordRun` |
| `internal/jobs/Runtime` | Execute due runs using existing policies | `EvaluateDue`, `RunNow` |
| `cmd/pa/scheduledJobRunner` | Execute agent turn and deliver output to Telegram | `Run(ctx, job)` |

## Data models

EP-020 reuses EP-019 schema without migration:

- `jobs` table: stores `instruction`, `schedule_expr`, `time_zone`, `delivery_chat_id`, `status`.
- `job_runs` table: stores run lifecycle and outcomes.

Created-job defaults:

- `status = active`
- `overlap_policy = single_instance`
- `timeout_policy = cancel_after_limit`
- `time_zone = pa_timezone`

NL create contract (manager-level):

- **Input:** `text`, `actor_user_id`, `delivery_chat_id`, `default_timezone`.
- **Success response fields:** `job_id`, `schedule_expr`, `timezone`, `next_run`, `instruction`.
- **Validation outcomes:** `not_a_create_request`, `invalid_time_syntax`, `internal_error`, `success`.
- **Path marker:** `creation_path` with values `deterministic_parser` or `native_tool_fallback` is included in audit outcome payload.

## Error handling

- Unsupported or malformed NL syntax returns deterministic guidance and does not persist a job (REQ-20.007).
- If scheduler is not ready, create requests return readiness response already used by `/jobs` path.
- Store/runtime failures return fail-fast errors and emit audit outcome `internal_error`.
- Deterministic parser is evaluated before fallback to keep behavior stable and predictable (REQ-20.011).
- Native-tool fallback is evaluated only for explicit schedule-intent messages with HH:MM signal to reduce false positives (REQ-20.011, REQ-20.013).
- Deterministic guidance message for malformed syntax: expected `"<instruction> and send it at HH:MM every day"`.

## Testing strategy

- Unit: parser behavior, manager create/reject flows, audit fields.
- Integration: routing behavior in `jobsCommandHandler`, unauthorized path in adapter.
- E2E: create-by-message -> list/show -> run-now -> delivery.
- Validation: `./bin/validate EP-020`, `make check`.

AC mapping:

| AC | Planned test level |
|----|--------------------|
| AC-20.001 | Unit (`manager_test`) |
| AC-20.002 | Unit (`manager_test`) |
| AC-20.003 | Unit (`manager_test`) |
| AC-20.004 | Unit (`manager_test`) |
| AC-20.005 | Integration (`jobs_runtime_test`) |
| AC-20.006 | E2E (`cmd/pa` EP-020 flow test) |
| AC-20.007 | Integration (`telegram/adapter_test`, `jobs_runtime_test`) |
| AC-20.008 | Unit/Integration (`manager_test`, log capture) |
| AC-20.009 | Unit/Integration (`manager_test`, fallback create path) |

## Requirement traceability

| Requirement | Design coverage |
|-------------|-----------------|
| REQ-20.001 | `jobsCommandHandler` NL route + manager create entry |
| REQ-20.002 | parser in manager extracts instruction/time |
| REQ-20.003 | manager persists via `Store.CreateJob` |
| REQ-20.004 | manager applies `pa_timezone` |
| REQ-20.005 | manager resolves delivery target from chat/session |
| REQ-20.006 | manager returns deterministic creation confirmation |
| REQ-20.007 | parser returns deterministic rejection with no side effect |
| REQ-20.008 | created jobs reuse existing `/jobs` list/show APIs |
| REQ-20.009 | existing runtime and `scheduledJobRunner` delivery path |
| REQ-20.010 | Telegram adapter allowlist gate remains authoritative |
| REQ-20.011 | deterministic-first gating + explicit schedule-intent fallback gating |
| REQ-20.012 | manager audit emits operation/outcome/actor/job_id |
| REQ-20.013 | native-tool fallback extraction and single create attempt on deterministic non-match |

## Risks and trade-offs

- MVP parser scope is intentionally narrow (daily HH:MM); richer recurrence is deferred.
- Using deterministic-first flow reduces accidental creation risk but may reject some user-intended phrasing without fallback.
- Reusing EP-019 runtime minimizes regression risk and implementation size.
- Adding fallback path increases parsing coverage but introduces additional prompt/tool-call variability.

Mitigations:

- Return deterministic guidance on parse failures with accepted syntax example.
- Emit creation audit outcomes including parsed schedule payload to support tuning.
- Limit fallback activation to explicit schedule-intent with HH:MM and enforce single-attempt create path.
- Track creation rejection ratio from logs; if ratio is high, expand syntax set in follow-up epic.
