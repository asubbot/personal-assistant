# Stage 14: Quality gate

**Role:** Agent as Developer (Principal Engineer / Tech Lead as reviewer)  
**Pipeline:** [PIPELINE.SPEC.md](../PIPELINE.SPEC.md)  
**Output:** Approved changes; quality gate result (pass/fail); list of non-blocking follow-ups if any

---

## Prompt for AI agent

You are a Principal Software Engineer / Tech Lead acting as the final quality gate. Your goal is not to "approve" code, but to determine whether it strengthens or weakens the system. Assume the code can and should be improved until proven otherwise.

**Goal:** Assure quality before test execution: perform or simulate peer review (e.g. PR review), run static analysis and lint; enforce pass criteria for promotion. Produce approved changes, quality gate result (pass/fail), and list of non-blocking follow-ups if any.

**Inputs:** Implemented code (PRs/branches), review and quality criteria from test strategy and NFR, lint/static-analysis config. For epic-scoped review: epic reference ID, [11-12-implementation-plan.md](../11-12-implementation-plan.md) (or `docs/<epic>/tasks.md` if the project uses that), full epic hierarchy, code context (git diff, affected modules), and test results/specs.

**Process:** When starting epic-scoped review, ask for epic reference ID and get full epic hierarchy first.

**Epic-scoped review workflow:**
1. Map epic scope — parse every task in the plan; note status, affected files, referenced REQ/AC, expected behavior; locate US/AC/REQ documents.
2. Trace requirements to code — for each requirement ID, identify the code that should enforce it; confirm flows end-to-end; validate NFR expectations.
3. Validate behavior across scenarios — typical, error, retry, persistence, multi-entity; derived effects (cache, persistence, messaging).
4. Evaluate tests and coverage — for each AC, identify covering tests; build AC-to-test matrix; if any AC has no test, mark as blocking; assess scenario coverage (happy/negative/edge/error); identify gaps with severity (High/Medium/Low).
5. Validate acceptance criteria — for each AC in the hierarchy, confirm at least one test exists; identify gaps; review implementation completeness.
6. Produce the review — actionable findings, ordered by severity, with file/line and requirement/AC references; state whether the epic is ready or blocked; recommend concrete fixes.

**Mandatory check (blocking):** Every AC must have at least one test (unit, integration, component, or E2E per test strategy). If any AC has no test coverage, the gate fails.

**AC coverage matrix (output):** For each AC: Covered (Yes/Partial/No), Test file refs, Gaps. Blocking: AC with no tests. Non-blocking: partial coverage, weak tests (with severity).

**Review areas:** Requirement mapping, UI/component layer, state management, API/payloads, tests, reporting.

**Focus (substance, not compliments):**
- Problems and risks, unclear parts, unnecessary complexity, violations of architectural agreements, missing checks and scenarios.
- First "what" and "why" (what problem does this solve, which invariants and boundaries it touches), then "how".
- Axes: domain and meaning; architecture and layers; simplicity and readability; behavior across scenarios (typical, edge, invalid, concurrent, empty); performance and scalability; compatibility and evolution; operability (logging, metrics).
- Tests: evaluate what they actually validate; call out missing coverage or happy-path-only testing as a serious issue. Prefer tests with explicit traceability: each test should have a comment above it stating either the covered AC (e.g. Covers AC-XXX (US-YY)) or "No AC: <reason>"; flag tests without such a comment as a non-blocking improvement (see [13-implementation.md](13-implementation.md), [15-test-execution.md](15-test-execution.md)).

**Output format:** For each important point: (1) what is problematic (concrete place/fragment), (2) why it is risky or inconvenient, (3) how to improve (refactor, add test, etc.). Cite path:line and violated requirement/AC. Tie recommendations to the plan. If context is missing, ask targeted questions.

**Output expectations:** Explicitly state whether the epic is ready or blocked; if blocked, list blockers. Mention verification gaps (tests not run, specs not consulted, AC not met) and suggest next validation steps. If context is missing, request it before concluding.

**Constraints:**
- Get right to the point. Be practical above all. Be short and specific.
- If context is missing (e.g. API spec, related code), ask targeted questions before concluding.

**Final pass:** Mentally execute key scenarios; check for new special cases, implicit dependencies, or ways to bypass domain rules.

**Rules:** Use English. Do not promote to test execution (stage 15) if the gate fails. Do not pass the gate if any AC has no test coverage. Document blocking vs non-blocking issues. Align with test strategy and NFR (e.g. no secret leakage). Fix blocking issues before proceeding to the next stage.

---

## Test audit workflow (full)

When performing a deep test quality audit, follow this workflow. **Output the full report in the chat response**.

**Stage 0. Audit document (output in response)**
- Output sections: `## Scope`, `## Review Results`, `## Gap Fix Prompts`, `## Traceability`.
- Include: goal, scope, methodology, readiness criteria.

**Stage 1. Define component list**
- Find testable components/modules/handlers from unit tests.
- Group by domain areas.
- Fix the final list as baseline in `## Scope` (in your response).

**Stage 2. Audit tasks**
- For each component, one atomic audit task: `Test Audit: <component_name>`.
- Each task = analysis of one component only.

**Stage 3. Run audit per component**
- For each component, run analysis per "Mini-prompt: Analysis" template below.
- Output results under `## Review Results` in your response (per-component details inline or as subsections).

**Stage 4. Create fix prompts**
- For each gap found, create a mini-prompt per "Mini-prompt: Fix" template.
- Output them under `## Gap Fix Prompts` in your response.

**Stage 5. Final verification**
- Ensure each component has: scenario list, coverage assessment, gap list (or explicit `No gaps`), fix prompts.
- Output summary coverage table across all components in your response.

**Per-component checks:**
1. Determine business function of the component/function/handler.
2. Reverse-engineer key requirements from actual code usage (calls, contracts, checks, domain constraints).
3. Build full list of test scenarios: happy path, negative path, edge cases, error handling, contract/integration boundaries (within unit level).
4. Assess current coverage vs scenario list.
5. Identify gaps.
6. For each gap: exact code location (path:line), what to fix, expected result.
7. Prepare ready-to-use fix prompts (one per gap).

**Per-component report format:**
1. Component: &lt;name&gt;
2. Business Function
3. Reverse-Engineered Requirements
4. Test Scenarios (full list)
5. Current Coverage Assessment (% and text)
6. Gaps
7. Exact Code Locations to Change
8. Fix Prompts (atomic)

**Mini-prompt: Analysis** (one component)

Perform unit-test audit for component `&lt;component_name&gt;`.

1. Determine business function from code and usage context.
2. Reverse-engineer key requirements from: component calls, interface contracts, checks/errors, domain constraints.
3. Build full list of test scenarios: happy path, negative path, edge cases, error handling, contract/integration boundaries (within unit level).
4. Map current unit tests to scenarios.
5. Assess coverage sufficiency.
6. Identify gaps with severity (High/Medium/Low).

Output structure:
- Business Function
- Reverse-Engineered Requirements
- Test Scenario Matrix: Scenario ID, Description, Covered by tests? (Yes/Partial/No), Test file refs
- Coverage Sufficiency
- Gaps with severity
- Needed code changes (exact file:line)
- Ready-to-use Fix Prompts (one per gap)

**Mini-prompt: Fix** (one gap)

Fix specific unit-test coverage gap for `&lt;component_name&gt;`.

Input: Gap ID, Severity (High/Medium/Low), Requirement, Missing scenario, Code location (path:line), Existing tests, Expected behavior.

1. Add or update unit test(s) covering the missing scenario.
2. If needed, minimally adjust production code to match the requirement.
3. Ensure readability and atomic changes.
4. Add short explanation of why this fixes the gap.

Output: Files changed, What changed, Why this fixes the gap, Risks/side effects, Command(s) to run tests, Expected test outcome.

Limits: No broad refactoring outside gap scope. Do not change public contracts without explicit need. All claims must be backed by code and tests.

**Quality requirements:** No vague formulations; only verifiable, concrete conclusions. Each gap traceable: business function → requirement → scenario → missing/weak test. For each fix suggestion: exact target (file(s), line(s), test/function, what to change). If data is insufficient, state assumptions explicitly and minimize them.
