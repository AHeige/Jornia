# Native Lua logging on 6 November 2026

Status: **completed**, including a repeat watching the correct target and verified restoration. This test used no internet access or GitHub synchronization.

## Scope

Check only whether `C_Log.LogMessage` is available to addons and whether **one** unique harmless marker reaches a local log quickly. Initially we expected `Logs/DeveloperLog.log`; the actual target was `Logs/General.log`. No player context, AI, chat transport, memory reading, filler or repeated send was included. Snapshot/AI-return work did not start.

First-party `apps/bridge-lab/native-log-probe.lua` loads through an existing experimental inbox without modifying a client binary or TOC. Load and `/wispnative status` read only the function type and three relevant CVars.

An explicit `/wispnative test <run-id>`:

1. Checks function/settings/timer APIs; missing functions abort without changes.
2. Saves exact `developerLogFilterNormal`, `developerLogWriteToFile` and `developerLog` values. Missing/unexpected values abort before changes.
3. Enables only values not already 1 and verifies readback. Failure triggers restoration.
4. Calls `C_Log.LogMessage` once with `WISPNATIVE1:<run-id>:<uptime>:<epoch>`. The same ID cannot be replayed.
5. Restores changed values after 60 seconds, `/wispnative stop` or logout/reload events. Verifies restoration, reports failure and allows manual recovery. Original values/receipts remain in local Jornia Bridge Lab SavedVariables.

The measurement interval ends before saving receipts through reload. Markers first appearing after restoration/reload/exit must be distinguished from fast active-period delivery. These historical exit checks are not accepted live-transport steps for later experiments.

## Independent file measurement

The existing read-only watcher gained `--protocol native`; the chat-log protocol stayed unchanged. Attempt one followed only `DeveloperLog.log`; the corrected repeat followed only `General.log`. It starts at the original file end, handles creation/append/detected truncation and retains only the matching marker's numeric times. No unrelated developer-log text is retained and the original file is never written.

Nominal poll interval: 50 ms. Record actual maximum interval, observation wall/monotonic times and native timestamp if available. Relate game and watcher clocks before calculating latency; Lua integer seconds are not millisecond precision. Results apply to this client, marker and traffic volume.

Run `5bf81eee6ba7` has private receipts/filtered observations under ignored `.cache/native-log-test/5bf81eee6ba7/`. No general log collection or raw client files are published.

## Local preliminary checks

- Lua tests: inert loading/status, one marker, replay prevention, timeout restoration, thrown logging error, setter changing state before throwing, denied setter, missing function and missing CVar.
- Isolated Python fixture: marker detection, numeric times and exclusion of unrelated text.
- Go tests used `GOPROXY=off`, `GOSUMDB=off` and existing dependencies. No external packages or finished third-party solutions were added.

## First attempt established function and destination

The client displayed `C_Log.LogMessage=function`. Original values: `developerLog=0`, `developerLogWriteToFile=1`, `developerLogFilterNormal=1`. Only `developerLog` changed. The receipt shows one accepted call and automatic restoration to 0 after 60 seconds without errors. The user checked all original values before reload.

After exit the marker appeared as an `[N][Lua]` line in **General.log**, not DeveloperLog.log. The continuous watcher had followed the wrong destination. An earlier targeted search found no marker, but cannot replace continuous measurement of the correct file. **The first attempt cannot establish first disk visibility or rule out live delivery.** DeveloperLog remaining empty is not a negative result for General.log writing.

## Repeat with the correct file

Run `d2b6fb185c8d` used the same first-party one-marker probe/restoration, watching General.log before startup. The first ID was not reused. Evidence is separate under `.cache/native-log-test/d2b6fb185c8d/`.

Actual Lua receipts show:

- client 1.60.1/build 70790/Interface 16001;
- exactly one marker submitted through an accepted call;
- only `developerLog` changed 0 → 1;
- the other two settings already 1 and unchanged;
- automatic restoration after about 60 seconds, verified readback and no errors;
- user verification of final values before reload.

| Check | Result |
|---|---|
| Function available in ordinary addon Lua | Yes |
| One labeled call accepted | Yes |
| Marker on disk within 60 seconds | No |
| Marker after restoration/reload | No; General.log still zero bytes |
| Marker at normal client exit | Yes; one matching General.log line |
| Native timestamp → first disk observation | **188.556 seconds** |

First disk observation: **13:46:40.686 CET**. Native line timestamp: 13:43:32.130. Native time/game uptime matched watcher wall/monotonic clocks within about 0.7 ms. Poll interval nominally 50 ms, maximum 104 ms, small relative to the delay.

This is a **manual exit-triggered result**, not mean/p95 latency. No filler was sent. Final General.log size was 39,138 bytes, below 64 KiB; this proves no buffer threshold. Bounded local volume or an available flush mechanism might produce earlier writes, but were not tested.

## Conclusion and uncertainty

We verified an **entirely local outbound write route from ordinary addon Lua to General.log**, without self-whispers, AI, process-memory access or pixels. A single small marker did **not deliver live** in the measured run. Do not adopt it as a live bridge or declare every other condition impossible.

Both attempts temporarily enabled developerLog. Since the target was General.log, that setting's necessity remains unproven and it may have been redundant. No variant tested this.

## Cleanup and restoration

Both watchers stopped. Temporary inbox 3 was restored byte-for-byte with WoW closed. Tools did not delete/rewrite original logs. Receipts were exported through the data-only parser rather than executing SavedVariables. Filtered evidence and repeat `summary.json` remain only in ignored caches. No publishing/network access occurred.

Steps 2/3 of that day's plan had not started. Further buffer-volume research would be a separately scoped outbound subtest.
