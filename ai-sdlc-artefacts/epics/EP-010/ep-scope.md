# Epic scope — EP-010 Distributed remote Go tool pipeline

| Field | Content |
|-------|---------|
| **ID** | EP-010 |
| **Status** | CANCELED (UX is not good for the product) |
| **Title** | Distributed remote Go tool pipeline |
| **Description** | Deliver an end-to-end, PA-orchestrated pipeline to build **static Go tool binaries** on remote nodes (builder container), transfer artifacts to PA, publish metadata in a **YAML-first catalog**, obtain **owner approval** before deploy, **push binaries to nodes** over SSH, and **invoke** them through the existing SSH + allowlist model—aligned with the distributed tooling vision. |
| **First version date** | 2026-04-07 |

## Branch and code retention (canceled epic)

- The implementation of this epic lives on a **separate Git branch** that **will not be merged** into `main`.
- That code is kept **for history and reference only** (design exploration, tests, operator doc snapshot). It is **not** a supported or shipped product feature.
- Product line remains on **`main`** without this pipeline unless a future epic revives the approach with revised UX.

## Glossary

- **PA (brain host):** Core PersonalAssistant instance (e.g. on Synology) holding LLM keys, agent loop, **local Git repository** for pipeline commits on PA, tool catalog, and orchestration; does not run arbitrary untrusted tool build logic for this epic’s target flow on itself.
- **Remote node:** SSH-reachable host where the builder container runs under PA-initiated commands; **less trusted** than PA for secrets.
- **Builder image:** OCI image with pinned Go toolchain, project/ai-sdlc inputs as needed, and reproducible build steps; PA orchestrates `docker pull` / `run` / `exec` on the node (vision **G**, **J**).
- **Job:** A bounded build/publish unit on PA with identity binding for artifact transfer (vision **H**): job ID plus secret or short-lived PA-issued JWT.
- **Artifact bundle:** Archive on the node (sources, `ai-sdlc-artefacts` when used, static binary) transferred **to PA** by default via **scp/sftp pull** from PA (vision **I**); subject to size limits (vision **K**).
- **Tool catalog (YAML):** Source of truth for agent-facing tool metadata, storage pointer, **digest**, release timestamp (UTC), and **remote executable path** on nodes (vision **L**); optional DB projection out of scope unless required later.
- **Deploy (N):** After **owner approval** (vision **D**), PA copies the static binary **PA → node** with `scp`/`rsync`, idempotent layout, versioned filenames and/or `current` symlink.
- **SSH step template:** Allowlisted remote command pattern (prefix or exact) used for builder orchestration or deploy; **separate** families for builder vs runtime vs deploy (vision **§7.3.2**, **N**).

## Scope (features/capabilities)

- **Builder image specification:** Documented and versioned definition of the builder image (base, Go version, embedded `ai-sdlc` / repo layout for optional `./bin/validate` and `make check` on the node per tool profile **C**).
- **PA-orchestrated remote build:** PA drives **only** allowlisted SSH commands (or an optional **versioned wrapper script** on the node plus a small set of explicit `docker` templates) to pull the builder image, run the container, execute build and optional validation steps; **no** LLM keys on the node; **no** using raw model prose as the command driver (vision **A**, **J**).
- **Job binding for artifacts:** PA associates artifact pull and subsequent git/catalog steps with a **job** using **job ID + secret** or **short-lived JWT** issued by PA (vision **H**); exact storage of secrets and rotation left to design stage.
- **Transfer node → PA:** Implement or integrate **scp/sftp pull** from PA for the final bundle; enforce **default and hard caps** for archive size consistent with vision **K**; document alternatives (e.g. rsync over SSH) as fallbacks when limits or policy require.
- **Git only on PA:** After bundle receipt, PA (or operator-assisted automation on PA) performs **git commit** on a **job branch** in the **local Git repository on PA**; nodes do **not** hold git credentials (vision **I**). **`git push` to any configured remote** (for example `origin`) is **out of scope** for this epic—operators may push outside the product or a later epic may add it.
- **Owner approval gate:** Workflow or explicit operator action records **owner approval** before any **deploy to nodes** (vision **D**); no automatic production deploy without passing this gate.
- **Deploy binary to nodes:** Idempotent **PA → node** copy of the **static** (`CGO_ENABLED=0` default) binary (vision **M**, **N**); layout includes versioned path and **`current`** (or equivalent); **separate** allowlist patterns for deploy commands vs builder steps.
- **YAML catalog extension:** Minimal schema and workflow to add or update catalog entries with **digest**, **released_at** (UTC), storage key/path, and **remote_path** used at runtime; optional cosign **out of scope** unless pulled in by a later decision.
- **Runtime alignment:** Documented mapping between **catalog.remote_path** (or equivalent) and **allowlist** entries so `RunOnNode` can invoke the deployed tool; same SSH channel as today; **EP-005** subsystem not required for MVP of this epic (vision **§7.1**, **§7.8**).
- **Limits and observability:** Configurable enforcement of vision **K** limits (remote command length, stdin, stdout/stderr capture, per-job log budget, archive size); truncation or spill to disk where specified in later design.
- **Tiered ai-sdlc profile:** Support **at least** a “light” path (minimal checks) and a “full” path (full `ai-sdlc` / `validate` where applicable) with explicit choice per job or tool class (vision **C**, **§10**).

## Success criteria

- One **documented end-to-end path** (can be driven manually or by tests): start job on PA → remote build on a configured node → bundle on PA → commit on job branch → owner approval → deploy to a target node → successful **allowlisted** remote invocation of the new binary **initiated by PA**. The **product entry** for operators is the **`pa`** binary with **`-remote-tool-builder`** (phases `build`, `accept`, `approve`, `deploy`, `build-accept`); see [docs/remote-go-builder-pipeline.md](../../../docs/remote-go-builder-pipeline.md).
- **Static** Go binary is produced with **`CGO_ENABLED=0`** by default for released artifacts; exceptions require owner-approved exception recorded in catalog or job metadata.
- **Artifact transfer** uses the agreed default (scp/sftp pull) and **refuses or truncates** per **K** when limits are exceeded (behaviour defined in requirements/design, not ambiguous).
- **Deploy** does not run before the **owner approval** gate is satisfied.
- **Catalog** contains a **YAML** record with **digest** and **released_at** for at least one shipped tool in the reference flow; runtime invocation uses a path **consistent** with that record and allowlist.
- **Builder** and **deploy** remote command patterns are **separate** in configuration/docs to reduce accidental reuse of dangerous templates.
- Automated tests cover **critical policy** (allowlist separation, limit checks, job binding where feasible); integration or manual steps documented for full SSH/docker path.

## Out of scope / deferred

- **EP-005 `pa-runner` / SSH subsystem** as the transport (may follow this epic; not part of EP-010 MVP).
- **Multi-platform** build matrix (vision **B**); single **GOOS/GOARCH** target for the epic unless scope is explicitly expanded later.
- **Reverse callbacks** from node/container to PA for LLM (vision **J**).
- **OCI image per tool** as runtime delivery; **OS packages**; **Ansible**-style fleet management (vision **N**).
- **SPIFFE**, mandatory **mTLS** for job binding, **OAuth device flow** for H (vision **E**, **H**).
- Replacing **EP-009** dynamic Python/Node sandbox creation; this epic targets **compiled Go tools** and catalog/deploy discipline.
- **`git push`** to a configured Git remote from PA as part of the pipeline (publishing job-branch commits upstream)—**out of scope**; local commits on PA remain in scope.

## Traceability

- **Scope:** Supports [scope.md](../../scope.md) goals for **nodes**, **security model** (allowlist, dedicated user), and **tool extensibility** with a reproducible, auditable path.
- **Strategy:** Aligns with [strategy.md](../../strategy.md): incremental MVP, testable behaviour, security checks for allowlists and secrets handling.
- **Vision (source of conditions A–N):** [pa-distributed-tooling-vision.md](../../analytics/pa-distributed-tooling-vision/pa-distributed-tooling-vision.md).
- **Related epics:**
  - [EP-001](../EP-001/ep-scope.md) — node model and SSH execution baseline.
  - [EP-004](../EP-004/ep-scope.md) — tool catalog and validation at execution.
  - [EP-005](../EP-005/ep-scope.md) — future structured remote execution; referenced as follow-on, not in EP-010 MVP.
  - [EP-009](../EP-009/ep-scope.md) — dynamic interpreted tools in Docker sandbox; complementary, not superseded by EP-010.
