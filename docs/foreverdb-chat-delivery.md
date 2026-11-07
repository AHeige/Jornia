# ForeverDB and native companion chat delivery — 6 November 2026

- [x] Store the approved developer credential privately; validate meta.client and build using authorized JSON requests. No key goes in a client, URL, repository or log.
- [x] Read current API rules and verify actual quest/class/recipe shapes. Visible ForeverDB.net attribution, no resale/relicensing, source/build/origin labels retained.
- [x] Implement owned bounded ingestion, separate provider lock, persistent quota reservations (8000/day safety budget), rate pacing, error cooldown and resumable page/detail jobs.
- [ ] Run database integration checks, complete the initial import and measure usable category coverage.
- [ ] Activate isolated scheduled VM refresh and verify failure/rollback behavior without changing unrelated services.
- [ ] Add visible attribution in desktop and native answers.
- [ ] Connect both native UI modes to one explicitly enabled local OCR reader; preserve single-send, loading, draft/error and character isolation.
- [ ] Integrate reusable checked block return rather than spending a LoD module per poll.
- [ ] Verify real AI replies in the running client, two consecutive questions and data-backed world/personal answers.
- [ ] Record latency, limitations, recovery and installed versions. No reload, logout or process restart as a transport step.

The approved developer identity is dev-0144. The initial authorized responses report build 1.60.1.70790 and catalog counts of 22,403 items, 32,109 spells, 6,614 quests, 947 vendors, 47 instances and 545 sets. These are provider record counts, not complete verified gameplay facts. On quests 376 and 3901, hand-in/start positions are explicitly labelled player observations, while some chain links remain labelled Classic data.

Current documentation: https://foreverdb.net/developers (read 6 November 2026). API use is limited to Wisp's application backend, with attribution. No bulk data download endpoint is exposed to users.
