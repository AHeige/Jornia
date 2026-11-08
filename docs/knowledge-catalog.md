# Jornia category and source catalog

Updated 6 November 2026. The goal is useful answers in defined areas, not the largest possible counter. This note records the catalog rollout before approved ForeverDB ingestion.

- [x] Stable registry of 25 categories.
- [x] Actual record counts and active API details, without claiming field completeness.
- [x] Separate raw facts from unimplemented planners/recommendations.
- [x] Chat access without model cost.
- [x] User submitted the ForeverDB developer-access application.
- [ ] Obtain an approved ForeverDB developer key.
- [ ] Verify authenticated responses, schema/build/origin and terms before enabling import.
- [ ] Integrate ForeverDB and test refresh/attribution.
- [ ] Build/verify talent builds, gear combinations and quest routes from sufficient facts.
- [ ] Add sourced area guides and explicitly subjective experience descriptions.

## Running registry

The internal `GET /api/knowledge/coverage` includes `categories` and `category_counts_overlap`. This is a server registry, not a new client navigation page. Existing desktop clients can use the questions below after the server update; no client version change is needed.

| Group | Categories |
|---|---|
| Quests and exploration | Quests, zones/places, NPCs, travel |
| Character and combat | Classes/races, specs/roles, abilities, talent trees |
| Items and economy | Items, gear/sets, professions, recipes, gathering, reputation |
| Content and activities | Dungeons, raids, bosses, PvP, events, collections |
| Story and experience | Lore/storylines, places/experiences |
| Derived recommendations | Character builds, gear combinations, quest routes |

Each entry describes necessary fields/relations and proposed next sources. `partial` means some records, `missing` means no dedicated records and `not_implemented` means no planner. `field_coverage: not_measured` is intentional: named quests do not establish complete objectives, prerequisites, rewards or chains. Items and gear overlap; category counts cannot be summed into a new total. Detail-document counts filter by active namespace and valid expiry.

Questions answered directly from registry status, even with AI selected:

- `What can you help me with?`
- `What do you know about quests?`
- `What do you know about talents?`
- `What do you know about professions?`
- `What do you know about quest routes?`
- `What do you know about experiences?`

Answers use route `knowledge_catalog`, a dated first-party source and zero model cost. They disclose gaps rather than adding model-memory facts. Named questions such as `What do you know about Astranaar?` follow normal retrieval. Private journals never enter category counts. Swedish query aliases are retained in the implementation as language support.

## Selected fast path: Blizzard and ForeverDB

Blizzard imports continue every 15 minutes. ForeverDB complements quest relationships/positions, talents, professions and build differences; it does not replace our memory, graph model or product logic.

Its [API documentation](https://foreverdb.net/developers) requires approved developer access and visible linked ForeverDB.net attribution. Published limits are 60 requests/minute and 10,000/day. This is not an unrestricted open-data license. Preserve every field's Classic/Forever/observation origin. Chain links, prerequisites and alternative requirements are distinct. Client data presence does not establish live availability.

After approval, put `FOREVERDB_API_KEY` in the operator's ignored `.env`, never the client or chat. First perform bounded metadata/schema/build checks, then prioritize:

1. Zones/quests: objectives, givers/recipients, positions and typed chain relationships.
2. Classes/talents: ranks, tiers and dependencies.
3. Professions/recipes: materials, output and learning sources.
4. Dungeons/gear: instances, bosses, items and set bonuses.
5. Experiences and derived planners once their required facts exist.

Do not start with another full item bulk import: Blizzard already broadly covers that area and it would consume the supplementation quota before filling important gaps. Do not activate another scheduler or empty importer before authenticated responses and reuse terms are verified. Lore and subjective opinions need separate evidence.

## Our world corpus

The user's preference is to collect data and own the knowledge store. Permitted API lists/details fit the architecture: paginated import into our normalized central graph. Ordinary questions query our database rather than calling ForeverDB anew per answer. Preserve versions, attribution, expiry and reuse terms. Approved access is not an unlimited right to mirror/republish a provider database.

Voluntary gameplay observations are a potential later source. Existing journals are not automatically uploaded or repurposed. Central observations need separate opt-in and a distinct schema excluding private history. Area guides and subjective experiences can be authored/reviewed separately.

Operational verification: 25 categories and 31 actual registry questions against the VM passed, including missing talents/lore/experiences and unimplemented planners. No model calls. Go tests, vet and isolated-schema PostgreSQL integration cases passed. API image `wisp-private-api:20261106-catalog1` was healthy; only Jornia API was recreated. The existing 15-minute ingestion schedule and unrelated containers remained. Rollback configuration: `.env.pre-catalog-20261106` on the VM. No ForeverDB import was active at this stage; see its later delivery checklist.
