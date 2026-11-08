# Completed quests and default character context — 8 November 2026

## What the real regression showed

The user completed and handed in **The Damned (376)** on Cone Thrower. The game displayed “The Damned completed”, the reward, and acceptance of Marla's Last Wish. Only the chat and read-only state were controlled for this slice; no gameplay automation, reload, logout or game restart was used.

Repeating **“How many Scavenger Paws do I still need?”** with the old context format produced **“No active quests require Scavenger Paws.”** It did not repeat the earlier “1 more” count, but it also did not explicitly recognize the turn-in. The approved snapshot contained active quests without a separate completion fact. Absence from an active log is not proof of completion.

The user also asked to make current-character context the default. They reported missing Brill information, which was checked independently rather than attributed to the transport.

## Changes

### Explicit, private completion evidence

- `State.lua` reads up to eight distinct recent `quest_completed` events from **this character's** in-memory Jornia journal. Those events originate in `QUEST_TURNED_IN`; removal/progress/ready-to-turn-in events do not qualify. It does not enumerate another character's history.
- Active IDs are excluded, so a currently active repeatable run is not labelled handed in. If available and true, the public `C_QuestLog.IsQuestFlaggedCompleted` flag confirms current client status; otherwise the actual recorded turn-in event remains the evidence, with its original time.
- Quest title/objective labels come from the completion event, earlier same-character observations of the same ID, or the public quest-title API. These labels link a question about paws/wings to its completed quest. Historical objective labels are not current item counts.
- The compact wire adds optional `d` rows under the existing character metadata. Legacy packets remain valid. Go receives a bounded `completed_quests` field with ID, title, observation time, confirmation kind and objective labels.
- Parser and API validation reject duplicate IDs, active/completed conflicts, unknown confirmation kinds, future observations and oversized text/arrays. Nothing is written into the shared world database by this feature.
- The Go-owned context continues to expose `state:snapshot` as supporting evidence. The model instruction explicitly distinguishes handed-in quests, ready-to-turn-in quests and missing quests, and forbids asking for more items for an already completed run.
- Wire budget remains 480 target body bytes. At most two recent completion facts are packed. A question matching a completed quest/title/objective prioritizes that evidence; optional gear, unrelated completion records, active rows and text may be omitted with the existing truncated marker. Unrelated completion metadata is dropped first when it would displace current quest guidance. A bounded snapshot is not a full completion-history export.

If journal recording was off and no completion candidate was recorded, this implementation does not magically discover all completed quests. Missing completion evidence remains unknown. It must not infer completion merely from the user's phrasing or a disappearing quest.

### Context on by default, with an opt-out

- Current character context is checked by default and stays checked after a successful send.
- The visible checkbox remains available. Explicit off/on preferences are scoped per character and stored as `share_current_context` in that character's existing local addon record; SavedVariables persists it when WoW normally saves.
- The view preserves the choice across modes and reopening. Other characters default independently; an absent preference is not treated as an opt-out.
- Sending still requires Enter/click. The snapshot is read at send time, not continuously uploaded. Disabling current context omits the approved live snapshot.
- Input clears immediately after acceptance; errors restore the submitted draft. The real screenshots in this slice verified empty input during/after sends and the context checkbox remaining checked.

This is the user's explicit product choice on 8 November 2026, replacing the previous off-by-default/per-send reset. Default-on does not authorize uploading a whole journal, screenshots or other players' data.

## Real in-game results after the change

The existing Lua window and transcript were preserved through an unused, syntax-checked LoD slot. The local Go companion and reader were restarted gracefully to load the new decoder, without restarting the game. The API was upgraded independently.

| Question | Observed native answer | Request | QR decode | Post-recognition to delivery |
| --- | --- | --- | ---: | ---: |
| How many Scavenger Paws do I still need? | “You already turned in The Damned quest. No more Scavenger Paws are needed for that completed run.” | `941663751794166386000001` | 24.9 ms | 13,020.7 ms |
| Hur många Duskbat Wings behöver jag fortfarande? | “Du har redan lämnat in The Damned. Du behöver inte samla fler Duskbat Wings för den uppdraget.” | `941663751794166412000002` | 21.4 ms | 9,724.6 ms |

Both used Berget and displayed one supporting source. The second question needed no checkbox click: current context remained enabled. The Swedish answer's minor grammatical error is preserved as observed. These are two real samples, not a general reliability or latency benchmark. The existing reusable return channel still accounts for several seconds of the wait.

## Brill coverage finding

A read-only query found **zero exact Brill entity names** in the stored Forever corpus. Only three current-build ForeverDB records mentioned the token Brill: **Forge Documents**, **Lift Seal**, and **The Deed to Brill**. Names such as “Brilliant …” are not Brill matches and were not counted as useful town coverage.

Those mentions do not supply a sourced town guide, level recommendation, travel route or proof that every referenced ability is available in Forever. No Brill facts were invented or imported under a different game version. The current coverage gap remains: prioritize settlements/quest hubs and source their location, travel connections, trainers and available quests. Adding more item/spell rows alone will not solve it.

## Deployment and verification

- VM API: `wisp-private-api:20261108-completed`, based on the previous guidance image with only the new first-party binary replaced. No migration needed. `FOREVERDB_ENABLED=true`, `WISP_QUESTLOG_ONLY=false` remain selected.
- API health passed. Every other pre-existing container retained its ID; database/gateway/admin and unrelated projects were not recreated.
- Private backups: `.env.before-completed-20261108`, `containers.before-completed-20261108`. Previous API image retained for rollback.
- Mac binary: `~/Applications/Jornia/jornia`; previous binary retained as `jornia.before-completed-20261108`. Existing SQLite and session configuration retained.
- Installed addon files backed up under `.cache/addon-ui-backup-20261108-203143`; later source installation preserves player files. The temporary slot file was restored after loading.
- `go test ./...` passed, including real Lua execution for completion/removal distinction, active-repeatable exclusion, and context defaults/opt-out isolation.
- Companion Python protocol tests: ten passed, including completion decoding, active-ID conflicts and false completion-proof rejection.
- Existing QR and return checks remain in force. Instant chat closing/background delivery and Windows native transport remain unresolved; this slice does not claim to fix them.
