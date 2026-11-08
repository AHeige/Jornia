# Personal context, settings and live journey

## Delivered

- [x] Replace the permanent Connect AI control with Settings.
- [x] Move the retained per-character context choice into Settings; preserve it across sends, modes and reopening.
- [x] Show reader/connection status and keep invite entry available inside Settings.
- [x] Refresh the open journey from more game events without opening a window or rewriting a chat draft.
- [x] Carry the selected character's observed name, race and class instead of asking a model to interpret numeric IDs.
- [x] Add bounded live journey excerpts for recognized journey questions.
- [x] Add a bounded, source-labelled class-training lookup.
- [x] Verify identity and spell questions through the real native Mac transport.
- [ ] Observe a new user-performed quest progression/level-up with the updated journey already open. The user offered to do this later; no gameplay automation was used.
- [ ] Add comprehensive spell effects, supported leveling-zone guidance, and lore coverage.

## Settings and consent

Settings contains the context checkbox, connection feedback and private invite
entry. There is no permanent Connect AI label in the main conversation. The
connection wording reports the last confirmed response; reader availability is
checked separately. This is not a continuous paid model health check.

With context enabled, Enter captures name, race/class display names, level,
location and bounded quest/progress data. The selected display identity comes
from the player's own addon APIs/current character, not another player's unit,
a model's guess about numeric IDs or a dated Armory profile. Turning context off
omits this snapshot. The preference is retained for the character.

Recognized journey questions also include at most three recorded Lua-memory
moments, with timestamps, locations and levels, marked explicitly partial. Notes
are excluded. This does not upload the entire SQLite journal or conversation.
Current journey routing recognizes a small set of English/Swedish phrases;
other questions/languages still use ordinary inference with observed identity.
That routing is not universal multilingual intent understanding.

The additional identity fields are optional protocol extensions. Older packets
remain valid. The QR byte budget is unchanged. For a journey question, optional
quest/equipment fields can be omitted with explicit missing/truncated markers to
make room; omission never means an empty or completed quest log.

## Live journey

The view subscribes to quest acceptance/removal/log changes, turn-ins, level and
zone changes, equipment, skills, death and resurrection. Updates are coalesced
at 350 ms, after the state's normal 250 ms observation update. They update an
already-open view; they do not open windows, send AI requests or manipulate the
character. The update path preserves the input, pending operation and context
preference. Labels distinguish accepted/progress/turned-in/removed quests and
other recorded changes rather than presenting every event as a generic moment.

Lua memory is current during play. SavedVariables and subsequent SQLite imports
still depend on the game saving files; this change does not make them a live
filesystem channel. A quest disappearing remains separate from a real turn-in.

## What the harness actually does

The production harness is owned Go code, not a LangChain agent loop. It registers
bounded tools, validates their calls, records tool status/timing, and imposes a
six-call / 48 KB / ten-second tool-workflow limit. Inference happens afterward.
The ordinary plan is still fixed:

1. `read_game_context`: validate the approved snapshot.
2. `retrieve_world_evidence`: choose bounded retrieval for the question.
3. `find_active_quest_options`: derive choices and remaining counts from observed quests.
4. `validate_navigation_options`: constrain any offered navigation to a fresh client-supported quest.

World retrieval uses current-build quest IDs for quest context, an explicit
class-training selection for recognized spell questions, or indexed lexical
search and bounded graph expansion for ordinary world questions. Inference
receives at most a 40 KB request, with a smaller world-fact budget, allowed
citation IDs, current observations and applicable history. Returned references
and navigation IDs are checked. These checks do not prove every natural-language
claim follows from its citations.

Basic English/Swedish identity questions have a direct factual path using the
observed identity; they do not need a model or world search. Other languages and
phrases retain inference with the same explicit identity fields.

The harness does not yet autonomously decompose a question, select arbitrary
tools, repeatedly investigate missing information, search the web, or retrieve
all previous conversation. Its current intent aliases are heuristics. More
indexed rows alone do not supply those capabilities.

## Class training and real coverage gaps

At audit time the current imported build had nine class details, 3,992 quest
details, 322 spell-list documents and no individual spell-detail documents.
The Mage detail contained 62 ability records, including historical/passive
records that must not all become recommendations. The zone catalog lacks
usable level ranges, and articles/posts provide only limited lore material.
Catalog counts are not field completeness or verified live availability.

`ClassOptions` reads the already cached class detail through its numeric class
identity and original source. It keeps trainer-labelled, rank-labelled entries
at/below the observed level plus the next listed training level, bounded to 12
current and four upcoming options. Unranked entries are conservatively omitted
because some are effects such as Chilled; legitimate unranked abilities may also
be absent. It preserves origin labels and excludes explicit historical or
unavailable rows. No extra provider call is made when a player asks.

This is a partial training reference, not the character's learned spellbook,
a proven rotation, a guarantee that a current trainer offers the spell, or a
complete class guide. Lore and leveling-region facts still require enrichment.

## Verification and deployment

All Go tests passed with isolated PostgreSQL integration tests; `go vet ./...`
passed. The Python bridge protocol suite passed all 11 tests. Tests cover
identity from current fields, private-note exclusion, future-history rejection,
metadata bounds, level/rank selection, historical effects, provider pause,
source expiry/build isolation, context preference and draft-preserving updates.

The UI was activated using installed unused LoD slots 095 and 096, then their
original files were restored. Current conversation, draft, position and size
were captured/restored through public widget APIs. No game reload, logout,
restart, movement, combat, loot or quest completion was used. The Mac desktop
and reader were updated; only the VM API container was recreated.

The first desktop restart was blocked by its exclusive old session file. The
native error returned the question as a draft without automatic resend. After
confirming the old process had stopped, the session file was backed up and a new
owner-only session created. Local connection checking then returned connected.

Final VM image: `wisp-private-api:20261108-personal-context-2`.

Observed native checks:

| Request | Result | Decode / total |
|---|---|---|
| `941745041794174600000002`, “Vem är jag?” | “Cone Thrower · Undead Mage · level 3. Du är i Tirisfal Glades.” Direct `character_context` route, one observation source | 18.4 ms / 6,390.8 ms |
| `941745041794174704000003`, “Hur hamnade jag här?” | Answer said no earlier route/origin to the current location was established; it did not invent one. This is not proof of a complete or useful journey explanation | 13.9 ms / 11,708.3 ms |
| `941745041794174897000004`, “Vilka spells kan jag lära mig på min level?” | Listed Fireball and Frost Armor, with visible ForeverDB attribution | 16.0 ms / 8,250.5 ms |

Settings was visually checked, including context off, closing/reopening, and
restoring on. A read-only journal check returned consent accepted, recording
true and three available recent moments. The journey view showed quest progress,
location and equipment entries in the uninterrupted session. A new user-performed
quest/level transition remains a separate acceptance check.
