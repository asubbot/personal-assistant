# Story scope — US-08 Pluggable LLM provider

**Story:** US-08  
**Title:** Pluggable LLM provider

---

## Formulation

As an operator, I want to choose and configure the LLM provider via configuration without code changes, so that I can avoid vendor lock-in.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-015](../../ep-acceptance-criteria.md#ac-015) | [REQ-008](../../ep-requirements.md#llm-and-logging) | LLM provider in config → core uses it without code change |
| [AC-016](../../ep-acceptance-criteria.md#ac-016) | [REQ-008](../../ep-requirements.md#llm-and-logging) | Provider switch in config + restart → new provider used |
| [AC-036](../../ep-acceptance-criteria.md#ac-036) | [REQ-025](../../ep-requirements.md#llm-and-logging) | LLM provider error → handled, no crash |
