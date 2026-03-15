# Story scope — US-12 Extensible tools

**Story:** US-12  
**Title:** Extensible tools with single contract

---

## Formulation

As a developer, I want to add new tools via a single contract without changing core orchestration code, so that capabilities can be extended in a modular way.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-022](../../ep-acceptance-criteria.md#ac-022) | [REQ-010](../../ep-requirements.md#scheduler-and-tools) | Tool registered with valid schema → single contract; invalid registration → reject/fail fast |
| [AC-023](../../ep-acceptance-criteria.md#ac-023) | [REQ-010](../../ep-requirements.md#scheduler-and-tools) | Invalid or out-of-schema tool input → validate and reject, tool not run |
| [AC-035](../../ep-acceptance-criteria.md#ac-035) | [REQ-010](../../ep-requirements.md#scheduler-and-tools) | Tool nil runner or runner error → error to caller, no execution |
