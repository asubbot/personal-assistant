# Epic scope - EP-020 Natural-Language Scheduled Job Creation from Telegram

| Field | Content |
|-------|---------|
| **ID** | EP-020 |
| **Status** | DONE |
| **Title** | Natural-Language Scheduled Job Creation from Telegram |
| **Description** | Allow authorized Telegram users to create scheduled agent jobs by sending natural-language requests like "Collect an AI news digest and send it at 09:00", using a hybrid creation path (strict parser first, native-tool fallback second) with automatic job creation and scheduled delivery on top of EP-019 runtime. |
| **First version date** | 2026-04-16 |

## Glossary

- **NL schedule request**: A free-text Telegram message that includes a task instruction and a delivery schedule expression in user-friendly form.
- **Schedule extraction**: Conversion of NL time intent (for example, "every day at 09:00") into a canonical schedule representation used by the scheduler.
- **Hybrid create flow**: Two-step creation strategy where deterministic parser is attempted first, and fallback extraction via native tool is attempted only for explicit schedule-intent messages.
- **Creation confirmation**: A deterministic response that includes created job id, interpreted schedule/timezone, and instruction summary.
- **Creation safety policy**: Validation and guardrails preventing malformed, ambiguous, unauthorized, or unsafe schedule-creation requests.

## Scope (features/capabilities)

- **NL job creation via Telegram chat**: Authorized users can request creation of scheduled jobs using natural language in regular chat flow (not only `/jobs` management commands).
- **Hybrid schedule interpretation**: The system first applies strict deterministic syntax extraction and then applies controlled native-tool fallback for explicit schedule-intent free-form phrases.
- **Automatic persistence and activation**: Valid requests create active jobs in the Job Store (`jobs.sqlite`) without manual DB editing.
- **Deterministic user feedback**: After creation, Telegram reply includes job id, interpreted schedule, timezone, and next run.
- **Delivery at requested time**: Created jobs are executed by existing EP-019 runtime and deliver generated results to Telegram at scheduled times.
- **Validation and rejection behavior**: Ambiguous or invalid time expressions return actionable errors; no silent fallback to unintended schedules.
- **Management compatibility**: Jobs created from NL requests are fully manageable via existing `/jobs list/show/pause/resume/run-now/delete`.
- **Auditability**: Creation attempts and outcomes are logged with actor id, operation type, parsed schedule payload, and outcome.

## Success criteria

- A user sends a message like "Collect an AI news digest and send it at 09:00 every day".
- The system automatically creates a scheduled job and confirms creation with a stable job id and interpreted schedule.
- A user sends an explicit schedule-intent free-form message like "Please collect AI news daily and send it at 09:00", and the system creates one scheduled job through fallback extraction.
- `/jobs list` shows the created job with expected schedule metadata.
- At requested time, the job executes as an agent turn and sends digest output to Telegram.
- Invalid/ambiguous NL schedule requests are rejected with deterministic guidance and no job creation side effects.
- Authorized-user and audit controls are enforced for creation operations.

## Out of scope / deferred

- Multi-step conversational setup wizard for complex recurrence rules.
- Rich natural-language recurrence beyond MVP patterns (for example, "every second Tuesday except holidays").
- UI beyond Telegram chat and existing command responses.
- Migration of historical free-text reminders from previous systems.

## Traceability

- Builds directly on EP-019 job store/runtime/management foundation.
- Extends Telegram interaction model from command-based management to natural-language creation flow.
- Keeps scheduler execution, overlap/timeout policies, and delivery semantics unchanged unless explicitly updated in later stages.
