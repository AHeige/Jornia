# Ashenvale: source review and comparison

Review date: 2026-11-05. Scope: a small useful geography/NPC/quest slice, not a complete zone import.

## What is now real knowledge

Default `forever` requests now use 15 curated entities and 17 relationships from four attributed Warcraft Wiki articles. The original six-entity fixture remains isolated under `fixture-ashenvale-v1` for regression testing. Production-facing answers no longer fall back to synthetic descriptions.

The selected articles explicitly describe their subject as remaining in Classic and Forever. Their claims were reviewed in web search-index excerpts. Direct wiki fetches returned HTTP 403, so there is no retrieved page revision ID or complete page snapshot. A local dated claim packet and its SHA-256 fingerprint are preserved in each source's provenance. An immutable dataset checksum also protects repeat imports.

`source_checked` means checked against these published community statements. It does **not** mean tested in the Forever client, confirmed by Blizzard, or guaranteed valid for a specific launch build. The API returns this warning, source URLs, review method, review date, attribution, and licence. `verified` is reserved for stronger verification and is not assigned to these records.

## Included claims and their evidence

| Facts included | Source |
| --- | --- |
| Ashenvale is in Kalimdor; Astranaar, Splintertree Post and The Zoram Strand belong to it; additional hubs include Forest Song, Zoram'gar Outpost and Maestra's Post; Blackfathom Deeps is at The Zoram Strand | [Ashenvale (Classic), shared Classic/Forever article](https://warcraft.wiki.gg/wiki/Ashenvale_(Classic)) |
| Raene Wolfrunner is at the inn in Astranaar, and starts the Raene's Cleansing chain | [Raene's Cleansing quest chain](https://warcraft.wiki.gg/wiki/Raene%27s_Cleansing_quest_chain) |
| The opening Raene's Cleansing quest starts with Raene, is categorized in Ashenvale, and is listed at level 19 with minimum level 18 | [Raene's Cleansing](https://warcraft.wiki.gg/wiki/Raene%27s_Cleansing) |
| The Barrens lead-in of The Ashenvale Hunt starts with Jorn Skyseer and sends the player to Senani Thunderheart at Splintertree Post | [The Ashenvale Hunt (The Barrens)](https://warcraft.wiki.gg/wiki/The_Ashenvale_Hunt_(The_Barrens)) |

Names of quest-chain outcomes and narrative revelations are intentionally omitted. Quest category uses `quest_in`; physical places use `located_in`. The Barrens lead-in is **not** misclassified as an Ashenvale-category quest. The places query follows nested locations, so it can include Blackfathom Deeps without listing NPCs or quests as places. Quest results explicitly say the list is incomplete.

The dataset is a CC BY-SA 4.0 adaptation with source and contributor-history links. See [attribution and licence](licenses/ashenvale-data.md). No Wowhead bulk export, artwork, maps, complete quest text, or reward table was imported.

## Comparison with the previous data

Five entities are shared: Kalimdor, Ashenvale, Astranaar, Splintertree Post and The Zoram Strand. All four real-world location relationships in the fixture are supported by the community review; no contradiction was established for those four claims. The new dataset adds ten entities and thirteen relationships. The synthetic Hidden Story and its edge do not enter `forever`.

The important change is evidence and useful coverage, rather than a claim that the old four geographic facts were false. Ashenvale's old description merely explained the development fixture; its new description describes the actual forest zone. New questions can locate Forest Song, Zoram'gar Outpost, Blackfathom Deeps and Raene, or ask who offers Raene's Cleansing.

`scripts/compare_ashenvale.py` asks the same 14 questions of both versions, without inference, and writes [the machine-readable comparison](ashenvale-comparison.json). This compares the old stored graph with the curated graph. It does not claim to inspect or validate a language model's training data.

## Conflicts and omitted claims

- Community sites disagree on a single recommended zone level range. The [shared wiki article](https://warcraft.wiki.gg/wiki/Ashenvale_(Classic)) lists 18–30, while [Lore Forever's review](https://loreforeverwow.com/guides/wow-forever-ashenvale) reports a different site's 20–32 range. No universal Forever level recommendation is served. A quest's own listed minimum is a separate fact.
- Several guides mix original-era places with later faction control. Silverwind Refuge's current ownership is omitted rather than inferred from a retail page or a guide's Forever label.
- Exact flight connections, coordinates, NPC combat levels, quest rewards, new quest totals, and spawn behavior were not imported. Shared pages contain content from multiple versions, which the applicability banner alone cannot resolve.
- The [wiki identifies Relathor Moonsong](https://warcraft.wiki.gg/wiki/Relathor_Moonsong) as Classic-only and absent from Forever. He is not included; the API returns unknown for his location. Unknown does not assert that an NPC does not exist.
- The corpus is community-sourced and partly inherited from Classic. It is not an exhaustive audit of Forever changes. A later client observation or authoritative update can revise a specific claim with a new source review.

## Reproduce

```sh
make eval
python3 scripts/compare_ashenvale.py
make eval-live
```

The first command runs both fixture and reviewed-data acceptance suites. The comparison does not spend inference credits. The live check makes one Berget request about Raene's location using the reviewed graph and checks the expected location, evidence and review metadata.

The next verification step is to check this small set against the exact Forever client build, recording build number, date and observations. Those records can then justify a stronger validation status on the corresponding claims.
