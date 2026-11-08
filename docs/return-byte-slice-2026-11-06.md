# Jornia reusable byte and text return on 6 November 2026

Later follow-up: [eight-byte acknowledged block transfer](return-block-slice-2026-11-06.md) is complete. This report preserves the earlier byte slice and its limits.

## Result

Controlled first-party text reached the actual Lua window through **the same reusable filenames**, with local OCR acknowledgements. No new inbox loaded per byte/message. The same receiver accepted several messages. A deliberately damaged packet was rejected before display.

| Test | Payload | Wire bytes | First byte → read final acknowledgement | Outcome |
|---|---:|---:|---:|---|
| Hi | 2 bytes | 9 | 4,611.7 ms | Text/final receipt verified in WoW/OCR |
| Again | 5 bytes | 12 | 8,399.6 ms | Same files reused, new text verified |
| BAD, wrong checksum | 3 bytes | 10 | 5,209.9 ms | Checksum failed — no text delivered; failure read |
| åäö | 6 UTF-8 bytes | 13 | 7,959.3 ms | Correct Unicode verified visually/final receipt |

All trials: Cone Thrower, Forever 1.60.1.70790/Interface 16001, same client PID 8007 as before the slice. No reload/logout/restart/game action/AI/server calls. No image left the machine.

This is a **small working text channel with reusable carriers**, not finished chat. Per-byte acknowledgement is too slow for long answers. Deliberate payload cap:96 bytes. Sustained traffic, Windows and real AI replies through this channel were not verified.

## Implementation

- `apps/bridge-lab/return-byte-bus.lua`: self-test, eight-bit receiver, clock edges, two stable reads, sequence/prefix receipts, frame checks, normal text display and timeout. No custom Lua network/filesystem API.
- `scripts/experiments/return_bus/bus.py`: stdlib sender toggles presence and waits for native-OCR receipts. It never loads addons or sends keys. Files are restored with SHA-256 checks.
- Python tests cover byte boundaries, UTF-8, damaged OCR receipts, reuse/restoration. A Go test executes actual Lua in the existing 5.1 environment, covering multiple messages, corruption, unavailable sound and character changes.

No large dependency or other product implementation was introduced. Code is first-party, using platform/test primitives.

## Carrying bytes through signals

Eight known filenames represent eight bits: present 1, absent 0. Another known name is an **alternating clock bit**. Set all data first, change clock last, then freeze data until a matching receipt.

Lua ignores unchanged clock levels. On an edge it samples data/controls twice, nominally 50 ms apart. It acknowledges sequence, byte value and Adler-32 of the received prefix. Vision reads visible text with another checksum. The sender requires transfer nonce, character binding, sequence, value and prefix digest before publishing the next byte.

This is not an atomic multifile operation. Data-before-clock, stable reads and frozen data make the bounded experiment controllable. Other scheduling/filesystem conditions remain to be tested.

Frame: WP + one length byte + payload + four big-endian Adler-32 bytes over header/payload. Valid completion gives WISPDONE; invalid checksum gives WISPFAIL/error 3. Payload is not displayed until complete validation. Checks/correlation detect corruption/staleness; they are not encryption/authentication against local attackers.

A permanent present control and missing control distinguish unavailable sound from all-zero data. Character changes stop transfer. Bad/missing receipts cause bounded waiting/failure, not automatic byte/AI replay.

## Live carriers and restoration

The running client lacked eight dedicated data resources. We borrowed **our own** already known `WispOCRInbox081..088` TOCs plus empty `WispBridgeLab/wispreturn_pulse.wav` for the clock. TOC contents were not read as packets; only presence mattered. Original bytes moved to same-directory `.wispbus-hold` names and back. No other addon's files were used.

This avoids reload for a running-session experiment, **not a product installation design**. Production needs dedicated owned carriers/startup/update rules. Updater and sender must not modify carriers simultaneously.

The ordinary OCR/AI reader paused for isolation; companion/server/database unchanged. No ordinary questions were sent/replayed.

Receiver code bootstrapped once through preinstalled unused WispBridgeInbox03. A UI update loaded through inbox 02 for close/progress. These load test code, not per-message return data. Later transfers use presence and `/wispbus` to rearm receiver variables, without reloading WoW/addons.

Afterward all nine carriers matched originals, no hold files remained, and inbox 02/03 were restored. The receiver remains in this UI's memory until it naturally ends, but saving/exiting is not needed to establish success.

## Performance boundaries

Observed payload rate: about **0.43–0.75 bytes/s** for very short messages. Seven framing bytes and one OCR receipt per byte dominate the conservative design. These times do not promise fast AI chat.

Individual OCR receipt scans took roughly 36–142 ms. Total time also includes capture cadence, stable Lua reads, files and stop-and-wait. Initialization/warm-up/waiting before first publication are excluded. OCR read the final receipt; actual text/errors were visually checked with CUA.

Lua reported zero API errors and 9.1–13.5 ms accumulated work in some sampling branches. This omits early returns/idle and full game FPS; **no FPS/CPU guarantee** follows.

Next experiment: reduce OCR round trips through multiple bytes per block, more parallel bits or an explicit timing window, retaining sequence/checksum/bounded replay. Measure PlaySoundFile/file-update costs first; unbounded per-frame calls are unjustified.

## Verification and limits

- Three Python tests and companion tests passed, including actual Lua. Local coverage:00/55/AA/FF boundaries, empty payload, UTF-8. Four live trials are listed, not an endurance test.
- No new filenames after receiver bootstrap. Reuse removes one-shot consumption **for this method**, not for the unchanged older OCR/AI prototype.
- Receipts drive each next byte. Lost receipts, paused capture, abrupt death and byte-error recovery remain limited. Fixed restore supports hold-file recovery; interrupted requests never automatically become new requests.
- Character changes/sound-down are simulated Lua tests, not actual character/settings changes. Sound settings unchanged.
- Static text positions stabilize OCR; these simple lab states do not replace the companion/journey design.
- No internet research, GitHub push, VM changes, paid models or public release in this slice.

Private logs: ignored `.cache/return-bus-2026-11-06/{hi,again,corrupt,utf8}.jsonl`. See `scripts/experiments/return_bus/README.md` for running/precautions.

After verification `/wispbus hide` stopped/hid the lab, visually checked. The ordinary OCR/AI reader resumed against the same companion without a question. Byte senders/capture processes stopped; no background file changes remained.
