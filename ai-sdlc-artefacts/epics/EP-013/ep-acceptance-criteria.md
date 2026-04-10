# EP-013 — Acceptance criteria

**Pipeline:** Stage 5.  
**Inputs:** [ep-scope.md](ep-scope.md), [ep-requirements.md](ep-requirements.md)

---

## Introduction

Testable conditions for runtime skills, `vec_skills`, tool union with `always_include`, marked merged system content, memory indexing guard, and turn-boundary behaviour. Traceability uses [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-13.001](#ac-13-001) | REQ-13.007 | Startup fails when SKILL.md contains a canonical marker line |
| [AC-13.002](#ac-13-002) | REQ-13.006 | Startup fails when skill references unknown tool id |
| [AC-13.003](#ac-13-003) | REQ-13.003 | Config load fails when always_include references unknown tool |
| [AC-13.004](#ac-13-004) | REQ-13.014, REQ-13.015 | Merged system begins with trust policy and wraps non-empty retrieved context in RETRIEVED_CONTEXT markers |
| [AC-13.005](#ac-13-005) | REQ-13.015 | Tool instruction aggregate wrapped in TOOL_INSTRUCTIONS markers when non-empty |
| [AC-13.006](#ac-13-006) | REQ-13.010, REQ-13.016 | When skills enabled and index ready, merged system contains RUNTIME_SKILLS block with selected skill body text |
| [AC-13.007](#ac-13-007) | REQ-13.011 | Tool list includes union of always_include, skill-declared tools, and vector-selected tools |
| [AC-13.008](#ac-13-008) | REQ-13.013 | When runtime skills disabled, handler behaviour matches pre-selection without skill packages |
| [AC-13.009](#ac-13-009) | REQ-13.018 | indexTurn refuses chunk containing forbidden marker line |
| [AC-13.010](#ac-13-010) | REQ-13.009 | vec_skills table accepts inserts after Clear+rebuild pattern |
| [AC-13.011](#ac-13-011) | REQ-13.005 | Load fails when SKILL.md missing required frontmatter |
| [AC-13.012](#ac-13-012) | REQ-13.017 | Same system string reused across simulated tool round (no rebuild of messages[0] mid-turn) |
| [AC-13.013](#ac-13-013) | REQ-13.020 | Integration test exercises core message path with mock LLM and runtime skills enabled |
| [AC-13.014](#ac-13-014) | REQ-13.012 | When skill rune budget exceeded, lower-ranked selected skill is dropped entirely |

---

## Acceptance criteria

<a id="ac-13-001"></a>**AC-13.001** (Trace: [REQ-13.007](ep-requirements.md#requirements))

**Given** a skill package whose `SKILL.md` contains a line exactly equal to `<<<PA_BEGIN_RETRIEVED_CONTEXT>>>` after trim  
**When** the application loads configuration with `runtime_skills.enabled` true  
**Then** startup SHALL fail with an error that references the skill directory or file path

---

<a id="ac-13-002"></a>**AC-13.002** (Trace: [REQ-13.006](ep-requirements.md#requirements))

**Given** a skill package whose `tools` field lists a tool id that exists neither in the tool catalog nor in the allowed native tool id set  
**When** the application loads configuration with `runtime_skills.enabled` true  
**Then** startup SHALL fail with an error that identifies the unknown tool id

---

<a id="ac-13-003"></a>**AC-13.003** (Trace: [REQ-13.003](ep-requirements.md#requirements))

**Given** `runtime_skills.always_include` contains a tool id not present in the catalog or allowed native set  
**When** configuration is loaded  
**Then** load SHALL fail with an error that identifies the tool id

---

<a id="ac-13-004"></a>**AC-13.004** (Trace: [REQ-13.014](ep-requirements.md#requirements), [REQ-13.015](ep-requirements.md#requirements))

**Given** a handler build with vector memory and a user message that yields non-empty retrieved context  
**When** the first LLM request is assembled  
**Then** the system message content SHALL start with the configured English trust policy text  
**And** the retrieved context SHALL appear between `<<<PA_BEGIN_RETRIEVED_CONTEXT>>>` and `<<<PA_END_RETRIEVED_CONTEXT>>>`

---

<a id="ac-13-005"></a>**AC-13.005** (Trace: [REQ-13.015](ep-requirements.md#requirements))

**Given** a handler with catalog tools selected so that aggregate system prompts are non-empty  
**When** the first LLM request is assembled in native tool-calling mode  
**Then** the aggregate tool instructions SHALL appear between `<<<PA_BEGIN_TOOL_INSTRUCTIONS>>>` and `<<<PA_END_TOOL_INSTRUCTIONS>>>`

---

<a id="ac-13-006"></a>**AC-13.006** (Trace: [REQ-13.010](ep-requirements.md#requirements), [REQ-13.016](ep-requirements.md#requirements))

**Given** `runtime_skills.enabled` true, a non-empty skill index, and a user message whose embedding search selects at least one skill  
**When** the first LLM request is assembled  
**Then** the system message SHALL contain a `<<<PA_BEGIN_RUNTIME_SKILLS>>>` / `<<<PA_END_RUNTIME_SKILLS>>>` block whose body includes text from the selected skill `SKILL.md` body

---

<a id="ac-13-007"></a>**AC-13.007** (Trace: [REQ-13.011](ep-requirements.md#requirements))

**Given** `always_include` lists tool A, the selected skill declares tool B, and vector pre-selection returns tool C  
**When** completion options are built  
**Then** the LLM tool list SHALL include tools A, B, and C when each exists in the catalog or native registry

---

<a id="ac-13-008"></a>**AC-13.008** (Trace: [REQ-13.013](ep-requirements.md#requirements))

**Given** `runtime_skills.enabled` false  
**When** a user message is handled with a ready tool index  
**Then** tool selection SHALL follow existing EP-004 pre-selection and fallback without requiring skill packages

---

<a id="ac-13-009"></a>**AC-13.009** (Trace: [REQ-13.018](ep-requirements.md#requirements))

**Given** a conversation handler with vector store and embedder configured  
**When** `indexTurn` is invoked with user text containing a line equal to a canonical marker after trim  
**Then** the operation SHALL return an error and SHALL not call `vectorStore.Add` for that chunk

---

<a id="ac-13-010"></a>**AC-13.010** (Trace: [REQ-13.009](ep-requirements.md#requirements))

**Given** an empty `vec_skills` table in a test database  
**When** the skill index build runs Clear then Add for at least one skill row  
**Then** Search returns that skill id for a matching query embedding

---

<a id="ac-13-011"></a>**AC-13.011** (Trace: [REQ-13.005](ep-requirements.md#requirements))

**Given** a `SKILL.md` without `name` or `description` in YAML frontmatter  
**When** the loader parses the skill directory  
**Then** the loader SHALL return an error

---

<a id="ac-13-012"></a>**AC-13.012** (Trace: [REQ-13.017](ep-requirements.md#requirements))

**Given** a completed first LLM call in a tool round with appended tool messages  
**When** the follow-up LLM call is made in the same `HandleMessage` invocation  
**Then** the first message `Content` (system) SHALL equal the system content used for the first call

---

<a id="ac-13-013"></a>**AC-13.013** (Trace: [REQ-13.020](ep-requirements.md#requirements))

**Given** a test harness with mock LLM provider, tool catalog, runtime skills enabled, and at least one valid skill package  
**When** `HandleMessage` runs for an allowed user text  
**Then** the mock SHALL observe a request whose system content includes the RUNTIME_SKILLS marker block  
**And** the process SHALL complete without panic

---

<a id="ac-13-014"></a>**AC-13.014** (Trace: [REQ-13.012](ep-requirements.md#requirements))

**Given** two skills selected by score ordering and a `max_skill_runes_per_turn` smaller than the combined body rune count  
**When** skill bodies are trimmed to satisfy the budget  
**Then** the lower-ranked skill body SHALL be omitted entirely and the higher-ranked skill body SHALL remain fully present
