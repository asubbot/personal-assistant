# Story scope — US-05 Dedicated PA user per node

**Story:** US-05  
**Title:** Dedicated PA user per node for SSH

---

## Formulation

As an operator, I want to configure one dedicated user account per node for PersonalAssistant SSH access, so that all actions are attributed to that identity.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-01.009](../../ep-acceptance-criteria.md#ac-01-009) | [REQ-01.013](../../ep-requirements.md#nodes-and-ssh) | One SSH user per node in config → core uses only that identity |
| [AC-01.010](../../ep-acceptance-criteria.md#ac-01-010) | [REQ-01.013](../../ep-requirements.md#nodes-and-ssh) | Multiple nodes → each uses its dedicated user, no shared account |
