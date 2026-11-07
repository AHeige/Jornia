# Wisp architecture and policy review

Reviewed 5 November 2026 from a Sweden/EU perspective. This review predates the companion implementation; see the [test guide](wow-first-test.md) and [verification report](verification.md) for subsequent versions and outstanding game tests. This is technical decision support based on published rules, not legal advice or Blizzard approval. Actual behavior, contracting entity, client version, distribution and funding must be assessed together before a public launch.

## Baseline architecture

- Central knowledge graph for public world knowledge and sources.
- Local SQLite in the companion for player events, history and preferences.
- No mandatory Wisp account or automatic central synchronization of player history.
- A central model proxy holds the Berget key and receives only context needed for a requested answer.
- The first companion is separate chat with manual entry/import. Automatic game integration was a separate, unapproved and unimplemented question at the time of this review.

SQLite supports local application storage without a separate database server or user administration. Its core is public domain; driver, extension and installer licenses require separate review. See [use cases](https://www.sqlite.org/whentouse.html) and [licensing](https://www.sqlite.org/copyright.html).

## Blizzard contracts and game integration

Blizzard's [addon policy](https://us.forums.blizzard.com/en/wow/t/ui-add-on-development-policy/24534/1) requires free distribution and publicly readable code. It prohibits advertising and donation requests in the in-game addon UI, harmful effects on the game/other players, and refers to the EULA and terms of use. It also addresses charges for addon-related services. A separate companion is not automatically exempt.

**Implication for Wisp:** keep the addon and connected prototype features free. Do not introduce paid AI features, mandatory proprietary payment keys or addon-linked subscriptions without a concrete contractual assessment. The policy expressly allows external website donations, but does not generally authorize every funding model.

Blizzard's [EULA](https://www.blizzard.com/en-gb/legal/08b946df-660a-40e4-a072-1fbde65173b1/blizzard-end-user-license-agreement) restricts unauthorized game-data collection, connections and third-party software. Its [anti-cheating agreement](https://www.blizzard.com/en-us/legal/simple/cd5930c0-2784-420c-a23d-1e0d6ff8599b/anti-cheating-agreement) addresses unauthorized gameplay changes and information collection. Contract breaches and lawbreaking are distinct, but contractual violations can still affect accounts or functionality.

**Prototype boundary:** no process injection, memory reading, packet interception, protection bypass, client-file modification or automated gameplay. Read-only behavior or use outside combat alone does not prove permission.

**Unresolved:** these documents do not give general approval for our combination of Lua addon, external companion, SavedVariables import and AI context. A documented API or readable file does not authorize every secondary use. Assess the exact method, especially a bidirectional/live bridge, before distributing it. Separate manual chat avoids many integration questions but is not a blanket exemption for third-party software.

The addon policy is old but still published. No Forever-specific Wisp approval was found. Direct EULA/API requests returned HTTP 403; relevant passages were reviewed through indexed excerpts from Blizzard's own pages. A launch review must check the contracts applying to the actual EU user and client.

## World knowledge and reuse rights

Warcraft Wiki specifies [CC BY-SA 4.0 for material it can license](https://warcraft.wiki.gg/wiki/Warcraft_Wiki:Copyrights). That does not let it relicense all underlying Blizzard property. Its [copyright policy](https://warcraft.wiki.gg/wiki/Warcraft_Wiki:Copyright_policy) also describes reliance on rights to game material. US fair use is not general Swedish/EU permission.

[CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/) requires attribution, a license link, identification of changes and compatible share-alike licensing for distributed adaptations. It does not automatically impose that license on all application code. A large mixed database and possible database rights need review before bulk import or publication.

The small existing Ashenvale selection includes attribution, license links, change descriptions and review limits in its [data notice](licenses/ashenvale-data.md). These are documented licensing measures, not complete rights clearance. `source_checked` describes factual sourcing/compatibility review, not legal approval.

**Continuing principle:** short structured facts, documented provenance and our own descriptions. Do not copy complete quest texts, books, maps, images, music or large database extracts without verified rights. Original wording and small scale can reduce some risks but do not settle permission. QuestieDB, Wowhead and other imports require separate review.

Using [Blizzard's Developer API](https://www.blizzard.com/en-us/legal/a2989b50-5f16-43b1-abec-2ae17cc09dd6/blizzard-developer-api-terms-of-use) adds terms covering charges, data handling/redistribution, keys, attribution and privacy. A public API record is not automatically freely redistributable. Separate API material from openly licensed knowledge packages. No Blizzard API integration existed in the prototype at the time of this review.

Use an independent visual identity and clearly state that Wisp is independent. Check the Wisp name before public distribution; this review is not a trademark search or registration.

## Local history and GDPR

Local storage reduces centrally held data. Account-free does not automatically mean anonymous. IP addresses and identifiable user information can be personal data under [IMY guidance](https://www.imy.se/verksamhet/dataskydd/det-har-galler-enligt-gdpr/introduktion-till-gdpr/personuppgifter/). Character names, messages and event combinations may also identify people. Processing can occur without persistent server storage.

Where Wisp determines purposes and means of server processing, its operator normally needs assessment as controller for that processing. [IMY's role guidance](https://www.imy.se/verksamhet/dataskydd/det-har-galler-enligt-gdpr/personuppgiftsansvariga-och-personuppgiftsbitraden/) distinguishes this from a provider's processor role. Roles for purely local processing depend on the implementation.

**Before tests with personal data:** define operator and legal basis per purpose; provide understandable privacy information; minimize data and recipients; use TLS and bounded logging; define retention and handle rights/incidents. An AI-context toggle is a product setting, not a complete GDPR assessment. If relying on consent, it must satisfy valid-consent requirements.

Store only the user's intended game history. Do not routinely collect guild chat, whispers, other players' messages or Battle.net details. Never receive Blizzard passwords. Provide export, deletion and local retention choices. Local database storage is not encryption or protection against other processes; use OS permissions and assess additional safeguards for the data. Determine handling of children and age limits before public distribution.

[PTS guidance](https://pts.se/internet-och-telefoni/kakor-cookies) covers storage/access on terminal equipment and the exception for functions necessary for a requested service. Calling something local does not settle consent. Assess the actual app function and telemetry; neither assume ordinary SQLite storage needs a general cookie banner nor that all desktop tracking is exempt.

## Berget

Berget's [terms, section 7](https://berget.ai/en/terms) state that prompt/model-output contents are not retained, while allowing technical logs and metadata. Zero retention does not mean no processing or no metadata. The terms target business/professional customers, favoring an operator-owned backend account over requiring each player to obtain a personal Berget account.

The directly retrieved [DPA](https://berget.ai/en/dpa), dated 22 October 2026, governs personal-data processing and lists subprocessors. Confirm applicability to our account/use, subprocessors, processing geography, security and actual log retention before sending personal history. A general [website privacy policy](https://berget.ai/en/privacy) alone is not a complete inference-processing agreement. The retrieved main terms were dated 16 October 2026; some indexed excerpts showed older dates.

At review time the API processed questions and public graph context; personal memory was not implemented. Application logs contained status, version, model, token counts and cost rather than question/answer text. This code observation does not cover future proxy, CDN, APM or production logs. Those layers must be included before promising a privacy level.

## AI disclosure and installation

The European Commission's [Article 50 guidance](https://digital-strategy.ec.europa.eu/en/faqs/transparency-obligations-under-article-50-ai-act) states that transparency requirements apply from 2 August 2026. Clearly tell players that Wisp is AI. Also assess other obligations applying to us as system provider versus the model provider; an AI label is not a complete AI Act assessment.

A normal user-installed companion can create SQLite in the user profile without local PostgreSQL, Docker or database administration. Recommended Windows location: `%LOCALAPPDATA%/Wisp/`. This is an implementation recommendation, not a legal requirement. Addon installation may need a separately chosen folder and file permission.

Explain file access, local storage, AI calls and uninstall behavior. Signed packages and verified updates are recommended safeguards, not Blizzard approval or a statement of general legal requirements. Check notices for every shipped component. Do not require a blanket antivirus/anti-cheat exception or administrator exemption.

## Consequences for next steps

Prototyping the central graph and separate local-history companion can continue with test data or deliberately supplied personal notes. This is a risk assessment, not a guarantee of legal approval.

Before game integration, document and assess the exact addon/export method. Before a public service, clarify controller responsibilities, Berget agreements, privacy information, data/trademark rights and distribution terms. Addon-linked payment models need separate contractual review. A lawyer familiar with EU/Swedish and gaming/IP matters should assess the concrete implementation before launch.

### Questions prepared for Blizzard, not sent

Describe a free, readable Lua addon and separate Windows companion without gameplay automation. Clarify for Forever:

1. May documented addon APIs record the player's own zone/quest events in SavedVariables?
2. May the user import saved data into a local companion for private history and AI conversations?
3. What restrictions apply to sending a chosen limited excerpt to external inference?
4. Is any method expressly allowed for returning companion replies to addon UI? We do not assume an approved live bridge exists.
5. How does the addon policy apply to a free companion and external funding?

The addon policy lists WoWUI@blizzard.com as a contact. No contact was made in this review. Any answer must be interpreted within its scope; no answer is not approval.
