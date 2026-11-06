# First Ashenvale slice: verification

Date: 2026-11-05.

The first sections record the original fixture milestone. The reviewed-data follow-up below supersedes its statements about the absence of `forever` knowledge.

Story: a local HTTP question reaches the Go API, retrieves a version-scoped PostgreSQL graph, and returns either a deterministic answer or a grounded Berget response with provenance and inference accounting.

## Evidence

| Boundary | Result |
| --- | --- |
| Docker Compose startup | PostgreSQL and API healthy; bootstrap completed successfully |
| Database schema and seed | Checksummed migration applied; repeat migration and seed succeeded; six fixture entities without duplicate inserts |
| HTTP → PostgreSQL → answer | All 12 fixture evaluation questions passed against the running container |
| Unsupported game version | Requests defaulting to `forever` returned unknown; no silent fixture fallback |
| Spoilers | Hidden synthetic story excluded from default graph responses and provider context; accessible only with explicit spoiler consent in ask requests |
| Provider adapter | Fake-provider tests cover authentication/request shape, provider errors, incomplete output, invalid JSON, missing usage, and partial usage |
| Evidence validation | Fabricated evidence ID withheld; inference usage retained in the unknown response |
| Live Berget | One real generated answer passed the live evaluation |
| Credentials | `.env` ignored by Git and excluded from Docker build context; key not included in responses or request logs |

`go vet ./...` and the Go test suite passed with `TEST_DATABASE_URL` set, including the isolated-schema PostgreSQL integration test. There is no browser UI in this milestone; the evaluation script is the HTTP client.

## Real provider observation

Question: “Briefly explain where Astranaar is in the supplied fixture.”

Returned answer: “Development fixture (Forever unverified): Astranaar is located in Ashenvale.”

- Model: `mistralai/Mistral-Small-3.2-24B-Instruct-2506`.
- Provider-reported input tokens: 439.
- Provider-reported output tokens: 48.
- End-to-end request latency: 1269 ms.
- Estimated inference cost: EUR 0.0001461, using EUR 0.30 per million input and output tokens.
- Evidence: `entity:ashenvale`, `entity:astranaar`, `edge:astranaar-in-ashenvale`.
- Source: `wisp-ashenvale-fixture-v1`, revision 1, validation status `fixture`.

This single-call observation verifies connectivity, response shape, and basic grounding. It does not establish model quality across real game questions or Forever factual correctness. Its cost exceeds the handoff's aspirational EUR 0.00001 per-message target; deterministic answers cost zero inference tokens, and broader routing economics still need measurement.

## Local environment findings

The default Docker endpoint was unavailable. The `colima` daemon responded with filesystem input/output errors, so it was left alone. Wisp runs as its own Compose project on the healthy `colima-logos-graph` daemon. A pre-existing host PostgreSQL process occupied port 55439; Wisp uses 55471 instead, without modifying that process.

The external filesystem produced macOS AppleDouble sidecars. The migration loader skips dotfiles, and Docker/Git ignore these files.

## Remaining scope

The seed is original development data. Verified Forever knowledge, general intent routing, conversational memory, addon integration, public authentication, daily spend budgets, and production hosting remain future work. Source citation membership is checked mechanically; semantic faithfulness still needs evaluation.


## Reviewed Ashenvale follow-up

The API now serves 15 source-reviewed entities and 17 relationships under `forever`, alongside the six original fixture entities under their own version. Four source records use `source_checked`; none claims in-game verification. Source review method, applicability limits, attribution and licence are returned by the API.

- Go vet and tests passed with PostgreSQL integration enabled, including repeat migrations/seeding, quarantine of unverified-source data, default-version answers, nested-place traversal, NPC/quest relationships, source metadata and fixture separation.
- All 30 HTTP acceptance cases passed against the rebuilt running stack: 12 fixture cases and 18 reviewed-data cases.
- The 14-question comparison found six newly answerable questions. All four original non-synthetic location relationships were supported by the review, with no established contradiction for those claims. The graph grew from five non-synthetic entities to 15; the synthetic story remains excluded from `forever`.
- One real Berget request about Raene Wolfrunner returned: “Raene Wolfrunner is located in Astranaar, an Alliance travel hub in Ashenvale.” It cited `entity:astranaar`, `entity:raene-wolfrunner`, and `edge:raene-in-astranaar`.
- Model: `mistralai/Mistral-Small-3.2-24B-Instruct-2506`; input tokens 759; output tokens 61; latency 1391 ms; estimated cost EUR 0.000246 at EUR 0.30 per million input and output tokens.

Evidence and limits: [source review](ashenvale-research.md), [old/new response comparison](ashenvale-comparison.json), [data attribution](licenses/ashenvale-data.md). This is a small community-source-backed slice, not an exhaustive zone dataset or a client-build verification. The review used indexed excerpts because direct wiki fetches returned HTTP 403.

## First playable companion slice

The first user-test bundle is built for Windows amd64 and macOS arm64. It includes a Lua addon, a self-contained companion binary, compiled local UI assets, installation walkthrough, optional Windows addon-copy script, dependency notices, and SHA-256 checksums. Archives were inspected for private SQLite databases, `.env` files, the actual configured Berget credential, node_modules, and macOS AppleDouble sidecars; none was present. Windows output was identified as PE32+ amd64; macOS output as Mach-O arm64. The packaged macOS binary was actually started and its embedded UI, CSRF token, empty local state and SQLite creation were checked. Cross-compilation does not establish successful execution on Windows.

Verified local story:

1. Execute the actual addon Lua in a Lua 5.1-compatible simulation with a controlled test character and API stubs.
2. Import its real serialized export packet through the browser UI into the companion's SQLite database.
3. Ask a world question through companion → central Go API → PostgreSQL → cited response.
4. Ask a memory question entirely locally using imported observations and the explicit player note.
5. Restart the companion process with its backend intentionally unavailable; read the same saved history and answer another local memory question.

The reproducible browser test also verified five rapid submits creating one transport request, repeat import without duplicate events, rejected requests preserving drafts, a lost post-commit response recovering from a persisted receipt with no replay, separation between two characters, and mobile layout without horizontal overflow. Run `PLAYWRIGHT_BROWSERS_PATH=/Volumes/Extern/CodexCaches/ms-playwright make e2e` on this Mac. Browser output and test data stay in ignored `.cache/`; screenshots are `companion-desktop.png` and `companion-mobile.png` in this directory.

Go unit/integration tests passed, including a privacy regression test that forgets one character, preserves another, and removes the raw deleted character ID from the retained deletion receipt. Model-step tests execute the actual LangChain ChatOpenAI integration against a fake provider and check usage accounting. A real browser AI question then exercised companion → Go retrieval → private Python/LangChain model step → Berget and displayed citations and estimated cost. An explicit Swedish-language follow-up returned “Raene Wolfrunner finns i Astranaar.” Model generation remains bounded with no agent loop, automatic retry, or LangSmith tracing. See `companion-live-answer.png` for the live-answer UI.

The installed public graph is now 40 entities/47 relationships with five source reviews. Package validation and all 30 existing API evaluation cases passed. The coverage endpoint explicitly reports incomplete coverage. A full QuestieDB import was not performed because its own licensing document describes unresolved historical rights and its Forever documentation disclaims complete new-content coverage; details are in `knowledge-expansion.md`.

## Windows execution verified (2026-11-05)

With the user's explicit transfer/execution approval, the exact Windows release ZIP was uploaded through Chrome Remote Desktop to DESKTOP-0NR0QR4, extracted on its Desktop, and its packaged `wisp.exe` started without administrator access or security-warning bypasses. Windows Chrome opened `http://127.0.0.1:8787/` successfully. No runtimes or dependencies were installed separately.

The browser UI accepted a synthetic schema-1 journal for Testspelare/Testrealm (two events, level 24, Ashenvale/Astranaar, explicit note about Raene). Import reported two new events and zero duplicates. Asking `Vad gjorde jag sist?` produced the observed zone, level, timestamp and explicit note locally. The Wisp console reported `%LOCALAPPDATA%/Wisp/player.db`.

The Wisp console was closed and the same executable relaunched. A fresh browser tab loaded the existing character and conversation. Sending the memory question again produced a second correct answer, demonstrating successful reads and writes after actual Windows process restart. No connection to the Mac backend was configured; this verified local memory, not remote world knowledge. Screenshot: [Windows test](companion-windows-test.jpg). Synthetic test data and the running companion were left available for user inspection.

**Still unverified:** actual addon frame rendering/events in the user's Forever client, Windows addon-copy script behavior in that installation, and cross-machine connectivity to the Mac server. The user confirmed WoW has not yet been installed on this Windows machine. No addon was installed and no game files were modified. No Blizzard approval of the export design or public deployment is claimed.


## Companion 0.2.0 and source-backed quests (2026-11-05)

The graph now contains 49 reviewed entities, 67 relationships and seven source reviews. Six legacy fixture entities remain separate. Added NPCs and quests cover the Bathran’s Hair → Orendil’s Cure → Elune’s Tear → The Ruins of Stardust → Fallen Sky Lake sequence and Pridewings of Stonetalon. Chain ordering is not a guarantee of character eligibility. Direct wiki requests returned HTTP 403; the review explicitly records indexed community-source evidence and excludes conflicting retail facts. No in-game verification is claimed.

All 46 API acceptance cases passed locally and against the VM (12 fixture, 34 reviewed-version cases), including unknown rewards, XP, levels and missing chain endpoints. Go unit/integration tests and vet passed. The real LangChain/Berget VM smoke test returned Raene in Astranaar with 767 prompt tokens, 61 completion tokens, EUR 0.0002484 estimated cost and 1415 ms model latency.

The browser E2E passed repeated demo activation (five clicks, one transport request), recovery after a lost response, no duplicate events, conversation preservation, character isolation, reload, actual companion process restart, offline local memory, reduced-motion coverage and mobile overflow checks. Demo records use a reserved synthetic identity and source classification, with an explicit banner and answer prefix; they cannot silently become normal imported observations. The connection check calls only healthz and incurs no inference.

Windows 0.2.0 was transferred and run on DESKTOP-0NR0QR4 with the user’s explicit approval. A dedicated Ed25519 identity was generated on Windows; only its public key was registered on the VM. The private identity remains in the protected local Wisp connection directory. The VM host key is pinned. Existing Testspelare history survived the update, and the separate three-event demo journal answered the memory question with an explicit synthetic-data label.

The actual Windows browser then completed companion → dedicated SSH tunnel → VM API → PostgreSQL retrieval → LangChain → Berget, answering “Raene Wolfrunner finns i Astranaar.” Its UI displayed source links, Mistral Small 3.2 and approximately EUR 0.000251. The health-only connection check reported connected. Screenshot: [Windows 0.2.0 connected](companion-windows-connected.jpg). No actual WoW installation or gameplay was available for addon verification.

A Windows graph-only follow-up asked “Vad kommer efter Orendil's Cure?” and returned Elune’s Tear, explicitly advising the player to check availability in game. Screenshot: [Windows quest chain](companion-windows-quest.jpg). The refreshed download archives contain the same tested companion binaries and updated START-HERE instructions; the already transferred Windows archive has the earlier documentation revision.

## Automatic local journey slice — 0.3.0 (2026-11-05)

Added opt-in addon recording, persisted companion folder permission, initial reads of all characters present in Wisp's own SavedVariables, automatic polling of later saves, and a per-character SQLite graph projection. Original observations remain local source evidence; the central server and Berget receive no journal context. Memory routing now supports “Visa min resa” and related fixed journey questions without any network request. This is an initial personal-memory slice, not general conversational reasoning over every character statistic.

Go tests/vet passed. Tests exercise real addon Lua in a Lua 5.1 simulation for first-use consent, no recording before acceptance, automatic new-character start after acceptance, login/logout events and pause. Data-only parsing rejects executable/truncated Lua and handles quoted byte escapes without executing the file. Store tests cover initial multi-character reads, stable deduplication, expected config revisions, pause/resume, saved-permission restart, deleted-character suppression, graph cascade/isolation, 205 observations split across three packets, rebuilding a missing graph from original events, and full history export beyond the UI's display window.

The actual companion browser E2E passed existing world/demo/memory scenarios plus the new automatic-journal flow: unchecked first-use consent, no pre-consent import, initial reads of two synthetic characters from a realistic WTF directory, five rapid activations with one POST, lost-response receipt recovery, later file-change import, pause/resume and offline process restart. Desktop/mobile and reduced-motion flows passed without horizontal overflow. Screenshots `companion-journey.png` and `companion-journey-mobile.png` show synthetic test fixtures, not data observed from WoW.

Limits: actual Forever addon UI/events/SavedVariables layout have not been exercised on the user's game client. The user has not installed WoW yet. Initial import reads only existing Wisp files; it does not recover historical adventures from generic settings or other addons. Collection covers sessions/areas/levels/completed quest IDs/notes, not inventory, equipment or active quests. SavedVariables delivery is delayed until WoW saves, normally reload/logout. The Windows 0.3.0 bundle was subsequently installed and verified; see the Windows update entry below.


## Windows updated to 0.3.0 (2026-11-05)

The user explicitly requested the Windows update. The release ZIP was uploaded through Chrome Remote Desktop, checked against SHA-256 `dd0cf7aa26b96f528bac6218932ac424c79277dee6e52ced2dd53c35f418109b`, and extracted to Desktop/Wisp-0.3.0. The old 0.2.0 Wisp process and its own SSH child were stopped; other programs were not stopped. Before launching the new executable, player.db and any sidecars were copied into `%LOCALAPPDATA%/Wisp/backups/before-0.3.0-<timestamp>`. The existing connection key was reused and never transferred.

The new executable ran with `-vm`, opened the browser, displayed the new automatic-journal first-use control, and preserved the existing three-event demo character and prior conversations. “Visa min resa” returned an explicitly synthetic local journey summary with three observations and two places, proving the new graph projection can be built from the existing Windows journal. Automatic WoW-folder reading was left paused because WoW is not installed. Screenshot: [Windows 0.3.0](companion-windows-0.3.0.jpg). No administrator rights or security-warning bypass was used; an inadvertently opened elevation prompt for PowerShell was declined and a normal console used instead.

The installed launcher is `Desktop/Wisp-0.3.0/wisp-0.3.0-windows-amd64/Start-Wisp-VM.cmd`. The prior version is retained as a fallback; it is not the current running companion. Cloud client updates and a friend-ready remote API enrollment are still proposed, not deployed: [distribution design](client-distribution.md).

## Private friend alpha 0.4.1 — 2026-11-06

Published https://wisp.logosab.com with public ZIP downloads and a separate authenticated tester gateway. The DNS A record was created in one.com without changing existing records. Caddy 2.11.7 validated and reloaded the appended Wisp virtual host using SIGUSR1; no proxy restart. Existing document-index HTTPS health was 200 before and after. All 14 non-Wisp containers retained their IDs and remained running; Wisp db/model IDs were also retained. Only the Wisp API was replaced and the isolated gateway added.

Validation passed:

- `go vet ./...`, `go test ./...`, and PostgreSQL integration tests against isolated schemas. New cases cover journal-only AI, allowed memory evidence, fabricated-reference rejection, explicit-context validation, cross-character exclusion, authenticated access, daily quotas, duplicate request IDs and revocation.
- Existing browser suite passed against the new targeted-retrieval API: repeated import, five-click single-flight, failure draft preservation, lost response recovery, character isolation, restart/offline memory and narrow layout. Initial attempts used an old API and then a port occupied by another process; that process was left untouched. The suite passed on an unused port with the new API.
- Journey browser suite passed consent-before-reading, multi-character import, file update, pause/resume, restart, graph persistence, offline answers and mobile reflow.
- `personal.mjs` passed no implicit upload, displayed preview, explicit journal sharing, synthetic marker, five clicks/one POST, lost post-commit response recovery without replay, sharing reset, credential-free export, normal/reduced-motion and narrow layout. Its model is synthetic; `companion-personal-ai.png` is labelled accordingly. The expanded question form was changed to normal document flow after visual inspection found it covering the answer.
- Real HTTPS gateway: anonymous health 401, authenticated health 200, private knowledge route 404, valid personal AI request 200 and repeated identity 409. TLS certificate validated normally. Downloaded ZIP bytes matched local SHA-256.
- The **packaged Mac arm64 client**, extracted from the public release, was started with a fresh private data directory and its default production endpoint. It saved a test code, imported the synthetic demo, previewed context, asked through local receipt handling → HTTPS gateway → Go knowledge API → LangChain → Berget, saved the answer and mapped memory citations back to local event links. The actual model answered: “Dina sparade demohändelser berättar att du har besökt Astranaar och Maestras Post i Ashenvale när du var nivå 24.” History export excluded the credential.
- Windows amd64 and Mac arm64 were cross-built. ZIP contents were scanned for actual Berget/Blizzard credentials and the newly issued tester code: none present. No env, database or private SSH files included. Windows 0.4.1 has **not** been executed on the remote Windows machine; the earlier installed version there remains 0.3.0. Actual Forever gameplay acceptance remains unverified.

Client 0.4.1 checksums:

- Windows: `e683755943e989ec91c8b3d2f614e28d3f81d5e87912ce6aa166fefd53e2a08b`
- Mac arm64: `576c4cf246679b96840fed5f15f9d70dbe601e14a8b91dfad52e1f78603200f5`

The access code was saved only in an ignored local invitation file, not in these docs or public releases. It expires after 30 days, with 50 accepted server requests per UTC day. Server request IDs and token hashes are durable; raw questions, journal excerpts and answers are not retained for server replay. An uncertain remote outcome still cannot be automatically reconstructed. Provider-side retention is not verified by these tests.

Official game-data ingestion was also deployed with build/locale-scoped metadata, source expiry, 28-day retention and a bounded worker scheduled every 30 minutes through the existing user's cron. The existing crontab was backed up and preserved. At initial rollout coverage reported 2,865 entities (including the 49 curated entities), 122 cached documents and 2,986 queued resources. These are metadata counts, not completeness or client verification. A bounded backfill was started; subsequent counts are dynamic.

Final delivery is **0.4.2**. 0.4.0/0.4.1 artifacts were retained unchanged while the desktop layout was corrected before handoff. The final browser test additionally asserts that a tall settings sidebar cannot push the question form away from the latest answer. Both final public ZIPs passed fresh credential scans and downloaded-byte checksum comparison. The packaged 0.4.2 Mac client also repeated the complete real HTTPS/LangChain/Berget journey question successfully. Windows execution and Forever gameplay remain unverified for this release.

Final SHA-256:

- Windows 0.4.2: `ce9e2bb4865ae689dc253306aec336ab2da81a26afc93495b721bb0fa9af6caa`
- Mac arm64 0.4.2: `c14f47ad235b8450f9777435968be4e91a19200a8f0c04f978a223b80cd0bacf`

The first scheduled-worker backfill completed normally after 200 bounded requests (23:18:17–23:21:37 UTC), increasing cached documents from 122 to 321. The 30-minute cron continues the queue; this is not a claim of a complete corpus. VM disk remained 16 GiB free. Wisp db/model restart counts remained zero.

## Visual download page and in-game direction — 2026-11-06

Published the redesigned download page at https://wisp.logosab.com with original forest concept art, actual companion demo imagery, and screenshots rendered from the new Lua UI prototype. The primary product direction is chat and journey inside WoW; the companion is background infrastructure. The page explicitly distinguishes this target from the external-chat 0.4.2 technical alpha.

The local renderer executes the real journal addon plus `apps/addon-preview/WispPanel.lua` using the pinned wow-addons-sandbox simulator. Chat/Journey tab switching, a note through the real journal slash-command handler, and the visibly disconnected chat-submit state were exercised. Example messages and character events are synthetic. The UI prototype is not included in the release addon or ZIPs. No actual Forever client, live in-game AI transport, or simulator parity was verified.

Validation: HTTP asset allowlist tests and gateway vet passed; tester access isolation/limits integration passed. Desktop and 390-pixel layout, keyboard FAQ interaction, image loading and anchor navigation were checked. Public CSS/images returned 200, private/unknown paths remained blocked, authenticated health passed, and existing ZIP checksums were unchanged. Only the gateway container was replaced; all 17 other running containers retained their IDs. No Caddy change was required.

Public-page evidence: `download-visual-hero.jpg` and `download-ingame-preview.jpg`. Renderer setup, asset provenance and image-generation prompt are recorded in `visual-preview.md` and `scripts/preview/README.md`.

### English visual revision — 2026-11-06

Replaced the download copy with shorter English text, near-black surfaces, a full-width forest hero and large stacked product previews. Translated the unreleased Lua panel and its synthetic example messages to English and re-rendered both screenshots. The unchanged desktop alpha screenshot is labelled as older Swedish UI. Prototype/transport and current-release limitations remain explicit.

Checked desktop and 390×844 layout (no horizontal overflow), download anchor/ZIP targets and keyboard FAQ expansion. Published four static files with backup; all running container IDs unchanged. Public bytes match local HTML, stylesheet and both newly rendered images. Client ZIPs and runtime services were not modified. Public screenshot: `download-english-dark.jpg`.

## Character-first experience and real Armory — 2026-11-06

Final delivery is **0.5.1**. The earlier 0.5.0 archives remain unchanged. The personal source repository is `AHeige/Wisp` (private); Wisp's existing VM endpoint remains infrastructure and the landing page credits AHeige.

Implemented and directly rendered: night expedition palette/art, original lantern-spirit avatar, character-first navigation, character roster, scoped drafts/history, dated journey chapters, event filters and observed place trail. Fourteen surfaces are scored before/after in `ux-review.md`; asset prompts/provenance are in `design/visual-assets.md`.

Actual browser checks included fresh startup with zero characters and unchecked consent; approved discovery of two character directories and one Wisp file; six events attached without a duplicate character; per-character draft/history isolation; clearing sharing on switch; quest/level filters; and an unavailable-server question retaining its draft with no automatic replay. The packaged 0.5.1 Mac process restarted the same local database, retained the two-folder roster and six events, and distinguished a directory-only character from a recorded journey. A measured 390×844 viewport had no overflow, and portrait positioning was corrected after visual inspection. The published FAQ opened with Enter. Reduced-motion CSS removes animations/transitions while textual pending status remains.

The user-provided **Kungen Forever / EU** was fetched from actual Blizzard Forever profile/equipment/media endpoints: level 28, Night Elf Warrior, 15 equipped items and official portrait. The local UI displayed source dates and zero invented journal events for this character. A separate real production HTTPS gateway lookup confirmed the same fields and render host. Profile data is local and expiring; it is not world graph data or implicit AI context. Server credentials were not copied to the client.

The packaged 0.5.0 Mac client submitted one explicitly approved canonical demo-context AI request through the production HTTPS gateway → Go API → LangChain → Berget. Pending was visible immediately; the actual answer was “You visited Astranaar and Maestra's Post in Ashenvale at level 24.” Sharing reset after confirmation. Final 0.5.1 preserved that answer and displayed it with technical model/cost metadata inside its evidence disclosure, without issuing the paid request again.

New Go cases cover Forever source/identity/media-host validation, invalid names/regions, local profile receipt deduplication, scope/expiry/credential-free export, directory/journal identity reuse, expanded-consent gating, mapped-folder forget suppression and offline English memory prompts. `go test ./...` and `go vet ./...` passed. The PostgreSQL tester-access test ran against isolated schemas with the profile route consuming the same quota and rejecting anonymous/duplicate access. Existing headless UI runners were adapted and syntax-checked, not executed during this pass; the browser evidence above came from direct interaction with the real rendered apps.

The Lua simulator executed the redesigned panel without Lua errors; tab switching, notes through the actual journal handler and disconnected send with draft preservation passed. It remains an unreleased simulator prototype, excluded from the ZIP. There is no verified in-game AI transport or actual Forever gameplay acceptance.

VM changes replaced only the isolated Wisp gateway (final image `wisp-private-gateway:20261106-6`), retaining all 17 other running container IDs. No Caddy reload, exposed database port or other service restart was needed. Profile concurrency is capped at two and tester quotas/replay protection apply. Public HTML/CSS/images match local bytes; both downloaded 0.5.1 ZIPs match their checksums. Anonymous health remains 401, private knowledge and original artwork paths remain 404. Actual provider credentials and the tester code were scanned absent from source candidates and both ZIPs.

Final SHA-256:

- Windows: `6b52c9116a65785e169bca176809a77350f1d00ee559b91bde3b067a4c3331f6`
- Mac arm64: `d56b47de600ac4a21a4976cd68d0a4e3859ca07f6d98e2a96891cba18787342c`

Screenshots: `ux-download-night.jpg`, `ux-download-mobile.jpg`, `ux-kungen-character.jpg`, `ux-mobile-character.jpg`, `ux-journey-night.jpg`, `ux-companion-night.jpg`. Windows 0.5.1 execution, native Forever layout/events and the bidirectional game bridge are not claimed as tested. Directory discovery proves only locally present identities; it cannot recover past adventures or characters absent from game files.
