# Reusable return bus, bounded macOS slice

Own Lua receiver + Python stdlib sender. No network, AI, game memory, global input hooks or third-party product logic. Existing native Vision helper reads only the visible Jornia receipt band.

This is experimental transport code, **not an installer or a distributable friend build**. It reuses eight existing, owned Jornia OCR lab TOC filenames as Boolean data lines (`WispOCRInbox081..088`) and our zero-byte `WispBridgeLab/wispreturn_pulse.wav` as a toggling clock. The permanent `wispreturn_present.wav` and never-created negative control check channel health. Dedicated data files should replace borrowed TOCs in a real installation. Do not run another updater or the normal OCR chat reader during this experiment.

Lua is installed once through an already installed, not-yet-loaded research slot; never reload/restart WoW for a test. `apps/bridge-lab/return-byte-bus.lua` is the receiver. In the verified run, `WispBridgeInbox03` bootstrapped it and inbox02 supplied an in-session UI-only close/progress update. Both originals were backed up and restored. Neither inbox carries subsequent test message data. No remaining research slot means a setup blocker, not permission to reload.

After all carrier files are present and the receiver is running:

1. `/wispbus` arms a new bounded transfer and shows `WISPREADY`. It resets protocol state, not the WoW UI or addon loader. `/wispbus hide` stops/hides it; Esc hides it.
2. Check the five-digit character binding in the receipt. `12715` was Cone Thrower's binding in this local test, not a universal value or a name matcher.
3. Run from the repo root, with a native helper built as documented in the OCR lab:

```sh
python3 scripts/experiments/return_bus/bus.py run \
  --root '/path/to/_forever_' --binding 12715 --text 'Hi'
```

The sender captures READY, freezes each byte until its matching OCR ACK, and finishes only after a checked `WISPDONE`. A repeated/old receipt cannot advance the current byte twice. `--corrupt` flips one checksum bit: the only expected success is a matching terminal checksum rejection, not text delivery. Re-arm Lua before each test. Captures and OCR text are not uploaded or saved; stdout contains status, receipt nonce and timings.

Publication uses eight data-file presence states, then toggles the clock **last**. Lua samples every 50 ms, requires the new clock and data to agree twice, checks the constant channel controls, and accepts one byte per clock transition. Clock values alternate; no return-to-zero handshake or new filename is needed between bytes. The sender does not touch data again until the receiver's byte/prefix checksum is echoed correctly through OCR.

Frame: ASCII `WP`, one byte of payload length, 0–96 payload bytes, then four-byte big-endian Adler-32 of header+payload. Text is displayed only after the full checksum validates. The OCR receipt includes a fresh transfer nonce, local GUID-derived binding, sequence, byte/length, digest and a separate weighted checksum. The checks detect ordinary corruption and stale frames; they are not cryptographic authentication. The payload itself is a controlled lab message, not an AI response. UTF-8 is assembled before display.

The code intentionally favors proof and diagnosis over throughput: every byte incurs an OCR round trip. There is no automatic retransmission, streaming, multi-request queue or reconnect recovery. The receiver bounds a transfer to 120 seconds; the sender bounds initial READY and per-byte waits. Character changes and failed self-tests do not silently produce zero data.

Normal completion/interruption restores all renamed carriers and verifies their hashes. A SIGKILL/power loss can leave `.wispbus-hold` files. Inspect and recover only the fixed own carrier paths:

```sh
python3 scripts/experiments/return_bus/bus.py restore --root '/path/to/_forever_'
```

Recovery refuses ambiguous files and symlinks rather than overwriting originals. It does not use a manifest containing arbitrary paths. Do not change sound settings for this test. If its controls fail, report the failure.

Tests:

```sh
python3 -m unittest discover -s scripts/experiments/return_bus -p 'test_*.py'
GOPROXY=off GOSUMDB=off GOCACHE="$PWD/.cache/go-build" GOMODCACHE="$PWD/.cache/go-mod" go test ./internal/companion
```

See `docs/return-byte-slice-2026-11-06.md` for real live measurements and limitations. The standard OCR AI experiment remains a separate finite-inbox flow until integration is explicitly built and tested.

## Eight-byte block experiment

`return-block-bus.lua` + `block.py` reuse 64 fixed own lab TOCs (`WispOCRInbox001..064`) and the same clock. **Do not run byte and block senders together.** There is no interprocess lock; isolate the normal OCR reader and updater, too. All 64 data files must be present at arming. Use a genuinely unused installed LoD slot to bootstrap owned code; no reload/restart as a substitute.

Arm `/wispblock` **before** starting the sender, then leave the receiver alone until completion/failure. Re-arming while sending can invalidate the baseline or change the transfer nonce. The panel must remain visible and unobscured. `/wispblock hide` stops and hides it.

```sh
python3 scripts/experiments/return_bus/block.py run \
  --root '/path/to/_forever_' --binding 12715 --text 'Again'
```

The sender waits for a checked READY and freezes each eight-byte block until a checked prefix receipt. Fields have fixed widths; omitted OCR spaces are tolerated, altered digits must still pass the checksum. Frame: `WB`, version byte 1, big-endian nonce uint32, binding uint16, payload length uint16, payload, Adler-32 uint32, then zero padding to a multiple of eight. Max payload 1024 UTF-8 bytes; max verified live payload 257 bytes. The nonce/binding are inside the frame as well as receipts. Repeated old receipts cannot advance twice. Checks are not cryptographic authentication.

After separately re-arming, `--fault crc`, `--fault stale` or `--fault padding` injects a checksum, nonce or padding fault. Only correlated rejection counts as a successful fault test. Padding requires a payload that actually has padding. Failed/timed-out transfers restore files, do not resend automatically, and do not show partial text. Restore this bank after abrupt process death using:

```sh
python3 scripts/experiments/return_bus/block.py restore --root '/path/to/_forever_'
```

Use the matching recovery width; byte-mode recovery does not restore the 64-bit data bank. See `docs/return-block-slice-2026-11-06.md` for measured speed, live failures and fixes, startup-slot usage, cleanup and performance limits. This remains a local controlled-message lab, not an integrated AI return channel or distributable client.
