# Story scope — US-19 Startup validation

**Story:** US-19  
**Title:** Startup validation — refuse to start on invalid config

---

## Formulation

As an operator, I want the system to validate all configuration (nodes, Telegram, LLM, embedding, paths) at startup and refuse to start with a clear error when invalid, so that I can fix configuration before serving.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-01.033](../../ep-acceptance-criteria.md#ac-01-033) | [REQ-01.024](../../ep-requirements.md#nodes-and-ssh), [REQ-01.003](../../ep-requirements.md#nodes-and-ssh) | Invalid/incomplete config → refuse start or clear error |
