# Automatic local adventure journal

The initial 0.3.0 journal gives Wisp a memory for each character. This feature does not change the central world graph or Berget calls. SQLite stores original observations and a rebuildable local graph; the player needs no separate graph server. The later 0.7.2 state extension is described below.

## First use

1. On first login the addon asks before enabling automatic recording. Consent is saved for the installation; subsequent characters start recording on login. Pause a character with `/wisp stop` and resume with `/wisp start`.
2. In the companion's **Automatic adventure journal** section, select the absolute path to the client folder containing `WTF`. Approve local file access and choose **Enable and import journals**.
3. The companion imports existing Wisp journals for all characters in those files, then checks for new saves every three seconds while running. The setting resumes automatically after restarting the companion.
4. Select a character and open **Your journey**, or ask **Show my journey**. The timestamp indicates when the game observation occurred. The latest file import is a different timestamp and is displayed separately.

This does not install Windows-login autostart. The companion must run to read new files. A character without Wisp data must first log in with the addon. Ordinary WoW settings cannot reconstruct an earlier journey. Other addons require separate, explicitly scoped adapters and are not imported here.

## Recorded observations

The initial version records recording-session start/end, zone/subzone, time, level, level-ups, turned-in quest IDs, personal notes, character identity/name/realm and client version/build. A recording session begins at an accepted start/login/UI reload; it does not guarantee a continuous real play session. XP, inventory, equipment, talents, active quests and chat were not collected in the initial version.

The addon observes events while playing, but SavedVariables normally reaches disk on `/reload` or logout. This mechanism does not provide continuous live positions to the companion. It forces no reload, reads no game-process memory and does not control gameplay. At the initial implementation stage, behavior and persistence still required verification in the user's actual Forever client; Lua simulation was not a substitute.

## Local graph and performance

The graph follows character → recording session/observation group → event → place/quest. Older events without session IDs are grouped by UTC date as imported observations, not proof of a real session. Every node and edge is scoped to the local character identity. Place and quest nodes are deduplicated. New events and graph projections are written atomically with the import receipt. The graph can be rebuilt from original events after an upgrade.

Projection and journey summaries need no LLM. Events are recorded on change, not per frame. Unchanged file sizes/timestamps avoid content reads; changed content receives a checksum. Stable event IDs prevent duplicate saves from creating duplicate events. Conflicting content under the same ID preserves earlier facts and returns an error rather than silently overwriting them.

The addon retains at most 2,000 events per character; older events can fall out before import. A manually copied export contains the latest 100. Automatic import splits larger journals into batches of at most 100, with smaller adaptive batches for detailed state/progress. Limits: 8 MiB per file, 100 characters per file, 100 journal files and 20,000 traversed directory entries per check, plus bounded directory depth. Exceeding a limit gives a visible error rather than a completeness claim. Imported local observations remain until deleted. The timeline shows the latest 50; export includes the complete local event and conversation history.

## File access, pause and deletion

The reader searches only for `Wisp.lua` in `SavedVariables` under the selected client's `WTF/Account`. It does not read other file contents or execute Lua from disk: only a bounded data literal `WispDB = {...}` is accepted. Code, other globals, excessive size/depth and incomplete files are rejected. Incomplete files are retried on later checks; saved history remains. Source files are unchanged.

**Pause automatic import** stops companion reads and preserves local history. `/wisp stop` stops addon recording for the current character. The interface explains these as separate activities.

**Forget selected character** also deletes the local graph and blocks automatic reimport of that character. The block uses a local hash of its identity; this is not an anonymity guarantee. The original addon file remains. An explicit later manual import lifts the block. Existing exports/backups are unaffected.

Journals, graphs and conversations remain local. World questions send the question and game version to the server. Selected historical excerpts can accompany an AI request after review. The experimental in-game bridge can include a fresh snapshot through a separate per-question choice; this is not a continuous live feed. Local storage does not imply encryption.

## Rich state in local prototype 0.7.2

The addon reads current state in Lua memory and saves a dated per-character checkpoint: level/XP, zone/subzone, class/race IDs, up to 50 active quests with bounded objectives/counters, equipped item IDs and professions where supported. Journey events include quest changes, equipment and death/resurrection. Paused recording stops new persistent events/checkpoints. Chat and other players' data are not collected.

SQLite separates checkpoints from historical events. An older file cannot overwrite a newer checkpoint. Reimport retains each event once. Empty Lua tables are normalized only for schema-defined lists; SavedVariables are never executed. Unavailable APIs and truncated lists are explicit. Complete history is not guaranteed if addon-buffer or file limits are exceeded.

For an approved question the addon refreshes state at send time. Its bounded question snapshot can carry up to four quests, objective counters, gear IDs and professions. Missing fields and truncation travel with it. Reading SavedVariables alone must never create context labeled live.
