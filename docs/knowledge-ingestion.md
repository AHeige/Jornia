# Forever data collection, versions and coverage

Investigated 5 November 2026. The goal is broad import of reusable data with continuing updates and explicit gaps. No investigated source provides a verified, complete and freely redistributable Forever corpus. This document records the Blizzard-first ingestion stage; subsequent ForeverDB work is tracked separately.

## Sources and actual access

**Official Blizzard Game Data API:** the primary source for this implementation. The developer of [blizzard_api 0.8.0](https://github.com/Pinta365/blizzard_api/releases/tag/0.8.0) reported the new Forever surface, namespaces and differences from Classic/Retail. That report was a discovery lead; Jornia then made its own authenticated requests and verified response namespaces. The first live run reported `static-1.60.1_70399-forever-eu`, EU region, en_GB locale. This is the API's reported version, not a claim that it matches the latest game client.

API data is used under [Blizzard's Developer API Terms](https://www.blizzard.com/en-us/legal/a2989b50-5f16-43b1-abec-2ae17cc09dd6/blizzard-developer-api-terms-of-use). Section 18 specifies a maximum 30-day TTL; Jornia uses 28 days and scheduled cleanup. This is not an open-data license. Credentials stay in ignored operator `.env` files on the development machine/private VM. They travel in OAuth/Authorization headers, never URL parameters or client bundles. This worker fetches no player profiles, guilds, auctions or account data.

**QuestieDB:** [LICENSE.md](https://raw.githubusercontent.com/Questie/QuestieDB/master/LICENSE.md), [PROVENANCE.md](https://raw.githubusercontent.com/Questie/QuestieDB/master/PROVENANCE.md) and the [Forever description](https://raw.githubusercontent.com/Questie/QuestieDB/master/docs/forever.md) were checked again. Historical rights remain explicitly uncertain and the Forever material is described as incomplete. No bulk import assumes everything is GPL.

**ForeverDB:** its [developer API](https://foreverdb.net/developers) offers complementary quests, giver/recipient relationships, positions and observations not established by Blizzard catalogs. It requires separately approved developer access, attribution and compliance. At this stage it was a candidate, not a completed import or assumed free database extract. Classic data and observations must remain distinct there too.

**Warcraft Wiki:** retain the narrow hand-reviewed adaptation. Direct bulk access returned 403; no restriction was bypassed. **ForeverDiff/pfQuest, 60.tools and GearInsight** are additional comparison candidates. A Classic baseline or code license proves neither Forever compatibility nor a general right to copy another site's database.

## Worker behavior

`apps/ingest` uses Go and existing PostgreSQL. It needs no LLM, embeddings or extra model cost.

1. Obtain a server-side OAuth Client Credentials token.
2. Query the Forever class index for the current API namespace. Check official hosts and Forever namespaces in response/resource links; no Classic/Retail fallback.
3. Follow allowed public game-data indexes and resource links. Discovery includes classes, races, specializations, professions, quests/categories/areas, items, spells, creatures, factions, item sets, journal encounters/instances, mounts and achievements where the API responds.
4. Search results can be capped. Split an ID range into smaller ranges until pagination fits limits. A capped first result never counts as the whole database. The explicit search range is 0..2147483647; it does not prove all game records are published.
5. Save API payload, checksum, retrieval time and expiry. Project IDs, names, selected numerical values and explicit references into the graph. Reject internal markers such as `[DNT]` from the served graph. This caught an actual internal test profession in an API index.
6. Identify each entity/source by namespace/build and locale. The same numerical ID in two builds becomes distinct graph records. Serve the active discovered API build; retain older raw data only within TTL. Classic and hand-reviewed community data remain separate sources, not automatic API corrections.
7. Save records/references atomically per response. New index data replaces only its own projection. Index references cannot overwrite fetched details. Empty fields mean unknown, not zero; a link does not establish quest prerequisites or drop probability.
8. Resume queued jobs after interruption, detect source versions, normally refresh resources after 24 hours and purge expired API copies. Reprojecting valid cache never extends its original expiry.

VM batches allow at most 500 game-data calls every 15 minutes, at least one second between requests and at most twelve minutes per run. Errors/404/429 persist as retry/not_available/rate_limited, not game facts. Tokens and secrets are not logged. The VM Compose profile separately caps resources. Graphs, documents and job state belong only to Wisp.

## Raw data, graph and coverage differ

An indexed quest name is not a complete quest. Full available JSON responses are retained temporarily, but not all fields are modeled or available to answer routing. Initial projection contains selected catalog facts and source references. NPC spawns, drop chances, full objectives, chains and eligibility must not be inferred merely from an ID.

`GET /api/knowledge/coverage` therefore always reports `complete: false`, servable records by type, active namespaces/locales, documents, details by resource type and queue status. `record_counts_are_not_field_completeness` explicitly distinguishes counts from complete fields. The total in-game dataset is unknown. Never hide 404 s, empty indexes, capped searches or API lag behind a 100% claim.

GIN word-index SQL finds anchors before bounded two-hop retrieval. The former first-100-record approach has been replaced in the server path. Go owns selection, sources and versions; at this stage LangChain remained a thin model step. Model context has a separate size budget. Ingestion makes no model calls.

## Running and reviewing

```sh
python3 scripts/ingest.py --max-requests 200
TEST_DATABASE_URL='postgres://wisp:wisp-local-only@127.0.0.1:55471/wisp?sslmode=disable' make check
```

The VM has a separate `ingestion` Compose profile and `wisp-ingest` command. Scheduling and deployment are documented in the VM notes. The central model does not store private player journals.

Do not retain raw copies indefinitely in dumps, logs or release bundles. Delete temporary transfer dumps after import. Future backups containing API data need the same or shorter retention. Clients purge cached answers carrying `expires_at` at next use; powered-off clients cannot physically purge. Handle this limit before public distribution rather than promising continuous deletion on a powered-off computer. Player-owned events are separate and are not removed by API cache policy.

## Operational update, 6 November 2026

Projection v2 retains additional explicit API fields, including requirements, rewards, quality, classes and item tooltips, as bounded JSON in entity descriptions. Links/media are removed, field size is capped and relationships are no stronger than their sources. Quest narratives are excluded. Missing/omitted fields remain unknown. An item link/reference establishes neither drop probability nor availability in the current world.

Valid documents with older projections are rebuilt, normally 200 per batch. `-reproject-limit 2000 -max-requests 1` performs a bounded larger rebuild with one namespace probe. Original retrieval time, expiry and actual refresh deadline remain. The local CLI default is 200 network calls; the VM runner chooses 500. No database or unrelated service resource limits were raised.

Separate Jornia cron jobs refresh every 15 minutes and run `-purge-only` daily at 03:17 VM time. `infrastructure/vm/refresh-knowledge.sh` provides locking, resource guards and atomic exit status. Purge contacts no provider and needs no valid API key. Backups/cache have the same TTL; content-free operational statistics can remain. See the [delivery checklist](delivery-knowledge-character-2026-11-06.md) for results and question tests.

At this update, ForeverDB's documented API required a separate approved key and visible attribution; no key had yet been supplied. QuestieDB historical rights remained uncertain and no bulk import was enabled. Original source status was rechecked on 6 November 2026. Later approved ForeverDB integration is described in [its delivery checklist](foreverdb-chat-delivery.md).
