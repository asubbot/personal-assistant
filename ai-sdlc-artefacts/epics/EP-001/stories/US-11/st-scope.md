# Story scope — US-11 Scheduled tasks

**Story:** US-11  
**Title:** Scheduled tasks (time or interval)

---

## Formulation

As an operator, I want to define scheduled tasks (time or interval) in configuration, so that the assistant can run periodic actions within the security model. Notify actions send messages to a configurable Telegram chat (REQ-023).

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-020](../../ep-acceptance-criteria.md#ac-020) | [REQ-009](../../ep-requirements.md#scheduler-and-tools), [REQ-023](../../ep-requirements.md#scheduler-and-tools) | Scheduled task runs within security model; notify → chat from config |
| [AC-021](../../ep-acceptance-criteria.md#ac-021) | [REQ-009](../../ep-requirements.md#scheduler-and-tools) | Task would violate security model → not executed |
| [AC-034](../../ep-acceptance-criteria.md#ac-034) | [REQ-009](../../ep-requirements.md#scheduler-and-tools) | Tasks file missing/invalid/duplicate names → empty list or error, no invalid tasks |
