# Richer quest guidance — 8 November 2026

**Your Journey in Azeroth. A little guidance for the road ahead.**

## Delivered behavior

The addon reads the selected quest's description and objective summary, plus any quest-specific waypoint text exposed by the game. It does not change quest selection or scrape unrelated quests. If selected-quest APIs are missing or fail, description remains unavailable; current objective counters still work.

Open the quest you want help with, open `/jornia compact`, approve current character context and ask where to go or what to do next. Each question captures current state again. Saved journal checkpoints remain dated history.

The private snapshot now supports `description` (1,024 UTF-8 bytes), `objective_text` (512), `waypoint_text` (240), and `text_truncated`. The receiver remains compatible with the earlier 640-byte ceiling. Rich snapshots now target 480 bytes to reduce OCR exposure. It prioritizes the selected/named quest, removes optional equipment/professions and extra quests if needed, then shortens narrative before objective/waypoint instructions. A partial narrative keeps its end because concrete instructions often follow introductory dialogue; it is explicitly marked partial. This is a limited excerpt, not semantic summarization or a complete route.

## Verified follow-ups

Migration `010_verified_quest_followups.sql` adds a small indexed review register. Runtime retrieval requires:

- Exact game version, numeric client build and an observed current quest ID.
- A source explicitly marked `verified`, with matching `provenance.effective_version` and no expired lifetime.
- A non-ForeverDB source. Pausing ForeverDB remains effective.

The model receives at most 16 reviewed relationships and their source metadata. A recorded next quest is not proof of current eligibility, an unlock or an optimal route. These records never create a waypoint for an unaccepted quest; navigation still needs a current, observed in-game quest marker and the player's click.

An operator must independently review the source and build before adding a register entry. `source_checked` wiki/API records and user event order are insufficient for this register. No production follow-up records have been manufactured to make the feature look complete. When no reviewed record exists, Jornia must explain the current objective and acknowledge that later continuation is unconfirmed. A lookup failure has a 500 ms deadline and falls back to current quest context.

## Validation

- Lua execution: selected quest text stays tied to its quest ID; another selection cannot inherit it.
- Lua wire: selected quest survives a full packet; Unicode remains valid; directions and truncation markers survive the budget.
- Python: legacy packets remain accepted; new text bounds are enforced.
- PostgreSQL isolated-schema test: exact build/version/quest, verified status, expiration and provider exclusion.
- Harness: observed directions and source-backed continuation reach inference; a failed continuation store does not prevent current quest help.

During the first live attempt, Vision produced a Cyrillic `с` in hexadecimal text. The strict parser rejected it before inference. Parsing normalizes visually identical hexadecimal homoglyphs and still requires the original checksum. New `WISPBEGIN4` packets use a 32-bit Adler checksum; legacy v3 packets remain readable. Alternative-candidate recovery is limited to v4 packets. Larger cards also exposed recognition errors; the reader can examine at most 256 combinations of Vision's own top-three line candidates, accepting only one unique checksum-valid, schema-valid packet. The transfer panel now uses larger text and groups four hexadecimal digits, avoiding long repeated digit runs. It still requires matching observations before inference and never repairs arbitrary bytes or resends a paid request automatically. This is bounded recovery, not proof of general OCR reliability.

Live results are recorded below after visual confirmation; unit tests alone do not establish an in-game success.

## API evidence

The selected quest text and waypoint mechanisms were checked against Blizzard UI code and generated API documentation mirrored at [QuestInfo.lua](https://github.com/Gethe/wow-ui-source/blob/live/Interface/AddOns/Blizzard_UIPanels_Game/Mainline/QuestInfo.lua) and [QuestLogDocumentation.lua](https://github.com/Gethe/wow-ui-source/blob/live/Interface/AddOns/Blizzard_APIDocumentationGenerated/QuestLogDocumentation.lua). These moving references describe API use, not guaranteed Forever support; local runtime checks remain necessary. No third-party implementation was copied or installed.

The native reader explicitly requests Vision Fast (`--fast`); historical experiments retain the helper's accurate default. The experimental font change was not retained. The tested source uses the standard game font with larger, grouped text; the live result must be checked before calling the new layout reliable. A malformed temporary lab package was corrected and syntax checks were added to the operator test procedure.

## Live result

The larger accurate-OCR attempts were rejected before inference. After switching the local Vision request to Fast and canonicalizing only fixed-alphabet numeric/hexadecimal glyphs and header spacing, a fresh native question succeeded in the same uninterrupted session. The checksum and snapshot freshness checks remained mandatory. No stale captured packet was sent to inference; an old private crop was used only to diagnose recognition, with a mocked clock in that offline diagnostic.

On Cone Thrower, “Where for The Damned?” returned guidance to the south in Tirisfal Glades, with 2 Scavenger Paws and 4 Duskbat Wings remaining. The southward direction was visible in the selected quest description and the counters matched the live quest log. One measured success: OCR 69.0 ms, return 8,640.7 ms across 18 blocks, post-recognition flow 10,204.0 ms. These are individual observations, not a reliability or latency benchmark.

The production follow-up register currently has zero reviewed entries. It is ready to accept independently verified, build-scoped relationships; this delivery does not claim to know unseen quest chains. The real-model synthetic checks passed observed-direction guidance and an explicit acknowledgement that an unsupported follow-up was unconfirmed. The VM remained healthy and unrelated container IDs did not change.

A second native request returned through the channel in 8,539.8 ms (OCR 69.8 ms, return 7,613.4 ms). Its model response failed citation validation and was withheld. The model citation instructions were tightened and the fallback now explains a validation failure rather than implying that world knowledge alone is missing. This is a response-quality failure, not a transport delivery success for the requested fact.

The final repeated native turn-in question produced a valid uncertainty response: the available description did not name a return recipient. That is distinct from the earlier citation-validation failure. Its OCR measured 71.0 ms and its post-recognition flow 7,014.2 ms, with a 6,366.0 ms return. A synthetic explicit-return case passed against the live model.

Structured local character identity is not included in the central snapshot schema. Approved game-provided quest text is sent as context and can itself contain names inserted by the game; this slice does not claim to anonymize arbitrary narrative text. It is not added to shared world knowledge.

## Product acceptance correction: transport must remain invisible

On 8 November the user explicitly rejected any visible transfer code in normal play. The live exchanges above prove a technical channel, not an acceptable finished chat experience.

The target is an ordinary in-game conversation: submit with Enter, retain the message in the conversation, show a brief inline waiting status, then show the reply. Keep input acknowledgement immediate and stable; a static status is sufficient while waiting and must not imply a confirmed answer. Technical details belong in an explicitly opened diagnostic mode.

Acceptance checks:

- No hexadecimal card, checksum, transport marker or extra technical window appears on submit.
- The question remains readable while waiting.
- Failure preserves the draft and explains the outcome without exposing the transport.
- Timeout, helper interruption and recovery do not reveal transfer data or replay a paid request.
- Delivery still occurs during an uninterrupted game session.

The current implementation does not meet these checks: its reader depends on the rendered encoded card. Simply hiding or making that card transparent breaks the read path. No invisible replacement has yet been implemented or verified; the current experiment is preserved until a replacement actually works.
