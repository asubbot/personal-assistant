# Acceptance criteria — US-05

**Story:** [08-user-stories.md](../../08-user-stories.md#us-05--dedicated-pa-user-per-node)

---

## AC-009 ([US-05](../../08-user-stories.md#us-05--dedicated-pa-user-per-node))

**Given** node configuration that defines one SSH user for PersonalAssistant, **When** the core connects to that node, **Then** it uses only that user identity.

---

## AC-010 ([US-05](../../08-user-stories.md#us-05--dedicated-pa-user-per-node))

**Given** multiple nodes, **When** the core connects to each, **Then** each connection uses the dedicated user defined for that node (no shared or alternate account).
