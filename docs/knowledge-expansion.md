# Expanding the Forever graph

The installed reviewed slice now contains **49 entities, 67 relationships and 7 source-review records**. The original synthetic fixture remains separate. The latest package adds 20 named Ashenvale locations and five neighboring Kalimdor zones, including Mystral Lake, Iris Lake, The Talondeep Path, Darkshore and Felwood. Named geography and adjacency are adapted from the same explicitly shared Classic/Forever wiki article; coordinates, current faction control, quest totals and patch-specific mechanics are not inferred.

Coverage is deliberately incomplete. `GET /api/knowledge/coverage` returns `complete: false`, installed counts by kind, scope and unverified domains. It does not fabricate a percentage of the entire game. The full WoW Forever universe has not been imported or verified.

New immutable reviewed JSON packages are discovered by the seed loader in filename order. Cross-package dependencies must precede dependent packages; the Ashenvale core precedes its geography extension. Whole-package checksums reject in-place changes to imported revisions. Validate with `make knowledge-check` before bootstrapping; it checks identities, references, review metadata and the current retriever's capacity. A new review revision plus an explicit entity/edge migration is required for corrections to existing entities.

The request path now uses targeted PostgreSQL retrieval with bounded neighborhoods and evidence/expiry/build filtering. It no longer relies on the first 100 entities of the whole scope. The official API import is a separate build-scoped metadata layer; see [ingestion](knowledge-ingestion.md). The legacy whole-graph debug reader is still bounded and is not a corpus export.

## Why a full QuestieDB import was not performed

The upstream [licensing document](https://raw.githubusercontent.com/Questie/QuestieDB/master/LICENSE.md) explicitly describes unresolved historic contributor permissions and an all-rights-reserved fallback for uncovered content. File/directory terms may override root terms. It is not sufficient to see GPLv3 in a repository or on an addon listing and assume every historic data row is cleared.

Its [Forever documentation](https://raw.githubusercontent.com/Questie/QuestieDB/master/docs/forever.md) also states that the transformed baseline and generated deltas are not complete new Forever content and leave gameplay restrictions/objectives and other behavior unresolved. [Provenance](https://raw.githubusercontent.com/Questie/QuestieDB/master/PROVENANCE.md) traces the initial data to Questie. No QuestieDB data files, game caches, client maps or executables were downloaded or executed for this prototype.

The source-review inventory is [pipelines/ingestion/source-review.json](../pipelines/ingestion/source-review.json). Expanding safely requires a reusable source or a field/file-specific rights review, a pinned revision, explicit Forever applicability, validation of relationships, and regression questions. Public knowledge imports must never include private user journal events.

See [data attribution](licenses/ashenvale-data.md), [legal review](architecture-and-legal-review.md) and [the current source comparison](ashenvale-comparison.json). Source-checked is community evidence, not a Blizzard endorsement or an in-game observation.

## Second Ashenvale slice

The quest-index package adds six selected quests and three NPCs, taking reviewed coverage to 49 entities and 67 relationships across seven source reviews. It supports the recorded order Bathran's Hair → Orendil's Cure → Elune's Tear → The Ruins of Stardust → Fallen Sky Lake, their givers, and selected NPC locations. “Next” means the next recorded chain entry, not a promise that a player is eligible. Current level requirements, rewards, coordinates and exact client availability remain unknown.

A practical source conflict was handled explicitly: the retail Ashenvale storyline has a different Bathran's Hair giver and intervening quests, while Classic pages and the shared Classic/Forever index contain the older sequence. The package uses the shared index, reviews the narrow Classic recipient claim separately, and imports none of the retail chain. This is community-source review, not in-game verification or complete Forever coverage. New evaluation cases include both answerable relationships and unknown rewards/eligibility/continuation questions.
