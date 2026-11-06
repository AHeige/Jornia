# Wisp OCR E2E lab (macOS 14+)

This is a first-party, finite-session experiment, not a general transport. No large OCR framework, copied companion, global keyboard hooks, game memory access, or chat logging. Vision/ScreenCaptureKit are platform primitives.

The addon is installed **before a normal game start**. The installer refuses a running game and an existing experiment. Never restart/reload a game for a transport test.

Build from the repository root:

```sh
mkdir -p .cache/ocr-test .cache/objc-module-cache
clang -fobjc-arc -fmodules -fmodules-cache-path="$PWD/.cache/objc-module-cache" \
  -mmacosx-version-min=14.0 -framework Foundation -framework AppKit \
  -framework Vision -framework CoreGraphics -framework ScreenCaptureKit \
  scripts/experiments/ocr/capture.m -o .cache/ocr-test/wisp-ocr-capture
python3 scripts/experiments/ocr/bridge.py install --root '/path/to/_forever_'
```

Use a built companion with `-ocr-lab -listen 127.0.0.1:8788`. This opt-in flag creates owner-only `ocr-session.json` next to its database and removes it on a graceful exit. It contains a temporary loopback token and verified GUID bindings. Do not share it. The normal desktop launcher does not enable the experiment. Stop the existing companion before opening the same database.

```sh
python3 scripts/experiments/ocr/bridge.py run --root '/path/to/_forever_'
```

If the reader reports `screen_permission_required`, screen recording permission is required for this specific local capture process. The `--permission` helper option requests it; no permission is silently changed. Wait for `Ready` before sending. The helper warms Vision first. It captures only the WoW window in memory, locates the visible magenta Wisp corners, and sends only the outbox band to Vision. Images are never saved or uploaded by the live reader. Closing the panel removes its outbox from view.

In WoW: `/wispocr`, type an English question, Enter. First test: `What can you tell me about Ashenvale?`. Limit 160 printable ASCII characters; spaces are normalized before rendering. Only the question goes to the existing Wisp API/Berget, with AI enabled. GUID/local character IDs, screenshots and journal excerpts are not part of the backend question payload. Normal local message history still stores question/answer for the linked character.

Loading states are plain text in a stable position: reading locally → locally validated/sending → API accepted/thinking → reply or failure. Static treatment is intentional: movement in the OCR card would impair recognition and distract from play; there is no flashing progress animation or artificial percentage. Text input stays intact and disabled during a pending question; errors re-enable it. Enter is handled solely by the addon's own edit box.

The reader requires two consecutive matching, complete cards, checksum and known unambiguous character binding. It never repairs question words. Stable request IDs use installation nonce + second timestamp + sequence; existing API mutation receipts prevent repeated paid inference after a reader restart. The checksum detects ordinary OCR corruption, not malicious tampering. Character changes discard the pending display; already accepted API requests may still complete for their original character.

Return uses **96 one-shot LoD inboxes**, polled every 600 ms while pending. On receipt/state changes the reader atomically updates all own inbox files; the next unused slot reads the newest state. No knowledge of file readiness or external keystroke is needed. A slot can be consumed once in the current UI lifetime: these are polling slots, **not 96 guaranteed conversations**. A slow request may exhaust the entire pool. There is a 55-second UI deadline. This experiment does not establish unlimited chat or a production update mechanism.

Metrics distinguish `ocr_ms`, `capture_ocr_ms` for the final matching read, `api_and_reply_ms` for API acceptance/polling/file writes, and total Enter→visible callback time displayed inside Lua. Two reads plus 300 ms reader cadence and up to 600 ms inbound polling add latency. Do not describe one OCR sample as total latency. `reply_files_written` is not proof of on-screen delivery: verify the actual game window.

Tests:

```sh
python3 -m unittest discover -s scripts/experiments/ocr -p 'test_*.py'
GOPROXY=off GOSUMDB=off GOCACHE="$PWD/.cache/go-build" GOMODCACHE="$PWD/.cache/go-mod" go test ./internal/companion
```

`fixture.m` renders a synthetic outbox into a local PNG for explicit fixture testing. `capture --file fixture.png` exercises the same region locator and Vision without game capture. This is not evidence of a live game/AI round trip. Generated binaries and images belong in `.cache/ocr-test`, never in a release.
