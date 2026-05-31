# EP-017 — Acceptance criteria

**Introduction:** Testable acceptance criteria for **EP-017** (intent classifier for prompt optimization: heuristic cascade, complexity tiers, tiered prompt assembly). Model classification stage criteria are **Obsolete** after [EP-036](../EP-036/ep-scope.md). Each AC traces to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-17.001](#ac-17-001) | [REQ-17.001](ep-requirements.md#req-17-001) | Two tiers (simple, full) are defined and selectable |
| [AC-17.002](#ac-17-002) | [REQ-17.002](ep-requirements.md#req-17-002) | Simple tier prompt excludes tools, RAG, dynamic tail, skills |
| [AC-17.003](#ac-17-003) | [REQ-17.003](ep-requirements.md#req-17-003) | Full tier prompt matches pre-epic assembly |
| [AC-17.004](#ac-17-004) | [REQ-17.004](ep-requirements.md#req-17-004), [REQ-17.005](ep-requirements.md#req-17-005) | Heuristic classifies greeting as simple |
| [AC-17.005](#ac-17-005) | [REQ-17.004](ep-requirements.md#req-17-004), [REQ-17.005](ep-requirements.md#req-17-005) | Heuristic classifies tool-bearing message as full |
| [AC-17.006](#ac-17-006) | [REQ-17.005](ep-requirements.md#req-17-005) | Heuristic returns ambiguous for borderline message |
| [AC-17.007](#ac-17-007) | [REQ-17.006](ep-requirements.md#req-17-006) | Heuristic performs no I/O or network calls |
| [AC-17.008](#ac-17-008) | [REQ-17.007](ep-requirements.md#req-17-007), [REQ-17.009](ep-requirements.md#req-17-009) | **Obsolete:** Model stage removed by [EP-036](../EP-036/ep-scope.md) |
| [AC-17.009](#ac-17-009) | [REQ-17.008](ep-requirements.md#req-17-008) | **Obsolete:** Model stage removed by [EP-036](../EP-036/ep-scope.md) |
| [AC-17.010](#ac-17-010) | [REQ-17.010](ep-requirements.md#req-17-010) | Cascade: heuristic → default `full` (EP-036; no model stage) |
| [AC-17.011](#ac-17-011) | [REQ-17.011](ep-requirements.md#req-17-011) | **Obsolete:** Model stage removed by [EP-036](../EP-036/ep-scope.md) |
| [AC-17.012](#ac-17-012) | [REQ-17.012](ep-requirements.md#req-17-012), [REQ-17.013](ep-requirements.md#req-17-013), [REQ-17.014](ep-requirements.md#req-17-014) | Simple tier prompt token count at least 50 % lower than full |
| [AC-17.013](#ac-17-013) | [REQ-17.015](ep-requirements.md#req-17-015) | Full tier produces identical prompt to pre-epic baseline |
| [AC-17.014](#ac-17-014) | [REQ-17.016](ep-requirements.md#req-17-016) | Classifier disabled via config restores pre-epic behaviour |
| [AC-17.015](#ac-17-015) | [REQ-17.016](ep-requirements.md#req-17-016) | Heuristic patterns configurable without code changes |
| [AC-17.016](#ac-17-016) | [REQ-17.017](ep-requirements.md#req-17-017) | INFO log contains tier, deciding stage, message length |
| [AC-17.017](#ac-17-017) | [REQ-17.018](ep-requirements.md#req-17-018) | **Obsolete:** Model stage removed by [EP-036](../EP-036/ep-scope.md) |
| [AC-17.018](#ac-17-018) | [REQ-17.019](ep-requirements.md#req-17-019), [REQ-17.020](ep-requirements.md#req-17-020) | make check passes; AC coverage verified |

---

## Acceptance criteria

<a id="ac-17-001"></a>**AC-17.001** (Trace: REQ-17.001)
Given the intent classifier is enabled in configuration
When the system starts
Then at least two complexity tiers (`simple` and `full`) SHALL be available for assignment by the classifier.

<a id="ac-17-002"></a>**AC-17.002** (Trace: REQ-17.002)
Given a user turn assigned to the `simple` tier
When HandleMessage constructs the main LLM request
Then the request SHALL contain no `tools` array, no RAG chunks, no Hermes tool-format block, and no runtime skill text in the system message.

<a id="ac-17-003"></a>**AC-17.003** (Trace: REQ-17.003)
Given a user turn assigned to the `full` tier
When HandleMessage constructs the main LLM request
Then the request SHALL include tools, RAG chunks, dynamic tail, and session history identically to pre-epic behaviour.

<a id="ac-17-004"></a>**AC-17.004** (Trace: REQ-17.004, REQ-17.005)
Given the heuristic stage is enabled with default patterns
When the user sends a greeting message (e.g. "привет", "hello", "ты здесь?")
Then the heuristic SHALL return the `simple` tier.

<a id="ac-17-005"></a>**AC-17.005** (Trace: REQ-17.004, REQ-17.005)
Given the heuristic stage is enabled with default patterns
When the user sends a message containing a tool-related intent (e.g. "напомни что я говорил вчера", "запусти задачу X")
Then the heuristic SHALL return the `full` tier.

<a id="ac-17-006"></a>**AC-17.006** (Trace: REQ-17.005)
Given the heuristic stage is enabled
When the user sends a borderline message that matches no confident pattern (e.g. "погода")
Then the heuristic SHALL return `ambiguous`.

<a id="ac-17-007"></a>**AC-17.007** (Trace: REQ-17.006)
Given the heuristic stage implementation
When classification runs
Then the heuristic SHALL complete without any network, LLM, or filesystem I/O calls (verifiable by test with no mocks for external services).

<a id="ac-17-008"></a>**AC-17.008** (Trace: REQ-17.007, REQ-17.009) **Obsolete:** Model classification stage removed by [EP-036](../EP-036/ep-scope.md); retained for historical REQ traceability.
Given the heuristic returned `ambiguous` and the model stage is enabled
When the model stage runs
Then it SHALL send a request to the classification provider containing only the user message and tier descriptions, and SHALL return a valid tier.

<a id="ac-17-009"></a>**AC-17.009** (Trace: REQ-17.008) **Obsolete:** Model classification stage removed by [EP-036](../EP-036/ep-scope.md); retained for historical REQ traceability.
Given configuration with a classification provider endpoint and model distinct from the main provider
When the model stage is invoked
Then the classification request SHALL be sent to the configured classification provider, not the main provider.

<a id="ac-17-010"></a>**AC-17.010** (Trace: REQ-17.010) **Amended (EP-036):** cascade is heuristic → default `full` when ambiguous; model stage removed.
Given the heuristic returns `ambiguous`
When the cascade resolves the tier
Then the assigned tier SHALL be `full`.

<a id="ac-17-011"></a>**AC-17.011** (Trace: REQ-17.011) **Obsolete:** Model classification stage removed by [EP-036](../EP-036/ep-scope.md); retained for historical REQ traceability.
Given the heuristic returns `ambiguous` and the model stage returns an error
When the cascade resolves the tier
Then the assigned tier SHALL be `full` and a WARN-level log entry SHALL be recorded with the error details.

<a id="ac-17-012"></a>**AC-17.012** (Trace: REQ-17.012, REQ-17.013, REQ-17.014)
Given a greeting message classified as `simple`
When the main LLM request is built
Then the prompt token count (system + user messages, no tools) SHALL be at least 50 % lower than the same message processed with the `full` tier.

<a id="ac-17-013"></a>**AC-17.013** (Trace: REQ-17.015)
Given a message classified as `full`
When the main LLM request is built
Then the assembled messages array and tools array SHALL be byte-identical to what the pre-epic code would produce for the same input and session state.

<a id="ac-17-014"></a>**AC-17.014** (Trace: REQ-17.016)
Given the intent classifier `enabled` flag is set to `false` in config
When a user message arrives
Then HandleMessage SHALL skip classification and follow the pre-epic full prompt path entirely.

<a id="ac-17-015"></a>**AC-17.015** (Trace: REQ-17.016)
Given updated heuristic pattern definitions in config (new regex added)
When the system reloads configuration
Then the heuristic stage SHALL use the updated patterns without a code deploy.

<a id="ac-17-016"></a>**AC-17.016** (Trace: REQ-17.017) **Amended (EP-036):** deciding stage values are `heuristic` or `default` (no model stage).
Given a user turn is classified
When the classification completes
Then an INFO-level log entry SHALL contain fields: assigned tier, deciding stage ("heuristic" or "default"), and message length in characters.

<a id="ac-17-017"></a>**AC-17.017** (Trace: REQ-17.018) **Obsolete:** Model classification stage removed by [EP-036](../EP-036/ep-scope.md); retained for historical REQ traceability.
Given the model stage is invoked for classification
When the turn completes and usage is reported
Then model-stage prompt and completion tokens SHALL appear in a separate log entry, and the user-facing Telegram footer SHALL report only the main-model tokens.

<a id="ac-17-018"></a>**AC-17.018** (Trace: REQ-17.019, REQ-17.020)
Given the delivered EP-017 branch
When `make check` and `./bin/validate EP-017` run
Then both SHALL exit with zero status.
