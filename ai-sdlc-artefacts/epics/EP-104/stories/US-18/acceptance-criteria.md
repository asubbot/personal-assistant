# Acceptance criteria — US-18

**Story:** [08-user-stories.md](../../08-user-stories.md#us-18--verify-node-availability)

---

## AC-032 ([US-18](../../08-user-stories.md#us-18--verify-node-availability))

**Given** the application is invoked with the designated parameter to verify node availability (e.g. `-verify-nodes`), **When** the application runs, **Then** it loads the validated configuration and for each configured node connects over SSH using that node's credentials, runs one allowlisted command (e.g. `uptime` or a documented probe), and reports success or failure per node to stdout or stderr; **and** the application exits without starting the normal serving mode (e.g. Telegram bot). **Given** at least one node fails to connect or the allowlist cannot be loaded, **When** the verify run completes, **Then** the application exits with a non-zero status.
