# EP-020 - Acceptance Criteria

This document defines testable acceptance criteria for natural-language scheduled job creation from Telegram chat.

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-20.001](#ac-20-001) | [REQ-20.001](ep-requirements.md#nl-request-intake), [REQ-20.002](ep-requirements.md#nl-request-intake) | Supported NL request is parsed as create operation |
| [AC-20.002](#ac-20-002) | [REQ-20.003](ep-requirements.md#job-creation), [REQ-20.004](ep-requirements.md#job-creation), [REQ-20.005](ep-requirements.md#job-creation) | Created job is persisted with active status, timezone, and delivery target |
| [AC-20.003](#ac-20-003) | [REQ-20.006](ep-requirements.md#user-feedback-safety-and-compatibility) | Creation confirmation is deterministic and includes required fields |
| [AC-20.004](#ac-20-004) | [REQ-20.007](ep-requirements.md#user-feedback-safety-and-compatibility) | Malformed request is rejected with no job side effects |
| [AC-20.005](#ac-20-005) | [REQ-20.008](ep-requirements.md#user-feedback-safety-and-compatibility) | NL-created jobs are visible/manageable via /jobs |
| [AC-20.006](#ac-20-006) | [REQ-20.009](ep-requirements.md#runtime-security-and-observability) | Created job executes and delivers output |
| [AC-20.007](#ac-20-007) | [REQ-20.010](ep-requirements.md#runtime-security-and-observability), [REQ-20.011](ep-requirements.md#runtime-security-and-observability) | Unauthorized and non-matching messages do not create jobs |
| [AC-20.008](#ac-20-008) | [REQ-20.012](ep-requirements.md#runtime-security-and-observability) | Creation attempts are audited with actor/op/outcome fields |
| [AC-20.009](#ac-20-009) | [REQ-20.013](ep-requirements.md#nl-request-intake), [REQ-20.006](ep-requirements.md#user-feedback-safety-and-compatibility) | Explicit free-form request is created via native-tool fallback |

## Acceptance criteria

<a id="ac-20-001"></a>**AC-20.001** (Trace: REQ-20.001, REQ-20.002)  
Given an authorized user sends "Collect an AI news digest and send it at 09:00 every day"  
When the message is processed  
Then the system SHALL parse instruction and HH:MM daily schedule as a create request.

<a id="ac-20-002"></a>**AC-20.002** (Trace: REQ-20.003, REQ-20.004, REQ-20.005)  
Given a parsed valid create request  
When creation succeeds  
Then the persisted job SHALL be active and include default timezone and current chat as delivery target.

<a id="ac-20-003"></a>**AC-20.003** (Trace: REQ-20.006)  
Given a successful creation  
When reply is returned to Telegram  
Then the reply SHALL contain Job ID, schedule, timezone, and next run timestamp.

<a id="ac-20-004"></a>**AC-20.004** (Trace: REQ-20.007)  
Given a request with malformed time syntax  
When the message is processed  
Then the system SHALL return deterministic guidance and SHALL NOT create any job.

<a id="ac-20-005"></a>**AC-20.005** (Trace: REQ-20.008)  
Given a job created from NL request  
When the operator runs `/jobs list` and `/jobs show <job_id>`  
Then the created job SHALL appear with expected metadata.

<a id="ac-20-006"></a>**AC-20.006** (Trace: REQ-20.009)  
Given a created active job is due or manually triggered  
When scheduler runtime executes the job  
Then generated output SHALL be delivered to Telegram delivery target.

<a id="ac-20-007"></a>**AC-20.007** (Trace: REQ-20.010, REQ-20.011)  
Given either an unauthorized user message or an authorized non-matching conversational message  
When message processing runs  
Then no scheduled job SHALL be created.

<a id="ac-20-008"></a>**AC-20.008** (Trace: REQ-20.012)  
Given a creation attempt is processed  
When audit logs are inspected  
Then each event SHALL include actor_user_id, operation, outcome, and job_id when available.

<a id="ac-20-009"></a>**AC-20.009** (Trace: REQ-20.013, REQ-20.006)  
Given an authorized user sends an explicit schedule-intent message that does not match strict template and contains HH:MM time  
When the message is processed  
Then the system SHALL create one job via native-tool fallback and SHALL return deterministic creation confirmation fields.
