# Jornia

**Journey in Azeroth.** A personal companion grounded in your current quest log.

An experimental World of Warcraft: Forever companion with a central knowledge graph and a private local adventure journal.

**Start with the transport research:** [Sending text into WoW through a sound API](docs/sending-text-through-wow-sound-api.md). Our Mac experiments reconstruct checked UTF-8 messages from reusable file-presence signals observed through `PlaySoundFile`, with local OCR acknowledgements. The same carrier filenames can transmit successive messages without consuming an inbox per reply. This is an experimental client behavior, not a supported IPC API or a Blizzard-approved integration.

Jornia combines a Lua addon, a Go desktop companion with private SQLite character journals, and a Go/PostgreSQL backend calling Berget. The addon maintains current character state; selected, dated state and journal excerpts can accompany a question. World facts retain source/build labels. SavedVariables persistence remains delayed until WoW saves; it is separate from the live transport.

**Status:** developer alpha. A real OCR → AI → native Lua exchange has been verified on Mac; reusable return blocks have separate measured success and corruption tests. Windows real-time transport, long-session reliability, signed installation and a self-contained setup for the current bridge remain unfinished. Historical experiment notes describe their own implementation stage, not necessarily today's complete product.

Source: **https://github.com/AHeige/Jornia**. Original software and documentation are MIT-licensed; see [LICENSE](LICENSE) and [NOTICE](NOTICE.md). Provider datasets, third-party material and game imagery are not relicensed by this repository. No provider database dump is included.

The existing [download page](https://aheige.github.io/wisp-site/) provides the 0.8.0 Windows and Mac desktop/journal alpha, not a packaged release of the current reusable native bridge. Testers receive a separate revocable access code; no provider key is distributed.

- [Client installation and VM-hosted updates](docs/client-distribution.md)
- [Automatic local journey: first-use consent, import and limits](docs/automatic-journal.md)
- [First WoW test: installation and walkthrough](docs/wow-first-test.md)
- [Companion mutation and recovery contract](docs/companion-mutations.md)
- [Knowledge scope, expansion and import limitations](docs/knowledge-expansion.md)
- [Architecture and legal/policy review](docs/architecture-and-legal-review.md)
- [Server startup and API contract](docs/local-development.md)
- [Ashenvale source review and comparison](docs/ashenvale-research.md)
- [Verification results](docs/verification.md)
- [Original project handoff](docs/project-handoff.md)

## Naming and compatibility

The product is now Jornia (formerly Wisp). Use `/jornia` or `/jorniamini` in game; `/wisp` and `/wispmini` remain aliases. Existing addon folder/SavedVariables IDs, transport markers, headers and deployed infrastructure identifiers retain their legacy names to preserve compatibility. Existing local `Wisp/player.db` is reused; new installations use a Jornia data directory. Historical captures and raw experiment records retain the names actually observed.

The current friend-test backend is quest-log-only. ForeverDB refresh is paused and its stored facts are excluded from served knowledge. No ForeverDB key is needed for this flow. See [the current delivery checklist](docs/questlog-companion-delivery-2026-11-08.md).

## Development

```sh
cp .env.example .env  # only if .env does not already exist
make up
make companion
```

On this Mac use `DOCKER_CONTEXT=colima-logos-graph make up COMPOSE=./.cache/docker-compose`. Berget credentials stay in the backend environment; they are not included in companion bundles.

```sh
make check
TEST_DATABASE_URL='postgres://wisp:wisp-local-only@127.0.0.1:55471/wisp?sslmode=disable' make integration
make knowledge-check
make eval
make e2e       # requires an installed Playwright Chromium; see test docs for this Mac's cache path
make package   # creates Windows amd64 and macOS arm64 test ZIPs under dist/
```

The Windows test bundle needs no Go, Python, Node, Docker or separate database installation. Start-Jornia.cmd uses the HTTPS endpoint and a separately supplied private test code. For the earlier developer Windows installation, `Start-Jornia-VM.cmd` opens a dedicated, host-key-pinned SSH tunnel to the VM. `Start-Jornia.cmd` runs locally without it; local memory remains usable offline. Select “Try a demo character” to try three explicitly synthetic Ashenvale events. The authenticated tester gateway and public release downloads run separately from the private API.

The original curated Ashenvale graph is supplemented by build-scoped Blizzard and ForeverDB importers. Source counts and usable detail coverage change as resumable imports run; catalog entries do not establish complete gameplay coverage. **It is not complete WoW Forever information.** Source reviews are community evidence with explicit limitations; the old synthetic fixture remains under its own version.

## Character-first experience (0.5.1)

Journey is the home screen. Switch characters through the persistent selector or roster; drafts, journals and conversations stay separate. First-use consent can discover all locally present character directories and import existing Jornia journals. An optional approved public Forever lookup adds the real portrait and equipped items with source dates, without turning Armory into live state or common world knowledge.

See the [UX assessment and iteration log](docs/ux-review.md), [start guide](docs/wow-first-test.md), [release notes](docs/release-0.5.1.md) and [visual asset provenance/prompts](docs/design/visual-assets.md). The in-game Lua interface remains a separate, unreleased prototype.
