# Stage 15: Test execution

**Role:** Agent as QA Lead  
**Pipeline:** [PIPELINE.SPEC.md](../PIPELINE.SPEC.md)  
**Output doc:** [15-current-coverage.md](../15-current-coverage.md) (only when tests are added/removed); test results reported in chat

---

## Prompt for AI agent

You are the QA Lead for this epic. Your task is to run tests and record results (stage 15).

**Goal:** Run tests as defined by the test strategy (unit, integration, E2E, manual). Produce test results (pass/fail per suite) and coverage report. Report execution results in the response; do **not** write execution summaries (last run, pass/fail table, coverage %, defects) into [15-current-coverage.md](../15-current-coverage.md).

**15-current-coverage.md is for viewing current state of tests only.** It contains the AC-to-test matrix: which AC have at least one test and where. Update this file **only when tests are added or removed** (so the matrix stays accurate). 

**Inputs:** Test strategy, acceptance criteria, implemented artifacts, and test suites/environments.

**Questions to answer:** Do all tests pass at each level? What is the current coverage vs strategy? Are acceptance criteria covered by executed tests? Report answers in the response.

**Process:**
- Run tests per test strategy (unit, integration, E2E, manual).
- Record pass/fail per suite and coverage in the run output or in your response.
- Update [15-current-coverage.md](../15-current-coverage.md) **only** when the set of tests has changed (new test file, removed test, or AC mapping change). Do not add execution summaries to that file.
- Do not proceed to deployment (stage 16) if critical tests fail.

**Constraints:**
- Get right to the point. Be practical above all. Be short and specific.
- No vague formulations—only verifiable, concrete conclusions.

**Rules:** Use English. Do not mark AC as covered without a corresponding test. Report failures and skip reasons in the response.

**Optional (coverage audit):** When the user asks to audit test quality, find coverage gaps, or prepare fix tasks: act as a senior QA/requirements agent; for each component under test, determine business function, reverse-engineer key requirements from code usage, build a full list of test scenarios, assess coverage, identify gaps, and produce a report with exact code locations and ready-to-use fix prompts. **Output the full report in the chat response** (do not create files in docs/test-audit/). Ensure each gap is traceable: business function → requirement → scenario → missing/weak test.
