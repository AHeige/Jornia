# Jornia return-channel research and live verification

6 November 2026.

## Conclusion

Distinguish **cached resource content** from **live file-presence checks**. Loaded Lua modules cannot be reread as mailboxes in the current method. But whether a known file exists can change repeatedly and be observed inside the game.

After reviewing external code we built a first-party probe. The user's Mac observed **ten off/on cycles and 20 Lua transitions** on one zero-byte filename without reload/restart. That proves a reusable boolean on this client, **not yet complete fast text transport** at this checkpoint.

Next: bounded checked bytes, then a short text packet/acknowledgement. Preserve outbound OCR. Do not blindly enlarge one-shot addon pools.

## Reviewed material

Public sources were downloaded into ignored `.cache/return-research-2026-11-06/` with pinned trees:

| Project | Revision | Relevant material |
|---|---|---|
| chelinho139/wow-ai | `60165ca5bb6c6452f18604dc7b4fe8678b607bed` | bridge/protocol.js, install-slots.js, WoWAI.lua, architecture |
| tommygeoco/neverquestalone | `5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07` | signals.mjs, slots.mjs, publisher.mjs, Transport.lua, signals tests, PROTOCOL.md |
| 0xInuarashi/wow-forever-codex | `85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8` | native.py, Native.lua, NativeProtocol.lua, font/cache tests and notes |

This was mechanism research, not permission to install, vendor, copy or wrap their implementations. All probe/product logic remains first-party under AGENTS.md.

Our existing `apps/ocr-lab/WispOCR/WispOCR.lua` and `scripts/experiments/ocr/bridge.py` were also reviewed: waiting ticks can consume new LoD modules, publication writes all 96 inboxes, and sent/thinking/final updates can require further loads. That implementation was unchanged during the research.

## 1. Older audio-content signals differ from presence

[The wow-ai architecture](https://github.com/chelinho139/wow-ai/blob/60165ca5bb6c6452f18604dc7b4fe8678b607bed/docs/ARCHITECTURE.md#signals-the-empty-wav-trick) describes empty → valid WAV as a readiness flag. Loaded audio resources may remain cached. The project still uses 200 one-shot modules with scheduled polling when signals fail. More modules enlarge a finite buffer; they do not provide reuse.

## 2. NeverQuestAlone uses removal and recreation

[signals.mjs lines 1–20/88–128](https://github.com/tommygeoco/neverquestalone/blob/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07/bridge/transport/signals.mjs) pulses by **removing/recreating the same known filename**, not rewriting audio. Files are empty; paired names reduce startup-absence races.

[The Lua receiver](https://github.com/tommygeoco/neverquestalone/blob/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07/addon/NeverQuestAlone/Transport.lua) calls PlaySoundFile, stops returned handles and interprets transitions. A permanent present control distinguishes missing signals from unavailable sound. [Protocol section 3](https://github.com/tommygeoco/neverquestalone/blob/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07/docs/PROTOCOL.md#3-bridge--game-doorbells) associates this with Forever 1.60.1.70009.

The module has recovery/bounded pulses. Payload still lives in finite addon modules: presence announces **when** to load one. This does not automatically implement the proposed text bus below.

Referenced verification file docs/VERIFICATION.md/E-015 could not be retrieved at that revision. Comments/tests are not independent live-client proof, so we performed our own probe.

## 3. Fonts carry bytes but retain a filename bank

[The font writer](https://github.com/0xInuarashi/wow-forever-codex/blob/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8/companion/native.py) encodes bytes in glyph widths. [Lua](https://github.com/0xInuarashi/wow-forever-codex/blob/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8/addon/CodexPixelBridge/Native.lua) uses SetFont/SetText/GetStringWidth, calibration and packet checks before normal display. This transfers content without an addon module per packet.

[Bank notes](https://github.com/0xInuarashi/wow-forever-codex/blob/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8/docs/font-bank.md) describe 65,535 filenames/hardlinks sharing baseline files. Atomic replacement breaks one link; writing through it would affect siblings. Filenames/metadata still cost resources. We did not install the bank.

[Recycling tests](https://github.com/0xInuarashi/wow-forever-codex/blob/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8/docs/font-recycling.md) concern two diagnostics on older 1.60.1.69913. Reuse occurs across a new client process, **not during uninterrupted play** or automatically across the full bank. Useful fallback capacity, not unlimited return.

## 4. Our live probe

Mac, Forever 1.60.1.70790/Interface 16001, Cone Thrower, PID 8007. The user explicitly permitted game control. The client was already at character selection; we entered the selected character without causing logout/reload/restart or gameplay actions.

Three owned resources were added to existing WispBridgeLab while at selection, before entering the world: two empty files and valid silent WAV. This does not establish new-name discovery during an already logged-in UI.

Our inert probe loaded through preinstalled unused WispBridgeInbox04. That consumes one slot **for measuring code**, not per signal. The probe changes no files, SavedVariables, AI or settings, samples every 250 ms and stops after 30 s.

Observed baseline:

| Resource | PlaySoundFile first result |
|---|---|
| Permanent empty present control | true |
| Empty pulse file | true |
| Valid silent WAV | true |
| Existing owned lab Lua | true |
| Existing owned lab TOC | true |
| Missing Lua control | nil |
| Missing WAV control | nil |

Thus truthiness is not evidence of valid audio on this build. Sound settings stayed unchanged, Sound_EnableAllSound 1.

External code toggled only owned empty wispreturn_pulse.wav: absent 0.5 s/present 0.5 s, ten cycles, restored in finally. Actual Lua showed **120 samples,20 transitions**; other controls unchanged. Same name throughout, no further LoD loading.

An earlier short run approached the 30 s limit and observed two of three cycles. It was not counted as complete; the ten-cycle repeat fit the new interval.

Inbox 04 restored byte-for-byte, SHA-256 `86951c73c6c144f3e0ca568cc9822d0231303ff601fa2581262d89c0c0412277`. Three probe resources remain. Probe auto-stopped after 120 samples; /wispreturn can rearm a bounded interval. No production journals/secrets changed; existing OCR/AI remained.

Code: apps/bridge-lab/return-presence-probe.lua and scripts/experiments/pulse_return_presence.py. Private timings/observations remain under ignored .cache/return-research-2026-11-06/wisp-probe/. Two clocks mean this is **not exact one-way latency**;250 ms is the sampling interval, not a latency guarantee.

## 5. Proposed reusable text bus at this checkpoint

**Our inference:** repeatedly observable presence allows several filenames to represent packet bits. Lua could reconstruct text without LoD inboxes. NQA demonstrates doorbells, not this complete text channel.

Eight known data files represent a byte; controls/validity/sequence prevent sampling during updates. Desktop freezes data until Lua displays an OCR-readable acknowledgement bound to session, character, question and sequence.

One file rename is not atomic publication of all files. Require ordered writes, full checksum and bounded recovery on rejection. Start with controlled zero/one bytes, not AI integration.

Requirements:

1. Same file set delivers changing bytes repeatedly without new names/reload/restart.
2. Reject corrupt/stale/wrong-character/wrong-request packets. Repeated receipts must not duplicate messages/paid inference.
3. Capture loss, reordered file updates, low FPS and unavailable sound yield wait/rejection, not false zeros.
4. Measure API/file cost, bytes/s and frame impact before choosing parallelism. Seven probes here do not establish 256-probe cost.
5. Verify question-to-answer E2E; single signal transitions are not finished replies.

Theoretical capacity before overhead: N files/8 bytes per stable sample. 64 files→8 bytes; 256→32 bytes. Writes, checks, receipts, recovery and UI budgets reduce throughput. These are capacities, **not benchmarks/promises**.

## 6. Decision at this stage

| Route | Evidence | Next role |
|---|---|---|
| Existing LoD inbox | Actual AI reply, finite pool | Preserve working reference |
| Reusable presence | Ten cycles/20 transitions on current Mac | Candidate checked byte/text transport |
| Presence readiness only | Primitive verified, integration absent | Reduce empty slot reads, not exhaustion |
| Font bank | External code/reports reviewed | Fallback, not installed/live-verified by Jornia |
| Reloading the same inbox | No reuse mechanism found | Do not promise |

The new possibility is **reusable observable presence**, not resetting the addon loader. Sustained fast reliable replies still require protocol experiments.

## Follow-up: byte transfer completed

A later instruction led to first-party byte transfer with clock, Adler-32, sequence/character correlation and OCR receipts. Several controlled texts including åäö used the same carriers; a damaged frame was rejected. More than a boolean probe, but still a slow bounded non-AI test. See [byte results](return-byte-slice-2026-11-06.md), then [block results](return-block-slice-2026-11-06.md) for later evidence.
