# Wisp visual OCR transport on 6 November 2026

## New user direction

A dedicated Lua chat is open in WoW. Enter should make the question readable from that window and send it through Wisp → backend → Berget → backend → addon reply. This now takes priority over SpeakText. Transport must not require reload, logout or client exit.

## First local evidence

Our Objective-C code ran macOS Vision on a previously attached user WoW image. No new screenshots, key monitoring, OS-permission changes or server uploads of image/OCR text occurred.

- Nine text lines detected.
- C_Log.LogMessage=function and visible settings were readable.
- First call: 7.673 seconds; subsequent calls: 0.210 and 0.185 seconds.
- A long function name gained a space despite confidence 1. Confidence does not prove correctness.

This is a **file-based OCR check**, not verified live capture. The image is a real game crop, not a fullscreen benchmark or a finished Wisp-outbox test. Three runs cannot establish product accuracy/latency.

Code: `scripts/experiments/read_image_ocr.m`, using Foundation/Vision primitives with first-party product logic. No external OCR distribution/finished companion was installed. Swift hit a local compiler/SDK-module issue; Objective-C worked. System development tools were unchanged.

## Proposed Enter flow

1. Lua handles Enter in its edit box and creates a stable readable **outbox card** with question, request ID and character correlation. It remains visible rather than flashing for one frame.
2. The companion locates the panel in the explicitly selected WoW window and reads its region. Ordinary chat, other apps and the full screen must not reach the backend.
3. Local OCR reads/validates IDs, delimiters and checksum, rejecting errors. Multiple identical reads can help but do not replace validation. One request ID must never trigger repeated paid calls.
4. Only validated question text and explicitly selected context reach the existing API. No global keylogger is needed and images can stay local.
5. Lua shows understandable pending/sent/answered states backed by actual receipts. Character changes/stale cards must not misroute answers.

Live Mac capture needs separate screen-access handling, not requested in the file test. Window identity, moving/scaling, font size, wrapping and reading errors require real-client checks. OCR-readable layout is part of the transport.

## Return is a separate problem

At the initial stage, two local unloaded-addon replies were proven. Each finite slot loads once per UI lifetime; readiness detection and many-turn endurance remained unresolved. A single correlated known-inbox reply can test E2E but is not unlimited finished chat. The OCR proposal does not silently authorize memory writes, hidden input or an untested return mechanism.

## Initial bounded test

**One question, no AI:** show a controlled Lua outbox, Enter, read locally during play. Require correct question/character/ID, no duplicates and measured Enter-to-validation. Reject malformed/clipped text. Only afterward test that request through AI/return.

At that checkpoint image OCR was verified but live capture, outbox UI, Enter detection and AI return were not. No network access/GitHub push occurred.

## E2E lab implemented later that day

A later instruction prioritized the whole chain. We built separate Wisp OCR Lab, a Python-standard-library controller, native Objective-C ScreenCaptureKit/Vision capture and explicit `-ocr-lab` in the Go companion. No large new dependency. See `scripts/experiments/ocr/README.md` for running/data limits.

Return now used 96 preinstalled LoD slots rather than one manually loaded inbox. The addon consumed the next slot every 600 ms while pending; the controller wrote sent/received/final state to all owned inboxes only on actual state changes. This finite cost does not prove unlimited chat.

Outbound validation required two identical complete reads and a checksum over ID/character/question. The initial lab capped questions at 160 ASCII characters. No automatic journal sharing; real excerpts require separate review. The initial bounded-test section above records the earlier plan, not remaining implementation status.

### First real-client run

After normal startup following installation, the user asked `What can you tell me about Ashenvale?` on Cone Thrower. Request `272478521793971663000001`:

- Final validating scan: **59.3 ms OCR**, **207.0 ms capture/localization/OCR**.
- Two matching scans required; not a full Enter-to-receipt measurement.
- Native `Question received — Wisp is thinking…` visibly confirmed outbound OCR → local companion → automatic inbox acknowledgement.
- AI failed: VM API logged `inference returned HTTP 502`. Controller API/file stage took 5,275 ms before publishing the error.
- Automatic `Could not finish — your question is kept` appeared in the same window, without reload/logout/exit/manual inbox loading.
- A separate small VM model check later succeeded through LangChain/Berget. The initial cause was then unknown and the original request was not automatically replayed.

Three Python protocol/return/error tests and companion Go tests passed, including Lua 5.1 correlated replies, wrong character, waiting and duplicate-send rejection. A synthetic crop/OCR fixture validated a control line Vision split into two observations: 149.6–214.4 ms. No browser/internet research or GitHub push; server/Berget calls served the requested test.

At that checkpoint live OCR, API receipt and automatic error return were proven; a successful AI reply still awaited confirmation.

### Completed real Berget reply in Lua

Two more failed runs (`…000002`: 81.1 ms OCR; `…000003`: 47.6 ms) led to safe diagnostics identifying **LengthFinishReasonError** in the model adapter. The old 220-output-token limit could abort structured responses. This was not established as transient/OCR failure; the SDK/adapter converted the abort to HTTP 502.

Fix: Go API/adapter allow at most **512 output tokens**. Go retains a concise-answer instruction: at most 60 words and four directly relevant evidence IDs; narrow answers if more evidence is needed. This is a prompt instruction, not a guaranteed word cap. No automatic retries/new libraries. Incomplete/invalid responses remain failures, never fabricated success.

**Request `272478521793972374000004` succeeded and was visually confirmed in the same uninterrupted Cone Thrower session.** The same Ashenvale question produced zone/named-place facts, sources and a warning that source_checked does not confirm current game state.

| Measurement | Value |
|---|---:|
| Final OCR of two matching scans | 46.2 ms |
| Capture/localization/OCR for that scan | 212.4 ms |
| Backend-reported response time | 4,382 ms |
| Local API acceptance/polling and reply publication | 4,631.5 ms |
| Enter → actual Lua callback shown in game | **6.1 seconds** |

Route `berget`, model `mistralai/Mistral-Small-3.2-24B-Instruct-2506`. Confirmed usage: 5,823 prompt tokens/**229 output tokens**, estimated EUR 0.0018156. Output exceeded the former 220-token cap. File writes alone were not proof: Lua displayed `Reply received — 6.1 s total` and the answer. Metrics overlap and must not be summed. Additional latency includes two scans, polling and acknowledgement delivery.

All Go packages, three Python bridge tests and two model-adapter tests passed. Fake-provider model tests were network-isolated and checked secrets/question text stay out of error logs. Native helper compiled against local SDK. CUA verified waiting, failures and success. Static text was chosen for readability/OCR stability; reduced-motion settings need not change.

### Operations and remaining limits

- VM model `wisp-private-model:20261106-ocr2`, API `wisp-private-api:20261106-ocr1`. Only these were recreated with existing limits/ports. **Twenty unrelated** running IDs remained; the preceding diagnostic step changed only model and preserved the other 21.
- Previous images/private rollback files `.env.pre-ocr-model-diagnostics` and `.env.pre-ocr-output-fix` remain. No keys here; no prune/database/firewall changes.
- Mac companion temporarily runs explicit -ocr-lab on 8788 with the existing database. Normal launcher unchanged; reader works only while its process runs. Public downloads/site unchanged.
- Return pool remains **finite**. One success/error returns do not establish normal conversation capacity or unlimited return.
- Fixed-height lab responses can clip long source titles/timing. Successful answer/uncertainty note were visible. Scrollable history, durable pending state, language/character coverage, resolutions and actual character-switch tests remain.
- This test sent **only question text**, not current quests/journal. Personal/current context is a separate slice. Working transport proves neither complete nor correct world data.

**Checkpoint status: first real OCR → AI → Lua chain verified without reload/logout/exit/memory reading/global key monitoring.**

## Follow-up: reusable return primitive

Later source review led to an independent live probe observing 20 file-presence transitions over ten cycles through PlaySoundFile on the same known filename. File contents and file presence have different observable behavior on this build. The working finite OCR/LoD flow remained unchanged at this checkpoint; full byte/text transport was not yet implemented. See `docs/return-channel-research-2026-11-06.md` and subsequent byte/block reports for continuation.
