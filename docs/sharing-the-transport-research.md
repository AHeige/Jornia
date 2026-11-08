# Sharing the Jornia transport research

Checked 7 November 2026. This is an outreach plan; no forum messages have been posted.

Lead with the measured reusable return channel, not claims of being the first AI addon or an approved transport. Link the article and the source, credit prior projects, and state the tested build and platform.

## Recommended order

| Priority | Community | Why it fits | Posting approach |
|---|---|---|---|
| 1 | [WoWInterface Developer Discussions](https://www.wowinterface.com/forums/forumdisplay.php?f=11) | Addon authors can evaluate client behavior, caching and Lua/API costs. | Choose General Authoring Discussion for a technical discussion. Ask for independent reproduction on other clients and sound settings. Check the selected board's rules before submitting. |
| 2 | [r/wowaddons](https://www.reddit.com/r/wowaddons/) | Direct audience for experimental addon development. | Use Development / WIP. The moderators explicitly reserve Release / Update for functional, tested addons. Disclose AI assistance in developing/documenting the project. |
| 3 | [Lua-l](https://www.lua.org/lua-l.html) | Embedded-Lua developers may appreciate a narrowly constrained host API becoming an observable channel. | Explain the host-specific result, not a new capability of Lua itself. Membership is required to post. Send a concise summary and links, not large attachments. |
| 4 | [Hacker News](https://news.ycombinator.com/) | The technical mechanism and limitations may interest systems programmers beyond WoW. | A regular article submission is appropriate now. Show HN is for a project people can actually try, not an article. HN prohibits generated/AI-edited post text and automated posting: write the submission and discussion yourself. |

WoWUIDev is another relevant developer community. Blizzard's [community forum thread](https://eu.forums.blizzard.com/en/wow/t/addon-community/460235) identifies it as an addon-development community. Inspect its current channels and rules after joining; no current invite or permission to promote this project has been verified.

## Reddit or WoWInterface draft

Suggested title: Sending text into a WoW addon through reusable PlaySoundFile presence signals

We have been experimenting with a return channel for a WoW Forever companion. The desktop process changes whether known addon-owned files exist; Lua observes their presence through PlaySoundFile's return value. There is no audible encoding.

The current lab uses 64 data filenames plus a clock, stable reads, checked packets and local OCR acknowledgements. The same carriers delivered multiple messages in one running Mac session. A 257-byte UTF-8 message took about 15 seconds; deliberately corrupted packets were rejected. It is a small, slow proof of concept, not a production-ready chat transport.

The underlying presence signal was informed by NeverQuestAlone's doorbells. Our experiment uses those signals to carry the payload itself rather than only announcing a fresh one-shot inbox.

Article: https://github.com/AHeige/Jornia/blob/main/docs/sending-text-through-wow-sound-api.md

Code: https://github.com/AHeige/Jornia

Tested on macOS, Forever 1.60.1.70790 / Interface 16001. We would particularly appreciate independent tests of resource-presence caching, sound-disabled behavior and callback cost on Windows. Documented addon APIs do not imply Blizzard approval. AI tools assisted with development and writing.

## Before submitting

- Read the community's current rules and select a technical/WIP category.
- Keep the article and source links accessible without a Jornia invite or an AI account.
- Do not describe the transport as ban-safe, unlimited, faster than pixel protocols, or the first of its kind.
- The 15-second sample measures transport, not AI generation. Do not combine it with the older 6.1-second AI/inbox demonstration.
- Prefer one specialist discussion first; answer technical questions before sharing more broadly. Do not ask for coordinated votes.

## Sources for posting requirements

- [r/wowaddons moderator flair guide](https://www.reddit.com/r/wowaddons/comments/1verqln/unsure_which_tag_to_use_read_this_before_posting/)
- [Lua-l scope and posting instructions](https://www.lua.org/lua-l.html)
- [Show HN requirements](https://news.ycombinator.com/showhn.html)
- [HN guidelines](https://news.ycombinator.com/newsguidelines.html)
