# EP-018 — Acceptance criteria

**Introduction:** Testable acceptance criteria for **EP-018 Tiered Prompt Cost Reduction** (`full_lite` tier, dynamic tool selection for `full_lite` and optional `full`, classifier three-way assignment). Static prompt density is **out of scope** (see [ep-scope.md](ep-scope.md)). Each AC traces to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-18.001](#ac-18-001) | [REQ-18.001](ep-requirements.md#req-18-001) | Configuration documentation lists the three-tier prompt matrix |
| [AC-18.002](#ac-18-002) | [REQ-18.002](ep-requirements.md#req-18-002) | `simple` tier omits tools, RAG, Hermes, runtime skills |
| [AC-18.003](#ac-18-003) | [REQ-18.003](ep-requirements.md#req-18-003) | `full` tier with `full`-dynamic off matches EP-017 `full` assembly |
| [AC-18.004](#ac-18-004) | [REQ-18.004](ep-requirements.md#req-18-004) | **Obsolete:** `full_lite` tier removed by [EP-036](../EP-036/ep-scope.md). |
| [AC-18.005](#ac-18-005) | [REQ-18.005](ep-requirements.md#req-18-005) | `full_lite` includes session exchanges like `full` when session memory on |
| [AC-18.006](#ac-18-006) | [REQ-18.006](ep-requirements.md#req-18-006) | `full_lite` omits runtime skill playbook from dynamic tail |
| [AC-18.007](#ac-18-007) | [REQ-18.007](ep-requirements.md#req-18-007) | `full_lite` with tools includes Hermes instructions |
| [AC-18.008](#ac-18-008) | [REQ-18.008](ep-requirements.md#req-18-008) | `full_lite` with zero tools omits Hermes instructions |
| [AC-18.009](#ac-18-009) | [REQ-18.009](ep-requirements.md#req-18-009) | **Obsolete:** Three-tier classifier removed by [EP-036](../EP-036/ep-scope.md). |
| [AC-18.010](#ac-18-010) | [REQ-18.010](ep-requirements.md#req-18-010) | **Obsolete:** Model classification stage removed by [EP-036](../EP-036/ep-scope.md). |
| [AC-18.011](#ac-18-011) | [REQ-18.011](ep-requirements.md#req-18-011) | **Obsolete:** Model failure path removed by [EP-036](../EP-036/ep-scope.md). |
| [AC-18.012](#ac-18-012) | [REQ-18.012](ep-requirements.md#req-18-012) | `always_include` tools survive merge before cap |
| [AC-18.013](#ac-18-013) | [REQ-18.013](ep-requirements.md#req-18-013) | Tool count never exceeds configured maximum when dynamic applies |
| [AC-18.014](#ac-18-014) | [REQ-18.014](ep-requirements.md#req-18-014) | Ranked order follows vector pre-selection when enabled |
| [AC-18.015](#ac-18-015) | [REQ-18.015](ep-requirements.md#req-18-015) | `full` + dynamic off does not apply epic max-tool cap |
| [AC-18.016](#ac-18-016) | [REQ-18.016](ep-requirements.md#req-18-016) | `full_lite` + pre-selection off uses fallback cap list |
| [AC-18.017](#ac-18-017) | [REQ-18.017](ep-requirements.md#req-18-017) | `full_lite` + tools on uses dynamic selection path |
| [AC-18.018](#ac-18-018) | [REQ-18.018](ep-requirements.md#req-18-018) | INFO log includes tier, tool count, dynamic flag, stage |
| [AC-18.019](#ac-18-019) | [REQ-18.019](ep-requirements.md#req-18-019) | Invalid EP-018 configuration rejected at load |
| [AC-18.020](#ac-18-020) | [REQ-18.004](ep-requirements.md#req-18-004), [REQ-18.006](ep-requirements.md#req-18-006), [REQ-18.013](ep-requirements.md#req-18-013) | **Obsolete:** `full_lite` vs `full` token delta removed by [EP-036](../EP-036/ep-scope.md). |
| [AC-18.021](#ac-18-021) | [REQ-18.020](ep-requirements.md#req-18-020), [REQ-18.021](ep-requirements.md#req-18-021) | `make check` passes; `./bin/validate EP-018` reports full AC coverage |

---

## Acceptance criteria

<a id="ac-18-001"></a>**AC-18.001** (Trace: REQ-18.001)  
Given the product configuration documentation for EP-018 is updated  
When a reader inspects the tier matrix section  
Then the documentation SHALL list `simple`, `full_lite`, and `full` and SHALL state which prompt components each tier includes.

<a id="ac-18-002"></a>**AC-18.002** (Trace: REQ-18.002)  
Given a user turn classified as `simple`  
When HandleMessage builds the main LLM request  
Then the request SHALL contain no tool definitions, no RAG chunk text, no Hermes tool-format block, and no runtime skill playbook text in the system message.

<a id="ac-18-003"></a>**AC-18.003** (Trace: REQ-18.003)  
Given dynamic tool selection for the `full` tier is disabled in configuration  
When a user turn is classified as `full`  
Then the main LLM request SHALL match the EP-017 `full` tier baseline for the same inputs (verified by automated structural or byte-identical comparison defined in the test).

<a id="ac-18-004"></a>**AC-18.004** (Trace: REQ-18.004) **Obsolete:** `full_lite` tier removed by EP-036; retained for historical REQ traceability only.  
Given a user turn classified as `full_lite`  
When HandleMessage runs  
Then the core SHALL not call semantic RAG retrieval for that turn and SHALL not inject retrieved memory chunk strings into the system message.

<a id="ac-18-005"></a>**AC-18.005** (Trace: REQ-18.005)  
Given session memory is enabled and a user turn is classified as `full_lite`  
When HandleMessage builds the message list  
Then session store exchanges SHALL appear in the same order as for the `full` tier for the same session key.

<a id="ac-18-006"></a>**AC-18.006** (Trace: REQ-18.006)  
Given a user turn classified as `full_lite`  
When the dynamic tail is assembled  
Then runtime skill playbook text SHALL be absent from the system message tail.

<a id="ac-18-007"></a>**AC-18.007** (Trace: REQ-18.007)  
Given a user turn classified as `full_lite` and the main completion includes at least one tool  
When the system message is built  
Then Hermes tool-format instructions SHALL be present in the system message.

<a id="ac-18-008"></a>**AC-18.008** (Trace: REQ-18.008)  
Given a user turn classified as `full_lite` and the main completion includes zero tools  
When the system message is built  
Then Hermes tool-format instructions SHALL be absent from the system message.

<a id="ac-18-009"></a>**AC-18.009** (Trace: REQ-18.009) **Obsolete:** Three-tier classifier removed by EP-036.  
Given the intent classifier is enabled  
When any user message is processed in a turn  
Then exactly one tier value in the set {`simple`, `full_lite`, `full`} SHALL be assigned before main-model prompt assembly.

<a id="ac-18-010"></a>**AC-18.010** (Trace: REQ-18.010) **Obsolete:** Model classification stage removed by EP-036.  
Given the heuristic returned `ambiguous` and the model stage is enabled  
When the classification request is sent  
Then the classification prompt body SHALL contain only the user message and the three tier labels `simple`, `full_lite`, and `full` with brief descriptions.

<a id="ac-18-011"></a>**AC-18.011** (Trace: REQ-18.011) **Obsolete:** Model failure path removed by EP-036.  
Given the model stage returns an error or unparseable output  
When the cascade completes  
Then the assigned tier SHALL be `full` and a WARN log entry SHALL include error details.

<a id="ac-18-012"></a>**AC-18.012** (Trace: REQ-18.012)  
Given `always_include` lists tool identifiers that exist in the catalog  
When dynamic tool selection runs for an applicable turn  
Then every configured `always_include` identifier SHALL appear in the merged set before the cap is applied.

<a id="ac-18-013"></a>**AC-18.013** (Trace: REQ-18.013)  
Given dynamic tool selection applies to a turn and `max_tools_for_llm_request` is set to N  
When the main LLM request is built  
Then the number of tools attached SHALL be less than or equal to N.

<a id="ac-18-014"></a>**AC-18.014** (Trace: REQ-18.014)  
Given tool vector pre-selection is enabled and dynamic selection applies  
When tools are ranked for the user message  
Then the relative order of candidates before capping SHALL match the existing pre-selection ranking output for that message.

<a id="ac-18-015"></a>**AC-18.015** (Trace: REQ-18.015)  
Given dynamic tool selection for the `full` tier is disabled  
When a `full` tier turn is assembled  
Then the tool assembly SHALL not apply the EP-018 maximum-tool cap (behaviour matches EP-017 `full`).

<a id="ac-18-016"></a>**AC-18.016** (Trace: REQ-18.016)  
Given `full_lite` and tool vector pre-selection is disabled  
When dynamic tool selection builds the candidate list  
Then the candidate identifiers SHALL be produced using the same fallback cap list rules as the current tool pre-selection path.

<a id="ac-18-017"></a>**AC-18.017** (Trace: REQ-18.017)  
Given `full_lite` and conversation tools are enabled  
When HandleMessage prepares tools for the main LLM request  
Then the tool identifier set SHALL be produced through the dynamic tool selection path from this epic.

<a id="ac-18-018"></a>**AC-18.018** (Trace: REQ-18.018)  
Given a completed main-model prompt assembly for a turn  
When logs are inspected at INFO level  
Then the log entry SHALL contain the assigned tier, tool count, whether dynamic tool selection ran, and the classifier stage name.

<a id="ac-18-019"></a>**AC-18.019** (Trace: REQ-18.019)  
Given a configuration file with an invalid EP-018 field value  
When the application loads configuration  
Then loading SHALL fail with an error message that identifies the invalid field or combination.

<a id="ac-18-020"></a>**AC-18.020** (Trace: REQ-18.004, REQ-18.006, REQ-18.013) **Obsolete:** `full_lite` tier removed by EP-036.  
Given a fixed fixture user message and session state for which `full` would include RAG or skills tail content  
When the same fixture is classified as `full_lite` with dynamic selection enabled and `always_include` only  
Then the main-model prompt rune count for `full_lite` SHALL be at least 15 percent lower than for `full` on the same fixture (threshold implemented as a named constant in the test).

<a id="ac-18-021"></a>**AC-18.021** (Trace: REQ-18.020, REQ-18.021)  
Given the delivery branch for EP-018  
When `make check` and `./bin/validate EP-018` are run from the repository root  
Then both commands SHALL exit zero and validation SHALL report every AC mapped to at least one automated test or an explicit manual scenario.
