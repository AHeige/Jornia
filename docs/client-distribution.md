# Private tester alpha distribution and AI connection

Updated 6 November 2026: website/downloads **https://aheige.github.io/wisp-site/**, protected API **https://wisp.logosab.com**. At this stage the client was 0.6.0, a portable ZIP for Windows amd64 and Mac arm64. Published versioned files and SHA-256 checksums are retained; new builds receive a new version number. This alpha has no signed installer or automatic updater. Download the next ZIP and close the older process before starting it; the same local data directory is used.

## Player flow

Download, extract and run `Start-Wisp.cmd`. Enter a separately supplied private test code under AI connection. The journal can be tried in demo mode without WoW and works locally offline. START-HERE explains addon installation and separate consent for recording and file import. Using journal excerpts in AI requests is a separate per-question choice with a preview.

The test code is a random 256-bit secret. It is sent in an Authorization header over HTTPS, stored only in the companion's local SQLite settings, and excluded from history exports and operation receipts. The public ZIP contains no test codes, provider secrets or private SSH keys. The server stores only the code's SHA-256 hash, label, expiry and quota. The first friend invitation lasts 30 days with at most 50 server requests per UTC day; local memory questions do not count. An invitation is personal and must be sent separately from the public link.

Revoke an invitation by setting `tester_access.revoked=true` for its label/hash in Wisp's database. The next server request is rejected while the local journal remains available. This is private test access, not a complete account system or paid product.

## VM boundaries

- Dedicated Wisp gateway limited to 96 MiB and 0.25 CPU; only downloads, the landing page, authenticated health and the authenticated ask route are exposed.
- Existing Caddy handles TLS for a separate virtual host. Only the Wisp gateway shares the proxy network. Neither the database nor model proxy is publicly exposed.
- The Wisp API retains a global limit of six model attempts per minute and two concurrent attempts. At this deployment stage, Python/LangChain calls Berget with bounded output and no automatic retries or tracing.
- Quotas and request identities persist in PostgreSQL. The same code/request ID is accepted only once, including after restart. A repeat returns 409; no model attempt is automatically repeated. The server does not retain question text, journal excerpts or model answers for replay. A lost result can therefore still require attention.
- Private journal context is not treated as world knowledge. The Wisp API logs aggregate request metadata only. This does not guarantee external providers' retention behavior.

The Caddyfile and Wisp configuration/database were backed up before the change. Caddy 2.11.7 reloaded the validated file through SIGUSR1 without a container restart. The existing document index returned HTTPS health 200 before and after. All 14 unrelated containers retained their IDs and kept running. Neither another application nor the VM was restarted.

## Future signed installation

MSIX/App Installer over HTTPS is a possible next step. It requires a trusted signing identity, stable package identity and verified data-directory migration. Build/sign outside the VM and keep signing secrets out of release directories. Preserve manual recovery and the offline journal if the update server cannot be reached.

Sources: [Microsoft update settings](https://learn.microsoft.com/en-us/windows/msix/app-installer/update-settings), [signing requirements](https://learn.microsoft.com/en-us/windows/msix/package/signing-package-overview), [packaged application file handling](https://learn.microsoft.com/en-us/windows/msix/desktop/desktop-to-uwp-behind-the-scenes), [Caddy reload signal](https://caddyserver.com/docs/command-line).
