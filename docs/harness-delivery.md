# Companion harness and delivery

Requested 6 November 2026. Initial target: question in the player's language → approved current game context → sourced next step → optional Show the way → validated Lua marker.

- [x] Thin first-party Go harness with tool registry, bounded execution, payload-free tracing and structured results.
- [x] Next-step evidence from a current addon snapshot; history/Armory cannot masquerade as current quest state.
- [x] Structured navigation actions; no model-generated Lua scripts or coordinates.
- [x] Language-independent questions/answers and UTF-8 through the OCR lab, without an ASCII-language allowlist.
- [x] Optional quest context and click-driven Lua waypoint; revalidate freshness, character and active quest on click.
- [ ] Full-flow tests covering language, errors, stale replies, character switching and absent native waypoint support.
- [x] Update wisp.logosab.com and downloads within existing isolation.
- [ ] Package checks: private files/secrets, binary identity, SHA-256, installation and available signing/notarization.
- [x] Document actual signing/antivirus status; no external-antivirus guarantees or disabled protections.

Starting point: published desktop alpha 0.6.0. OCR/return labs were separate. Reusable block return was experimental. No WoW reload/restart may substitute for transport or loading test code. ForeverDB access had been requested, not yet granted; navigation must use actual client support and decline when data is missing.

## Verification and remaining gates

- `go test ./...` passes, including actual Lua in a 5.1 environment, consent, malformed actions, stale snapshots, wrong character, combat, missing APIs and absent waypoint confirmation.
- Five Python protocol tests pass. The Vision fixture reads a Japanese UTF-8 question and quest snapshot through the hex protocol: 358.6 ms in a synthetic run, not new live latency.
- Five real Berget questions with explicitly synthetic current snapshots pass in French, Spanish, Japanese, Arabic and Swedish, each with supporting state citations and a validated action. See `harness-live-evaluation.json`. Initial iteration failures are retained separately; model errors are never automatically retried.
- API `wisp-private-api:20261106-harness3` runs Go directly against Berget. Only Jornia API was replaced; other containers remained. The old model proxy is stopped, with image/private rollback configuration retained in Compose profile `legacy-model`.
- Mac companion updated locally with its database preserved. Both ZIPs passed CRC, SHA-256 and private-filename checks. The Mac executable served embedded UI using a separate test database. Windows was cross-compiled, not executed in this test.
- Live checks found C_QuestLog.GetNextWaypoint and C_Map.SetUserWaypoint. **Quests 376 and 3901 on Cone Thrower returned no native positions.** The adapter must not offer a marker in that case. A later slice needs sourced positions and a separately validated map fallback.
- The new live OCR flow was **not accepted at this stage**: its first card clipped, and two waiting tests exhausted the finite return pool. Font, fixed text height and crop were corrected; synthetic Vision passed. Neither live attempt reached Berget. Corrections loaded through unused slots 094/095, then their files were restored. We performed no reload/logout/restart and left the game running. The finite pool was effectively exhausted for that UI session.
- Mac downloads are blocked pending Developer ID Application identity, a notarytool Keychain profile and Apple acceptance. See `macos-signing.md`. An Apple Developer account is confirmed but no valid signing identity is installed. Windows signing is also unavailable.

**Next concrete test at that point:** run the corrected snapshot/OCR chain in a new naturally occurring game session, or integrate the separately proven reusable block return before more finite-session tests. Do not request reload/restart to bypass the limit. Do not present this as a general friend installation with finished in-game navigation.

Publication: [0.7.0 Windows developer alpha](https://github.com/AHeige/wisp-site/releases/tag/v0.7.0) is published with SHA-256; the downloaded file matched the locally tested hash. GitHub Pages commit `11efefb` and the wisp.logosab.com reference were updated. The Mac button remains blocked pending signing. Only public release assets were uploaded at this delivery stage, not source, keys or player journals. See `release-0.7.0-download.png`.

Cleanup: the exhausted lab reader stopped and its window was hidden. The updated ordinary companion continued on port 8788 with local data preserved. Extra test app/SSH tunnel processes stopped. Subsequent integration work is tracked separately in `foreverdb-chat-delivery.md`.
