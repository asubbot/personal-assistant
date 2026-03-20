# EP-006 Manual tests

**Purpose:** Manual verification for **tool-call escalation**, **baseline / rollback**, and **observability** where real LLM APIs, Telegram, or operator log review are needed. Automated coverage is in `internal/core`, `internal/escalationpolicy`, and `tests/integration` (see [ep-implementation-plan.md](ep-implementation-plan.md)).

**Reference:** [strategy.md](../../strategy.md) §2.3 (Manual testing), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-requirements.md](ep-requirements.md), [README.md](../../../README.md) (Config: `tools.llm_escalation`).

*Each scenario has a **Trace** line (links to AC / REQ). Record pass/fail in a release checklist or test log when signing off a deployment.*

**If a step says “provoke” and you are unsure how:** read **[How to execute the steps (operators)](#how-to-execute-the-steps-operators)** first (§ A–E).

## Navigation

| AC | REQ | Name |
|----|-----|------|
| — | — | **[How to execute (operators)](#how-to-execute-the-steps-operators)** |
| [AC-06.011](ep-acceptance-criteria.md#ac-06-011) | [REQ-06.014](ep-requirements.md#nfr--security-testability-observability) | [Escalation disabled — no provider advance on qualifying failure](#mt-esc-off) |
| [AC-06.005](ep-acceptance-criteria.md#ac-06-005) · [AC-06.006](ep-acceptance-criteria.md#ac-06-006) | [REQ-06.006](ep-requirements.md#escalation-policy-and-chain) · [REQ-06.007](ep-requirements.md#escalation-policy-and-chain) | [Escalation enabled — qualifying failure advances to next provider](#mt-esc-on) |
| [AC-06.007](ep-acceptance-criteria.md#ac-06-007) | [REQ-06.008](ep-requirements.md#exhaustion-and-stop) | [Max escalations per user message](#mt-max-esc) |
| [AC-06.008](ep-acceptance-criteria.md#ac-06-008) | [REQ-06.009](ep-requirements.md#rollback-at-end-of-turn) | [Baseline reset on the next user message](#mt-baseline-reset) |
| [AC-06.013](ep-acceptance-criteria.md#ac-06-013) | [REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | [Hermes (text-tool) parse failure and escalation](#mt-hermes) |
| [AC-06.009](ep-acceptance-criteria.md#ac-06-009) | [REQ-06.010](ep-requirements.md#observability) · [REQ-06.011](ep-requirements.md#observability) · [REQ-06.012](ep-requirements.md#nfr--security-testability-observability) | [No secrets in escalation and LLM logs](#mt-no-secrets) |
| [AC-06.004](ep-acceptance-criteria.md#ac-06-004) | [REQ-06.005](ep-requirements.md#error-classification) | [Non-qualifying tool failures — no escalation](#mt-non-qual) |
| [AC-06.002](ep-acceptance-criteria.md#ac-06-002) | [REQ-06.002](ep-requirements.md#baseline-and-configuration) | [Invalid escalation config at startup](#mt-invalid-config) |
| [AC-06.010](ep-acceptance-criteria.md#ac-06-010) | [REQ-06.013](ep-requirements.md#nfr--security-testability-observability) | [Operator checklist after deploy](#mt-operator) |

---

## How to execute the steps (operators)

EP-006 escalation runs **after** a **`Complete`** returns (e.g. tool calls) and something **qualifying** happens—usually a **tool / node execution** error wrapped as *may escalate*, or a **Hermes parse** error on the text-tool path. The steps below say “provoke …” because your catalog, nodes, and models differ; pick **one** method you can do safely in a **staging** environment.

### A — Qualifying failure via `run_on_node` (most reproducible)

What counts: SSH connect failure, remote command error, or similar node path → policy treats it as **qualifying** (see implementation: `escalationpolicy.WrapNodeOutcome` / `MayEscalate`).

1. In [config](../../../config/config.json) (or your deploy config): **`tools.llm_escalation.enabled` true**, at least **two** `llm_providers`, valid `baseline_index` and `max_per_user_message`.
2. Keep a tool that uses **`run_on_node`** (e.g. `node_time`, `run_echo`—whatever exists in your [tool catalog](../../../config/tools.yaml)).
3. **Break only execution** (not catalog validation):
   - **Easiest:** set the node’s **`host`** to an **unreachable** IP/hostname, **restart** the app, send a Telegram message that still causes the model to request that tool; or  
   - stop the SSH daemon / container on the target node while keeping config unchanged; or  
   - temporarily point **`private_key_path`** to a wrong file for that node (connection fails).
4. Watch **application logs** (stdout/file per your setup): expect **`tool invocation`** with an error, then (if escalation on) **`llm tool escalation`** with `failure_class=tool_execution` and `from_index` / `to_index` or provider labels.
5. **Restore** node/SSH/config after the test.

### B — Two qualifying failures in the **same** user message (for max-escalation scenarios)

The handler can run **multiple tool rounds** in one user turn. You need **two** separate failures that both qualify, without ending the turn between them.

1. Set **`max_per_user_message`** to **1** (to test “no policy escalation” on tool failures, use **`enabled: false`** instead—not `max_per_user_message: 0`, which the app rejects at load).
2. Use a **staging** setup as in **A**, where the first tool call fails (e.g. node down).
3. **Prompt the model** so that after the first failed tool round it issues **another** tool call in the **same** turn (e.g. “run X on the node, then run Y” using tool names from your catalog). If the model stops after one tool, try a stronger model or a clearer instruction; **automation** covers the policy math—manual check is best-effort.
4. **Pass interpretation:** after the **first** qualifying failure you should see **one** escalation; after the **second** qualifying failure in that same message, logs should show **no further** index advance beyond the cap.

### C — Unknown tool / allowlist (non-qualifying)

- **Unknown tool:** Use a chat model with **native tool calling**; ask in a way that sometimes emits a **fake** tool name not in [tools.yaml](../../../config/tools.yaml). If the model always behaves, this scenario is **hard to force manually**—rely on unit tests and spot-check when you see a bad tool name in the wild.
- **Allowlist / cmdsafe:** Ask for an action that resolves to a command **not** on the node allowlist, or arguments that expand to **shell metacharacters** after substitution—should get validation / policy errors and **no** escalation for that reason alone.

### D — Hermes parse failure

Requires **text-based tools** (`supports_tools: false` and text-tool mode per your provider config). Provoke the model to output **almost** valid Hermes markup but **invalid** JSON or broken tags (model-specific). If it never fails, treat this scenario as **optional** and rely on automated tests.

### E — Logs and secrets

Use **`PA_LOG_LEVEL=info`** (or `debug` if you accept noise). Search log files for substrings of **known** secrets (copy a **prefix** of a **test** API key into grep—do **not** paste full production secrets into shell history). Cross-check [README § Log redaction](../../../README.md#log-redaction-secrets).

---

<a id="mt-esc-off"></a>

## Escalation disabled — no provider advance on qualifying failure

**Trace:** [AC-06.011](ep-acceptance-criteria.md#ac-06-011) · [REQ-06.014](ep-requirements.md#nfr--security-testability-observability)

**Prerequisites:**

- `tools.llm_escalation.enabled` **false** (or block omitted per project defaults).
- At least one tool that can fail in a **qualifying** way when escalation would be on (e.g. node / `run_on_node` error returning a typed `MayEscalate` path — in practice the handler still evaluates qualification, but must **not** advance provider).

**Steps:**

1. Deploy with **`tools.llm_escalation.enabled` false** (or omit the block). Optionally keep two `llm_providers` so the binary path matches production shape.
2. Provoke a **node/tool execution** failure as in **[How to execute § A](#how-to-execute-the-steps-operators)** (e.g. unreachable `host` for the tool’s node, then message that triggers that tool).
3. Inspect logs: you may see **`tool invocation`** errors, but **no** line **`llm tool escalation`** with **`action=escalate_policy`** for that tool failure.

**Pass:** User-visible outcome is deterministic; **no** escalation along the ordered provider list solely because of that tool failure.

---

<a id="mt-esc-on"></a>

## Escalation enabled — qualifying failure advances to next provider

**Trace:** [AC-06.005](ep-acceptance-criteria.md#ac-06-005) · [AC-06.006](ep-acceptance-criteria.md#ac-06-006) · [REQ-06.006](ep-requirements.md#escalation-policy-and-chain) · [REQ-06.007](ep-requirements.md#escalation-policy-and-chain)

**Prerequisites:**

- `tools.llm_escalation.enabled` **true**, `len(llm_providers) >= 2`, valid `baseline_index` and `max_per_user_message`.
- A way to provoke a **qualifying** error on the first provider’s turn (e.g. misconfigured API for provider 0 only, or a tool/node failure classified as `MayEscalate`).

**Steps:**

1. Enable escalation and two providers per **Prerequisites**. Use **[§ A](#how-to-execute-the-steps-operators)** so the **first** `Complete` returns a **tool call**, then execution **fails** (e.g. SSH down / bad host).
2. Send one Telegram (or test) message that causes the model to call that tool (clear instruction + tool name from your catalog).
3. Inspect logs: **`llm tool escalation`** with `failure_class=tool_execution`, indices or labels moving **from baseline to the next** `llm_providers` entry.
4. Confirm the user gets a **final** reply or clear error **without** a tight loop of repeated identical escalations.

**Pass:** Next `Complete` after the qualifying failure uses the **next** provider in configuration order (no skipping).

---

<a id="mt-max-esc"></a>

## Max escalations per user message

**Trace:** [AC-06.007](ep-acceptance-criteria.md#ac-06-007) · [REQ-06.008](ep-requirements.md#exhaustion-and-stop)

**Prerequisites:**

- Escalation enabled; set **`max_per_user_message`** to a small value (e.g. **1**).
- Scenario with **two** qualifying failures in the **same** user message (e.g. two tool rounds that both fail in a qualifying way).

**Steps:**

1. Set **`max_per_user_message`** to **1** (for “no advance” from **disabled** policy escalation, use **`enabled: false`**; `max_per_user_message: 0` with `enabled: true` is invalid—see [AC-06.007](ep-acceptance-criteria.md#ac-06-007) and unit tests for exhausted-cap behaviour).
2. Follow **[§ B](#how-to-execute-the-steps-operators)** to get **two** qualifying tool failures in **one** user turn (node broken + model issues two tool calls in sequence). If you only achieve **one** failure, you can still **partially** sign off: verify **one** escalation line, then rely on automated tests for the second failure.
3. Inspect logs: after the cap is reached, **no** further `to_index` advances for that message; remaining completes use the **last** allowed provider until the turn ends.

**Pass:** Escalation count for that user message does not exceed **`max_per_user_message`**; deterministic stop.

---

<a id="mt-baseline-reset"></a>

## Baseline reset on the next user message

**Trace:** [AC-06.008](ep-acceptance-criteria.md#ac-06-008) · [REQ-06.009](ep-requirements.md#rollback-at-end-of-turn)

**Prerequisites:**

- Escalation enabled; **baseline_index** not necessarily **0** (e.g. **1**) to make log inspection obvious.
- Two consecutive user messages in the same chat.

**Steps:**

1. Set **`baseline_index`** to **1** (or any non-zero index you use in prod) so logs are easy to read (`m1` vs `m0`).
2. On message **A**, use **[§ A](#how-to-execute-the-steps-operators)** so a tool fails and escalation moves to the **next** provider (e.g. index **2** if baseline was **1**).
3. Send message **B** (new user text).
4. In logs, find the **first** `llm call` (or equivalent) for message **B**: provider label/index should match **baseline** again (e.g. back to **`m1`** if `baseline_index` is **1**).

**Pass:** End-of-turn rollback: each new user message starts from the configured baseline for the first completion of that message.

---

<a id="mt-hermes"></a>

## Hermes (text-tool) parse failure and escalation

**Trace:** [AC-06.013](ep-acceptance-criteria.md#ac-06-013) · [REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs)

**Prerequisites:**

- Text-based tool path enabled (`supports_tools` false and text-tool mode per [EP-004](../EP-004/ep-scope.md) config).
- Escalation enabled with at least two providers.

**Steps:**

1. Configure text-tool / Hermes path per [EP-004](../EP-004/ep-scope.md) (provider **`supports_tools: false`**, text tools enabled in config).
2. Try prompts that elicit **malformed** `<tool_call>` blocks or invalid JSON inside them (**§ D**). This is **model-dependent**; if you cannot trigger a parse error, skip and rely on unit tests.
3. Inspect logs for **`failure_class=hermes_parse`** and optional escalation to the next provider.
4. With **`max_per_user_message`** exhausted, confirm the user still gets a **single** clear failure outcome and **no** infinite retry loop.

**Pass:** Behaviour matches AC-06.013; logs show classified Hermes failure when applicable.

---

<a id="mt-no-secrets"></a>

## No secrets in escalation and LLM logs

**Trace:** [AC-06.009](ep-acceptance-criteria.md#ac-06-009) · [REQ-06.010](ep-requirements.md#observability)–[REQ-06.012](ep-requirements.md#nfr--security-testability-observability)

**Prerequisites:**

- Known API keys or tokens in config (use **test** keys only).
- Escalation and/or tool failures exercised in the same session.

**Steps:**

1. Run **[§ A](#how-to-execute-the-steps-operators)** once with escalation **on** (and optionally once with escalation **off**) so log volume is realistic.
2. From a **test** API key or token, take a **short unique prefix** (e.g. first 8 chars) and **grep** stdout log file / journal / Docker logs / `llm_logs` (if [paths.llm_log_dir](../../../README.md) is set). Do **not** log the full secret in the ticket.
3. Repeat for Telegram token shape if applicable (see [README redaction table](../../../README.md#log-redaction-secrets)).

**Pass:** Matches are only inside redacted placeholders (e.g. `[REDACTED]`) or absent; no full secrets on `llm tool escalation` or `tool invocation` lines.

---

<a id="mt-non-qual"></a>

## Non-qualifying tool failures — no escalation

**Trace:** [AC-06.004](ep-acceptance-criteria.md#ac-06-004) · [REQ-06.005](ep-requirements.md#error-classification)

**Prerequisites:**

- Escalation enabled.

**Steps:**

1. **Unknown tool id:** See **[§ C](#how-to-execute-the-steps-operators)** (native tool calling + model-invented tool name). If you cannot trigger it, note “not observed” and rely on automated tests.
2. **Allowlist / cmdsafe:** Ask for a node action whose **expanded command** is **not** on the allowlist, or includes **`;` `|` `&`** etc. after substitution—should error **before** successful execution.
3. Inspect logs: **no** `llm tool escalation` **for that reason** (unknown tool / allowlist / cmdsafe are non-qualifying).

**Pass:** Provider index unchanged for those failure classes; user sees the usual validation / policy error text.

---

<a id="mt-invalid-config"></a>

## Invalid escalation config at startup

**Trace:** [AC-06.002](ep-acceptance-criteria.md#ac-06-002) · [REQ-06.002](ep-requirements.md#baseline-and-configuration)

**Steps:**

1. **Example A:** In JSON config set `"llm_escalation": { "enabled": true, "max_per_user_message": 2, "baseline_index": 0 }` under `tools` but leave **`llm_providers`** with **only one** entry — save and start `./pa` (or your Docker entrypoint).
2. **Example B:** With **two** providers, set `"baseline_index": 99` (out of range).
3. **Expected:** Process exits during config load with an error mentioning escalation / baseline / provider count (wording per [internal/config/load.go](../../../internal/config/load.go) messages).

**Pass:** Startup fails with a **clear** config error; service does not run with inconsistent escalation settings.

---

<a id="mt-operator"></a>

## Operator checklist after deploy

**Trace:** [AC-06.010](ep-acceptance-criteria.md#ac-06-010) · [REQ-06.013](ep-requirements.md#nfr--security-testability-observability) (complements automated tests)

**Steps:**

1. Confirm `tools.llm_escalation` matches intent (enabled/disabled, `baseline_index`, `max_per_user_message`).
2. Run one **happy-path** chat message (no tools).
3. Run one **tool** message (success path).
4. Optionally run **Escalation enabled — qualifying failure** and **Baseline reset** scenarios above.

**Pass:** Behaviour and logs are consistent with configuration; no unexpected provider switching when escalation is off.
