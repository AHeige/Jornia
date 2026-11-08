# Jornia acknowledged block return on 6 November 2026

## Result

Eight bytes per acknowledgement work between desktop and Lua. The same 64 data carriers and clock were reused. No inbox is consumed per block/reply. No AI/network/reload/logout/restart was involved.

| Trial | Payload | Wire bytes / blocks | Transport time | Result |
|---|---:|---:|---:|---|
| New byte reference, Again | 5 bytes | 12 / 12 byte receipts | 5,329.9 ms | Delivered |
| Block, Again, first version | 5 bytes | 24 / 3 | 1,284.7 ms | Delivered |
| Block, Again, final version | 5 bytes | 24 / 3 | 1,438.7 ms | Delivered |
| Longer text including åäö | 257 bytes | 272 / 34 | 14,977.4 ms | Complete text verified/displayed |
| Deliberately bad checksum | 3 bytes | 24 / 3 | 1,336.9 ms | Rejected, no text |
| Wrong transfer nonce | 3 bytes | First of three blocks | 423.9 ms | Immediately rejected, no text |

The final short block trial was about **3.7× faster** than the reference. Single runs, not statistical benchmarks. Long-packet payload rate: **17.16 bytes/s**, still too slow for long quick-chat answers. Timing starts at first block publication and ends at checked final OCR receipt; warm-up/inference excluded.

The byte reference was remeasured after screen changes. All trials used Mac/Cone Thrower/Forever 1.60.1.70790/Interface 16001, same PID 8007,1920×1080. CUA checked text/errors/metrics. No images reached a model/server.

## Implementation

- `apps/bridge-lab/return-block-bus.lua`: first-party 64-bit receiver, two stable samples, controls, current character/nonce, prefix receipts, complete validation and scrollable response.
- `scripts/experiments/return_bus/block.py`: Python stdlib, blocks frozen until checked OCR acknowledgement, timeout, fault injection/restoration. Uses our existing native Vision/ScreenCaptureKit helper.
- `bus.py` supports only fixed 8-bit/64-bit carrier banks. Recovery never accepts arbitrary manifest paths.

Wire: WB, version 1, nonce uint32, character binding uint16, payload length uint16, payload, Adler-32 uint32. Integers big-endian; pad to eight-byte blocks; each byte's carrier bits little-endian. Payload cap 1,024 UTF-8 bytes; largest live trial 257. Full 1,024-byte size is only locally protocol-tested.

Set data first, toggle clock last. Lua detects the edge and requires two identical samples nominally 50 ms apart plus valid controls. Sender requires matching nonce/binding/sequence/prefix length/digest before publishing another block. Display text only after full checksum/padding validation.

Receipt fields are fixed-width with their own checksum. OCR may omit spaces between numeric fields, but altered digits must fail validation. This detects accidental corruption/stale packets, not cryptographic authentication.

## Issues found live

First long attempt never sent: Vision merged digest/receipt-checksum numbers. Fixed-width parsing was corrected without removing checksums. A regression fixture uses the actual observed line.

One retry overlapped rearming/sender startup and accepted an earlier READY. It stopped/restored. **Arm before starting the sender; never rearm mid-transfer.** No automatic resend.

The first long transfer then stalled at block 4: prefix digest 2,673,674,982 exceeded signed 32-bit formatting and Lua emitted no acknowledgement. Replacing `%010d` with `%010.0f` allowed full delivery, including final digest 3,876,742,076. This points to WoW integer formatting; no error stack was collected. Tests guard against integer digest formats because host Lua does not reproduce the limit.

Failed attempts are not counted as delivered. Private logs include successes/failures in ignored `.cache/return-block-2026-11-06/`.

## Game cost and limits

Successful 257-byte trial:6,019 probes,190.3 ms accumulated instrumented callback work,1.75 ms maximum callback,zero API exceptions. FPS baseline/min/mean 60/60/60. Counter includes waiting after arm, not self-test startup, metrics rendering or whole-game CPU/GPU work. A short stationary 60 FPS-capped run proves no general combat/hardware performance claim.

This slice has no interprocess lock, queue, automatic recovery/resend, streaming or Berget product integration. One sender only; no concurrent updater/OCR reader. Lost receipts stop, never trigger new inference. Windows/sound settings/client variants are untested. Presence visibility remains an empirical client property.

## Bootstrap and restoration

Initial receiver loaded through preinstalled unused WispBridgeInbox01. Formatting fix loaded through unused WispOCRInbox096 after read-only verification. Our `/run C_AddOns.LoadAddOn(...)` bootstrapped test code, not an end-user workflow. All packets then used the same reusable files and `/wispblock` rearming, without more addon loads.

Carriers temporarily borrowed **our own** WispOCRInbox001..064/*_Camelot.toc plus empty WispBridgeLab/wispreturn_pulse.wav. Dedicated product carriers remain necessary. All 65 files were restored/content-checked, no `.wispbus-hold` left. Bootstrap 01/096 restored byte-for-byte. Lab stopped/hidden; ordinary OCR/AI reader resumed Ready without a question.

Six Python tests and all internal/companion tests passed. Coverage includes binary boundaries, UTF-8, length limits, reuse/restoration, corrupt/repeated receipts, checksums/padding, stale nonce, character change and unavailable sound control. Last two are simulations, not actual player/settings changes.

Next: reduce acknowledgement round trips and integrate reusable return with AI. At this slice's checkpoint the old OCR/AI flow still used finite inboxes; it did not silently adopt this channel.
