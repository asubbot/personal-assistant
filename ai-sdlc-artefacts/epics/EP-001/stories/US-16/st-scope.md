# Story scope — US-16 Secret leakage protection

**Story:** US-16  
**Title:** Secret leakage protection (prompt injection)

---

## Formulation

As an operator or security-conscious user, I want the assistant to never expose secret values in LLM context, user-facing responses, or logs, so that credentials cannot be extracted via crafted prompts. Redaction SHALL use built-in patterns (defined in code and not overridable by configuration) and optional additional patterns from config.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-028](../../ep-acceptance-criteria.md#ac-028) | [REQ-017](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Context build with fake secret → context does not contain it |
| [AC-029](../../ep-acceptance-criteria.md#ac-029) | [REQ-017](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Prompt-injection message with fake secret → reply and logs do not contain it |
| [AC-030](../../ep-acceptance-criteria.md#ac-030) | [REQ-017](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Flow using secrets → log stream does not contain fake secret values |
| [AC-038](../../ep-acceptance-criteria.md#ac-038) | [REQ-026](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration), [REQ-027](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Built-in redaction applied to LLM log and app log before write |
| [AC-039](../../ep-acceptance-criteria.md#ac-039) | [REQ-027](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | No or empty log_redaction → built-in only, start success |
| [AC-040](../../ep-acceptance-criteria.md#ac-040) | [REQ-028](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Additional patterns in config → applied with built-in; ids not equal built-in |
| [AC-041](../../ep-acceptance-criteria.md#ac-041) | [REQ-029](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration) | Reserved id or invalid regex in redaction config → refuse start, clear error |
