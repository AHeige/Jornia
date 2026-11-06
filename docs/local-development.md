# Local development

Current companion and WoW test instructions are in [the first-test walkthrough](wow-first-test.md). The central stack now includes a private LangChain model service; the Go API retains retrieval and deterministic answers.

## Start

Requires Docker with Compose and an available Docker daemon. Go 1.26+ and Python 3 are needed for host-side checks.

1. Copy `.env.example` to `.env` if `.env` does not already exist. Add `BERGET_API_KEY` for live inference. Never commit the key.
2. Run `make up`.
3. Run `make eval`.

On this Mac the working daemon is the `colima-logos-graph` context. Wisp uses its own Compose project, containers, network, and volume alongside existing services. The separate `colima` daemon returned filesystem input/output errors and was not modified. A repository-local Compose v5.7.0 binary was downloaded from Docker's official GitHub release and checked against the release's SHA-256 file. Until Compose is installed globally, use:

```sh
DOCKER_CONTEXT=colima-logos-graph make up COMPOSE=./.cache/docker-compose
DOCKER_CONTEXT=colima-logos-graph make logs COMPOSE=./.cache/docker-compose
```

This does not change the global Docker context. `.cache/` is ignored and is not required on other machines with Compose installed. The default Docker context on this machine did not have a running daemon when this foundation was built.

Compose starts PostgreSQL, runs migrations and idempotent seeding of both the original fixture and reviewed Forever data, then starts the API. PostgreSQL persists in the `wisp_postgres-data` Docker volume. Both published ports bind only to loopback. Default ports: API 8090, database 55471; override in `.env` if necessary.

The default database credentials are deliberately local-development credentials. This Compose configuration is not an internet-facing deployment. It has no public authentication, TLS termination, backups, daily spend budget, or production abuse protection.

## Try Ashenvale

Default `forever` requests use the curated community knowledge slice; see [source review and comparison](ashenvale-research.md). The fixture remains available for isolated regression checks.

```sh
curl -s http://127.0.0.1:8090/api/wisp/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Where is Raene Wolfrunner?"}'
```

The following examples explicitly exercise the old fixture:

```sh
curl -s http://127.0.0.1:8090/healthz
curl -s 'http://127.0.0.1:8090/api/knowledge?game_version=fixture-ashenvale-v1'
curl -s http://127.0.0.1:8090/api/wisp/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Where is Astranaar?","game_version":"fixture-ashenvale-v1"}'
```

Supported deterministic forms are `Where is <name>?`, `Tell me about <name>`, `What is <name>?`, `What places are in <name>?`, `Who gives <quest>?`, `Who do I turn in <quest> to?`, and `What quests are in <zone>?`. Swedish equivalents are also accepted for these patterns. Unsupported forms return `unknown` without inference. This is a narrow initial parser, not a general intent classifier.

Add `"mode":"generate"` to explicitly request a model answer:

```sh
curl -s http://127.0.0.1:8090/api/wisp/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Briefly explain where Astranaar is in the supplied fixture.","game_version":"fixture-ashenvale-v1","mode":"generate"}'
```

Live inference requires a key in `.env`. After changing `.env`, run `make up` again to recreate the API with the new environment. `docker compose restart` alone does not reload environment variables.

The key stays in the private model-service container environment; the API calls that service over the internal Compose network. Requests send the question and a bounded public graph neighborhood to Berget. There is no private memory or account implementation in this slice.

## API contract

`POST /api/wisp/ask` accepts:

| Field | Meaning |
| --- | --- |
| `question` | Required nonempty string, at most 1000 bytes |
| `game_version` | Defaults to `forever`; uses reviewed community knowledge; `fixture-ashenvale-v1` selects synthetic test data |
| `mode` | `auto` (default, deterministic only in this slice) or `generate` |
| `spoilers` | Defaults to false; true explicitly includes spoiler-labelled data |

The answer contains `status` (`answered` or `unknown`), `text`, `game_version`, `route`, `sources`, `evidence_ids`, `warnings`, `model`, `usage`, `estimated_cost_eur`, and `latency_ms`.

Deterministic answers have null model and usage and zero inference cost. Model answers report provider model and usage; missing usage or pricing produces null cost. Estimates use configured EUR-per-million token rates, not a provider billing receipt. Update both rates when changing models; remove the rates if pricing is unknown.

`generate` retrieves exact named entities and one-hop neighbors, excluding spoilers before inference unless requested. It does not call the model when no matching knowledge exists. Missing configuration returns 503. Model/knowledge failures return 502. Local model limits return 429: at most six requests per minute and two concurrent requests per process. There are no automatic inference retries.

Model answers must return JSON with evidence IDs from the retrieved context. Invalid IDs, unsupported statuses, absent evidence, empty text, and truncated JSON are withheld. Evidence membership checks do **not** prove that every statement in the generated text follows from that evidence; grounded-answer evaluation remains necessary.

`GET /api/knowledge?game_version=...` returns a typed JSON graph with public provenance. It always filters spoilers. This initial graph reader is limited to 100 entities and 200 edges per version; it must be replaced with targeted database retrieval before growing beyond the current small curated slice.

`GET /healthz` checks database connectivity. It does not call the provider or certify migration/data readiness.

## Verify

```sh
make check
TEST_DATABASE_URL='postgres://wisp:wisp-local-only@127.0.0.1:55471/wisp?sslmode=disable' make integration
make eval
make eval-live
```

`make check` runs vet and unit tests; PostgreSQL integration tests skip unless `TEST_DATABASE_URL` is set. `make integration` requires that variable, creates a uniquely named test schema, and removes only that schema afterwards. It tests migrations and seeding twice, graph retrieval, HTTP behavior, version isolation, spoilers, and a fake OpenAI-compatible provider. Unit tests exercise provider errors, truncated output, and missing usage.

`make eval` runs 12 fixture and 18 reviewed Forever acceptance cases against the real running API with no model calls. `make eval-live` makes one real provider call and prints the answer, provenance, model, usage, latency, and estimated cost. The live check validates response structure and a basic expected fact; it is not a conversational-quality benchmark.

`WISP_API_URL` overrides the evaluation base URL when using another API port.

## Stop and preserve data

Run `make down` (with the same Compose override if needed). It removes this stack's containers and network while preserving the named database volume. Do not add `--volumes` unless intentionally deleting local knowledge data.

## Structure and migrations

- `apps/api`: binary entry point, bootstrap command, configuration, shutdown.
- `internal/storage`: typed graph access, embedded immutable SQL migrations, fixture seeding.
- `internal/wisp`: answer routing, retrieval, evidence checks, local inference limits.
- `internal/ai`: configurable Berget Chat Completions adapter.
- `internal/httpapi`: bounded HTTP requests, validation, health, JSON responses.
- `tests/integration`: PostgreSQL-backed tests in isolated schemas.
- `tests/evaluations`: fixture and reviewed-data API cases.

Add new numbered SQL files for schema changes. Migration checksums reject edits to previously applied migrations. Bootstrap applies them transactionally under a database advisory lock. Seed versions are immutable and inserted idempotently; add a new fixture version to change its data.

No GitHub service, paid infrastructure, addon bridge, Windows companion, deployment, or vector index has been created. The small wiki-derived slice is attributed under CC BY-SA 4.0; no bulk third-party dataset has been imported.


## Reviewed data updates

`sources.provenance` records review date, method, local claim fingerprint, attribution, licence, and the absence of an upstream revision when unavailable. `source_checked` denotes a published community-source review; it is distinct from `verified`. Unverified-source entities and edges are excluded from retrieval.

Dataset files are immutable after import. Corrections must use a new review revision and an explicit migration of affected entity/edge records; simply changing an already imported JSON file is rejected by its checksum. The `forever` namespace identifies the game branch, not a confirmed client build. Run `python3 scripts/compare_ashenvale.py` to compare the current sourced slice with the original fixture.
