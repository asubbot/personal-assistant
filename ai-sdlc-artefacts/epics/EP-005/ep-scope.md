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

## Out of scope (this epic)

- HMAC or signed envelopes (optional follow-up).
- Docker-in-Docker execution on the node (separate epic/decision).
- Removing legacy string-based execution globally before operators migrate all nodes.
