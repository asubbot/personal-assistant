# EP-019 — Acceptance criteria

**Introduction:** Testable acceptance criteria for **EP-019 Scheduled Agent Jobs and Legacy Scheduler Replacement**. The document defines when scheduled agent jobs, Telegram management commands, legacy removal, and non-functional controls are acceptable for delivery. Each AC traces to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-19.001](#ac-19-001) | [REQ-19.001](ep-requirements.md#job-model-and-scheduling) | Job Store keeps stable unique Job ID |
| [AC-19.002](#ac-19-002) | [REQ-19.002](ep-requirements.md#job-model-and-scheduling) | Scheduler loads persisted jobs before management |
| [AC-19.003](#ac-19-003) | [REQ-19.003](ep-requirements.md#job-model-and-scheduling) | Schedule evaluation uses job time zone |
| [AC-19.004](#ac-19-004) | [REQ-19.004](ep-requirements.md#job-model-and-scheduling) | Next run timestamp is available per job |
| [AC-19.005](#ac-19-005) | [REQ-19.005](ep-requirements.md#job-execution-and-delivery) | Due job creates a Job Run |
| [AC-19.006](#ac-19-006) | [REQ-19.006](ep-requirements.md#job-execution-and-delivery) | Job Run executes via standard agent-turn path |
| [AC-19.007](#ac-19-007) | [REQ-19.007](ep-requirements.md#job-execution-and-delivery) | Success result is delivered to Telegram target |
| [AC-19.008](#ac-19-008) | [REQ-19.008](ep-requirements.md#job-execution-and-delivery) | Failure delivery includes reason class |
| [AC-19.009](#ac-19-009) | [REQ-19.009](ep-requirements.md#job-execution-and-delivery) | Run timeout policy is enforced |
| [AC-19.010](#ac-19-010) | [REQ-19.010](ep-requirements.md#job-execution-and-delivery) | Overlap policy single-instance skips due run |
| [AC-19.011](#ac-19-011) | [REQ-19.011](ep-requirements.md#telegram-job-management) | `list` returns required job fields |
| [AC-19.012](#ac-19-012) | [REQ-19.012](ep-requirements.md#telegram-job-management) | `show` returns detailed job card |
| [AC-19.013](#ac-19-013) | [REQ-19.013](ep-requirements.md#telegram-job-management) | `pause` changes job status to paused |
| [AC-19.014](#ac-19-014) | [REQ-19.014](ep-requirements.md#telegram-job-management) | `resume` changes job status to active |
| [AC-19.015](#ac-19-015) | [REQ-19.015](ep-requirements.md#telegram-job-management) | `run-now` enqueues immediate run |
| [AC-19.016](#ac-19-016) | [REQ-19.016](ep-requirements.md#telegram-job-management) | `delete` returns confirmation challenge |
| [AC-19.017](#ac-19-017) | [REQ-19.017](ep-requirements.md#telegram-job-management) | Confirmed delete removes job |
| [AC-19.018](#ac-19-018) | [REQ-19.018](ep-requirements.md#telegram-job-management) | Unauthorized command is rejected and audited |
| [AC-19.019](#ac-19-019) | [REQ-19.019](ep-requirements.md#legacy-replacement-and-configuration) | Legacy `scheduled_tasks` config fails startup |
| [AC-19.020](#ac-19-020) | [REQ-19.020](ep-requirements.md#legacy-replacement-and-configuration) | Docs/examples show only new job schema |
| [AC-19.021](#ac-19-021) | [REQ-19.021](ep-requirements.md#non-functional-requirements) | Audit log records lifecycle and management events |
| [AC-19.022](#ac-19-022) | [REQ-19.022](ep-requirements.md#non-functional-requirements) | `list` responsiveness is validated per profile |
| [AC-19.023](#ac-19-023) | [REQ-19.007](ep-requirements.md#job-execution-and-delivery), [REQ-19.011](ep-requirements.md#telegram-job-management), [REQ-19.016](ep-requirements.md#telegram-job-management) | End-to-end daily digest plus operator management flow |

---

## Acceptance criteria

<a id="ac-19-001"></a>**AC-19.001** (Trace: REQ-19.001)  
Given a new Scheduled Agent Job is created  
When the job is persisted in Job Store  
Then the stored record SHALL include a stable Job ID that is unique among all stored jobs.

<a id="ac-19-002"></a>**AC-19.002** (Trace: REQ-19.002)  
Given persisted jobs exist in Job Store  
When the PersonalAssistant System starts  
Then Scheduler SHALL load all persisted jobs before processing Management Commands.

<a id="ac-19-003"></a>**AC-19.003** (Trace: REQ-19.003)  
Given a Scheduled Agent Job has Cron Expression and Time Zone  
When Scheduler computes due runs  
Then due evaluation SHALL use the configured Time Zone of that job.

<a id="ac-19-004"></a>**AC-19.004** (Trace: REQ-19.004)  
Given a persisted Scheduled Agent Job  
When the job appears in management output  
Then output SHALL include a computed next run timestamp.

<a id="ac-19-005"></a>**AC-19.005** (Trace: REQ-19.005)  
Given a Scheduled Agent Job reaches due time  
When Scheduler tick processes schedules  
Then Scheduler SHALL create one Job Run record for that due occurrence.

<a id="ac-19-006"></a>**AC-19.006** (Trace: REQ-19.006)  
Given a Job Run has started  
When execution begins  
Then the run SHALL use the standard agent-turn orchestration path for LLM, tools, and memory.

<a id="ac-19-007"></a>**AC-19.007** (Trace: REQ-19.007)  
Given a Job Run completes successfully  
When delivery is performed  
Then a Telegram message with generated result SHALL be sent to the configured Delivery Target.

<a id="ac-19-008"></a>**AC-19.008** (Trace: REQ-19.008)  
Given a Job Run ends with failure  
When failure delivery is performed  
Then a Telegram message SHALL include the run failure reason class.

<a id="ac-19-009"></a>**AC-19.009** (Trace: REQ-19.009)  
Given run timeout policy is configured  
When a Job Run exceeds allowed execution window  
Then Scheduler SHALL stop execution according to policy and mark run outcome accordingly.

<a id="ac-19-010"></a>**AC-19.010** (Trace: REQ-19.010)  
Given overlap policy is single-instance and one run is active  
When next due occurrence is reached  
Then Scheduler SHALL skip that due occurrence and record a skip event.

<a id="ac-19-011"></a>**AC-19.011** (Trace: REQ-19.011)  
Given an Authorized Operator sends `list`  
When the command is processed  
Then response SHALL include Job ID, schedule, Time Zone, status, and next run for each job.

<a id="ac-19-012"></a>**AC-19.012** (Trace: REQ-19.012)  
Given an Authorized Operator sends `show` with valid Job ID  
When the command is processed  
Then response SHALL include instruction summary, Delivery Target, last run status, and next run timestamp.

<a id="ac-19-013"></a>**AC-19.013** (Trace: REQ-19.013)  
Given an Authorized Operator sends `pause` with valid Job ID  
When the command is processed  
Then target job status SHALL become paused and subsequent schedule triggers SHALL not start runs.

<a id="ac-19-014"></a>**AC-19.014** (Trace: REQ-19.014)  
Given an Authorized Operator sends `resume` with valid Job ID  
When the command is processed  
Then target job status SHALL become active and future due triggers SHALL be eligible for execution.

<a id="ac-19-015"></a>**AC-19.015** (Trace: REQ-19.015)  
Given an Authorized Operator sends `run-now` with valid Job ID  
When the command is processed  
Then Scheduler SHALL enqueue an immediate Job Run for that job.

<a id="ac-19-016"></a>**AC-19.016** (Trace: REQ-19.016)  
Given an Authorized Operator sends `delete` with valid Job ID  
When the command is processed  
Then the system SHALL return a confirmation challenge containing a Confirmation Token bound to that Job ID.

<a id="ac-19-017"></a>**AC-19.017** (Trace: REQ-19.017)  
Given an Authorized Operator sends deletion confirmation with valid token and matching Job ID  
When the confirmation is processed  
Then Job Store SHALL remove the job and subsequent `show` SHALL return job-not-found.

<a id="ac-19-018"></a>**AC-19.018** (Trace: REQ-19.018)  
Given a Telegram user is not an Authorized Operator  
When the user sends any Management Command  
Then the command SHALL be rejected and an audit event with user ID and command name SHALL be written.

<a id="ac-19-019"></a>**AC-19.019** (Trace: REQ-19.019)  
Given configuration contains legacy `scheduled_tasks` schema fields  
When configuration loader validates input  
Then startup SHALL fail and error output SHALL name each unsupported legacy field.

<a id="ac-19-020"></a>**AC-19.020** (Trace: REQ-19.020)  
Given product documentation and config examples for scheduling  
When the documentation set is reviewed  
Then all scheduling references SHALL use only the new Scheduled Agent Job schema.

<a id="ac-19-021"></a>**AC-19.021** (Trace: REQ-19.021)  
Given a management operation or run lifecycle transition occurs  
When audit logging is inspected  
Then each event SHALL include timestamp, actor identity, Job ID, operation type, and outcome.

<a id="ac-19-022"></a>**AC-19.022** (Trace: REQ-19.022)  
Given a deployment profile defines measurable responsiveness thresholds for `list`  
When the defined acceptance test for that profile is executed  
Then the `list` command behavior SHALL satisfy that profile threshold.

<a id="ac-19-023"></a>**AC-19.023** (Trace: REQ-19.007, REQ-19.011, REQ-19.016)  
Given a daily digest job exists and an Authorized Operator manages jobs via Telegram  
When the scheduled time is reached and operator performs `list` then `delete` with confirmation  
Then digest delivery SHALL appear in Telegram, `list` SHALL show expected job state before deletion, and post-confirmation `list` SHALL no longer include the deleted job.
