# Jornia — product engineering rules

## Own the logic

- Jornia must own its product logic. Implement character identity, journey modelling, retrieval decisions, AI orchestration, synchronisation, transports and the companion experience in first-party code.
- Do not adopt large libraries/frameworks or build Jornia on another project's finished solution. Prefer a small implementation we can understand, test and maintain ourselves.
- Small, narrowly scoped code libraries are acceptable for basic primitives, protocols, drivers and formats. Keep dependencies minimal; do not delegate product behaviour to them.
- Research other projects to understand mechanisms and evidence. Research is not permission to install, vendor, copy or wrap their solutions. Experiments and production implementations must follow the same ownership rule.
- Apply this rule to existing code as well as new work. Review existing large dependencies when touching their area and replace delegated product logic with our own implementation. Preserve working user data and verified behaviour during migration; do not silently disable features to remove a dependency.

This is the user's settled product direction, recorded on 6 November 2026.

## Realtime transport experiments

- Only test transports that can deliver during an uninterrupted game session. Do not use `/reload`, logout or client shutdown as a required transport step, positive control or test completion step. The user ruled these experiments out on 6 November 2026.
- Do not restart/reload the game to load or read test results without a new explicit instruction. Prefer an already installed, not-yet-loaded experiment slot and read-only runtime observations. If these are unavailable, report the setup blocker rather than substituting a reload.
- Current investigation: the user explicitly proposed OCR of the visible Jornia Lua chat/outbox on 6 November 2026. Prioritise that path; SpeakText is parked. The user’s earlier rejection of visual transport does not prohibit this newly requested OCR approach. Text appearing only after a session ends still does not qualify as realtime delivery.

## Repository language

- Write documentation, comments, development instructions, default UI copy and operator messages in English.
- Keep multilingual query aliases, explicit localized replies and Unicode/language test fixtures intact. Repository language must not remove supported player languages.
- Preserve historical measurements, identifiers, source URLs, raw evaluation outputs and recorded user data. Explain non-English evidence in English rather than rewriting what a test originally observed.

## Player-facing transport experience

- The player must never see transport code, encoded packets, checksums or diagnostic markers during normal use. This is an explicit product requirement from 8 November 2026.
- The intended chat flow is: write a message, press Enter, see a short waiting status in the same chat, receive the answer. Do not require a separate technical transfer window.
- The current visible OCR transfer card is a laboratory mechanism, not an accepted player experience. Do not call it ready for normal users until invisible transport has been demonstrated during uninterrupted play, including failure and recovery behavior.
- Preserve verified behavior while replacing the mechanism. Hiding the Lua card alone disables its OCR path and does not satisfy this requirement.

### Small in-chat QR exception (8 November 2026)

- The user subsequently approved one small QR tile at a fixed position in the top-right corner of the chat. This is an explicit exception to the earlier fully invisible transport requirement, not permission to restore the large encoded OCR card.
- Keep raw packets, checksums and diagnostic labels out of the player experience. The QR tile must stay inside the chat, preserve readable messages and input, and require no extra player action.
- Measure real quest context, repeated messages, movement/resizing, failure and manual recovery during uninterrupted play. A successful generated QR fixture alone does not establish a usable native chat or Windows readiness.

## Game-client test scope (8 November 2026)

- The user stopped the quest-completion/gameplay test because of botting concerns. Earlier permission to move, fight, loot or turn in quests must not be reused to resume that test.
- Continue with chat/companion UI and read-only character-state verification. Any later gameplay-control test requires a new explicit instruction.
