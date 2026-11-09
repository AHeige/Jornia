# Mac pilot release

Decision: 9 November 2026. Ship a small Mac pilot first. Windows native chat and its installer are deferred. Do not replace the published 0.8.0 downloads with an untested native build.

## The first player journey

Download Jornia → move Jornia.app to Applications → open it → select the Forever game folder → install the addon before the player's next normal game launch → explicitly allow local screen capture → connect AI with an invite → open the in-game chat → ask a question with current character context → receive a reply in that chat.

The app must explain that capture is local, images are not uploaded, and only submitted questions and approved context go to the backend. No development tools, terminal commands or separate manually launched reader should be required. The existing per-character context opt-out and journal consent remain in force.

## Delivery order

- [x] Scope the first candidate to Apple Silicon. Final binary inspection found that the bundled CPython requires macOS 26, so this candidate declares 26+; macOS 14 support needs a compatible runtime, and Intel Macs are not validated.
- [x] Add a Mac-only selection to the existing packaging tool: `--platform darwin`. This still builds the older browser/journal bundle; it does not yet include native chat.
- [x] Prepare signing to include nested capture/runtime binaries before sealing the outer app, reject external symlinks, and verify the nested signatures.
- [x] Handle SIGTERM in the companion's graceful shutdown path so managed shutdown can remove temporary session credentials and close the local database.
- [x] Build one app lifecycle for the companion, reader and capture helper: one instance, private session ownership, startup ordering, visible status and graceful quit. Never delete another process's session to force startup. Packaged start/stop and permission-denied recovery pass with a synthetic capture helper.
- [x] Bundle the Python standard-library reader with a private relocatable runtime. Players do not need Homebrew, Xcode or Python installed. Runtime and linked-library licences are included.
- [x] Bundle the complete native addon and owned return-channel assets. The older journal-only download remains unchanged.
- [x] Add game-folder selection/validation, preserve SavedVariables and local journals, back up replaced addon files, and refuse installation while a game or bridge is running. Disposable-folder tests cover journal preservation, backups and interrupted carriers.
- [ ] Present screen-capture consent and actionable denied/revoked-permission states under the stable signed app identity. Request only the permissions actually needed.
- [ ] Verify manual recovery from a stopped reader/helper and an interrupted transfer. Restore only owned carrier files; preserve ambiguous originals and explain the failure. Never automatically repeat a paid question.
- [ ] Verify chat closure immediately after Enter, character switching, resizing/movement and repeated questions. If delivery requires the chat to remain visible, state that limitation clearly in the pilot; do not claim background delivery.
- [ ] Observe new player-performed quest progress with the journey already open, then ask a current-state question. Do not automate gameplay or require reload/exit for this acceptance test.
- [ ] Add a new release version and accurate native-pilot instructions; build an immutable candidate containing no credentials, local databases or capture output.
- [ ] Install a valid Developer ID Application identity locally and configure a Keychain notarization profile. There were zero valid signing identities at the 9 November check. Never put private keys/passwords in the repository or chat.
- [ ] Sign, notarize, staple and assess the complete app; publish its checksum. Test the actual downloaded artifact with quarantine intact on a Mac without development dependencies.
- [ ] Test the chosen update path: quit safely, verify the new signed app before replacing it, retain private data, and preserve a rollback. Manual replacement is sufficient for a first pilot if explicitly documented; automatic updates are not implemented by this checklist.
- [ ] Publish a clearly labelled Mac pilot and give it to a small number of testers. Leave Windows availability unchanged and distinguish its browser companion from native chat.

## Acceptance questions

The pilot should answer identity, current quest progress, remaining objectives and a grounded next step when evidence exists. Test `Who am I?`, `What am I doing?`, `What should I do next?` and a question about a quest just turned in by the player. It must distinguish a completed turn-in from an active quest ready to hand in, and admit missing route or lore evidence.

Data ingestion can continue independently. Complete world coverage is not a prerequisite for the pilot and must not be advertised. A live developer-installed Mac round trip is evidence for that setup, not evidence of clean-machine installation, trusted distribution or Windows support.

## Current release blockers

A self-contained local candidate now exists; see [0.9.0 native Mac notes](release-0.9.0-native-mac.md). The existing public 0.8.0 ZIP still contains the browser companion and journal addon only. The final installed app needs its own screen-capture approval and live acceptance test. Developer ID signing/notarization and clean-machine installation are outstanding. Do not publish a new download or declare this checklist passed until the unchecked delivery steps have evidence.

## Preparation checks, 9 November

Three bundle-inspection tests passed: nested signing order, rejection of links outside the app, and internal symlink deduplication. Companion tests and vet passed. An isolated app on an ephemeral loopback port exited successfully on SIGTERM, removed its temporary session file and retained its database. These checks did not stop the player's active companion or game. Actual Developer ID signing and clean-machine installation remain untested.

Related: [Mac signing](macos-signing.md), [personal context and live journey](personal-companion-context-2026-11-08.md), [current published package](release-0.8.0.md).
