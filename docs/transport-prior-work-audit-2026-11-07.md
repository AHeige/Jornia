# Wisp transport prior work audit

Research date: 7 November 2026. This is a technical prior-work review, not a patent search or a determination of Blizzard policy compliance.

## Finding

The general mechanism is established: observable shared resource state can encode bits, with a clock and acknowledgement. Lampson's 1973 paper already gives a file-interlock example. In WoW Forever, rdimascio publicly reported the missing/present distinction on 28 September 2026, and NeverQuestAlone already documents repeatable file-presence observations through PlaySoundFile. Wisp learned that primitive from reviewing its implementation.

The narrower distinction supported by this review is that Wisp's lab uses multiple presence observations to carry the reply payload itself, repeatedly through the same filenames. The closest inspected implementations instead use presence as notification for one-shot Lua slots, or recover reply bytes from a finite font bank. We did not identify an exact match to the complete Wisp lab among the selected sources. That result cannot establish worldwide first use or absence of unindexed/private work.

Publish this as an implementation and measurement report building on prior work. Do not present it as discovering a new communication principle, the first AI inside WoW, or the first use of the sound API as a signal.

## Strongest preceding work

1. **1973: shared resource state as data and clock.** Butler Lampson's [A Note on the Confinement Problem](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/acrobat-24.pdf), example 5, pages 2–3 of the hosted copy, represents Boolean values through file interlocks and sends bits using data/sendclock/receiveclock. The paper credits examples 5 and 6 to A. G. Fraser. This is strong conceptual prior work, not a WoW implementation or a PlaySoundFile discovery. Do not confuse the web crawler's recent hosting date with the paper's original October 1973 publication.
2. **NeverQuestAlone: the directly relevant WoW primitive.** [Protocol section 3](https://github.com/tommygeoco/neverquestalone/blob/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07/docs/PROTOCOL.md#3-bridge--game-doorbells) explicitly describes the present/deleted/recreated behavior of previously known files and truthy results independent of playable contents on its tested build. The [signal writer](https://github.com/tommygeoco/neverquestalone/blob/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07/bridge/transport/signals.mjs) creates/removes empty files; the [Lua receiver](https://github.com/tommygeoco/neverquestalone/blob/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07/addon/NeverQuestAlone/Transport.lua) observes bells. Reply contents are separately published through finite slots in protocol section 4. Its snapshot predates our experiment; repository dates do not establish the first-ever public disclosure of every idea.
3. **wow-ai: native AI with slots and sound readiness checks.** [Architecture](https://github.com/chelinho139/wow-ai/blob/60165ca5bb6c6452f18604dc7b4fe8678b607bed/docs/ARCHITECTURE.md) and [Lua implementation](https://github.com/chelinho139/wow-ai/blob/60165ca5bb6c6452f18604dc7b4fe8678b607bed/addon/WoWAI/WoWAI.lua) show optical outbound messages, sound-based checks and LoadAddOn delivery. They do not show the Wisp presence-bit payload bus in the reviewed files. Its empty-versus-playable sound explanation differs from the truthy-empty-file behavior measured in our client; build-specific empirical behavior matters.
4. **Forever Bridge: another native byte return path.** [README and implementation](https://github.com/0xInuarashi/wow-forever-codex/tree/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8) describe checked reply bytes encoded in measured font widths. A bank of 65,535 previously unused names is finite. This rules out claims that Wisp is the first to return external bytes to native Lua without a reload per exchange.
5. **2014: covert channels inside WoW already studied.** Brian Rowe and Daryl Johnson's [A Behavior-Based Covert Channel in a MMO](https://www.worldcomp-proceedings.com/proc/p2014/SAM4152.pdf) transfers bits between game clients using cast behavior. It is a different path from our desktop-to-addon carrier, but broad claims to inventing a WoW side channel would also be unjustified.

## An earlier public sound-presence report

[chelinho139/wow-ai PR 15](https://github.com/chelinho139/wow-ai/pull/15), opened **28 September 2026 at 20:47:19 UTC**, is a particularly important additional source. rdimascio reports missing → nil and empty/garbage/truncated/valid → true on Mac, and proposes presence-based signals. Head commit: `a1800ed116fe6067a0b007b45f21246f5b32c665`; the PR was open when checked. We retained its patch and metadata as well as reviewing the text. It proves this observation was already publicly reported; its claimed platform generality is not established by our review.

The maintained [rdimascio/claude-wow architecture](https://github.com/rdimascio/claude-wow/blob/b22f5de8f7e95de4bfbd4ee58adf1bd8d4dfbaf0/docs/ARCHITECTURE.md) still describes replies through a 200-slot inbox pool, rather than a presence-bit payload bus. Selected signal code was read as well. [PR 53](https://github.com/rdimascio/claude-wow/pull/53) has a useful contrary platform observation: a comment reports deleted files still reading as present on Classic Era. This is evidence against assuming the mechanism is portable to every WoW client, not a fresh Wisp live measurement.

## Additional nearby implementations

- [VIXAL-OS/wow-agent-bridge](https://github.com/VIXAL-OS/wow-agent-bridge/tree/960fc5a51e397b33e8b5ec077ebf5cea0fed91d8): checked font-metric return on 3.3.5a. Its README notes that empty and valid files both return 1 through PlaySoundFile on that client. That is an earlier related observation, although it does not by itself demonstrate toggling presence to send payload bytes.
- [MrDoSSS/wow-forever-assistant](https://github.com/MrDoSSS/wow-forever-assistant/blob/b0a55f314796ecab64300f8a51e3ae8181653857/addon/ForeverAssistant/Transport/Inbound.lua): the inbound implementation advances through unused slots, with exhaustion handling. The bridge delivery writer defines a 200-slot pool.
- [DSilence/t3wow](https://github.com/DSilence/t3wow/tree/fd2abfce1625c3c01e20a296bed61dcce8c699f4): documentation describes finite live-update slots and screenshot-based outgoing messages. Documentation-only comparison in this audit.
- [btsouth/hermes-wow](https://github.com/btsouth/hermes-wow/tree/cb4466edd64c74b2bdb8686e643c73e3af880e5b) and [Nardo86/azeroth-companion](https://github.com/Nardo86/azeroth-companion/tree/dfc69c031872cefbf88d9a8bd1aff5e6482e1ca9): reviewed documentation describes reload/SavedVariables synchronization. This is weaker evidence than auditing every current transport source file.
- Other READMEs were screened to expand discovery, not certified as lacking the mechanism. See scope below.

## Search method and limits

The review combined general web searches, GitHub repository/code searches, selected code retrieval and older forum/paper searches. Queries included PlaySoundFile with bridge, presence, IPC, doorbell, binary, payload, bitstream, rename and communication; and WoW/Warcraft with file existence, reusable return, sound API, named pipe and covert channel. Search hits describing injection, ordinary music playback, external overlays or client-to-client addon messages were distinguished from the specific native return path.

Selected GitHub issues and pull requests were searched and read, including the September presence-signal report. GitHub searches were exploratory. Some results were broad or noisy, phrase hits included our now-public article, and search indexes cannot be assumed complete. Wisp's own repository was excluded as independent evidence. No zero-result query was treated as proof of novelty. General searches for AI repositories included many irrelevant projects; the audit does not claim to have inspected all results or all forks.

For the closest projects, source snapshots were pinned to the default branch commit resolved during this audit. Only selected relevant files were read, not all files/history in each repository. No third-party project was installed or executed, and the other authors' claimed live behavior was not independently retested. Private Discord messages, deleted repositories, inaccessible forums, unindexed branches, every issue/PR and nonpublic research remain outside coverage.

| Repository snapshot | Commit prefix | Review scope |
|---|---|---|
| [rdimascio/claude-wow](https://github.com/rdimascio/claude-wow/tree/b22f5de8f7e95de4bfbd4ee58adf1bd8d4dfbaf0) | `b22f5de8f7e9` | Selected signal/protocol source, architecture and public PRs |
| [0xInuarashi/wow-forever-codex](https://github.com/0xInuarashi/wow-forever-codex/tree/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8) | `85953dbecbc2` | Selected transport/protocol files and documentation |
| [DSilence/t3wow](https://github.com/DSilence/t3wow/tree/fd2abfce1625c3c01e20a296bed61dcce8c699f4) | `fd2abfce1625` | README only |
| [LoveWood233/wow-airi-companion](https://github.com/LoveWood233/wow-airi-companion/tree/654f6b5e90f1ce093f890372770cb7a7129dd52d) | `654f6b5e90f1` | README only |
| [MrDoSSS/wow-forever-assistant](https://github.com/MrDoSSS/wow-forever-assistant/tree/b0a55f314796ecab64300f8a51e3ae8181653857) | `b0a55f314796` | Selected transport/protocol files and documentation |
| [Nardo86/azeroth-companion](https://github.com/Nardo86/azeroth-companion/tree/dfc69c031872cefbf88d9a8bd1aff5e6482e1ca9) | `dfc69c031872` | Protocol and architecture documentation only |
| [Qiliang/WowRadar](https://github.com/Qiliang/WowRadar/tree/285b8f12943240d2b911ba6441417e5edd9716b8) | `285b8f129432` | README only |
| [VIXAL-OS/wow-agent-bridge](https://github.com/VIXAL-OS/wow-agent-bridge/tree/960fc5a51e397b33e8b5ec077ebf5cea0fed91d8) | `960fc5a51e39` | Selected transport/protocol files and documentation |
| [btsouth/hermes-wow](https://github.com/btsouth/hermes-wow/tree/cb4466edd64c74b2bdb8686e643c73e3af880e5b) | `cb4466edd64c` | Selected documentation only |
| [chelinho139/wow-ai](https://github.com/chelinho139/wow-ai/tree/60165ca5bb6c6452f18604dc7b4fe8678b607bed) | `60165ca5bb6c` | Selected transport/protocol files and documentation |
| [cuter74-dev/wowTerminal](https://github.com/cuter74-dev/wowTerminal/tree/28fde0155f56e35eccbdd22864b6ecb15df909aa) | `28fde0155f56` | README only |
| [newtype-01/wow-codex](https://github.com/newtype-01/wow-codex/tree/7a664320ef90a67c799fe78794023edc7afe46c7) | `7a664320ef90` | README only |
| [nukeador/wow-context-bridge](https://github.com/nukeador/wow-context-bridge/tree/41246a41e028273c7c322acd559abdd4c379eab7) | `41246a41e028` | README only |
| [tommygeoco/neverquestalone](https://github.com/tommygeoco/neverquestalone/tree/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07) | `5eb98d5c33ad` | Selected transport/protocol files and documentation |

The read-only source snapshots, query results and manifests are retained locally under ignored `.cache/novelty-audit-20261107/`. They are research material, not vendored product code. Public links above allow readers to inspect the same revisions. Git timestamps help pin a comparison; they are not proof of original authorship or the exact first publication time.

## Wording to use

> We implemented and measured a reusable file-presence payload channel into native WoW Lua through PlaySoundFile. It builds on previously published presence signals, including rdimascio's report and NeverQuestAlone's implementation, and established shared-resource communication techniques. We have not found the same complete arrangement in the reviewed sources, and we do not claim first invention.

The defensible result is measured implementation behavior: repeated carriers, checked UTF-8 transfer and rejection of selected malformed packets on a stated build. Long-session portability, a favorable policy decision, universal safety and a worldwide novelty claim do not follow from those results.

## Changes made after this review

The public article now cites the September report and Lampson/Fraser, labels the general protocol principle as established, and narrows its comparison to reviewed implementations. Its original measurements remain unchanged. No external forum post is part of this audit.
