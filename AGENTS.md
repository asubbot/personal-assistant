# Project Instructions

Instructions for AI agents working in a **git-backed software repository**.

## How to read this file
- **From Cooperation through Security (basics):** baseline agent and workspace habits; broadly reusable in similar repos.
- **`## This repository (PersonalAssistant)`:** paths, tooling, and where to open SDLC files for **this** repository only.
- **Normative pipeline** (stages, delegation, artefact rules, agent execution expectations): **[ai-sdlc/specification/pipeline.spec.md](ai-sdlc/specification/pipeline.spec.md)** and the stage skills it maps to. **[ai-sdlc/README.md](ai-sdlc/README.md)** is the **directory index** for `ai-sdlc/` (what each path is for).

## Cooperation with the user
- Work **with** the user: when several valid options exist (design, naming, artefact placement, approach, or interpretation), list them (e.g. A / B) with short pros/cons if useful and **ask for a choice**. Do not decide alone; wait for an explicit choice.
- **Chat language:** match the user’s language unless they ask otherwise. (Code, commits, and in-repo technical docs stay English per **Language** below.)

## Principles
- **KISS** — prefer the smallest change that solves the problem; avoid unnecessary abstraction and scope creep.
- **Fail fast** — detect invalid state and errors early; do not swallow failures without a clear, documented reason.
- **Explicit JSON configuration** — product **`config.json`** must list **every** documented top-level key exactly once. The allowed set is enforced at load (`internal/config`, `validateConfigRootObjectKeys` / `ConfigRootJSONKeys`). Optional product blocks are **disabled with JSON `null`**, not by omitting the key. Unknown top-level keys are rejected. Missing keys, invalid values, or structural drift must fail **config load** so the process does not start with an implicit or partial configuration.
- **nolint:gocyclo** - DO NOT use nolint:gocyclo

## Language
- All code comments, UI/user-facing messages in the product, and commit messages must be in English.

## Research / Docs-first
- **Third-party libraries, APIs, and platforms:** use **official documentation**, keeping the **user’s keywords**. Prefer official docs over issues or blogs; fall back only if official docs are insufficient.

## File changing (general)
- **Product source and build configuration:** do not change without the repository owner’s **explicit allowance**, except where they have already approved a bounded change (e.g. a task from an agreed implementation plan).
- **Delivery-process artefacts** (requirements, design, plans, reviews, etc., when this repo defines them): write or update them **only** through the process and skills the owner points you to for **this** repository (see **This repository** below).
- **Commits:** do not commit without the owner’s explicit allowance—including after delegation or multi-step work. Commit messages in English; when helpful, reference the skill or plan step.
- **Merge ≠ push:** merging branches (e.g. `git merge`, or finishing a merge locally) only updates **local** history unless a **push** follows. **Do not push** to a remote (`git push`, PR “push” actions, etc.) unless the user has **explicitly asked** for that remote update in the current request or otherwise clearly authorized it for this step. A request to “merge” or “commit and merge” alone is **not** permission to push.
- **Merge** implement as real merge, it is NOT **fast-forward**
- **Secrets:** never commit real tokens, passwords, or private keys; do not paste them into the repo or examples. Use placeholders and patterns from the repo’s README and configuration documentation.

## Security (basics)
- Do not weaken security or reliability for convenience without an explicit trade-off discussion with the owner.

## This repository (PersonalAssistant)

- **Product code layout:** treat **`cmd/`**, **`internal/`**, **`tests/`**, and project **build files** (e.g. `Makefile`, Go module files) as product source unless the owner narrows scope further.
- **Agentic SDLC:** definitions live under **`ai-sdlc/`**. Use **[ai-sdlc/README.md](ai-sdlc/README.md)** as the **directory index** (what each path is for). The **step-by-step pipeline** (stages, delegation, artefact rules, AC validation commands) is only in **[ai-sdlc/specification/pipeline.spec.md](ai-sdlc/specification/pipeline.spec.md)** and the **`*.skill.md`** files it maps to.
- **Pipeline outputs:** documents produced by the SDLC (scope, strategy, epic files, etc.) go under **`ai-sdlc-artefacts/`** at the repo root, not inside `ai-sdlc/`.
- **Checks after substantive code edits:** when you may run commands, run **`make check`** after non-trivial changes unless the owner says otherwise.
- **Optional editor tooling (Cursor): Sourcerer MCP** — optional semantic search over the workspace; do not use it instead of the docs you were instructed to follow. Typical setup needs an embeddings API, a local index (often `.sourcerer/`, **gitignore**), and respects **`.gitignore`** — follow upstream package docs.
- **Sensitive domains in this codebase:** treat **config paths**, **SSH**, **Telegram**, and **LLM logs** as high-sensitivity when touching related code; follow epic requirements (e.g. redaction, allowlists) where they apply.
