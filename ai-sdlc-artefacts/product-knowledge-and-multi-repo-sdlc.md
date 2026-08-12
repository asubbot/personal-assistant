# База знаний и multi-repo SDLC — заметки для обсуждения

**Статус:** Черновик для обсуждения (не нормативный; не заменяет [pipeline.spec.md](../ai-sdlc/specification/pipeline.spec.md), пока не согласован и не принят).

**Контекст:** Архитектурные обсуждения (2026-07). Документ разделяет **два уровня**:

1. **База знаний (БЗ)** — отдельный репозиторий-**справочник** по компании / проектам / продуктам; двухуровневая модель **Raw → Canonical**; обновляется автоматически (ingest + AI-синтез + validate). **Не** командная вики с ручным копирайтингом.
2. **`ai-sdlc-artefacts`** — артефакты SDLC **конкретного продукта** (эпики, REQ, AC, design); контракт на поставку; процесс **ai-sdlc** на текущем этапе **не меняем**.

Секции про multi-repo SDLC (§5–§8) — **перспектива** после появления нескольких сервисов; фокус ближайшей работы — **модель БЗ (§3–§4)**.

---

## 1. Постановка проблемы

### 1.1 Сейчас (PersonalAssistant)

- SDLC-артефакты лежат в `ai-sdlc-artefacts/` в том же git-репозитории, что и Go-код (~510 файлов, 40+ эпиков).
- **Плюс:** атомарная поставка — epic-ветка / PR может менять spec, код и тесты вместе; `make validate` на одном дереве.
- **Минус:** продуктовый справочник и контракт delivery смешаны с кодом одного repo; нет общего слоя «памяти компании» для AI-native ритма.

### 1.2 Два уровня (целевое разделение ответственности)

| Уровень | Вопрос | Где живёт | Кто / как обновляется |
|---------|--------|-----------|------------------------|
| **База знаний** | *Что мы знаем о компании, продуктах, контексте?* | Отдельный repo `pa-knowledge` | Ingest → Raw; cron + AI → Canonical; validate |
| **Product SDLC** | *Что обязаны поставить в этом продукте / PR?* | `ai-sdlc-artefacts/` в repo продукта (пока PA) | Pipeline ai-sdlc, осознанные изменения артефактов |

БЗ — **не** база коммитов команды и **не** замена эпиков. Эпики остаются SOT для `make validate ac` и delivery, пока процесс ai-sdlc не эволюционирует отдельно.

### 1.3 Цели

- **БЗ:** единый **автоматический справочник** (Raw → Canonical), успевающий за динамикой AI-native компании без ручного ведения страниц.
- **Multi-repo (позже):** каждый микросервис в своём repo со своим PR и независимым deploy; общий процесс ai-sdlc с pin в consumer-репо.

---

## 2. Согласованные ограничения

### 2.1 База знаний

| Ограничение | Смысл |
|-------------|--------|
| **Отдельный репозиторий** | `pa-knowledge` — не часть repo сервиса. |
| **Два слоя: Raw → Canonical** | Промежуточный ручной/полуручной слой (derived-wiki) **не** используем. |
| **Raw append-only** | Записи в Raw **не корректируются и не удаляются**; только **добавление** новых записей с метаданными (время, тип, источник). |
| **Ingest в первую очередь автоматический** | Raw пополняется из тулов/каналов (экспорты, webhooks, агенты); **ручное** добавление — исключение. |
| **Canonical — machine-synthesized** | Справочные страницы пишет AI по крону; люди **не** ведут canonical как вики. |
| **Инкрементальный синтез** | Обработка **только нового** Raw; пересборка **всех связанных** canonical-страниц + их validate. |
| **Validate — gate для canonical** | Невалидный прогон не заменяет текущий canonical (rollback на предыдущую версию). |
| **OKF conformance для canonical** | Слой `canonical/` — **OKF v0.1 bundle** ([SPEC.md](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)); синтез и validate **не должны** нарушать conformance; расширения pa-knowledge — только дополнительные front matter keys, не ломающие спеку. |
| **Роль человека** | Структура repo, правила обработки (промпты, типы Raw, маппинг сущностей), редкое ручное добавление Raw. |

### 2.2 Multi-repo SDLC (перспектива; ai-sdlc не трогаем сейчас)

| Ограничение | Смысл |
|-------------|--------|
| **PR на сервис** | Каждый микросервис мержится своим PR. |
| **Согласованность spec/code** | В PR сервиса код и тесты соответствуют SDLC-артефактам, на которые ссылается PR. |
| **Независимый deploy** | Сервисы выкатываются своим ритмом. |
| **Один эпик → один сервис** | `EP-XXX` относится ровно к одному микросервису. |
| **Общий процесс** | ai-sdlc с pin в каждом consumer-репо (как `ai-sdlc.version` сейчас). |

**Явно отклонено:**

- **Ручной canonical в БЗ** — не масштабируется для AI-native компании; справочник = синтез из Raw.
- **БЗ как SOT для REQ/AC** — delivery-контракт остаётся в `ai-sdlc-artefacts` (пока не согласован иной перенос).
- **Double diamond как два обязательных merge на эпик** — ломает атомарность spec+code в одном service PR.
- **Platform meta-repo**, собирающий все сервисы одним PR — не требуется.
- **MCP как обязательная инфраструктура БЗ** — вне scope; filesystem + Cursor по умолчанию.

---

## 3. Модель базы знаний: Raw → Canonical

### 3.1 Назначение

БЗ — **справочник**: ответы на «что происходит», «как устроены продукт/сервисы», «о чём говорили», с навигацией (страницы, wikilinks, MOC). Обновляется по расписанию, а не когда «нашли время обновить вики».

```text
внешние источники          pa-knowledge
(tools, channels)                │
       │                         │
       └── ingest (auto) ──►  raw/     append-only, метаданные
                                   │
                    cron + AI agent │
                                   ▼
                            canonical/   справочник (страницы, связи)
                                   │
                            validate job
                                   ▼
                            meta/validation-reports/
```

### 3.2 Слой Raw

**Семантика:** наблюдения «как есть» — без интерпретации и без последующих правок.

| Правило | Детали |
|---------|--------|
| **Append-only** | Существующие записи не редактируются и не удаляются. Уточнение факта = **новая** запись с новым timestamp. |
| **Метаданные обязательны** | Минимум: `ingested_at`, `type`, `source` (канал/тул). Опционально: `external_id`, `content_hash`, теги. |
| **Автоматический ingest — норма** | Чаты, транскрипты, экспорты доков, события из тулов, snapshot'ы из pipeline (например merge эпика → raw) — через агентов/connectors. |
| **Ручное добавление — исключение** | Короткие явные факты (`decisions/`, заметка оператора) когда автоматический канал недоступен. |

**Предложение layout и имени файла:**

```text
raw/
  YYYY/MM/DD/<type>-<source>-<seq>.md   # или .json для структурированных экспортов
```

**Пример front matter:**

```yaml
---
ingested_at: 2026-07-13T14:30:00Z
type: meeting-transcript          # enum: см. meta/raw-types.yaml
source: zoom-export
external_id: mtg-abc123
content_hash: sha256:…
---
```

**Типы Raw** (справочник типов и схем — в `meta/raw-types.yaml`, задаёт человек). Эскиз: **§12.1**.

### 3.3 Слой Canonical

**Семантика:** machine-synthesized **справочник** — читаемые страницы и связи, собранные из Raw. Единственный слой, который читают люди и агенты «по умолчанию» (кроме аудита источников).

**Формат interchange:** каталог `canonical/` оформляется как **Open Knowledge Format (OKF) v0.1 bundle** — directory of markdown concept files + YAML frontmatter, cross-links, optional `index.md` / `log.md`. Спека: [okf/SPEC.md](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md); обзор: [Google Cloud blog](https://cloud.google.com/blog/products/data-analytics/how-the-open-knowledge-format-can-improve-data-sharing).

| Правило | Детали |
|---------|--------|
| **OKF conformance (обязательно)** | Каждый concept-файл (не `index.md` / `log.md`) — markdown с parseable YAML front matter и **непустым `type`**. Связи — стандартные markdown-ссылки. Зарезервированные имена и cross-linking — по §6–§8 OKF. Job B **hard fail** при нарушении §9 Conformance. |
| **Расширения pa-knowledge** | Доп. поля (`sources`, `synthesized_at`, `agent`, …) **разрешены** (OKF: consumers preserve unknown keys). Они **не** заменяют обязательные OKF-поля и **не** отменяют reserved filenames. |
| **Автор — AI-агент** | Генерация по крону (или по событию «новый Raw»); ручное редактирование страниц **не** предусмотрено. |
| **Только инкремент от нового Raw** | Job обрабатывает записи Raw, ещё не учтённые в `meta/synthesis-state` (или аналог). |
| **Пересборка связанных страниц** | Новый Raw может затрагивать несколько сущностей → агент **перегенерирует все связанные** canonical-страницы (не весь vault целиком, если не нужно). |
| **Provenance** | Каждая страница ссылается на Raw-записи-источники (`sources: [...]` — расширение; в body — citations по §8 OKF при цитировании внешних resource). |
| **Текущая правда справочника** | Последний canonical, прошедший validate (включая OKF); иначе — предыдущая версия. |

**Предложение layout (OKF bundle внутри repo):**

```text
canonical/                            ← OKF v0.1 bundle root
  index.md                            ← okf_version: "0.1" + оглавление (§6 OKF)
  log.md                              ← опционально: хронология прогонов синтеза
  products/
    index.md
    personal-assistant.md             ← concept; path = identity
  services/
    index.md
    core.md
  mocs/                               ← доп. index.md для навигации (OKF-совместимо)
```

Реестр допустимых значений **`type`** для concept-файлов — в `meta/synthesis-config/okf-types.yaml` (producer-side; центрального registry OKF нет).

**Пример front matter canonical-страницы (OKF + расширения):**

```yaml
---
# OKF v0.1 (обязательно / рекомендуется)
type: Product
title: Personal Assistant
description: Deployable AI assistant product (service core).
timestamp: 2026-07-13T15:00:00Z
tags: [personal-assistant, core]
resource: https://github.com/asubbot/PersonalAssistant

# pa-knowledge extensions (optional)
sources:
  - raw/2026/07/10/meeting-transcript-zoom-001.md
  - raw/2026/07/12/chat-export-slack-payments-003.json
synthesized_at: 2026-07-13T15:00:00Z
agent: kb-synth-v1
related_entities: [product/personal-assistant, service/core]
---
```

### 3.4 Пайплайн (два cron-job'а)

**Job A — Synthesize** (после ingest или по расписанию):

1. Выбрать необработанный Raw (`meta/synthesis-state`; эскиз **§12.2**).
2. Определить затронутые сущности / связанные canonical-страницы (правила в `meta/synthesis-config/`).
3. Перегенерировать эти страницы из актуального набора Raw-источников.
4. Commit в ветку / staging; запустить Job B.

**Job B — Validate**:

1. **OKF conformance** — [§9 SPEC.md](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md): front matter + `type` на concept-файлах, reserved filenames, cross-linking; **hard fail** при нарушении.
2. **Ссылки** — markdown links между concepts, `sources[]`, относительные пути.
3. **Согласованность формулировок** — противоречия между canonical-страницами одной сущности (rule-based + опционально LLM-отчёт).
4. **Согласованность с Raw** — утверждения на canonical покрыты источниками; нет «фактов без Raw» (anti-hallucination).
5. При **pass** — promote canonical (merge / tag `canonical-current`); при **fail** — отчёт в `meta/validation-reports/`, canonical не обновляется.

### 3.5 Роль человека

| Область | Действия человека |
|---------|-------------------|
| **Структура** | Layout `raw/`, `canonical/`, `meta/`; соглашения об именовании; MOC-шаблоны. |
| **Правила обработки** | `meta/raw-types.yaml`, `meta/synthesis-config/` (промпты, маппинг type → сущности, пороги validate). |
| **Ingest** | Настройка connectors к тулам/каналам (основной поток данных). |
| **Raw (редко)** | Ручная запись в Raw при отсутствии автоматического канала. |
| **Canonical** | **Не пишет.** Исправление справочника = новый/уточняющий Raw + следующий прогон синтеза. |

### 3.6 Obsidian (опциональный UI)

- **Vault root:** корень `pa-knowledge`.
- **Чтение:** canonical + graph; Raw — для «показать источник».
- **Плагины (для обсуждения):** Obsidian Git (sync clone), Dataview (дашборды по front matter), Linter; Mermaid в страницах canonical.
- Obsidian — **навигация**, не отдельный SOT и не редактор canonical.

---

## 4. Топология репозиториев

```text
pa-knowledge/                         ← база знаний (справочник)
  raw/                                ← append-only, метаданные
  canonical/                          ← OKF v0.1 bundle (AI-синтез)
  meta/
    raw-types.yaml
    synthesis-config/
      okf-types.yaml                  ← реестр type для OKF concepts
    synthesis-state/                  ← какой Raw уже обработан
    validation-reports/
    agent-runs/
  .obsidian/                          ← опционально, team plugins

PersonalAssistant/                    ← продукт (вырожденный случай: один deployable)
  ai-sdlc-artefacts/                  ← SDLC артефакты продукта (без изменений процесса)
  ai-sdlc.version + ai-sdlc/
  … код …

pa-svc-<name>/                        ← перспектива: repo микросервиса
  ai-sdlc-artefacts/ или sdlc.ref      ← TBD при split
  knowledge.ref                       ← опционально: SHA pa-knowledge для контекста агента, не для validate ac

github.com/asubbot/ai-sdlc            ← процесс + validate (pin only)
```

**Связь БЗ ↔ SDLC (пока слабая, односторонняя):**

- События delivery (merge эпика, release) **могут** попадать в Raw как `type: sdlc-artifact-export` — canonical обновит справочную страницу продукта/эпика.
- `make validate ac` **не** читает canonical; читает `ai-sdlc-artefacts/` в repo продукта.

---

## 5. Multi-repo SDLC (перспектива)

*Секция сохранена для будущего split; **не** в фокусе текущей итерации.*

### 5.1 Сущности планирования

**Эпик (`EP-XXX`)** — полный набор артефактов pipeline; ровно один микросервис; расположение **в repo продукта** (`ai-sdlc-artefacts/epics/`) до отдельного решения о переносе.

**Инициатива (`INIT-XXX`)** — опциональный cross-service слой; при появлении нескольких сервисов может жить в canonical/MOC БЗ или в отдельной папке SDLC — TBD.

### 5.2 Фазы pipeline (логический double diamond)

```text
Фаза Product (stages 3–8)     ai-sdlc-artefacts в repo продукта (или будущий product repo)
        ↓ handoff
Фаза Engineering (9–11)     тот же repo + код сервиса
```

При multi-repo: продуктовые фазы могут переехать в отдельный product repo; БЗ при этом остаётся **справочником**, не заменой `ep-requirements.md`.

### 5.3 Связь service PR с артефактами (кандидаты)

| Механизм | Описание |
|----------|----------|
| **`sdlc.ref` / submodule** | Pin SDLC-артефактов для validate в CI сервиса |
| **`knowledge.ref`** | Опционально: pin БЗ для контекста агентов; **не** gate для AC |

---

## 6. Эволюция (раздельные треки)

### 6.1 База знаний (приоритет)

1. Создать `pa-knowledge` с layout §3–§4.
2. Определить `meta/raw-types.yaml` и первые ingest connectors.
3. MVP агента синтеза: новый Raw → пересборка связанных OKF concept-файлов в `canonical/`.
4. MVP validate: **OKF conformance** (hard fail) + ссылки + sources + базовые противоречия.
5. `meta/synthesis-config/okf-types.yaml` — словарь `type` для concepts.
6. Опционально: Obsidian vault.

### 6.2 ai-sdlc и multi-repo (отложено)

Изменения в upstream ai-sdlc (`KNOWLEDGE_ROOT`, stage skills, initiative) — **только после** стабилизации модели БЗ и решения о судьбе `ai-sdlc-artefacts` при split. Список кандидатов см. предыдущие версии документа; не дублируем как обязательный план.

---

## 7. Сравнения (кратко)

| Инструмент / подход | Роль |
|---------------------|------|
| **pa-knowledge (Raw → Canonical)** | Автоматический справочник; `canonical/` = OKF v0.1 bundle |
| **ai-sdlc-artefacts** | SDLC-контракт конкретного продукта; validate ac |
| **ai-sdlc** | Общий процесс (не меняем в этой итерации) |
| **Obsidian** | UI над canonical в pa-knowledge |
| **Beads / Linear** | Ops / tasks; не замена Raw или canonical |

---

## 8. Эскиз миграции

### 8.1 База знаний (первая итерация)

1. Создать remote **pa-knowledge** с `raw/`, `canonical/`, `meta/`.
2. Завести `meta/raw-types.yaml` и `meta/synthesis-config/`.
3. Подключить 1–2 автоматических ingest-канала (например экспорт чатов / git activity).
4. Запустить cron synthesize + validate.
5. Опционально: открыть vault в Obsidian.

### 8.2 PersonalAssistant / SDLC (без срочных изменений)

1. `ai-sdlc-artefacts/` **остаётся** в repo PA.
2. Опционально: экспорт сводок эпиков в Raw БЗ (`type: sdlc-artifact-export`) для справочных страниц.
3. Multi-repo split и `knowledge.ref` — когда появится второй сервис.

---

## 9. Открытые вопросы

### База знаний

1. **Именование repo:** `pa-knowledge` vs другое; хранить ли большие blob'ы в git LFS / object storage с указателями в Raw?
2. **Схема метаданных Raw:** единый JSON Schema / front matter; реестр `type` — эскиз **§12.1**.
3. **Гранулярность Raw:** один файл на событие vs батч за день по каналу.
4. **Маппинг «связанные страницы»:** `meta/synthesis-config/entity-registry.yaml` (эскиз **§12.3**) vs вывод агентом.
5. **Validate:** какие противоречия — hard fail vs warning; OKF broken cross-links — warning per spec, missing `type` — hard fail.
6. **Rollback canonical:** tag `canonical-current` vs ветка `canonical-published`.
7. **Первые ingest-каналы:** приоритет списка тулов (Telegram, git, Linear, SDLC export, …).
8. **OKF `type` vocabulary:** зафиксировать в `meta/synthesis-config/okf-types.yaml` (Product, Service, Epic, …) vs свободные строки.

### Граница БЗ ↔ SDLC

8. **Экспорт эпиков в Raw:** автоматически при merge / вручную / не делать на MVP?
9. **Конфликт canonical vs эпик:** политика для агентов — «для delivery всегда эпик»; нужна ли явная ссылка из canonical на `ai-sdlc-artefacts` path?

### Multi-repo (отложено)

10. **`knowledge.ref` в service repo:** нужен ли для runtime агентов или достаточно clone БЗ по расписанию?
11. **Перенос `ai-sdlc-artefacts`:** остаётся в product repo или отдельный `pa-product-*` repo?
12. **Initiative layer:** canonical MOC в БЗ vs папка в SDLC.

### Obsidian

13. **Scope:** read-only dashboard vs основной просмотр справочника.
14. **`.obsidian/` в git:** team plugins vs личный `workspace.json` в gitignore.

---

## 10. Предлагаемый порядок обсуждения

1. Подтвердить **модель БЗ Raw → Canonical** и ограничения §2.1.
2. Утвердить **метаданные Raw** и **append-only** (вопросы 2–3).
3. Описать **первый ingest** и **правила пересборки связанных страниц** (вопросы 4, 7).
4. Специфицировать **validate** (OKF conformance + promote/rollback) (вопросы 5–6, 8).
5. Решить **границу с SDLC** на MVP (вопросы 8–9).
6. Multi-repo и ai-sdlc эволюция — **после** п.1–5.

---

## 11. Ссылки (в репозитории)

- [pipeline.spec.md](../ai-sdlc/specification/pipeline.spec.md) — текущий нормативный pipeline (без изменений в этой итерации)
- [VALIDATION.md](../ai-sdlc/tools/validate/VALIDATION.md) — validate для SDLC-артефактов
- [AGENTS.md](../AGENTS.md) — пути `ai-sdlc-artefacts/` в PA
- [docs/installation.md](../docs/installation.md) — nested clone ai-sdlc
- [OKF v0.1 SPEC.md](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md) — interchange-формат для `canonical/`
- [Introducing the Open Knowledge Format](https://cloud.google.com/blog/products/data-analytics/how-the-open-knowledge-format-can-improve-data-sharing) — обзор от Google Cloud

---

## 12. Приложение: эскизы `meta/`

Черновики для обсуждения (не норматив). Пути — относительно корня `pa-knowledge/`.

### 12.1 `meta/raw-types.yaml` — реестр типов Raw

Определяет допустимые значения `type`, обязательные метаданные и маппинг на сущности canonical.

```yaml
# meta/raw-types.yaml
version: 1

# Общие поля front matter для любого Raw (кроме type-specific)
common_required:
  - ingested_at   # RFC3339 UTC
  - type          # ключ из types.*
  - source        # идентификатор канала/коннектора

common_optional:
  - external_id   # id в системе-источнике (дедуп ingest)
  - content_hash  # sha256 тела для идемпотентности
  - tags          # свободные метки

types:
  meeting-transcript:
    description: Транскрипт или заметки встречи
    format: markdown
    required: [external_id]
    default_entities: []          # агент может уточнить при синтезе
    ingest: manual|zoom-export    # кто может создавать

  chat-export:
    description: Пакет сообщений из чата (Slack, Telegram, …)
    format: json
    required: [external_id, content_hash]
    entity_hints:                 # поля JSON → entity id (JMESPath или путь)
      channel: tags.channel
    ingest: telegram-bot|slack-export

  git-activity:
    description: Событие из git (merge, tag, release notes)
    format: json
    required: [external_id]
    entity_hints:
      repository: repo.full_name
    related_entity_rules:
      - match: { repo: "asubbot/PersonalAssistant" }
        entities: [product/personal-assistant, service/core]

  doc-snapshot:
    description: Копия внешнего документа или README на момент ingest
    format: markdown
    required: [content_hash]
    ingest: http-fetch|manual

  decision:
    description: Явное решение оператора (ручной Raw, исключение)
    format: markdown
    required: []                  # оператор обязан заполнить body
    default_entities: []          # обязательно указать в body или tags
    ingest: manual
    notes: Используется когда нет автоматического канала; уточнение = новая запись

  sdlc-artifact-export:
    description: Снимок SDLC-артефакта из product repo (опционально на MVP)
    format: markdown
    required: [external_id, source]
    entity_hints:
      epic: tags.epic_id
      product: tags.product_id
    ingest: pa-export-hook|manual
```

**Дедуп ingest:** коннектор перед записью проверяет пару `(type, external_id)` или `content_hash`; при совпадении — **не** создаёт новый Raw (append-only не нарушается, дубликат не пишется).

### 12.2 `meta/synthesis-state/` — учёт обработанного Raw

Один файл на запись Raw (или индекс + журнал — на выбор реализации). MVP: **файл состояния на Raw-путь**.

**Путь:** `meta/synthesis-state/raw/<mirror-of-raw-path>.yaml`  
Пример: Raw `raw/2026/07/13/meeting-transcript-zoom-001.md` → state `meta/synthesis-state/raw/2026/07/13/meeting-transcript-zoom-001.yaml`

```yaml
# meta/synthesis-state/raw/2026/07/13/meeting-transcript-zoom-001.yaml
raw_path: raw/2026/07/13/meeting-transcript-zoom-001.md
ingested_at: 2026-07-13T14:30:00Z
type: meeting-transcript

# Жизненный цикл обработки
status: synthesized          # pending | synthesizing | synthesized | validate_failed | skipped
first_seen_at: 2026-07-13T14:31:00Z
last_processed_at: 2026-07-13T15:00:00Z
agent: kb-synth-v1

# Какие canonical-страницы пересобраны из-за этого Raw (включая ранее обработанные источники)
affected_pages:
  - canonical/products/personal-assistant.md
  - canonical/services/core.md

# Ссылка на прогон (см. agent-runs)
run_id: 2026-07-13T150000Z-synth-007
```

**Индекс для job A (опционально):** `meta/synthesis-state/index.yaml`

```yaml
version: 1
updated_at: 2026-07-13T15:00:05Z
pending_count: 3
pending:
  - raw/2026/07/13/chat-export-telegram-002.json
  - raw/2026/07/13/git-activity-merge-004.json
last_successful_run_id: 2026-07-13T150000Z-synth-007
```

**Правила:**

- Новый файл в `raw/` → `status: pending` (создаётся state при ingest или при первом scan job A).
- Job A берёт только `pending` (и опционально `validate_failed` после исправления конфига).
- После успешного validate все затронутые Raw в прогоне → `status: synthesized`.
- Повторная обработка того же Raw **без нового файла** не делается; новая информация = **новый** Raw-путь.

### 12.3 `meta/synthesis-config/` — правила пересборки связанных страниц

```yaml
# meta/synthesis-config/entity-registry.yaml
version: 1

entities:
  product/personal-assistant:
    canonical_page: canonical/products/personal-assistant.md
    moc: canonical/mocs/products.md
    raw_match:
      - type: sdlc-artifact-export
        tags.product_id: personal-assistant
      - type: git-activity
        repo: asubbot/PersonalAssistant

  service/core:
    canonical_page: canonical/services/core.md
    raw_match:
      - type: git-activity
        repo: asubbot/PersonalAssistant
      - type: decision
        tags.service: core
```

```yaml
# meta/synthesis-config/agent.yaml
version: 1
agent_id: kb-synth-v1
prompt_template: meta/synthesis-config/prompts/page-synth.md

# При новом Raw: union страниц из entity-registry + entity_hints типа
resolve_related_pages:
  - entity_registry   # raw_match rules
  - type_hints        # entity_hints из raw-types.yaml
  - agent_inference   # опционально; отключить на MVP для предсказуемости

regenerate:
  mode: affected_only  # не полный vault
  include_mocs: true   # обновить MOC, если изменилась дочерняя страница
```

**Алгоритм job A (кратко):**

1. Для каждого Raw с `status: pending` вычислить `affected_pages` (registry + hints).
2. Для каждой страницы из union собрать **все** Raw, которые на неё мапятся (не только новый).
3. Перегенерировать страницу из полного набора источников.
4. Записать `affected_pages` и `run_id` в state каждого обработанного Raw.

### 12.4 `meta/agent-runs/` — журнал прогонов

```yaml
# meta/agent-runs/2026-07-13T150000Z-synth-007.yaml
run_id: 2026-07-13T150000Z-synth-007
kind: synthesize
started_at: 2026-07-13T15:00:00Z
finished_at: 2026-07-13T15:04:12Z
agent: kb-synth-v1
trigger: cron

inputs:
  new_raw:
    - raw/2026/07/13/meeting-transcript-zoom-001.md
  pages_regenerated:
    - canonical/products/personal-assistant.md
    - canonical/services/core.md

output:
  commit_sha: abc1234          # после commit staging
  validate_run_id: 2026-07-13T150412Z-val-007

status: validate_passed        # failed | validate_passed
```

### 12.5 `meta/validation-reports/` — результат job B

```yaml
# meta/validation-reports/2026-07-13T150412Z-val-007.yaml
validate_run_id: 2026-07-13T150412Z-val-007
synthesize_run_id: 2026-07-13T150000Z-synth-007
started_at: 2026-07-13T15:04:12Z
status: passed                 # passed | failed

checks:
  okf_conformance:
    status: passed
    spec_version: "0.1"
    violations: []             # hard fail если не пусто
  links:
    status: passed
    broken: []
  cross_page_consistency:
    status: passed
    warnings: []
  raw_coverage:
    status: passed
    unsupported_claims: []     # hard fail если не пусто

promote:
  action: tag                  # tag | merge
  ref: canonical-current
  previous_ref: def5678
```

При `status: failed` поле `promote` отсутствует; canonical на `canonical-current` **не** меняется.

### 12.6 Пример полной цепочки (один новый Raw)

```text
1. telegram-bot → raw/2026/07/13/chat-export-telegram-002.json
                 meta/synthesis-state/.../chat-export-telegram-002.yaml (pending)

2. job A       → пересборка canonical/… (union по entity-registry; output OKF-conformant)
                 state → synthesized, run_id записан

3. job B       → validate (OKF + links + raw coverage) → meta/validation-reports/...-val-008.yaml

4. pass        → git tag canonical-current на commit с обновлённым canonical/
```

### 12.7 `meta/synthesis-config/okf-types.yaml` — словарь OKF `type`

Producer-side реестр (OKF не требует central registry). Агент синтеза **обязан** использовать значения из списка для concept-файлов в `canonical/`.

```yaml
# meta/synthesis-config/okf-types.yaml
okf_version: "0.1"

types:
  Product:
    description: Продукт / deployable в портфеле
    path_prefix: canonical/products/

  Service:
    description: Микросервис или логический сервис
    path_prefix: canonical/services/

  Epic:
    description: Справочная страница эпика (не SDLC SOT)
    path_prefix: canonical/epics/

  Playbook:
    description: Runbook / operational procedure

  Reference:
    description: Прочий concept; fallback для agent_inference
```

Validate: неизвестный `type` на concept-файле — **warning** (per OKF consumer tolerance); отсутствующий или пустой `type` — **hard fail**.

---

*Владелец документа: TBD. Обновлять по мере принятия решений; правила БЗ — в конвенции pa-knowledge; правила delivery — в ai-sdlc / product repo.*
