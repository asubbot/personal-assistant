# Epic scope — EP-019 Scheduled Agent Jobs and Legacy Scheduler Replacement


| Field                  | Content                                                                                                                                                                                                                                                                                                    |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ID**                 | EP-019                                                                                                                                                                                                                                                                                                     |
| **Status**             | DONE                                                                                                                                                                                                                                                                                                       |
| **Title**              | Scheduled Agent Jobs and Legacy Scheduler Replacement                                                                                                                                                                                                                                                      |
| **Description**        | Replace the current `scheduled_tasks` execution model with first-class scheduled agent jobs that can run natural-language instructions on a schedule and deliver results to Telegram. Because the product is pre-production, remove the legacy scheduler path without backward compatibility requirements. |
| **First version date** | 2026-04-16                                                                                                                                                                                                                                                                                                 |


## Glossary

- **Scheduled agent job**: A persisted cron-based job that triggers a full agent turn (LLM + tools + memory path) from a configured instruction payload.
- **Legacy `scheduled_tasks`**: The current JSON task list model (`name`, `schedule`, `action`, `params`) executed by `internal/scheduler`, where `notify` sends static text and tool actions log results.
- **Delivery target**: The Telegram destination for job results (default notify chat or explicit chat configuration per job).
- **Isolated run**: A scheduled execution context that does not pollute normal interactive chat state unless explicitly configured.

## Scope (features/capabilities)

- **New scheduling model**: Define and implement a job schema focused on scheduled agent turns (instruction/message payload, cron schedule, timezone, delivery settings, run mode).
- **Result delivery**: Scheduled job output is sent to Telegram as a user-visible message, with deterministic handling for success and failure notifications.
- **Telegram job management (MVP)**: Add operator commands to list current jobs and manage lifecycle from Telegram (`list`, `delete`, `pause`, `resume`, `run-now`) with stable job IDs and clear command responses.
- **Execution policy**: Define run-time safeguards for scheduled jobs (timeouts, overlapping runs policy, retries policy, and logging/audit shape).
- **Management safety controls**: Restrict job-management commands to authorized users, require explicit confirmation for destructive actions (at least `delete`), and audit all management operations.
- **Legacy removal (no compatibility)**: Remove the current `scheduled_tasks` code path and related config contract; do not provide migration adapters or dual-mode fallback.
- **Configuration cleanup**: Replace legacy configuration keys and examples with the new job schema and startup validation rules.
- **Documentation and tests**: Update docs and test suites to cover only the new scheduling model and remove legacy scheduler expectations.
- **Implementation references**: During design and implementation, it is allowed to study and reuse suitable ideas from OpenClaw and GoClaw GitHub projects as reference architectures, without introducing compatibility constraints with those projects.

## Success criteria

- A configured daily job like "Collect an AI news digest and send it at 09:00" executes as a scheduled agent turn and sends the generated digest to Telegram.
- Authorized Telegram users can view current jobs and execute lifecycle actions (`list`, `delete`, `pause`, `resume`, `run-now`) with deterministic, user-visible outcomes.
- The application no longer depends on or executes legacy `scheduled_tasks` action semantics (`notify`/tool-only scheduler path removed).
- Startup validation rejects invalid new job definitions with actionable errors; no compatibility fallback is provided for legacy task schema.
- Destructive job operations require confirmation and are recorded in logs with actor and job ID.
- Updated docs describe only the new scheduling model and its operational controls (delivery, retries/overlap behavior, observability).
- Automated checks covering the new scheduler path pass in CI (including `make check` and AC validation in later stages).

## Traceability

- **Scope:** Extends the scheduler capability from [scope.md](../../scope.md) toward practical proactive assistant behavior while preserving reliability/security orientation.
- **Strategy:** Aligns with MVP evolution in [strategy.md](../../strategy.md): keep Telegram as the primary interaction channel while making automation actually useful for daily workflows.
- **Design inputs:** For stage 6 and stage 8 work, OpenClaw and GoClaw GitHub projects are approved as external implementation references for ideas and patterns, with no compatibility obligation for PersonalAssistant.

