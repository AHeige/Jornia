# Ashenvale curated community data: attribution and licence

`internal/storage/datasets/ashenvale-forever-v1.json` and `internal/storage/datasets/ashenvale-geography-v1.json` form a selected adaptation of Warcraft Wiki material, attributed to **Warcraft Wiki contributors**, under [Creative Commons Attribution-ShareAlike 4.0 International](https://creativecommons.org/licenses/by-sa/4.0/).

The [wiki copyright policy](https://warcraft.wiki.gg/wiki/Warcraft_Wiki:Copyrights) licenses material it can lawfully license under CC BY-SA 4.0 and separately identifies Blizzard ownership of Warcraft material. No game artwork, audio, maps, quest dialogue, or bulk database export is included here. This notice does not claim ownership or grant rights to Blizzard's trademarks or game assets, and it does not set the application's code licence.

Source articles and contributor histories:

- [Ashenvale (Classic)](https://warcraft.wiki.gg/wiki/Ashenvale_(Classic)) · [history](https://warcraft.wiki.gg/wiki/Ashenvale_(Classic)?action=history)
- [Raene's Cleansing quest chain](https://warcraft.wiki.gg/wiki/Raene%27s_Cleansing_quest_chain) · [history](https://warcraft.wiki.gg/wiki/Raene%27s_Cleansing_quest_chain?action=history)
- [Raene's Cleansing](https://warcraft.wiki.gg/wiki/Raene%27s_Cleansing) · [history](https://warcraft.wiki.gg/wiki/Raene%27s_Cleansing?action=history)
- [The Ashenvale Hunt (The Barrens)](https://warcraft.wiki.gg/wiki/The_Ashenvale_Hunt_(The_Barrens)) · [history](https://warcraft.wiki.gg/wiki/The_Ashenvale_Hunt_(The_Barrens)?action=history)

Changes: selected facts were condensed into short descriptions and graph relationships. Article narratives, dialogue, images, rewards, and complete tables were omitted. The original synthetic fixture remains separate under its existing CC0 notice.

Reviewed on 2026-11-05 using web search-index excerpts, including applicability banners explicitly mentioning Classic and Forever. Direct wiki-page retrieval returned HTTP 403. Upstream MediaWiki revision IDs were not available; the dataset records that limitation instead of inventing them. `review_sha256` fingerprints the locally reviewed claim packet, not the upstream page. The dataset also receives an immutable whole-file checksum on import.

Redistributors of this adaptation must preserve attribution, identify their changes, link the licence, and meet its share-alike conditions. See the linked licence for the complete terms.


Legal review note (2026-11-05): this notice documents attribution and the wiki-adaptation licence; it is not a clearance of all underlying Blizzard rights, a bulk-database redistribution approval, or permission for game-client extraction. See [architecture and legal/policy review](../architecture-and-legal-review.md). The data's `source_checked` status describes factual source review only.

## Quest chain expansion (2026-11-05)

`ashenvale-quests-v1.json` adapts selected factual entries from [Ashenvale quests](https://warcraft.wiki.gg/wiki/Ashenvale_quests?oldid=6550117) by Warcraft Wiki contributors, plus the recipient of [Orendil's Cure (Classic)](https://warcraft.wiki.gg/wiki/Orendil%27s_Cure_(Classic)). This selected adaptation is offered under CC BY-SA 4.0 to the extent the wiki contributors hold those rights. It does not license Blizzard assets. Changes: short original descriptions, graph labels and relationships; no dialogue, images, maps, rewards or copied quest records. Source review is based on indexed excerpts because direct requests returned HTTP 403. The shared index indicates Classic/Forever applicability; the Classic recipient page alone is not proof of the exact Forever client behavior. Retail storyline entries and unconfirmed level/reward values were excluded. Per-claim attribution and review metadata ship with the package and API sources.
