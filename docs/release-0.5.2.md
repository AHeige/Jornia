# Wisp 0.5.2 — character identity correction

A character directory such as `93/Sch-Nice` and its account journal (`Sch`, internal realm label) previously appeared as separate characters. Wisp now links them using its own per-character installation/GUID marker, not name/realm guesses. The first directory identity remains stable when unclaimed; an already imported GUID takes precedence over an empty preliminary duplicate. Independent private content is preserved and forgotten identities stay excluded. Directory names remain available for full display names.

The Mac client and main addon were updated in place, with a private offline database/binary/addon backup. Existing account journals import without modification. Restart WoW after the addon TOC update, then log in and save once. A fresh character's end-to-end first-discovery test is still pending. The real Sch-Nice marker was saved successfully; its previously forgotten empty folder is correctly still excluded.

Verification: `go test ./...`, `go vet ./...`; regression cases for surname/numeric realm mismatch, journal-before-marker repair, stable preliminary ID, repeated scans, reused folders/different GUID, same names across accounts, malformed/executable markers, forget suppression, and actual Lua marker creation. Windows cross-built, not executed.

The experimental first-party Bridge Lab is **not bundled** in this release. Two one-shot, post-startup desktop-file → in-game Lua text deliveries were confirmed in the actual Forever Mac client. It is not a live bidirectional AI transport; see `realtime-bridge-research.md`.

Archives are immutable local candidates; public download page remains 0.5.1 until the new-character gameplay check is complete.

- Windows SHA-256: `c93b17561fd8ce4eb31f4636899381262eb9078d349ac9aad991b73a14978198`
- Mac arm64 SHA-256: `415ad3601f74a73bf91753e45c702242c59b44f69a382dc785e9cb79cf44c23f`
