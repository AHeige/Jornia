# Wisp — product engineering rules

## Own the logic

- Wisp must own its product logic. Implement character identity, journey modelling, retrieval decisions, AI orchestration, synchronisation, transports and the companion experience in first-party code.
- Do not adopt large libraries/frameworks or build Wisp on another project's finished solution. Prefer a small implementation we can understand, test and maintain ourselves.
- Small, narrowly scoped code libraries are acceptable for basic primitives, protocols, drivers and formats. Keep dependencies minimal; do not delegate product behaviour to them.
- Research other projects to understand mechanisms and evidence. Research is not permission to install, vendor, copy or wrap their solutions. Experiments and production implementations must follow the same ownership rule.
- Apply this rule to existing code as well as new work. Review existing large dependencies when touching their area and replace delegated product logic with our own implementation. Preserve working user data and verified behaviour during migration; do not silently disable features to remove a dependency.

This is the user's settled product direction, recorded on 6 November 2026.

## Realtime transport experiments

- Only test transports that can deliver during an uninterrupted game session. Do not use `/reload`, logout or client shutdown as a required transport step, positive control or test completion step. The user ruled these experiments out on 6 November 2026.
- Do not restart/reload the game to load or read test results without a new explicit instruction. Prefer an already installed, not-yet-loaded experiment slot and read-only runtime observations. If these are unavailable, report the setup blocker rather than substituting a reload.
- For the next investigation, prioritise the local SpeakText/voice-helper path. Text appearing only after a session ends does not qualify as realtime delivery.
