# Epic scope — EP-033 Memory Summarization Retry

| Field | Content |
|-------|---------|
| **ID** | EP-033 |
| **Status** | DONE |
| **Title** | Memory Summarization Retry |
| **Description** | Add deterministic retry behavior for failed day summarization jobs in `memoryjob` so temporary LLM or runtime failures do not silently leave missing day summaries. Keep scope narrow: retries only, without expanding catch-up to wider date ranges. |
| **First version date** | 2026-04-21 |

## Glossary

- **Day summarization job**: Automatic memory summarization run for one calendar day in `pa_timezone`, executed as `catchup_day` or `summarize_yesterday`.
- **Retry policy**: Bounded retry plan with fixed attempt limit and backoff delays.
- **Backoff**: Delay before the next retry attempt after a failed execution.
- **Retryable failure**: Temporary failure where rerunning the same day job can succeed without changing configuration.
- **Non-retryable failure**: Failure that should fail fast and not be retried automatically.
- **Dedupe key**: Stable key used to avoid multiple queued retry jobs for the same day target.

## Scope (features/capabilities)

- Add bounded retries for failed day summarization flows in `memoryjob`.
- Apply retry policy to `catchup_day` and `summarize_yesterday` only.
- Keep month and year catch-up flows unchanged in this epic.
- Keep startup catch-up date window unchanged in this epic.
- Schedule retries in the existing `memoryjob` queue/execution model instead of adding a second worker subsystem.
- Ensure retries respect existing user-turn deferral behavior for background jobs.
- Prevent duplicate retry scheduling for the same target day.
- Emit structured retry logs with attempt number, next delay, and terminal-failure signal.
- Keep current success path and vector reconciliation behavior unchanged.
- Add deterministic tests for retry scheduling, backoff timing, dedupe, and retry exhaustion.

## Success criteria

- When `catchup_day` fails with a retryable error, the runner schedules retry attempts with bounded backoff and eventually succeeds if a later attempt succeeds.
- When `summarize_yesterday` fails with a retryable error, the runner schedules bounded retries and keeps existing priority/defer behavior.
- The system does not enqueue duplicate retry chains for the same day target.
- Retry exhaustion is explicit in logs and does not loop indefinitely.
- Existing month/year summarization behavior remains unchanged.
- Existing day summarization success path remains unchanged.
- `make check` passes.
- `./bin/validate EP-033` passes.

## Traceability

- **Scope:** Supports reliability goals in [scope.md](../../scope.md) by reducing silent misses in automatic memory summarization.
- **Strategy:** Aligns with [strategy.md](../../strategy.md) fail-fast and testability priorities using bounded retries plus explicit verification.
- **Related epics:** Extends automatic summarization runtime behavior from [EP-002](../EP-002/ep-scope.md) without changing long-term memory model from [EP-016](../EP-016/ep-scope.md).
