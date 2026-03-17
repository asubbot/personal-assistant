# Story scope — US-16 Secret leakage protection

**Story:** US-16  
**Title:** Secret leakage protection (prompt injection)

---

## Formulation

As an operator or security-conscious user, I want the assistant to never expose secret values in LLM context, user-facing responses, or logs, so that credentials cannot be extracted via crafted prompts. Redaction SHALL use built-in patterns (defined in code and not overridable by configuration) and optional additional patterns from config.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-01.028](../../ep-acceptance-criteria.md#ac-01-028) | [REQ-01.017](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Context build with fake secret → context does not contain it |
| [AC-01.029](../../ep-acceptance-criteria.md#ac-01-029) | [REQ-01.017](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Prompt-injection message with fake secret → reply and logs do not contain it |
| [AC-01.030](../../ep-acceptance-criteria.md#ac-01-030) | [REQ-01.017](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Flow using secrets → log stream does not contain fake secret values |
| [AC-01.038](../../ep-acceptance-criteria.md#ac-01-038) | [REQ-01.026](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration), [REQ-01.027](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Built-in redaction applied to LLM log and app log before write |
| [AC-01.039](../../ep-acceptance-criteria.md#ac-01-039) | [REQ-01.027](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | No or empty log_redaction → built-in only, start success |
| [AC-01.040](../../ep-acceptance-criteria.md#ac-01-040) | [REQ-01.028](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Additional patterns in config → applied with built-in; ids not equal built-in |
| [AC-01.041](../../ep-acceptance-criteria.md#ac-01-041) | [REQ-01.029](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Reserved id or invalid regex in redaction config → refuse start, clear error |
