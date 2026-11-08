# Jornia — Project handoff for Codex

I've verified Berget.ai's current offering. They do have a very small, exceptionally cheap model worth testing. And your Mac + Ubuntu VM setup is a good fit.

## Berget.ai: which model should Jornia use?

Qwen 3.5 2B

Berget.ai · Decision model · Preview

Cheapest candidate

Input / million tokens

# €0.042

Output / million tokens

# Free

At 200 input tokens, that works out to approximately €0.0000084 per call.

The catch: this is a preview decision model, not a proven general conversational model. We need to test whether it supports our intended endpoint and delivers acceptable conversational quality.

[Berget model catalogue](https://berget.ai/en/models?utm_source=chatgpt.com)

For production, I'd benchmark two additional models.

| Model                 | Input / 1M | Output / 1M | Role                        |
| --------------------- | ---------- | ----------- | --------------------------- |
| Qwen 3.5 2B           | €0.042     | Free        | Experimental intent routing |
| Mistral Small 3.2 24B | €0.30      | €0.30       | Conversational default      |
| GLM 5.3 Flash         | €0.25      | €0.50       | More complex reasoning      |

Berget recommends Mistral Small 3.2 for focused Q&A. The service is OpenAI API-compatible and runs inference on Swedish infrastructure.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://berget.ai\&sz=32)

Berget AI+2



My recommendation: Don't commit to one model yet. Build a model-independent adapter, use deterministic graph answers whenever possible, and benchmark these models using real WoW questions.

A 200-input/50-output response on Mistral would cost approximately €0.000075. That's above your original target, but still only €7.50 per 100,000 such responses.

Now, here is the project brief I'd give Codex.

# Jornia — AI Companion for World of Warcraft

**Project brief · Version 0.1 · November 5, 2026**

## 1. Vision

Build **Jornia**, a lightweight, intelligent, text-based AI companion for World of Warcraft: Forever.

Jornia understands Azeroth through an extensive, open-source knowledge graph, remembers a player's personal adventures, and offers natural, context-aware conversations.

The goal is to enhance the experience of playing WoW without disrupting immersion, introducing obligations, or encouraging fear of missing out.

**Product philosophy: A companion that remembers your journey, not one that manages your objectives.**

### Guiding principles

- Immersion first. Jornia only responds when invited unless the player explicitly enables proactive assistance.
- No avatars, voice, or unnecessary graphical interfaces in the MVP.
- Text-first interactions, ideally through an optional native WoW chat tab.
- No grinding, daily quests, retention tricks, or artificial urgency.
- Exploration, curiosity, mastery, storytelling, and community.
- Strong memory and world knowledge with minimal inference costs.
- Open-source shared knowledge graph and community contributions.
- Player information remains private by default.
- Easy installation, ideally one installer and no mandatory registration.
- No subscription dependency for the initial experience.

## 2. Primary user experience

A player downloads and installs Wisp.

The installer detects their WoW installation and configures both the Lua addon and a lightweight local Windows companion.

The player opens WoW and can optionally access Jornia through a dedicated chat interface.

Example:

**Player:** Jornia, I'm level 24. Where should I explore?

**Jornia:** Ashenvale has some beautiful places you might enjoy. Want somewhere dangerous, peaceful, or mysterious?

**Player:** Dangerous. No spoilers.

**Jornia:** I have a few ideas. I'd start by heading north. Let me know if you'd like a hint.

After several weeks:

**Player:** Jornia, what were we doing last time?

**Jornia:** Last time, you were exploring Ashenvale and looking for a way into an area above your level. Want to continue?

Jornia should use retrieved facts and verified player history. It must not invent memories, player locations or game events.

## 3. System architecture

### Development environment

- Primary development machine: Mac mini M4, 24 GB RAM.
- Development agent: OpenAI Codex.
- Source control: GitHub.
- Local development: Docker Compose.
- Deployment target: Ubuntu VM running on separate hardware.
- WoW addon and Windows companion testing: Windows machine with WoW: Forever installed.
- AI inference: Berget.ai external API only.
- No locally hosted or self-hosted LLM.

### Main components

**A. WoW Lua addon**

Responsibilities:

- Native, unobtrusive chat interface.
- Capture permitted character and exploration events.
- Persist supported state through SavedVariables.
- Display previously available knowledge and contextual hints.
- Expose a `/wisp` command.
- Never automate combat or player actions.

The addon must remain transparent, free, and compliant with Blizzard's development policies.

**B. Windows companion**

Preferred language: Go.

Responsibilities:

- Install and configure the WoW addon.
- Maintain a local SQLite database.
- Persist conversations and personal adventure history.
- Synchronise data with the Jornia backend.
- Provide a small external chat interface as a fallback.
- Handle secure authentication and anonymous sessions.
- Manage the connection to WoW using supported, compliant mechanisms.

Important: WoW addons cannot directly make HTTP requests. SavedVariables normally synchronise at logout or UI reload. A real-time in-game communication bridge must be investigated and validated before implementation.

Avoid process injection, memory modification, simulated gameplay input, and anti-cheat bypass techniques.

**C. Jornia API**

Preferred language: Go.

Responsibilities:

- Authentication and anonymous installation identities.
- Conversational API.
- Knowledge retrieval.
- Personal memory retrieval.
- Model selection and routing.
- Token usage and cost tracking.
- Abuse protection and rate limiting.
- Client synchronisation.

**D. Knowledge graph**

Initial technology:

- PostgreSQL.
- Relational entity and edge tables.
- pgvector.
- PostgreSQL full-text search.
- Versioned source provenance.

Evaluate a dedicated graph database later if actual graph traversal requirements justify it.

**E. AI provider**

Berget.ai Serverless Inference.

Endpoint:

`https://api.berget.ai/v1`

Candidate models:

- `Qwen 3.5 2B` — experimental decision/routing model; determine exact API model ID and endpoint support from live catalogue.
- `mistralai/Mistral-Small-3.2-24B-Instruct-2506` — initial conversational baseline.
- `zai-org/GLM-5.3-Flash` — optional complex-query fallback.

Model IDs must be configurable, never hardcoded into business logic.

Keep Berget credentials exclusively in the server environment.

## 4. Knowledge architecture

Create a separate open-source project or package called **wisp-knowledge**.

The graph should represent the World of Warcraft universe as structured, interconnected and searchable knowledge.

### Domains

**World**

Continents, zones, subzones, locations, NPCs, creatures, factions, instances and travel connections.

**Gameplay**

Quests, prerequisites, items, equipment, classes, abilities, professions, encounters, challenges and mechanics.

**Lore**

Characters, historical events, factions, relationships, stories and geographic associations.

**Player memory**

Personal exploration, character milestones, dungeon experiences, conversations, player preferences and explicitly recorded memories.

Public world knowledge and private player knowledge must be stored separately.

### Knowledge sources

Investigate and validate:

- Blizzard Developer API.
- QuestieDB.
- Existing structured WoW datasets.
- Warcraft Wiki, subject to content licensing.
- Community-authored contributions.
- User-observed game events.

Do not assume existing Vanilla/Classic data is correct for Forever.

All imported information must record:

- Source and source revision.
- Canonical entity identifier.
- Game version and patch compatibility.
- Validation status.
- Source licence or redistribution conditions.
- Timestamp or effective version.
- Confidence when appropriate.

Do not publish third-party datasets until redistribution rights have been verified.

### Retrieval

Use a hybrid approach:

1. Exact entity lookup.
2. Relational graph traversal.
3. Full-text keyword search.
4. Vector similarity where useful.
5. Optional reranking.
6. Compact context assembly for the LLM.

Avoid embedding every field indiscriminately. Embeddings should primarily support narrative descriptions, semantic questions and lore retrieval.

Maintain spoiler-awareness at retrieval and response time.

## 5. Low-cost AI architecture

The target is an average variable LLM cost approaching €0.00001 per user message, where feasible.

This is an optimisation goal, not a guaranteed maximum.

Implement the following routing strategy:

**Path A — Deterministic answers**

Known facts such as NPC locations, quest dependencies and structured item information can be retrieved directly and rendered using templates.

No LLM call required.

**Path B — Small-model answers**

Use a cheap model for classification, short responses, and simple conversational transformations when benchmarks show adequate quality.

**Path C — Conversational model**

Use Mistral Small 3.2 for questions requiring natural dialogue, explanations, personalisation, or synthesis from multiple sources.

**Path D — Advanced fallback**

Only escalate to a stronger model when necessary.

### Cost controls

- Strong retrieval before inference.
- Compact, structured context.
- Semantic caching of reusable public answers.
- Exact caching for deterministic queries.
- Short responses by default.
- Strict output-token limits.
- Avoid unnecessary repeated LLM calls.
- Per-request token and latency instrumentation.
- Daily spend budgets and rate limits.
- Explicit handling of unavailable or uncertain information.

Evaluate cost per **successful grounded answer**, not merely cost per inference call.

## 6. Memory and privacy

Each installation receives an anonymous local identity.

Local SQLite persists:

- Conversations.
- Exploration history.
- User preferences.
- Pending synchronisation events.
- Personal notes.
- Last known game state from permitted data collection.

Optional registration links the local identity to a cloud account.

Cloud synchronisation must be opt-in.

Design for consent, deletion, export and reasonable retention limits.

The public graph must never expose private player history.

Use stable event IDs and idempotent synchronisation.

## 7. Community model

Jornia should initially be free and community-oriented.

Open-source components should include:

- Knowledge graph schemas.
- Data ingestion tools.
- Validation scripts.
- Publicly redistributable knowledge datasets.
- Lua addon source.
- Documentation and contribution guidelines.

The backend may be open-source later, but that is not a requirement for the MVP.

Funding could eventually come from voluntary community support, provided Blizzard's addon policy and relevant terms permit the arrangement.

No paid in-game features or addon-related subscriptions should be introduced without resolving those restrictions.

## 8. Suggested repository structure

```
wisp/
├── apps/
│   ├── addon/
│   ├── companion/
│   └── api/
├── packages/
│   ├── knowledge/
│   ├── retrieval/
│   ├── ai/
│   └── contracts/
├── pipelines/
│   └── ingestion/
├── infrastructure/
│   ├── docker/
│   └── deployment/
├── tests/
│   ├── integration/
│   └── evaluations/
├── docs/
├── AGENTS.md
├── docker-compose.yml
└── README.md
```

Organise by responsibility rather than accumulating unrelated files in flat directories.

Prefer small cohesive modules, typed boundaries, reusable services, explicit errors and meaningful tests.

Avoid speculative abstractions or introducing distributed infrastructure before it is needed.

## 9. MVP development plan

### Phase 1 — Core backend

Create the monorepo, Docker Compose setup and PostgreSQL schema.

Implement:

- Health endpoint.
- Knowledge entity schema.
- Relationships schema.
- Basic query service.
- Berget.ai provider adapter.
- `/api/wisp/ask` endpoint.

### Phase 2 — First knowledge import

Start with Ashenvale as the reference zone.

Import a limited, legally reusable dataset of zones, NPCs, quests and locations.

Construct relationships and provenance metadata.

Demonstrate deterministic graph queries.

### Phase 3 — AI evaluation

Create a benchmark suite containing at least 100 questions across:

- Factual knowledge.
- Exploration.
- Lore.
- Quest dependencies.
- Spoiler avoidance.
- Conversational quality.
- Unknown or ambiguous information.

Compare the candidate Berget models for correctness, latency and actual cost.

### Phase 4 — Local companion

Build the Windows companion with SQLite storage, an external text chat interface and secure API communication.

The companion must remain usable without a WoW connection.

### Phase 5 — WoW addon integration

Create the native chat UI and permitted event collection.

Investigate supported integration methods.

Do not depend on an unverified real-time bridge.

Use an external companion window or overlay as a fallback when native real-time chat cannot be achieved compliantly.

### Phase 6 — Ubuntu deployment

Deploy backend services using Docker Compose.

Configure:

- PostgreSQL persistence.
- HTTPS reverse proxy.
- Environment-based secrets.
- Database migrations.
- Backups.
- Health checks.
- Logging and cost metrics.
- Restricted administration access.

Only promote a tested version to the Ubuntu environment.

## 10. Acceptance criteria for the first milestone

A developer can clone the repository on macOS, configure a Berget.ai API key, and start the stack locally.

The system can:

1. Store a zone and several related entities.
2. Query those entities through a typed API.
3. Accept a natural-language question.
4. Retrieve relevant grounded context.
5. Generate a short response using Berget.ai.
6. Return a structured answer containing the text, source references, model identifier, token usage and estimated cost.
7. Correctly acknowledge missing information rather than hallucinating.
8. Run automated integration tests.

The WoW addon, Windows installer and hosting deployment are not required for this first milestone.

## 11. Instructions for Codex

Begin by analysing this project brief.

Create a minimal, maintainable foundation rather than implementing every component immediately.

First:

1. Research and validate the exact Berget.ai API model identifiers and capabilities.
2. Review candidate knowledge sources and licences.
3. Define the initial architecture and contracts.
4. Create the repository structure.
5. Implement a local PostgreSQL graph schema and seed data.
6. Implement the first grounded Jornia API endpoint.
7. Add tests and an initial evaluation harness.
8. Document decisions, known limitations and the next implementation steps.

Do not deploy, provision paid infrastructure, ingest restricted content or create external services without explicit approval.

**Prioritise a working vertical slice over architectural complexity.**

The first meaningful success is a real conversation with Jornia, grounded in a small but trustworthy graph of Azeroth.

## What I'd do first on your Mac

I wouldn't give Codex the entire implementation at once. Start with the brief above and ask it to deliver Phase 1 and a small part of Phase 2.

One useful technical decision: keep the Berget API provider abstracted from the knowledge and retrieval layers. If their preview 2B model doesn't meet our quality requirements, switching to Mistral should be a configuration change, not a rewrite.

Berget offers €5 in initial trial credits, so you can begin benchmarking without committing to a monthly plan.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://berget.ai\&sz=32)

Berget AI



For the first session, success is simple: ask Jornia about Ashenvale through an API on your Mac, get an accurate graph-grounded answer, and see exactly how much the inference cost.

After that, move the services onto Ubuntu and tackle the WoW connection separately.

You now have a defined product, a technical direction, a cost strategy and a first milestone. The main unresolved risk is not AI or graph technology—it's whether we can deliver the desired native in-game experience within Blizzard's rules.