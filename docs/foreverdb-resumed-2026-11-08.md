# ForeverDB resumed — 8 November 2026

The user explicitly requested resuming ForeverDB after the provider-independent quest-log experiment proved too narrow for broader world guidance. The separate provider-independent mode remains supported, but it is no longer the selected production configuration.

## Applied configuration

On `/home/leon/wisp-private`:

- `FOREVERDB_ENABLED=true` enables existing current-build, unexpired source records in world retrieval.
- `WISP_QUESTLOG_ONLY=false` restores the catalog and general world-retrieval paths. Per-question approved character snapshots remain supported and take precedence over dated state.
- `.foreverdb-enabled` restores the import runner's operator opt-in.
- The provider refresh entry is restored to the existing user crontab: `7 * * * * /home/leon/wisp-private/refresh-foreverdb.sh # wisp-foreverdb-refresh` (VM time).
- The official-source worker continues every 15 minutes; daily purge remains at 03:17. Those entries were preserved.

Only the Jornia API container was recreated, using the existing `wisp-private-api:20261108-guidance2` image. Database, gateway, admin and all other pre-existing containers retained their IDs. Private `.env`, crontab and container-inventory backups were saved before mutation. No game reload, client restart, client secret distribution, public port or firewall change was needed.

The import retains a separate lock, 192 MiB/0.5 CPU container limits, a 12-minute deadline, 200-request default batch, 8,000/day shared safety budget, provider pacing/cooldowns and free-disk/available-memory guards. An immediate bounded refresh was also started to verify the resumed integration.

## Verified behavior and important gap

API health returned OK and the running container reports the intended flags. A real model call for **“Tell me about Silverpine Forest”** returned an answered result with four `foreverdb:1.60.1.70862:detail:…` sources, each titled **“Data from ForeverDB.net”**.

The general question **“Where should I level 10–15?”** still returned unknown and requested current context. Switching providers back on does **not** establish useful level-range retrieval. Inspection also found current-build zone projections containing only a slug: a stored zone name or large record count does not prove that a level range or travel recommendation is available to the model.

Existing native reply metadata marks ForeverDB-derived answers. `Companion.lua` displays the clickable **“Data from ForeverDB.net — https://foreverdb.net”** credit when that marker is present. The authorized provider response, version and field provenance remain attached in storage; blending facts in an answer does not remove attribution. The provider's collection/structure/observations and Blizzard's underlying game content should not be conflated. No provider dataset is relicensed as Jornia's open-source code.

A read-only current-build inventory found **3,568 zone rows but only 83 distinct zone names**, alongside 3,991 quest and 23,111 item rows. These are projection rows, not independent confirmations or complete canonical entities. Incidental references and duplicated names make raw counts particularly misleading. The stored current provider build was `1.60.1.70862`.

Inspection of the cached current-build `/api/v1/zones?limit=100` response found only `id`, `name`, `newInForever`, `parentId`, `type` and `url` across its rows. No level-range field was present in that catalog. The next slice must therefore inspect supported richer endpoints or obtain separately sourced guidance; it cannot fix this solely by renaming a missing JSON field. Map hierarchy (`parentId`) should also be preserved rather than mistaken for a level recommendation.

## Incremental enrichment priorities

Use the existing [data freshness strategy](data-freshness-strategy.md), with the following practical order. This is the next implementation plan, not a claim that these capabilities have shipped.

1. **Make beginner zone advice answerable.** Inspect actual zone list/detail response fields; preserve level ranges, faction constraints, map IDs, travel connections and their original Classic/client/Forever labels. Distinguish full zone records from incidental zone-name references. Add a bounded level-range/faction retrieval tool rather than hoping lexical name matching answers an unnamed-zone question. Test the explicit requested range independently of a level-2 player's current level.
2. **Deepen active-quest guidance.** Prioritize current quests and adjacent quest hubs for objectives, observed start/turn-in locations, prerequisites and follow-ups. Preserve observed vs inferred vs Classic fallback distinctions; player event order never proves a prerequisite.
3. **Expand by useful category.** Classes/talents, trainers/professions, item acquisition, dungeons and travel follow in measurable slices. Keep the full inventory of categories, but count a usable answer only when its required fields and relationships are present.
4. **Drive scheduled work from gaps and changes.** Continue resumable bounded imports, version locking, expiry and provider cooldowns. Prioritize changed builds and missing useful fields over refreshing every unchanged detail. Keep last reviewed history for audit; never silently merge conflicting builds.
5. **Prove coverage with questions.** For each category report distinct canonical entities, required-field coverage, freshness, source labels and answer success on representative questions. Include unknown/conflict cases and private current-character context. Raw record totals and provider list counts are not completeness measures.
6. **Add independent evidence gradually.** Official APIs/change notes, original sourced editorial guidance and later separately opted-in minimized observations can corroborate or fill gaps. Do not centralize private journeys or enable contribution collection by this operational change.

A successful acceptance case is: a player asks for a level 10–15 zone, Jornia finds suitable sourced candidates, explains faction/travel assumptions, distinguishes dated/general guidance from live quest state, and shows the source. Recommendations must not be manufactured from absent level or faction fields.

## Rollback

Set `FOREVERDB_ENABLED=false` and `WISP_QUESTLOG_ONLY=true`, remove only the `# wisp-foreverdb-refresh` cron entry, remove the operator marker and recreate only `api`. Preserve all stored provider records and player data. Restore individual settings from the private timestamped backups; never overwrite newer unrelated cron entries or run global Docker cleanup.
