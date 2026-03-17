# Story scope — US-08 Pluggable LLM provider

**Story:** US-08  
**Title:** Pluggable LLM provider

---

## Formulation

As an operator, I want to choose and configure the LLM provider via configuration without code changes, so that I can avoid vendor lock-in.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-01.015](../../ep-acceptance-criteria.md#ac-01-015) | [REQ-01.008](../../ep-requirements.md#llm-and-logging) | LLM provider in config → core uses it without code change |
| [AC-01.016](../../ep-acceptance-criteria.md#ac-01-016) | [REQ-01.008](../../ep-requirements.md#llm-and-logging) | Provider switch in config + restart → new provider used |
| [AC-01.036](../../ep-acceptance-criteria.md#ac-01-036) | [REQ-01.025](../../ep-requirements.md#llm-and-logging) | LLM provider error → handled, no crash |
