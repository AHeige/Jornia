# Mac distribution signing

Checked again on 9 November 2026. The user has Apple Developer access. This Mac currently has **zero valid code-signing identities**. Command Line Tools and `notarytool` are installed; full Xcode is not installed. Apple Developer membership alone does not sign Jornia.

1. Create or import a **Developer ID Application** certificate together with its private key into this Mac's login Keychain. An Apple Development or Mac App Distribution certificate is not sufficient for this distribution. If importing an existing identity, do so locally in Keychain Access; do not post a P12, password or private key in chat or Git.
2. Store notarization credentials locally using interactive `xcrun notarytool store-credentials Jornia` (Apple ID, team ID and app-specific password, or an App Store Connect API key). Secrets stay in Keychain. Do not paste them in a shell command, repo or chat.
3. Run `security find-identity -v -p codesigning` to obtain the installed identity's SHA-1.
4. Build the candidate with `python3 scripts/package_companion.py --platform darwin`. The Mac candidate contains `Jornia.app`. It is **not a trusted distribution** until the following succeeds.
The 0.8.0 archives already exist. Do not rebuild or replace published artifacts; the native pilot needs a new version and release notes. The command below illustrates signing the existing browser-only candidate, not a native-chat release.

5. Run `python3 scripts/sign_macos.py dist/jornia-0.8.0-darwin-arm64/Jornia.app --identity IDENTITY_SHA1 --notary-profile Jornia --output dist/jornia-0.8.0-macos-arm64-notarized.zip`.

The script discovers and signs nested Mach-O binaries and code bundles from the inside out, then signs the app bundle with hardened runtime and secure timestamp, submits to Apple, requires `Accepted`, staples and validates the ticket, requires Gatekeeper assessment, then creates an immutable final ZIP and SHA-256. It never disables Gatekeeper or strips quarantine. No relaxed entitlements are requested. Test the **downloaded final ZIP**, including first launch, on a clean Mac with default security settings before recommending it to friends.

The standard companion uses loopback HTTP and the normal browser, not Electron. The experimental screen capture helper is not part of this app; the signer now includes nested executables, but packaging the helper still requires a permissions UX, a standalone runtime and a new notarization/test pass. See [the Mac pilot checklist](mac-pilot-release.md).

Windows remains unsigned until a separate Authenticode/Artifact Signing identity is available. Apple's certificate cannot sign Windows. Signing and notarization establish publisher/integrity and Apple's automated checks; they cannot guarantee that all antivirus products will accept a download.

Sources: [Apple Developer ID](https://developer.apple.com/developer-id/), [Apple notarization requirements](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution), [custom notarization workflow](https://developer.apple.com/documentation/security/customizing-the-notarization-workflow).
