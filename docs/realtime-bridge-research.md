# Wisp transport investigation — 6 November 2026

## Result

We proved a limited **desktop → local file → in-game Lua** return channel in the user's actual Apple Silicon Forever client, 1.60.1 / build 70790 / Interface 16001. This corrects the earlier blanket assumption that external text can only enter the UI after `/reload`.

Four inert, preinstalled load-on-demand inbox addons were present when WoW started. After login, our own Python writer replaced slot 1's Lua file with a fixed callback carrying hex-encoded test data. The user ran `/wispbridge inbox 1` without reloading. The real game displayed request `file-probe-1793959889670938000` and the new message written **after startup**, rather than the startup control text. Native screenshot inspection confirmed the exact request and text. Slot 2 independently displayed `file-probe-1793960007090121000`, including Unicode `åäö`; the user then reloaded. The resulting Wisp lab SavedVariables file contains both receipts.

This was test text, not an AI answer, and user-triggered delivery, not automatic polling. Each addon slot is consumed once per UI lifetime. Four slots are installed for the experiment. `/reload` resets addon loading, but is not an invisible or desirable conversational requirement. A finite pool might serve a session; it is not an unlimited socket. Repeated turns, combat, large answers, stale files, switching characters, loader failures and slot-pool cost still need measurement.

All implementation in `apps/bridge-lab` and `scripts/experiments` is ours. No researched third-party bridge was installed, executed or copied. The dependency/ownership rule is in `AGENTS.md`.

## Keep the two directions separate

| Direction | Required data | Present evidence |
| --- | --- | --- |
| Game → companion | Character GUID, current level/zone, active quests/objectives, completed quests, player's explicit question | Our journal records real events, but disk persistence still requires WoW saving. No live external event channel is proved on this Mac. |
| Companion → Lua UI | Correlated response text and source metadata | Two live, post-startup file-inbox deliveries proved without reload. Bounded, one-shot slots; not a continuous channel. |

A successful outbound bridge does not prove answers can return to the addon. An overlay is an external window, even if visually positioned over WoW. An ordinary addon has immediate event/state access inside the game; that alone does not give a desktop process that access.

## Local inspection and measurements

- Wisp's actual account journal contained six events after the initial play test, including quest 4641 completed at level 1 in Durotar / The Den. The quest submission had been recorded correctly. The empty selected directory identity caused the apparent missing history.
- The real character directory is `93/Sch-Nice`; `UnitName`/`GetRealmName` in the account journal report `Sch` and `CAMWOW EU5 Realm 1 PvE`. Name matching cannot establish identity. 0.5.2 now writes a per-character GUID/installation marker in Wisp's own file.
- After the user's new reload, that marker exists in the real character directory. The old empty folder row had already been explicitly forgotten before the update: the backup contains a forget receipt/exclusion and no directory link. The importer correctly continues to respect that exclusion. A fresh character remains the required end-to-end first-discovery test; this is not reported as a live duplicate-merge test.
- `QuestCache.log` exists. Sampled lines record IDs and cache query/classification/denial activity. They do **not** establish acceptance, completion, objective progress or a reliable journal contract. Other cache files were inventoried, not mined for credentials or rewritten.
- The user ran our inert capability probe in the actual game. Saved results: `io.open`, `os.execute`, `socket`, `require`, `loadfile`, `dofile`, `DebugLog` are nil; addon loader, chat logging, combat logging and `ConsoleExec` are functions. Presence is not permission to perform arbitrary IO. The probe calls none of these functions.
- The installed WoW executable is signed with Hardened Runtime and lacks `com.apple.security.get-task-allow`. Our own minimal task-port permission probe against the running WoW PID returned `KERN_FAILURE (5)`. It performed no memory reads/writes, debugger attachment, injection, binary modification or policy change. This establishes failure for this unprivileged attempt, not impossibility under every OS configuration.

Raw game receipts and screenshots remain local in ignored cache/WoW files, outside published source.

## Candidate methods

| Method | Outbound | Inbound | Assessment |
| --- | --- | --- | --- |
| SavedVariables | Real journal at reload/logout | Editing while the game owns its state can be overwritten; no live polling API | Durable baseline. Never concurrently rewrite the player's journal as a mailbox. |
| Predeclared load-on-demand inboxes | No outbound channel by themselves | **Proved twice in the real client** | Strongest new finding. Bound each slot to a session, GUID and request; preserve fixed code/data separation. Do not consume a slot until a response has been written. |
| Chat log | Potential game-generated text after client buffering | No corresponding arbitrary local-text input | Must measure this build's buffering, not assume a real-time file tail. Our probe did not enable or collect other players' chat. |
| Combat log | Combat observations, with its own write timing | No arbitrary replies | May enrich context, but cannot substitute for accepted quests or question text. |
| Quest/cache files | Some world/cache activity | No demonstrated external mailbox | Useful research evidence; semantically incomplete for player state. No client cache patch was attempted. |
| Addon-owned Lua buffer + read-only process access | Demonstrated by another project's Windows/Linux implementation | Read-only access cannot insert answers | Conceptually plausible outbound driver. Our Mac task-port attempt failed. No offsets, injectors, third-party scanner or system-security weakening installed. |
| Clipboard | Player copies an explicit export/question | Player pastes a response into an addon edit box | Predictable user-assisted control/fallback; not automatic live context. |
| Audio encoding / TTS | Hypothesis: addon emits generated audio symbols captured externally | Companion speaks externally; dynamic sound files require testing | Separate research option requiring capture permissions and latency/error measurements. No audio capture enabled. |
| Screen encoding | Possible side channel | Does not solve text insertion | User rejected this direction. No pixel bridge added. |
| Addon network messages | WoW-server-mediated addon communication | Requires another authorised endpoint speaking that protocol | Not a localhost socket. A second client/server relay is a separate architecture, not solved by local files. |
| Native injection / binary patch | Could change the runtime's capabilities | Could enable a custom IPC binding | Neither implemented nor proved. Different maintenance/security surface; not necessary to establish the successful file-inbox result. |

FIFO/symlink tricks do not create a new Lua write call or make WoW serialize continuously. Replacing a player-owned save path can block or break saving; this was not done. Filesystem notifications observe writes, not in-memory quest changes.

## Our next prototype

1. Keep event collection, first-use consent and durable journalling in our addon. Add active quest/objective snapshots to the model separately; accepting a quest is not completing one.
2. Retain the existing Go/SQLite authoritative importer and stable IDs. Treat transport frames as typed data with protocol version, session nonce, installation/GUID, sequence and capture time. A live snapshot must never silently masquerade as older saved state.
3. For a testable Lua chat, implement a question edit box and a bounded inbox pool. The user can explicitly copy a question to the desktop while outbound transport is unresolved; answer delivery can use the newly proved inbound channel. Do not label this fully automatic.
4. Only write fixed callback code with encoded text, never AI-generated Lua. Bind replies to the intended character/session/request, reject duplicate or stale frames, render sources as text, and keep paid requests acknowledged with no automatic replay.
5. Test outbound options separately on this exact build. Prefer a minimal first-party implementation. The Mac memory permission failure is a real gate; do not pretend a Windows reader is portable or disable protections as an installation step.
6. Measure end-to-end: question capture → context age → AI latency → fresh inbox write → in-game display, plus restart/slot exhaustion/error recovery. A prepared static test reply is not evidence for the AI leg.

## Reproduce the proven experiment

Install our lab while WoW is closed:

```sh
python3 scripts/experiments/bridge_lab.py install --root "$HOME/World of Warcraft/_forever_"
```

Start WoW and enable Wisp Bridge Lab plus its four inbox addons. `/wispbridge probe` records capability types without invoking IO/network/logging APIs. After WoW is already running, write a reply from Terminal:

```sh
python3 scripts/experiments/bridge_lab.py reply --root "$HOME/World of Warcraft/_forever_" --slot 1 --text 'Written after login'
```

Then run `/wispbridge inbox 1` **without reload first**. Compare the displayed request ID against the writer's output. `Startup control text` would be a negative result. Slot reuse is explicitly refused. Use slot 2 for an independent second result. A later `/reload` saves the lab receipts. Do not reset files while a slot is being loaded.

The lab is separate from the 0.5.2 downloadable addon and normal Wisp import reads only its own `Wisp.lua`, not the lab report. Remove the five `WispBridgeLab`/`WispBridgeInbox01..04` addon directories with WoW closed to uninstall the lab; the main Wisp addon is independent.

## Source evidence

- Blizzard-authored UI API declarations, pinned Forever UI source `eac30af597e5aaa43a60f6675ef5fdcea100375f`: [C_AddOns loader](https://github.com/Gethe/wow-ui-source/blob/eac30af597e5aaa43a60f6675ef5fdcea100375f/Interface/AddOns/Blizzard_APIDocumentationGenerated/AddOnsDocumentation.lua), [chat API restrictions](https://github.com/Gethe/wow-ui-source/blob/eac30af597e5aaa43a60f6675ef5fdcea100375f/Interface/AddOns/Blizzard_APIDocumentationGenerated/ChatInfoDocumentation.lua). Declarations are evidence for signatures, not proof of native file-cache behaviour; our live experiment supplies that proof for the two tested slots.
- [BabelChat's own memory-reader design](https://github.com/Yumash/BabelChat/blob/feadee70a85c7f1cbf8e794ba11d97cdfd774a89/docs/tech/memory-reader.md) and [project README](https://github.com/Yumash/BabelChat/blob/feadee70a85c7f1cbf8e794ba11d97cdfd774a89/README.md): an addon-owned Lua buffer, Windows/Linux external readers and an external overlay. The author reports long chat-log buffering. Those are their measurements, not Forever/Mac measurements by Wisp. No code reused.
- [Apple: debugging-tool entitlement](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.cs.debugger) and [Hardened Runtime](https://developer.apple.com/documentation/xcode/configuring-the-hardened-runtime): task-port access is constrained; a process belonging to the same user is not a blanket entitlement to read it. Corroborated by the local permission failure above.
- [Jocys' own implementation description](https://www.jocys.com/TTS/#How_It_Works): its game-to-monitor channel uses pixels and provides no return path. This was investigated, not adopted.

Technical feasibility and permission to distribute/run a transport are separate questions. This note records mechanisms and evidence, not a claim that Blizzard permits a process reader or that the experiment is production-ready.

## Uppföljning: uppmätt chatlogg 6 november

[Det genomförda A/B-testet](chatlog-test-2026-11-06.md) gav åtta mottagna självviskningar men ingen diskleverans före klientavslut. Tre direkta av/på-växlingar och en av-period på faktiskt 270 ms ändrade inte det. Alla åtta markörer observerades först vid normalt avslut. Det tidigare öppna loggspåret har därmed ett konkret negativt realtidsresultat för den prövade byggversionen och trafikvolymen, med en lyckad positivkontroll. Eventuella volymtrösklar är inte testade.

## Uppföljning: offlineanalys av klientfiler

[Den lokala filanalysen](local-client-file-audit-2026-11-06.md), genomförd utan internet, hittade den exakta native-signaturen `C_Log.LogMessage(message)` samt utvecklarloggens fil och konfigurationssträngar i den installerade klienten. Nästa föreslagna utgående test är en enda uttryckligt märkt native loggpost. API-tillgänglighet, rättigheter och diskfördröjning är ännu inte prövade. Inga klientinställningar eller binärer ändrades i analysen.
