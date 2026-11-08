# Proposed way forward

Status: discussion draft, 2026-11-05. This document proposes scope; it does not record approved implementation decisions.

## Product to prove first

Make one short conversation with Jornia useful, grounded, and transparent about what it knows. Start with an API and a small Ashenvale dataset. Keep the Windows installer, addon bridge, cloud accounts, and deployment outside this milestone.

The first demonstration should answer a supported world question with source references, acknowledge an unsupported question, and report the model, token usage, and estimated inference cost when a model is called.

## Resolve the inputs

Before writing the application:

1. Identify the exact game/server and client version meant by “WoW: Forever,” including differences from available Classic or Vanilla datasets. This determines which facts and addon APIs apply.
2. Verify Berget's current API model IDs, endpoint support, pricing, and usage reporting. Select one conversational baseline; defer multi-model routing until measurements justify it.
3. Review a small candidate knowledge source for licensing, redistribution rights, provenance, and game compatibility. Use clearly labelled original fixtures for development while these checks are pending.
4. Define what personal context would be sent to the API or inference provider. Local history storage and cloud synchronization are distinct from sending selected context for remote inference; make that understandable before building memory features.

Track real-time addon communication as an independent feasibility investigation. The external companion should provide a viable experience even if a supported native bridge is unavailable.

## Smallest implementation

Use a Go API and local Docker Compose PostgreSQL as proposed in the handoff. Begin with relational entities, edges, source revisions, and version compatibility. Add full-text search when needed; defer vector search until examples show a retrieval gap.

Keep public knowledge and private player data separate. Do not introduce personal memory synchronization in the first slice.

Agree on a small request/response contract before implementation. A response should distinguish answered, unknown, and clarification-needed outcomes and include source references. Model, usage, and cost fields should accurately represent deterministic answers and unavailable provider accounting rather than inventing values.

Build in this order:

1. Health endpoint, migrations, and a tiny original fixture dataset with explicit provenance and game-version scope.
2. Exact entity lookup and deterministic answers for a narrow set of supported questions.
3. One configurable Berget adapter and a grounded question endpoint, with explicit unknown handling and bounded responses.
4. Integration tests and a small evaluation set covering supported facts, missing facts, ambiguity, and spoilers. Expand toward the handoff's 100-question benchmark once real licensed knowledge is available.

Create application directories as implementation requires them. Keep knowledge schemas and import conventions extractable into `wisp-knowledge` later, without creating a second repository now.

## Completion evidence

- A documented local startup works from a clean checkout.
- A supported question returns a grounded answer with traceable sources.
- An unsupported question acknowledges missing information.
- Deterministic answers require no model call.
- A live provider smoke test records actual usage when available and estimates cost from documented pricing.
- Automated tests run with a fake provider; live inference is a separate opt-in check.
- Credentials and personal data stay out of Git.

## Next conversation

First settle the target game/client and the preferred first interaction: a direct factual question, or a spoiler-aware exploration suggestion. The factual path is the proposed first implementation because its correctness is easier to demonstrate; exploration can follow using the same knowledge foundation.

No external services, paid infrastructure, deployment, or third-party dataset publication is included in this proposal.
