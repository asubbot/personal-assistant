# Acceptance criteria — US-04

**Story:** [08-user-stories.md](../../08-user-stories.md#us-04--per-node-allowlist)

---

## AC-007 ([US-04](../../08-user-stories.md#us-04--per-node-allowlist))

**Given** a node with an allow list of commands/tools, **When** the core invokes an action on that node, **Then** only commands or tools on the allow list are executed.

---

## AC-008 ([US-04](../../08-user-stories.md#us-04--per-node-allowlist))

**Given** a node whose allow list does not include a requested action, **When** the core would invoke that action, **Then** the system does not execute it and reports or logs the denial.
