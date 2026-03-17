# Story scope — US-03 Node config

**Story:** US-03  
**Title:** Node config — define and validate at startup

---

## Formulation

As an operator, I want to define nodes (host, SSH user, authentication) in configuration and have the system validate at startup, so that configuration errors are caught before serving.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-01.005](../../ep-acceptance-criteria.md#ac-01-005) | [REQ-01.003](../../ep-requirements.md#nodes-and-ssh), [REQ-01.024](../../ep-requirements.md#nodes-and-ssh) | Invalid node config or missing/invalid config file → refuse start or clear error |
| [AC-01.006](../../ep-acceptance-criteria.md#ac-01-006) | [REQ-01.004](../../ep-requirements.md#nodes-and-ssh) | Running core uses SSH and validated config only for nodes |
