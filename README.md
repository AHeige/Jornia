# Wisp

An experimental World of Warcraft: Forever companion with a central knowledge graph and a private local adventure journal.

**Product direction: chat and journey belong inside WoW in a Lua interface.** The external chat in 0.5.1 is a technical alpha. The companion is background infrastructure; a verified bidirectional game bridge remains to be built. See [ingame experience](docs/ingame-experience.md) and the [rendered Lua prototype](apps/addon-preview/WispPanel.lua).

The first end-to-end slice includes a Lua addon with user-initiated export, a self-contained Go companion with SQLite and a local chat UI, a central Go/PostgreSQL knowledge API, and a bounded Python/LangChain model step calling Berget. First-use opt-in enables automatic reads of Wisp SavedVariables from the selected WoW folder and a per-character journey graph in SQLite. Explicit per-question preview can now include recent journal excerpts in an AI answer. There is no agent harness, mandatory player account, automatic personal-history upload, or real-time bridge back into WoW.

Personal repository: **https://github.com/AHeige/Wisp** (private).

Private alpha download: **https://aheige.github.io/wisp-site/** (0.5.1). Testers receive a separate revocable access code; no provider key is distributed.

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

## Development

```sh
cp .env.example .env  # only if .env does not already exist
make up
make companion
```

On this Mac use `DOCKER_CONTEXT=colima-logos-graph make up COMPOSE=./.cache/docker-compose`. Berget credentials stay in the model service's environment; they are not included in companion bundles.

```sh
make check
TEST_DATABASE_URL='postgres://wisp:wisp-local-only@127.0.0.1:55471/wisp?sslmode=disable' make integration
make knowledge-check
make eval
make e2e       # requires an installed Playwright Chromium; see test docs for this Mac's cache path
make package   # creates Windows amd64 and macOS arm64 test ZIPs under dist/
```

The Windows test bundle needs no Go, Python, Node, Docker or separate database installation. Start-Wisp.cmd uses the HTTPS endpoint and a separately supplied private test code. For the earlier developer Windows installation, `Start-Wisp-VM.cmd` opens a dedicated, host-key-pinned SSH tunnel to the VM. `Start-Wisp.cmd` runs locally without it; local memory remains usable offline. Select “Prova med demokaraktär” to try three explicitly synthetic Ashenvale events. The authenticated tester gateway and public release downloads run separately from the private API.

The curated graph has 49 entities and 67 relationships around Ashenvale, alongside a growing build-scoped Blizzard API metadata cache. **It is not complete WoW Forever information.** Source reviews are community evidence with explicit limitations; the old synthetic fixture remains under its own version.

## Character-first experience (0.5.1)

Journey is the home screen. Switch characters through the persistent selector or roster; drafts, journals and conversations stay separate. First-use consent can discover all locally present character directories and import existing Wisp journals. An optional approved public Forever lookup adds the real portrait and equipped items with source dates, without turning Armory into live state or common world knowledge.

See the [UX assessment and iteration log](docs/ux-review.md), [start guide](docs/wow-first-test.md), [release notes](docs/release-0.5.1.md) and [visual asset provenance/prompts](docs/design/visual-assets.md). The in-game Lua interface remains a separate, unreleased prototype.
