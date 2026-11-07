# Chat-log latency on 6 November 2026

Status: **completed** on Forever 1.60.1, build 70790 / Interface 16001, Apple Silicon Mac, with Cone Thrower. Steps 2 and 3 of that day's plan had not started.

## Setup

First-party probe `apps/bridge-lab/chat-probe.lua` was installed into existing `WispBridgeInbox03/Inbox.lua`. It needs no new TOC and loads through `/wispbridge inbox 3`. Loading is inert. It sends fixed test markers only on explicit `/wispchat a` or `/wispchat b`, uses no AI and stores no unrelated chat.

`/wispchat status` reads logging state without changing it. Start saves prior logging state, character GUID, build and times. Stop, logout and a ten-minute timeout restore state. A GUID change stops the next send attempt. Native WoW logging may write other chat locally during the interval; our watcher processes only exact markers for our run ID. Nothing goes to AI or external analytics.

- **A:** normal chat logging, no action after receipt.
- **B:** 0.1 seconds after `CHAT_MSG_WHISPER` for our exact marker, call `LoggingChat(false)` then `LoggingChat(true)`. The delay separates toggling from the original event callback. Actual receipt and pre/post-toggle times are recorded.

Self-whispers pass through the WoW server. A successful send is not receipt: the probe waits for `CHAT_MSG_WHISPER`, distinct from outbound `CHAT_MSG_WHISPER_INFORM`.

## Measurement

`watch_chatlog.py` reads only `Logs/WoWChatLog.txt`, starting at its original end. Nominal polling is 50 ms; the actual maximum interval is recorded. It follows file creation, append and detected replacement/truncation. It never writes the log or retains unmatched chat.

Each match records variant, sequence, sender uptime/integer epoch, first disk-observation wall time and monotonic time. If a native timestamp exists, only that timestamp is retained, not the rest of the line. Native-to-disk calculations assume WoW uses the Mac's local timezone; validate against actual lines before using results. Lua integer seconds are not millisecond precision. Relative game send/receive/toggle times can be compared directly.

Unseen A trials are censored when B toggling begins. If B flushes earlier A markers, label those toggle-triggered, not fast ordinary logging. Reload/logout occurred only after the measurement interval and restored settings to save Lua receipts. These historical controls are not the later accepted live-transport workflow.

Raw results remain in ignored `.cache/chatlog-test/`; original logs are neither deleted nor published.

## Preliminary checks

- Lua tests verify inert load/status, B triggered by received rather than outgoing echo, restoration of both original false/true states and inert delayed callbacks after stop.
- An isolated Python fixture verifies append detection, timestamp parsing and exclusion of unrelated chat.
- An initial `/run` attempt encountered WoW's generic custom-script warning before execution. This is not a transport result. Tests use addon commands instead; ordinary Wisp use should not require pasted `/run` scripts.

## Results

Run `8624d79bb0a3`: all eight unique messages arrived in game. The accidental extra A send was retained as its own trial, not an error or duplicate. Send-to-receipt took **241–260 ms**; the server round trip did not explain the disk delay.

| Trial | Count | Received in game | On disk before exit | Result |
|---|---:|---|---|---|
| A1–A4, ordinary logging | 4 | 4/4 | 0/4 | No fast delivery |
| B5–B7, immediate off/on | 3 | 3/3 | 0/3 | Toggle did not flush |
| B8, multi-frame gap | 1 | 1/1 | 0/1 | Actual off interval 270 ms; no fast delivery |

The final control distinguished same-call toggling from a genuine off interval. `chat-gap-probe.lua` recorded `logging_while_off=false`, then `logging_after_toggle=true`. All three direct toggles also recorded logging restored. Event/toggle times come from saved Lua receipts.

The file stayed at zero bytes throughout A, both B variants, `/wispchat stop` and `/reload`. Normal full-client exit produced **3,810 bytes**. The independent watcher saw **16 matching lines for eight unique IDs**, all in one disk observation at **12:37:14.292 CET**. Two lines per ID represent both sides of a self-whisper, not sixteen trials.

Native-log timestamps to first disk observation ranged about **228–620 seconds**. This manually controlled exit-triggered result is not normal mean/p95 latency. Native timestamps and high-resolution game receipt times matched watcher wall/monotonic clocks within about 2 ms. Nominal polling was 50 ms, maximum 141 ms; that cannot explain minutes of delay.

**Conclusion:** at this traffic volume on this client, chat logs are not a sufficient live transport. The tested toggle-flush hypothesis received no support. Exit's positive control establishes that payloads were logged and the path/watcher were correct.

This does not prove every chat-log method impossible. The file stayed below 4 KiB; volume-triggered buffering or other explicitly tested flush mechanisms remain hypotheses. We measured no such threshold and sent no filler traffic. That would require a separately agreed step-1 subtest.

## Restoration and evidence

Original logging was false. The test stopped manually before reload/exit and restored that state. Both watcher processes stopped. Temporary inboxes 3/4 were restored byte-for-byte from pretest copies with WoW closed and are inert again. Original chat logs remain.

Filtered observations, operator notes, addon receipts and machine-readable summaries remain only in `.cache/chatlog-test/8624d79bb0a3/`, unpublished. The bounded data-only parser read Lua receipts; no Lua execution. `summarize_chatlog.py` rebuilds the summary from filtered files.

**Decision:** do not use the tested chat-log mechanism as the live bridge. Step 1 A/B testing finished; snapshot/AI-return work did not start automatically. Any next decision must establish whether another outbound log route is fast and repeatable enough to investigate.
