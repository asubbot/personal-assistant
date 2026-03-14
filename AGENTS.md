# Project Instructions

## Cooperation with the user
- The agent works **in cooperation with the user**. When multiple valid options exist (design, naming, artefact location, implementation approach, or interpretation of the request), present them clearly (e.g. A / B or 1A / 2B) with short pros/cons if helpful, and **ask the user to choose**. Do not decide autonomously. Proceed only after an explicit user choice.

## File changing
- Don't change source files without my allowing
- Don't commit without my allowing

## Architecture
- Use KISS and "fail fast" approach
- Source of truth for requirements and design: **ai-sdlc-artefacts** (e.g. epics/ep-104 for requirements, design, implementation plan). Process definition: **ai-sdlc/specification/** (pipeline, skills, templates).

## Language
- All code comments, UI/user-facing messages, and commit messages must be in English.


## Research / Docs-first
- When solving an issue, first search official documentation using the USER's keywords (preserve the user's wording).
- Prefer official docs over GitHub issues/blog posts; fall back only if official docs lack the answer.

## Workflow with subagents
- **Plan first:** For multi-step or non-trivial tasks, create a plan in planning mode (CreatePlan). The plan is the single source of truth for scope and order of work.
- **Verification per step:** Each plan step must include a verification block: how to confirm the step is done correctly (commands, tests, acceptance criteria). A step is not done until verification passes.
- **One step per subagent:** Execute each plan step by delegating to a subagent (mcp_task). Pass a self-contained task description: what to do, which files, and the verification criteria from the plan. Do not bundle multiple plan steps into one subagent run without my approval.
- **Review after each step:** After the subagent finishes, review the result: changes, adherence to project rules (e.g. KISS), test/lint outcome, and whether new dependencies or complexity are justified. Summarise in chat: agree or disagree and why.
- **Stop on failure or doubt:** If the subagent did not fulfil the step (verification failed, wrong approach, or review disagrees), stop and report in chat: what was done, what failed, and options (retry with clarified prompt, fix manually, skip, change plan). Do not continue or retry automatically without my decision.
- **Parallel steps:** If the plan has steps with no dependencies, you may run several subagents in parallel. Do so only when the plan allows it and when review per step stays clear.
- **Commits:** Do not commit subagent results until I approve. After approval, commit with a message that references the plan step; one commit per step unless I say otherwise.

## About this file
- I expect you to suggest improvements to this file when you see ways to make it clearer, more complete, or better aligned with how we work.
