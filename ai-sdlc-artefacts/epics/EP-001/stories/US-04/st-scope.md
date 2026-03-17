# Story scope — US-04 Per-node allowlist

**Story:** US-04  
**Title:** Per-node allowlist — security model

---

## Formulation

As an operator, I want a documented security model that defines, per node, which commands or tools are allowed, so that only permitted actions run on each node.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-01.007](../../ep-acceptance-criteria.md#ac-01-007) | [REQ-01.005](../../ep-requirements.md#nodes-and-ssh) | Node allow list → only allowlisted commands/tools executed |
| [AC-01.008](../../ep-acceptance-criteria.md#ac-01-008) | [REQ-01.005](../../ep-requirements.md#nodes-and-ssh) | Requested action not on allow list → not executed, denial reported/logged |
