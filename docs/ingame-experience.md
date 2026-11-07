# Wisp is in-game first

Confirmed by the user on 6 November 2026: **AI chat and journey management belong inside WoW, in Lua.** The desktop companion supplies background storage, connectivity and updates. External chat in 0.4.2 is a technical alpha, not the intended final experience.

## The view

`apps/addon-preview/WispPanel.lua` is an executable, unreleased interface prototype with Conversation and My Journey tabs. The Lua renderer loads it alongside the existing journal addon. Fixture code supplies a demo character and clearly labeled example answers. The prototype is not part of the published 0.4.2 ZIP and changes no installed WoW files.

Simulator verification covered tab switching, actual synthetic WispDB journal entries, adding a local note and explicit send rejection when the AI bridge is absent. The prototype runs no model. At this stage its new UI behavior had not been verified in Forever.

## Data responsibilities

- The addon owns current state through available game APIs and its in-memory journal. A question needs a new client snapshot, not the desktop application's older SavedVariables copy.
- The companion owns the durable local archive, bounded private context, authenticated server connection and updates. Journals do not become shared world knowledge.
- The server owns sourced world-fact selection, versions, cost limits and the thin LangChain/Berget step used at this stage.
- Replies must be bound to question, character and session. Reconnecting must not replay paid questions or deliver an answer to another character.

## Original transport investigation

Moving a chat field into Lua does not replace the bridge. An automatic, verified channel is needed out of the client and back. The user initially objected to pixel transport; it must not be adopted silently.

Primary reference: [wow-ai](https://github.com/chelinho139/wow-ai) describes a Forever implementation using prepared load-on-demand addons for replies and a visual outbound channel. This is a candidate inbound pattern, not Wisp verification, Blizzard permission or evidence of a working outbound log channel. No third-party bridge was installed.

The next verification at that stage required a real Forever client: measure log-write latency and controlled answer loading. Distinguish quest shown/accepted/abandoned/turned in; test rapid questions, interruption, restarts and character switching. Rendering and an external AI chat do not prove the in-game chain.

The download page explicitly distinguishes the intended in-game experience, executable Lua prototype and downloadable external technical alpha.

## Two window modes, 6 November 2026

The Lua visual prototype now supports a full companion window and a movable/resizable compact chat, sharing drafts, messages and pending state. See [window modes and verification](lua-window-modes.md). This remains a separate UI prototype; it has not replaced the live OCR lab or been bundled in the public ZIP. Subsequent transport work supersedes the early investigation notes above: a finite OCR → Berget → Lua roundtrip and a separate reusable block-return experiment have been proven, while the newest snapshot/waypoint slice still has explicit live-test limits. Go now owns server orchestration and talks directly to Berget.

The subsequent user request installed the two-mode UI into the Mac client. Native character/journal/artwork and resizing have now been tested. At that installation stage the new view explicitly showed its AI chat was disconnected; the OCR lab remained separate. See `lua-window-modes.md` for installation commands, backups and that transport boundary. Later integration work is tracked in [the delivery checklist](foreverdb-chat-delivery.md).
