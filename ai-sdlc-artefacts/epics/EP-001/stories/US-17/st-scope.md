# Story scope — US-17 Debug-level LLM logging

**Story:** US-17  
**Title:** Debug-level LLM conversation logging

---

## Formulation

As a developer or operator, I want to enable debug logging for LLM conversations via `PA_LOG_LEVEL=debug`, so that I can inspect the full request (including memory and vector context) and response when troubleshooting. By default (INFO level) only metadata is logged.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-01.031](../../ep-acceptance-criteria.md#ac-01-031) | [REQ-01.021](../../ep-requirements.md#llm-and-logging) | PA_LOG_LEVEL=debug → full request/response; INFO → metadata only |
