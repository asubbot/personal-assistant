# Epic scope — EP-005 SSH subsystem execution channel (pa-runner)

| Field | Content |
|-------|---------|
| **ID** | EP-005 |
| **Status** | NEW |
| **Title** | SSH subsystem execution channel (pa-runner) |
| **Description** | Deliver a named SSH subsystem (e.g. `pa-exec`) on each node that runs a Go **pa-runner** binary: encrypted channel and client authentication remain SSH; the execution payload is a **versioned structured request** (e.g. JSON with `argv[]`) so the node never interprets the payload via a shell. PersonalAssistant gains a client path that opens the subsystem and sends validated `argv` after tool validation and PA-side policy. Node onboarding installs and configures `Subsystem` in `sshd_config` for a dedicated PA user. Legacy `session.Run(command string)` remains available per-node until operators migrate. |

## Glossary

- **pa-runner:** Server-side Go binary started by `sshd` as an SSH subsystem. Reads one or more framed requests from stdin, executes via `execve` (or equivalent) with explicit `argv`, returns stdout/stderr and exit status to the client over the subsystem channel. No shell parses the user payload.
- **Subsystem (SSH):** Client request type `subsystem` with a name (e.g. `pa-exec`) that maps in `sshd_config` to a server program. Provides a dedicated bidirectional stream for an application protocol.
- **Execution request (v1):** Structured message from PA to pa-runner carrying at minimum a non-empty `argv` array (absolute path to binary as `argv[0]`, further arguments as separate elements). Format, limits (max JSON size, max args, max arg length), and optional protocol version field are fixed in a short spec produced in later stages.
- **Transport mode (per node):** Configuration flag per node: **legacy** (remote command as today: string over SSH exec / shell) vs **subsystem** (PA uses subsystem + structured request). Enables gradual rollout.

## Scope (features/capabilities)

- **Protocol specification:** Define version 1 request/response framing (e.g. single-line JSON per invocation or length-prefixed frame), hard limits on body size and argument count/length, and how exit code and stderr are surfaced to the PA client. Document compatibility expectations between PA and pa-runner versions.
- **pa-runner (Go):** Implement the subsystem entry: parse stdin per spec, validate structural constraints, `execve` the requested `argv` without invoking `/bin/sh -c`, propagate stdout/stderr and exit code (or structured error) back on the channel. Fail closed on malformed or oversized input.
- **sshd configuration:** Document and automate (e.g. extend deploy script) `Subsystem pa-exec /usr/local/bin/pa-runner` (or agreed path) for the dedicated PA user; ensure only intended keys can open sessions to that user. No node-side command allowlist required for this epic if policy remains PA-only; optional minimal hygiene (e.g. reject non-absolute `argv[0]`) may be specified in requirements.
- **PA SSH client:** Extend node execution so that when transport mode is **subsystem**, PA opens `pa-exec`, sends the execution request built from validated tool output (template substitution already done in PA → final `argv` split or explicit array from catalog), reads response. Reuse existing host key and key-based auth.
- **Per-node configuration:** Config field (or convention) to select legacy vs subsystem per node; default preserves current behaviour until subsystem is enabled and pa-runner is installed.
- **Deployment / onboarding:** Update or add operator steps (script or doc) to install pa-runner binary on target architectures, register subsystem in `sshd_config`, reload sshd, verify handshake with PA or CLI (e.g. `ssh -s user@host pa-exec` with test payload).
- **Testing:** Integration tests using the existing Docker SSH testbed pattern: subsystem session, send v1 request, assert stdout/exit code; regression tests for legacy mode unchanged.

## Success criteria

- A documented **v1 execution protocol** exists under this epic (requirements or system design).
- **pa-runner** builds for at least the same target OS/arch as current PA node support; subsystem session executes a multi-argument command without shell interpretation (verified by a test that would fail under `sh -c` injection).
- **PA** can run at least one tool on a test node via **subsystem** when transport mode is set, and still run via **legacy** when not.
- Operator instructions or script allow registering subsystem and pa-runner on a fresh dedicated user without breaking manual SSH admin access for other users.
- Unit and/or integration tests cover protocol parsing in pa-runner and PA subsystem client path; existing legacy SSH tests still pass.

## Traceability

- **Scope:** [scope.md](../../scope.md) — Core manages nodes over SSH under a validated security model; deployment and adding nodes kept simple. This epic hardens the execution wire format while keeping SSH as transport (encryption, host authentication, client keys).
- **Strategy:** [strategy.md](../../strategy.md) — Increment after MVP; integration tests for SSH-related behaviour; security checks explicit. Aligns with delivery increment **after** stable tool execution ([EP-004](../EP-004/ep-scope.md)) and node model ([EP-001](../EP-001/ep-scope.md)).
- **Related epics:** [EP-001](../EP-001/ep-scope.md) defines MVP node access and dedicated user; [EP-004](../EP-004/ep-scope.md) defines tool validation and `run_on_node`; EP-005 replaces the remote execution **transport** for nodes that opt in, without changing PA as sole source of command policy.

## Security and architecture notes (design guidance)

These points inform later **requirements** and **system design**; they do not expand functional scope unless explicitly adopted there.

- **Privilege isolation:** If **one process** both parses the subsystem wire format and holds access to powerful resources (e.g. Docker Engine via `docker.sock`), a single vulnerability in parsing or request handling can have a **large blast radius** (roughly host-level impact). **Splitting** into two processes from the **same binary**—a narrow **gateway** (protocol only, minimal rights) and an **executor** (only the executor opens `docker.sock` / runs children)—reduces that coupling. EP-005 does not require a split process; record it as an optional hardening path when Docker or similar is in play.

- **Protocols (reference stack):** **PA → node:** SSH, subsystem stream, **v1 structured payload** (no remote shell parsing of the payload). **Gateway ↔ executor (optional future layout):** local **IPC** (e.g. Unix domain socket with length-prefixed or line-framed JSON, or gRPC over UDS)—not internet-exposed. **Executor → Docker:** Docker Engine API, typically **HTTP over the Unix socket** (`docker.sock`), separate from SSH.

- **`docker run --privileged`:** The flag greatly widens container capabilities (weaker default seccomp, broader device/capability surface). **Prefer not to use** for assistant-driven workloads. **Current PA product path:** enforcement is mainly the per-node **command allowlist** and tool policy on the PA side; a **prefix-only** template check (e.g. EP-009) does **not** by itself forbid `--privileged` inserted after an allowed `docker run --rm --network bridge` prefix. Operators must keep allowlists **tight**; a **code-level denylist or, better, argv construction from a whitelist** of permitted Docker options closes that class of gap.

- **If pa-runner builds Docker invocations — primary control vs string checks:** Do **not** rely on **validating a finished shell-style command string** as the main gate. Instead **assemble `argv` from a whitelist**—e.g. `[]string{"docker", "run", "--rm", …}` passed to `execve` / `exec.Command` with **no shell**—and **do not allow unconstrained user-controlled fragments** in the “Docker flags” portion of that slice. The structured request should map only into **fixed fields** (allowed image refs, network enum, memory/CPU knobs, container argv). Then evasions such as `--privileged=true`, odd spacing, or shell-style escaping are **not** a problem for a bespoke string parser: disallowed combinations simply **never appear** in `argv` because only enumerated tokens and vetted values are emitted. Treat **substring / regex checks** on a joined string at most as a **secondary** safety net, not the primary policy.

## Out of scope (this epic)

- HMAC or signed envelopes (optional follow-up).
- Docker-in-Docker execution on the node (separate epic/decision).
- Removing legacy string-based execution globally before operators migrate all nodes.
