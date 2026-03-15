# Story scope — US-16 Secret leakage protection

**Story:** US-16  
**Title:** Secret leakage protection (prompt injection)

---

## Formulation

As an operator or security-conscious user, I want the assistant to never expose secret values in LLM context, user-facing responses, or logs, so that credentials cannot be extracted via crafted prompts. Redaction SHALL use built-in patterns (defined in code and not overridable by configuration) and optional additional patterns from config.

## Requirements

[REQ-017](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration), [REQ-026](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration), [REQ-027](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration), [REQ-028](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration), [REQ-029](../../ep-requirements.md#secret-protection-prompt-injection--exfiltration)

## Acceptance criteria

[st-acceptance-criteria.md](st-acceptance-criteria.md) — AC-028, AC-029, AC-030, AC-038–AC-041
