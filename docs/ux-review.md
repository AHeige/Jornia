# Jornia experience review

Owner: Codex · 2026-11-06 · implemented and reviewed

## Product direction

Jornia is a personal companion and a character's field journal. The destination is an in-game Lua experience; the desktop app supplies local history and the AI connection. Improving the desktop alpha must not imply that the in-game transport is complete.

Primary journey: **Open Jornia → choose a character → remember the last session → explore recorded places and milestones → ask Jornia → return to the same character later.** First use offers a journal connection or a clearly labelled demo. Settings support this journey; they are not the home screen.

English UI, dark forest/aged gold palette, local original concept art, restrained fantasy typography, readable text and explicit freshness. No invented XP, playtime, class, portraits, quest names or map coordinates. The character schema currently contains name, realm and ID; levels and places come from recorded events.

## Scoring rubric

Scores are a design assessment, not user research. 1–3: obstructs the task; 4–5: usable technical prototype; 6–7: clear but visually or behaviorally unfinished; 8: coherent, visually inspected and functional; 9: polished across states and widths; 10: exceptional and validated with players. Do not award 10 without player evidence. A high visual score does not certify game compatibility or AI correctness.

## Baseline audit

Inspected the actual 0.4.2 companion at localhost:18980, the published download site and executed Lua preview sources. Baseline screenshot: `ux-before.jpg`.

| Surface | Baseline /10 | Problem | Required improvement | Final /10 |
| --- | ---: | --- | --- | ---: |
| Download / discovery | 8 | Strong art; older product screenshots no longer match direction | Refresh product imagery after redesign | 9 |
| First launch / empty journal | 4 | Folder settings compete with the welcome; unclear order | One clear start: connect journal or try demo | 8 |
| Character identity / switching | 4 | Small select and a paragraph; no sense of roster | Persistent identity, accessible selector and character roster | 9 |
| Journey overview | 3 | Hidden under a disclosure; no visual hierarchy | Character landscape, last recorded place, counts and next action | 9 |
| Timeline / milestones | 4 | Flat event list; everything looks alike | Dated chapters, distinct event types, readable notes and filters | 8 |
| Places along the way | 3 | Places embedded in text | Ordered observed route; no invented geographic map | 8 |
| Jornia / conversation | 5 | Generic bubbles, no companion presence | Clear Jornia identity, character scope, useful prompts, adjacent composer | 8 |
| Evidence / context sharing | 6 | Correct safeguards but verbose and visually weak | Compact explicit choice, exact preview, inspectable sources | 8 |
| Journal setup / manual import | 5 | Long sidebar form always visible | Dedicated setup view, consent kept explicit, clear save/pause result | 8 |
| Connection / privacy / export | 5 | Scattered under sidebar disclosures | Coherent settings, readable status, retained export/delete protections | 8 |
| Pending / failure / recovery | 7 | Good receipt mechanics, generic presentation | Preserve acknowledged runtime and drafts, clear scoped feedback | 8 |
| Mobile / keyboard / reduced motion | 6 | Long admin-first reading order | Character then journey, touch targets, keyboard navigation, stable reading | 8 |
| In-game Lua prototype | 4 | Plain rectangle and red default controls | Matching character/journal/companion hierarchy, no fake live AI | 8 |
| Distribution / handoff | 6 | ZIP works; cannot see new UI in existing release | Versioned package, updated notes and matching previews | 8 |

## Interaction and motion decisions

- Journey and Companion are peer views of the selected character. Characters is a roster; Settings holds rare administrative work.
- Character changes restore the correct draft and conversations, clear sharing consent and never show another character's answer or evidence.
- First-use and empty filters offer a next action. Demo history remains visibly synthetic.
- Read-only filters and navigation update locally. Existing acknowledged `Mutations` runtime owns all writes and paid requests. No automatic replay, implicit journal sharing or optimistic historical facts.
- No motion library exists. Reuse daisyUI button/input feedback; one shared 140ms opacity treatment may acknowledge a user-driven view change. History, polling and illustrations remain static. Reduced motion removes transitions and animation.
- Functional checks: startup without data; demo activation; real multi-character import/switch/reload; distinct drafts and histories; local memory question; error draft preservation; context review/reset; timeline filters; journal pause/setup; narrow screen, keyboard and reduced motion; Lua preview execution. Actual Forever and live in-game transport remain outside the verified prototype.

## Iteration log

### 0 — Evidence and design decision

The baseline is a technical settings sidebar plus generic chat. The largest intervention is information architecture: make the recorded adventure the primary object, introduce a persistent character identity, and put Jornia beside that history through a dedicated companion view. The existing local data and receipt model are retained.

### 1 — Structure and first visual pass

Implemented a local-art welcome scene, persistent character selection, Journey / Companion / Characters navigation, character crest and last recorded state, chronological journal chapters, event filters and an ordered place trail. Settings now has its own view. First rendered pass: welcome 8/10, journey 8/10, companion 7/10. Remaining issues found by inspection: pending indicator's custom display overrode the hidden utility; default prompts needed English offline answers; multi-character switching required a draft/consent race review. Those are being corrected before final scoring.

## Character appearance / equipment direction

User asked during the redesign whether actual state/gear can render their character. The current schema only includes identity plus journal observations, not race/class/equipment/appearance. Do not substitute the hero illustration or crest for an actual model. The illustration is explicitly labelled.

Candidate in-game path: native character model populated from the logged-in player, subject to Forever client verification. Desktop path: explicit addon snapshots of public identity, appearance and equipped items, or a dated Armory/media result. A Nov 5 [first-party report from the API library author](https://us.forums.blizzard.com/en/blizzard/t/wow-forever-api-what-works-so-far-plus-a-ts-client/60334) describes equipment and character-media under undocumented Forever profile endpoints; it is not an official availability guarantee or live state. Do not add guessed API data or render gear as verified before testing against a real character. This is a follow-up data capability, not a reason to block the actual journey and character-switching redesign.

### 2 — Real flows and corrections

Rendered the actual companion with two imported synthetic characters, six events each. Verified quest/level filtering, preserved per-character drafts and conversations, and cleared sharing consent on character change. Added local English memory prompts that stay offline. Corrected view navigation retaining the previous page's scroll position; added direct journal-evidence navigation. New directory discovery imports identities without invented history and links uniquely matching later journals without duplicating the roster. Consent version 2 prevents silently expanding earlier file permissions.

### 3 — User-directed visual reset and real character

The user rejected the original droplet avatar and requested an entirely different graphic identity. Replaced forest-green/serif branding with midnight navy, steel-blue surfaces, amber action/milestone color, cyan places and violet notes. New original lantern-spirit avatar and moonlit expedition art replace the previous mascot/scene. Inspiration: [Arionis, color design in games](https://arionisgames.com/blog/the-role-of-color-design-in-modern-games/), used for consistent mood and informative color roles, not as proof of universal color psychology. Event type labels and glyphs carry meaning as well as color.

Verified the user-provided **Kungen Forever / EU** against Blizzard on 2026-11-06: profile level 28, Night Elf Warrior, 15 equipped items, official main character render and avatar. The actual browser UI fetched the snapshot, showed equipment names/rarities and source times, and kept the character's journey empty because no journal had been imported for it. This is an official static render, not a WebGL/interactive 3D model and not live state. Personal API snapshots remain local and outside the knowledge graph and AI context. Server-side credentials are never packaged.

Actual narrow viewport measured at 390×844: portrait loaded; no horizontal overflow. First pass cropped the portrait poorly; a measured second pass changed mobile object-fit/position so the full character is visible. Desktop and mobile screenshot evidence: `ux-kungen-character.jpg`, `ux-mobile-character.jpg`. A fresh startup had no characters and unchecked consent; after approval one import found two character directories and one Jornia file, attached six events to the matching character and preserved the other as an empty directory identity.

## Final design assessment

Scores evaluate the implemented interaction and visual design. They do not certify Forever compatibility, Windows execution or model accuracy. No 10: there has been no independent player usability study. The in-game score applies to the executed Lua prototype only.

### 4 — Release walkthrough

The final delivery is **0.5.1**; previously built 0.5.0 archives remain unchanged. The final polish puts model/cost metadata inside evidence disclosures and makes setup, errors and console instructions English. The published download site and packaged app share the new avatar, palette and art. Conventional local character discovery and optional dated Armory lookup are separate choices.

| Requirement | Authoritative evidence |
| --- | --- |
| Clear first-use journey | Fresh runtime at :18984: zero characters and unchecked consent before approval; two character folders + one Jornia file after approval; six entries attached to the matching character. |
| Switch characters without mixing history | Actual browser: Thalara/Bramble drafts restored separately, conversations scoped, journal-sharing permission cleared. Packaged 0.5.1 restart preserved the two-folder roster and six entries without duplicates. |
| Journey as primary surface | Rendered character scene, event-type filters, dated chapters, ordered observed places and a direct companion action. Screenshots `ux-journey-night.jpg` and `ux-kungen-character.jpg`. |
| Real character and gear | Real HTTPS gateway → Blizzard lookup for Kungen Forever / EU returned level 28, Night Elf Warrior, 15 items and official render; actual local UI displayed them with dates and zero fabricated journal entries. |
| AI companion works | Packaged 0.5.0 made a real explicitly approved demo-context request through HTTPS → Go → LangChain → Berget. Its answer listed Astranaar and Maestra's Post at level 24. Final 0.5.1 restored that answer, kept sharing unchecked and moved model details into sources. |
| Pending/failure are understandable | Actual submit disabled immediately and showed pending. Unavailable-backend run retained its draft and showed no-replay uncertainty. Go tests cover receipts, conflicts, isolation, expiry and mapped-folder forget suppression. |
| Dark gaming identity and new avatar | Original lantern-spirit avatar and expedition landscape embedded locally; compass mark, midnight surfaces, amber actions, rarity colors and typed timeline markers. All fourteen surfaces above assessed. |
| Narrow screen and keyboard | Measured 390×844 width with no overflow; corrected portrait crop; keyboard-opened published FAQ. Reduced-motion CSS disables transitions/animations globally, retaining textual status. |
| In-game design represented honestly | Real Lua executed in simulator: tab switch, note through the actual journal handler, disconnected send preserving the draft, no Lua errors. Preview captions explicitly state not released/live/Forever-verified. |
| Reviewable handoff | This assessment, source in personal private `AHeige/Wisp`, immutable 0.5.1 ZIPs and checksums, scanned for actual provider credentials and tester code. Public downloaded ZIP bytes match the build. |

The existing headless browser runners were adapted to the new view navigation and English selectors and syntax-checked; they were not executed in this review. Browser evidence here is the direct rendered walkthrough, with Go and PostgreSQL tests reported in `verification.md`. Windows execution, actual Forever play and the bidirectional in-game AI bridge remain unverified/unfinished and are disclosed in the site and guide.

### Personal GitHub publication

Application source is private in [AHeige/Wisp](https://github.com/AHeige/Jornia). The static site and previously public preview assets are isolated in [AHeige/wisp-site](https://github.com/AHeige/wisp-site), with public versioned ZIPs. The verified download page is https://aheige.github.io/wisp-site/; unauthenticated downloads match both local SHA-256 hashes. The previous Logos download root forwards there. Only the protected API remains on the existing VM endpoint. No company repository or private player database was published.

The Pages export is reproducible with `python3 scripts/build_download.py --pages-output /path/to/site`: it copies only referenced public assets, rewrites project-relative paths and points downloads at public GitHub release assets. Source and executable packages remain separate from the public website repository.
