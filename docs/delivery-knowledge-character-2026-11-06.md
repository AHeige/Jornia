# Jornia world knowledge, character questions and test packages

Requested 6 November 2026. Work in order and check off only verified results.

- [x] 1. Inventory sources, database, VM resources and schedules.
- [x] 2. Update available Forever knowledge and improve collection without mixing versions or assuming unknown reuse rights.
- [x] 3. Install/verify bounded scheduled server jobs, resumption, freshness, purge and operational status.
- [x] 4. Verify/extend selected-character, equipment, active-quest and journey questions with correct freshness/isolation.
- [x] 5. Broad question matrix: world, Forever differences, character/journey, unknown facts, conflicts and failures. Report real AI calls separately from deterministic tests.
- [x] 6. Build/verify packages with documented capabilities/limits. Publish through the previously approved GitHub flow if supported.
- [x] 7. Verify installation/operations, update documentation and report results/gaps.

## Scope

“All facts ever” is a coverage ambition, not a measurable completion state. No source establishes complete game knowledge. Import broadly from permitted sources, show coverage and retain explicit unknowns. Journals remain local; private character data must not become central world facts. Packaging must not imply the reusable experimental Lua return is integrated.

## Results by step

Measurements, runs and artifacts were recorded after verification.

### 1. Inventory

The VM had 11 GiB free disk and about 5.3 GiB available RAM; Jornia's database was 72 MB. Existing half-hourly ingestion worked: 30,329 entities, 6,888 raw/detail documents, 24,215 queued jobs; the last eight batches succeeded. These are not complete quest/NPC facts. Active API namespace `static-1.60.1_70399-forever-eu` predates test client 70790. Unrelated containers were inventoried first. ForeverDB required approved developer access; QuestieDB historical rights remained unclear. Neither database was copied without clarified access.

### 2–3. Collection and operations

A new 500-request batch finished at 15:54:49 UTC, followed by reprojection of 2,000 valid cached documents to v2. Explicit API facts about requirements, rewards, quality, item class and tooltips became usable. Reprojection retained original retrieval time, expiry and refresh deadline. No quest narratives, game artwork or separate community database were bulk-imported.

The old cron line was replaced: every 15 minutes, up to 500 requests, at least one second apart, with a twelve-minute deadline. Credential-independent purge runs daily at 03:17 in the VM timezone. `flock` and a database lock prevent overlap. Refresh skips below 3 GiB free disk or 512 MiB available RAM; purge may still run. The ingester retains 192 MiB/0.5 CPU limits. Unrelated containers, proxy and firewall were unchanged. A separate importer image allows updates without API/model/database restarts.

Purge reported `purged`, the runner exited 0, and cron started the next real batch at 16:00:01 UTC. Jobs resume from PostgreSQL. `ingestion-last.log`, its predecessor and atomic `ingestion-status.json` provide bounded diagnostics; `ingestion_runs` and `/api/knowledge/coverage` report actual status/coverage. Missing sources/404/429 create no facts. Completeness remains false.

### 4. Personal questions

Ordinary level/class/equipment questions work offline when their evidence exists. AI can receive an explicitly reviewed projection of journal plus dated Armory, including profile-only context with an empty journal. Names, GUIDs, realm, portraits, notes and conversation history are excluded. Profile change/expiry invalidates approval. Character switching and confirmed completion reset sharing. `character:profile` carries date/expiry; journal level is distinct from an older profile.

Active quests, bags, gold and talents were not provided in this release. Questions must decline to guess; live collection was not implemented. Explicit negative tests cover this. Browser tests verified preview/consent on desktop/mobile and reduced motion.

### 5. Question tests

Final verification at this stage: 46/46 existing VM acceptance cases, 50/50 new deterministic cases (18 API-type representatives in two languages plus unknowns/version boundaries), 40/40 local questions in the extracted Mac binary, and 18/18 real Berget questions. The AI script's 32/32 also includes 14 negative non-model cases already counted in the broad matrix; they are not double-counted. Total: 154 question cases before the separate HTTPS package test. This bounded regression does not guarantee arbitrary answers.

Testing found/fixed: API catalog entries with names matching reviewed quests obscured relations; a closing parenthesis caused a missed SQL match; oversized world context blocked otherwise answerable personal questions; prompts needed clearer journal/profile separation. Initial AI run: 29/32; corrected run: 32/32. New intentional calls tested changes; uncertain paid requests were not automatically replayed.

Go tests/vet, PostgreSQL cases in isolated temporary schemas and four browser suites passed. Development and extracted Mac binaries passed preview, profile sharing, identity masking, offline help, mobile and reduced-motion checks. Model responses were checked for status, expected facts and allowed citations; personal answers were manually reviewed. Estimated final AI-matrix cost: EUR 0.011606; first run EUR 0.010887. API import makes no model calls.

### 6. Packages

Version 0.6.0 built for Windows amd64 and Mac arm64. SHA-256:

- Windows: `4fd19b1f617f6a8a1ba541718c95e5fe470254c7b4e7183661ce36250ea3a42a`
- Mac: `91be60f95dcc2f57c7b0c76271b91cb4af067d25b38836e53cc6093420ca92ba`

Archive integrity, architecture (PE32+ x86-64 / Mach-O arm64), private-file checks and checks against configured secrets passed. The extracted Mac binary ran with a fresh data store. Forty offline questions and four real HTTPS questions through gateway → Go → LangChain → Berget passed with synthetic profile/journal. One correctly distinguished journal level 16 from older profile level 15. The short-lived test code was revoked and its token file deleted. The friend's invitation/quota remained unchanged.

Packages contain the product addon and desktop chat, not OCR/block return. Windows was cross-compiled, not run on the friend's machine. Both binaries are unsigned. At this delivery stage private source was separate from the public website/release repository.

### 7. Coverage and limits

After manual/scheduled refresh: 30,578 servable entities, 7,886 valid API documents and 23,467 remaining jobs. Initially: 30,329 entities/6,888 documents. Raw/detail documents increased by 998; two class-index probes replaced existing documents. Database: 87 MB; VM free disk about 11 GiB. Active namespace remained `static-1.60.1_70399-forever-eu`; no new client build was discovered or claimed verified.

Coverage included 7,051 quest records, 17,537 items, 877 NPC records and 1,895 abilities. Counts do not establish full objectives, drop tables or spawn positions. Automatic batches continue but cannot guarantee all facts absent from the source. ForeverDB access was the next dependency; no key or blanket redistribution right was assumed.

All 158 final question cases passed: 154 plus four HTTPS package cases. This does not prove arbitrary correctness. Both ZIPs are unsigned and Windows was not run in this delivery. Live/block channels remain separate research. No personal player events were needed for verification.

Published release: [v0.6.0](https://github.com/AHeige/wisp-site/releases/tag/v0.6.0). Both ZIPs were downloaded anonymously and exactly matched the locally tested hashes. The download-page update is commit `1213876`; Pages deployment was checked separately.

Final checks: GitHub Pages reported `built`, workflow `success`, and the live page showed 0.6.0 with correct Windows/Mac links. Web commit: `1213876`. API image `20261106-knowledge5` was healthy; separate ingester `20261106-knowledge2` and model/gateway/database remained. A Forever profile request with Retail scope returned HTTP 400 before inference. Timezone: Europe/Stockholm, so purge is 03:17 Swedish VM time. All inventoried unrelated project containers retained their IDs.

[Downloads](https://aheige.github.io/wisp-site/) · [Release/checksums](https://github.com/AHeige/wisp-site/releases/tag/v0.6.0).

This checklist is complete for the verified delivery. Complete game knowledge, approved ForeverDB access, active-quest live collection and integrated in-game return remained product work at that stage, not finished features.
