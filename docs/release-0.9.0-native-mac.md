# Jornia 0.9.0 — native Mac candidate

Built on 9 November 2026. This is a local pilot candidate, not a notarized public release.

## One app

`Jornia.app` includes a small first-party Cocoa control window, the Go companion, the Python standard-library reader with its own CPython runtime, the native capture helper, and the complete in-game QR chat addon with owned return-channel assets. Players do not need Homebrew, Python, Go, Xcode or terminal commands. The optional local journey dashboard still opens in the normal browser; this is not Electron.

The control window lets the player choose their Forever folder, install/update the addon, request local screen-capture permission, start/stop the connection, and open the local journey. Quitting stops the reader first, restores owned connection files, then stops the local server. A second active reader or app is refused. Interrupted sessions are preserved before recovery; a potentially live owner is never overwritten.

## Try it on this Mac

The locally installed candidate is `~/Applications/Jornia.app`. Choose `/Users/am/World of Warcraft/_forever_`, allow screen capture for Jornia, then start in-game chat. The existing addon is already installed on this Mac; no reinstall/reload is required for this launch test. Open `/jornia` after logging in and ask a question. Keep the chat visible until the answer arrives.

On a new installation, install the addon before the player's next normal game launch; the installer refuses to modify an open game. The game-folder selection does not silently grant journal recording/upload consent. Existing addon journal consent and desktop journal settings are preserved. The local journey dashboard exposes journal-import settings for first-time setup.

Existing `~/Library/Application Support/Wisp/player.db` is reused; otherwise data is stored under `~/Library/Application Support/Jornia`. The app does not import or bundle a developer database, session credential, provider secret or invite code. Enter an invite through the existing in-game Settings or local dashboard if needed. Screen images stay local; only submitted text and approved context go to the configured backend.

## Compatibility and distribution

- Apple Silicon only. This candidate requires **macOS 26+**, because the builder's CPython and linked libraries target macOS 26. The capture helper itself targets 14, which does not make the complete app compatible with 14. The packager derives the app minimum from every native binary.
- The app is ad-hoc signed for local development and passes nested signature verification. It is **not Developer ID signed or notarized**. Keep Gatekeeper enabled; do not share it as a trusted public download.
- No automatic app updater or background/closed-chat delivery is included. Native Windows is deferred.
- Existing public 0.8.0 downloads are unchanged and still do not contain this native bridge.

To build a fresh candidate without overwriting an existing directory:

```sh
python3 scripts/package_macos.py --output dist/jornia-native-candidate
python3 scripts/macos/smoke_candidate.py dist/jornia-native-candidate/Jornia.app
```

The build copies and relocates CPython and linked native libraries, includes their notices, rejects absolute developer dependencies, verifies imports with a clean environment, seals nested code and writes a file-hash manifest. Temporary sealing happens on APFS because signing on the external ExFAT workspace creates AppleDouble siblings; the checkout and build output stay in the external workspace. Only generated staging metadata is removed, never player-file quarantine or OS protections.

## Verified and pending

- Native window rendered and folder selection tested on the actual Mac.
- Starting while the previous developer reader owned the files showed a useful refusal and preserved its session.
- The previous owned developer processes were stopped gracefully; the app started its own local server/reader. macOS denied capture permission for the new app, which was shown as an actionable error, and owned files were restored. User approval is still needed for the new app identity.
- Packaged-runtime smoke tests use disposable game/data folders and a synthetic capture helper: start/ready/quit and denied-permission recovery both pass. Both retain the database, remove temporary credentials and restore every addon file hash. These tests make no paid AI requests and do not control the player's game.
- Five installer/session-ownership tests, eleven protocol tests, three signing-order tests and existing companion Go tests pass.
- The candidate ZIP was extracted into a fresh temporary folder. Executable permissions, nested signatures and both lifecycle smoke cases passed there too. HTTPS certificate validation succeeded against the protected backend (expected unauthenticated HTTP 401). This is not a clean-machine/Gatekeeper acceptance test.
- Live chat through the final installed app still needs the new screen-capture permission and a logged-in character. Earlier developer-installed live chat evidence is not a substitute for this check.
- Clean-machine installation, older macOS support, Developer ID signing, notarization, update/replacement and rollback remain release gates. See [the pilot checklist](mac-pilot-release.md).
