# Code review / Quality gate — <epic or change>

**Author:**  
**Date:**  
**Scope:** Epic ref ID or PR/branch

---

## Verdict

- **Ready for test execution:** Yes | No (blocked)
- **Blocking issues:** (list or "None")
- **Non-blocking follow-ups:** (list or "None")

## AC coverage matrix

| AC | Covered (Yes/Partial/No) | Test file refs | Gaps |
|----|---------------------------|----------------|------|
| AC-01 | … | … | … |
| … | … | … | … |

*AC with no tests must be marked as blocking. Rows ordered by AC ascending.*

## Findings (by severity)

### Blocking

- **Location (path:line):** What is wrong; why it is risky; how to fix. Reference REQ/AC.

### Non-blocking

- **Location:** What could be improved; recommendation.

## Verification

- `make check` (or equivalent): pass/fail
- Other checks run: …
