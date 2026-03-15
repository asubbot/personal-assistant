# Story scope — US-18 Verify node availability

**Story:** US-18  
**Title:** Verify node availability via CLI parameter

---

## Formulation

As an operator, I want to run the PersonalAssistant binary with a dedicated parameter to verify that SSH access to all configured nodes works, so that I can confirm credentials and allowlist without starting the bot.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-032](../../ep-acceptance-criteria.md#ac-032) | [REQ-022](../../ep-requirements.md#nodes-and-ssh) | Verify-nodes: connect per node, run allowlisted command, report, exit without serving |
