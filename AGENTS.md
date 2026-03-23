# Project Instructions

## Cooperation with the user
- The agent works **in cooperation with the user**. When multiple valid options exist (design, naming, artefact location, implementation approach, or interpretation of the request), present them clearly (e.g. A / B or 1A / 2B) with short pros/cons if helpful, and **ask the user to choose**. Do not decide autonomously. Proceed only after an explicit user choice.
- **Chat language:** Reply in the language the user uses unless they ask otherwise. (Code, commits, and in-repo technical docs remain English per **Language** below.)

## File changing
- **Product code and behaviour** (`cmd/`, `internal/`, `tests/`, build files, etc.): do not change without my explicit allowance.
- **Project artefacts** under `ai-sdlc-artefacts/`: follow the relevant skill (e.g. draft in chat until I approve **save** where the skill says so).
- **Commits:** do not commit without my explicit allowance.
- **Secrets:** never commit real tokens, passwords, or private keys; do not paste them into the repo or examples. Use placeholders and existing secret-file patterns from README/config docs.

## Architecture
- Use **KISS** and **fail fast**.
- Source of truth for requirements and design: **`ai-sdlc-artefacts`** (active epic folders under `epics/EP-XXX/`: scope, requirements, design, implementation plan, acceptance criteria). Process definition: **`ai-sdlc/specification/`** ([pipeline.spec.md](ai-sdlc/specification/pipeline.spec.md), [skills](ai-sdlc/specification/skills/README.md), templates).
- **Heavy or SDLC tasks:** **read and follow** the matching skill under **`ai-sdlc/specification/skills/`** first. The skill is the workflow (outputs, verification, when to write files). Do not invent a parallel process.

## Heavy tasks, skills, and optional plans/subagents
- **Primary:** non-trivial work (epics, audits, requirements, code review, consistency checks, etc.) is driven by the **skills**—open the right `*.skill.md` and execute it.
- **Optional:** if you also use Cursor **plans** or **subagents**, align with the same rules: **one step at a time** with clear verification, **review** before moving on, **stop** on failure or doubt and report options (retry, fix manually, skip, change plan)—no automatic retries or bundling multiple steps without my approval. **Parallel** delegated work only when steps are independent and you can still review each outcome clearly.
- **Commits:** do not commit delegated or multi-step work until I approve; commit messages in English and, when helpful, reference the skill or plan step.

## Language
- All code comments, UI/user-facing messages in the product, and commit messages must be in English.

## Research / Docs-first
- **Behaviour of this repo:** prefer **`ai-sdlc-artefacts`** and the current codebase over external sources.
- **Third-party libraries, APIs, and platforms:** search **official documentation** using the USER's keywords (preserve the user's wording). Prefer official docs over GitHub issues or blog posts; fall back only if official docs lack the answer.

## Optional tooling (Cursor): Sourcerer MCP
- **What it is:** an optional MCP server for **semantic search** over the workspace (code and Markdown chunks), to navigate faster and reduce reading whole files. It is **not** part of the SDLC pipeline.
- **Pipeline rule:** Stages and skills use files under **`ai-sdlc-artefacts/`** and **`ai-sdlc/specification/`** as the source of truth in git. Do not treat Sourcerer as a substitute for reading skills or approved artefacts.
- **When it helps:** large repos, exploratory “where is X?” questions across code and docs. For small epic files, normal **read** / **grep** is often enough.
- **Operator note:** typical setup needs an **embeddings API** (e.g. OpenAI), local index data (often under `.sourcerer/`—**gitignore** it), and respects **`.gitignore`**. Follow the upstream package docs for install and env.

## Quality checks
- After non-trivial code changes, run **`make check`** (fmt, vet, lint, tests with coverage) when you are allowed to execute commands, and fix failures before handing off—unless I say otherwise.

## Security (basics)
- Treat config paths, SSH, Telegram, and LLM logs as sensitive contexts; follow requirements in epics (e.g. redaction, allowlists) when touching related code.
- Do not weaken security or reliability for convenience without an explicit trade-off discussion with me.

## About this file
- Suggest improvements to this file when you see ways to make it clearer, more complete, or better aligned with how we work—especially after we change process or tooling.
