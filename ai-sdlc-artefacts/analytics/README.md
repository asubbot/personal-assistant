# Analytics — external project comparisons

This directory holds **comparison and research reports** on open-source or external projects relevant to PersonalAssistant (architecture, security, tooling, deployment), plus a curated list of **external analytical articles**. Each repo report is self-contained markdown with a fixed analysis date and pinned repository revision where possible.

Reports are produced with the project skill [`ai-sdlc/specification/skills/project-comparison-report.skill.md`](../../ai-sdlc/specification/skills/project-comparison-report.skill.md) and aligned with epic design in [`ai-sdlc-artefacts/epics/`](../epics/).

## Reports in this folder

| Slug | Report |
|------|--------|
| PicoClaw | [picoclaw/picoclaw-analysis.md](picoclaw/picoclaw-analysis.md) |
| AAF (Autonomous Agent Framework) | [aaf-autonomous-agent-framework/aaf-analysis.md](aaf-autonomous-agent-framework/aaf-analysis.md) |
| NanoClaw | [nanoclaw/nanoclaw-analysis.md](nanoclaw/nanoclaw-analysis.md) |
| Spacebot | [spacebot/spacebot-analysis.md](spacebot/spacebot-analysis.md) |
| Topsha | [topsha/topsha-analysis.md](topsha/topsha-analysis.md) |
| GoClaw | [goclaw/goclaw-analysis.md](goclaw/goclaw-analysis.md), [goclaw/threat-model.md](goclaw/threat-model.md) |

## Similar projects on GitHub (for further research)

*Not endorsements. Verify license, security posture, and maintenance before reuse. Grouping is approximate (many projects span several columns).*

### Personal assistant / gateway / multi-channel

| Project | URL | Memory / RAG | Tooling / agents | Notes |
|---------|-----|:------------:|:----------------:|-------|
| **OpenClaw** | https://github.com/openclaw/openclaw | Partial (workspace, skills) | Skills, gateway | Multi-channel (Telegram, Slack, …); TypeScript. |
| **PicoClaw** | https://github.com/sipeed/picoclaw | Session / plugins | Tools, agent loop | Go; compared in [picoclaw-analysis.md](picoclaw/picoclaw-analysis.md). |
| **AAF** | https://github.com/th0r3nt/AAF-Autonomous-Agent-Framework- | SQL + Chroma + graph (Kuzu) | Skills, sandbox DinD | Python, Docker; compared in [aaf-analysis.md](aaf-autonomous-agent-framework/aaf-analysis.md). |
| **Open Interpreter** | https://github.com/OpenInterpreter/open-interpreter | Local context | Code execution, tools | Local/code-focused assistant. |
| **Continue** | https://github.com/continuedev/continue | IDE context | Tools in editor | VS Code / JetBrains; coding workflow. |

### OpenClaw-adjacent assistants (ecosystem)

*Multi-channel / “Claw family” projects often named alongside [OpenClaw](https://github.com/openclaw/openclaw) in community comparisons (e.g. [Habr — *Халява уходит из разработки Агентов*](https://habr.com/ru/articles/1010236/)). **Several unrelated repos share similar names**—verify maintainer and license before use.*

| Project | URL | Memory / RAG | Tooling / agents | Notes |
|---------|-----|:------------:|:----------------:|-------|
| **IronClaw** | https://github.com/nearai/ironclaw | Hybrid search, persistent memory | WASM sandbox tools, multi-channel | Rust; OpenClaw-inspired, security-focused. Other `ironclaw` repos exist on GitHub. |
| **ZeroClaw** | https://github.com/zeroclaw-labs/zeroclaw | Pluggable memory | Traits for tools, channels, runtime | Rust; “minimal” assistant infra. Prefer **zeroclaw-labs**; upstream has warned about impersonation forks. |
| **MicroClaw** | https://github.com/microclaw/microclaw | As per project | Chat-oriented agent loop | Rust; inspired by NanoClaw / OpenClaw ecosystem. |
| **NemoClaw** | https://github.com/NVIDIA/NemoClaw | OpenClaw instance in sandbox (onboarding) | OpenShell sandbox, declarative egress/FS/process/inference policies, `nemoclaw` CLI | NVIDIA reference stack for **secure** OpenClaw; **alpha**; [README](https://github.com/NVIDIA/NemoClaw?tab=readme-ov-file), [docs](https://docs.nvidia.com/nemoclaw/latest/). |
| **NullClaw** | https://github.com/nullclaw/nullclaw | Per project | Tools, channels | Zig; very small static binary narrative. |
| **GitClaw** | https://github.com/open-gitagent/gitclaw | Git-committed `memory/` etc. | Declarative YAML tools/skills in repo | Git-native agent layout (`agent.yaml`, hooks). |
| **AstrBot** | https://github.com/AstrBotDevs/AstrBot | Knowledge base, multimodal | Plugins, MCP, agent sandbox | Python; Telegram, QQ, Discord, Slack, Feishu, … **AGPL-3.0**. |
| **GripAI** | https://github.com/5unnykum4r/grip-ai | Hybrid BM25 + vector | Shell, browser, files, DAG multi-agent | Python; Telegram/Discord/Slack; article spelling *GripAi*. |
| **Moltis** | https://github.com/moltis-org/moltis | Hybrid vector + full-text | MCP, multi-channel, Docker sandbox | Rust single-binary; voice + chat channels. |

### Coding agents: composable skills and SDLC workflows

Packaged **skills** (prompts + procedures) and workflows for IDE-embedded agents: spec/plan/TDD, subagents, code review—not a single chat gateway like Telegram bots, but relevant for **tooling, process, and extensibility** patterns.

| Project | URL | Memory / RAG | Tooling / agents | Notes |
|---------|-----|:------------:|:----------------:|-------|
| **Superpowers** | https://github.com/obra/superpowers | Workflow outputs (e.g. saved design/plan docs) | Composable **skills** library; **subagent-driven** implementation; TDD, debugging, git worktrees, review | Agentic skills framework & SDLC methodology; plugins for Claude Code, Cursor, Codex, OpenCode, Gemini CLI, etc. MIT. |

### Long-term memory, RAG, knowledge graphs

| Project | URL | Memory / RAG | Tooling / agents | Notes |
|---------|-----|:------------:|:----------------:|-------|
| **Khoj** | https://github.com/khoj-ai/khoj | Documents, search | Assistants | Self-hosted second brain + chat. |
| **Mem0** | https://github.com/mem0ai/mem0 | User/assistant memory API | Pluggable | Memory layer for apps; vector + graph options. |
| **Letta** | https://github.com/letta-ai/letta | Agent memory (blocks, archival) | Agent runtime | Evolved from MemGPT lineage; persistent agents. |
| **GraphRAG** | https://github.com/microsoft/graphrag | Graph + LLM summaries | Pipelines | Microsoft reference for graph-enhanced RAG. |
| **Cognee** | https://github.com/topoteretes/cognee | Graph + vectors | Pipelines | Knowledge graph + memory for AI apps. |
| **LlamaIndex** | https://github.com/run-llama/llama_index | Indexes, RAG | Workflows, tools | Data framework for LLM apps. |
| **Haystack** | https://github.com/deepset-ai/haystack | RAG pipelines | Agents, tools | Production-oriented NLP/RAG. |

### Agent frameworks (orchestration, multi-step, tools)

| Project | URL | Memory / RAG | Tooling / agents | Notes |
|---------|-----|:------------:|:----------------:|-------|
| **LangGraph** | https://github.com/langchain-ai/langgraph | Via LangChain | Graph/state machines | Durable agents, cycles, human-in-the-loop. |
| **AutoGen** | https://github.com/microsoft/autogen | Conversable agents | Multi-agent, tools | Microsoft multi-agent framework. |
| **CrewAI** | https://github.com/crewAIInc/crewAI | Task context | Role-based crews | Multi-agent “teams”. |
| **Semantic Kernel** | https://github.com/microsoft/semantic-kernel | Plugins, memory connectors | Planners, functions | .NET / Java / Python enterprise SDK. |
| **smolagents** | https://github.com/huggingface/smolagents | Minimal | Code-acting agents | Hugging Face lightweight agents. |

### Self-learning and improvement loops

**Problem space:** systems where an agent **repeatedly** observes outcomes, **reflects**, updates **memory or behaviour** (prompts, plans, tool use, or even **own code**), and **retries**—a closed **Do → Learn → Improve** cycle rather than a single request–response. Overlaps with long-running autonomy, meta-learning, and self-modification. **Risks:** runaway loops, cost spikes, unsafe self-editing, and weak auditability—relevant when comparing to PA’s bounded tool and scheduler model.

| Project | URL | Loop / focus | Notes |
|---------|-----|--------------|-------|
| **Ouroboros** | https://github.com/razzant/ouroboros | Autonomous cycle: goals → plan → act → journal → reflect → **self-modify code** → repeat | Long-running agent; emphasises minimal human intervention; strong overlap with “recursive self-improvement” narrative. |
| **Self-Improving Coding Agent** | https://github.com/MaximeRobeyns/self_improving_coding_agent | Benchmark → agent edits **its own codebase** → re-evaluate | Coding-agent focus; explicit improvement iterations on the repo. |
| **recursive-agents** | https://github.com/hankbesser/recursive-agents | Meta-framework: **critique / refine** own outputs; revision history | Transparency and iterative refinement rather than full autonomy. |
| **AgentLoop** | https://github.com/Guri10/AgentLoop | Closed-loop **control**: state → LLM decision → action until goal | Illustrates looped decision-making architecture. |
| **AutoGPT** | https://github.com/Significant-Gravitas/AutoGPT | **Plan–act–delegate** loop with memory and tools | Classic autonomous agent stack; many forks and plugins. |
| **BabyAGI** | https://github.com/yoheinakajima/babyagi | **Task creation** from objective → execute → reprioritise queue | Early “infinite task list” pattern; lightweight reference implementation. |
| **MetaGPT** | https://github.com/geekan/MetaGPT | Multi-role software org **simulation**; iterative deliverables | Not pure self-edit loop; strong “agent society” and process iteration. |
| **OpenEvolve** | https://github.com/codelion/openevolve | **Evolve** programs with LLM + tests (iterative mutation) | Code/evolution loop; benchmark-driven improvement. |

### Self-hosted chat UI / gateway to models

| Project | URL | Memory / RAG | Tooling / agents | Notes |
|---------|-----|:------------:|:----------------:|-------|
| **LibreChat** | https://github.com/LibreChat-AI/LibreChat | Conversations | Plugins, agents | ChatGPT-like UI; multiple providers. |
| **Lobe Chat** | https://github.com/lobehub/lobe-chat | Plugins | Tool calling | Modern UI; plugin ecosystem. |
| **Open WebUI** | https://github.com/open-webui/open-webui | RAG attachments | Functions / tools | Ollama-friendly web UI. |

## Analytical articles and essays

*Third-party opinion and architecture pieces—not project endorsements. Add rows for articles that inform PA design (security, tools, agents, MCP). Prefer stable URLs; note language if not English.*

| Title | URL | Topics |
|-------|-----|--------|
| *Халява уходит из разработки Агентов* (Habr) | https://habr.com/ru/articles/1010236/ | Shell vs specialised tools; MCP and **tool poisoning**; guardrails limits; sandbox/confinement; **controlled interpreter** vs terminal; human-in-the-loop at tool layer; E2E tests with mocked tools; bottom-up agent design |

## Adding a new report

1. Pick a short directory slug (e.g. `vendor-product`).
2. Follow `project-comparison-report.skill.md`: external repo revision, PA revision, design baseline epic, implementation notes.
3. Add a row to **Reports in this folder** above.

## Adding an analytical article link

1. Add a row to **Analytical articles and essays** with title, URL, and short topic tags (comma-separated).
2. If the piece is paywalled or volatile, consider archiving a citation (DOI, PDF) in the epic or audit notes.

## Disclaimer

Links, project descriptions, and article summaries are for **research and comparison** only. They do not imply affiliation; upstream features, threat models, and third-party claims may change after the report date.
