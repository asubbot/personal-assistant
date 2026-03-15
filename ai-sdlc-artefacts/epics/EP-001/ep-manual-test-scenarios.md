# EP-001 Manual test scenarios

**Purpose:** Document manual verification steps for acceptance criteria that are not fully covered by automated tests. See [ep-req-ac-test-coverage.md](ep-req-ac-test-coverage.md) for the full coverage matrix.

**Reference:** [strategy.md](../../strategy.md) §2.3 (Manual testing), [ep-acceptance-criteria.md](ep-acceptance-criteria.md).

---

## AC-004 — Image builds and runs on DS220+ (REQ-002) {#ac-004}


**Criterion:** Image builds and runs on DS220+ (or equivalent x86_64) without code change.

**Steps:**

1. Build the Docker image (from repo root):
   - `docker build -t pa:test .` (or use the path to the Dockerfile documented in the project).
2. Run the container on an x86_64 host (Synology DS220+ or equivalent):
   - Mount config (and any required secrets/data) per project documentation.
   - Start the container; confirm the core starts and exposes or uses the configured interfaces (e.g. Telegram webhook, config mount).
3. **Pass:** Container runs; no code change was required for the target platform.

**Optional:** Repeat on a generic x86_64 Linux host if DS220+ is not available; document the environment.

---

## AC-032 — Verify-nodes (REQ-022) {#ac-032}


**Criterion:** With the designated parameter (e.g. `-verify-nodes`), the application loads config, connects to each node over SSH using that node’s credentials, runs one allowlisted command (e.g. probe), reports success or failure per node, and exits without starting normal serving mode. Non-zero exit if config/allowlist load fails or any node fails.

**Steps:**

1. Prepare valid config with at least one node (host, dedicated user, auth, command allowlist). Ensure the allowlist includes a safe probe command (e.g. `uptime`).
2. Run:
   - `pa -verify-nodes` (or the binary name and flag as documented in the project).
3. **Pass:** Application connects to each node, runs the allowlisted probe, prints per-node result to stdout/stderr, exits with 0 (all nodes OK).
4. **Pass (failure path):** With invalid config, missing allowlist, or unreachable node, application exits with non-zero status and does not start Telegram/serving.

**Note:** Requires real or test SSH nodes (or documented mock setup) for full validation.

---

## Optional manual checks (strategy §2.3)

- **Node access and allowlist:** Verify node access and allowlist behaviour via CLI or documented steps (overlap with AC-032 and integration tests).
- **LLM log format:** Optionally inspect LLM log output for format and completeness (AC-017, AC-018).
- **Architecture and docs:** Review architecture and documentation where required by acceptance criteria.
