# Quest-log companion delivery — 8 November 2026

## Intent

Help a player understand their current quests before claiming broad world knowledge. Jornia combines an explicitly approved current snapshot with private local journey history. A live response must work during an uninterrupted game session.

## Delivery checklist

- [x] Use Jornia as requested, acknowledging existing unrelated products at https://jornia.app/en/ and https://jornia.es/. This is a product choice, not a uniqueness or trademark clearance claim.
- [x] Pause ForeverDB workers and retrieval; preserve existing records.
- [x] Deploy quest-log-only mode on the VM without replacing unrelated services.
- [x] Derive remaining objectives from observed counters; distinguish unknown context and an observed empty quest log.
- [x] Prioritize named/selected quests within the bounded OCR snapshot.
- [x] Register a first-use character locally without requiring a saved journal or reload.
- [x] Verify provider exclusion against PostgreSQL and preserve stored rows.
- [x] Complete the real, logged-in Mac native chat test with current quests.
- [x] Build and inspect renamed desktop packages.
- [x] Rename the public repository and update the download page. Source changes are published after final verification.

## Deployment evidence

The quest-log API deployment returned healthy status and HTTP 200 coverage. Coverage reported ForeverDB as paused, stored data preserved, and Blizzard as the active source. Comparing container IDs before and after deployment showed no unrelated container changes. The isolated PostgreSQL integration test passed provider exclusion, idempotence, build separation and preservation checks.

## Boundaries

The shared catalog is not complete world knowledge. In this slice, the model must ground guidance in current objective text, counts and supplied evidence. Without verified target coordinates it cannot offer a map destination. SavedVariables still provide delayed long-term journal persistence; current question context uses the experimental Mac OCR path. The Windows desktop journal build does not establish native Windows chat readiness.

## Compatibility

The canonical checkout stays at `/Volumes/Extern/Wisp`. Existing `Wisp` addon folders, SavedVariables names, protocol identifiers, HTTP headers and infrastructure hostnames remain stable so installed clients and journals continue working. New public branding and repository links use Jornia.

## Real in-game result

Two consecutive questions were visually confirmed on Cone Thrower during one uninterrupted session, with no reload. The observed quest log showed 4/6 Scavenger Paw and 2/6 Duskbat Wing for The Damned. “What should I do next?” returned “Collect 2 Scavenger Paws and 4 Duskbat Wings for The Damned.” A Swedish follow-up returned “Det saknas 2 Scavenger Paw.” Both responses used Berget and one source, with ForeverDB disabled.

Reader measurements (not a latency benchmark): first question OCR 257.8 ms, acknowledged return 8,975.7 ms, post-recognition flow 10,853.1 ms; second OCR 257.0 ms, return 6,784.0 ms, flow 7,712.2 ms. These do not include all user-perceived time before OCR detection. Return transport is the largest measured component.

The first six-case synthetic API run passed its initial limited checks: remaining count, next step, completed quest, missing snapshot, empty snapshot and stale snapshot rejection. The first five called the real model; the stale case was rejected before inference. A stricter second run caught a poor empty-quest response withheld by evidence validation (5/6 passed). The citation contract was tightened rather than claiming that run as a success. Model prose still requires review.

## Next guidance slice

The user wants explanations of where to go and what follows, not only remaining counters. Capture quest description and location instructions through public addon APIs, prioritize the selected quest within the transfer budget, then combine that observation with separately verified world evidence for unseen quest chains. A current quest log alone does not establish the next unaccepted quest, an optimal route or NPC coordinates. Continue to distinguish observation, supported recommendation and missing knowledge.

The stricter ready/empty recheck passed both cases after supplying an explicit allowed citation set. Both browser suites passed (consent, no implicit upload, no paid replay, profile privacy and responsive layouts); Go tests, vet, five Python protocol checks and local documentation link checks passed. Jornia 0.8.0 desktop archives were inspected and published, and the public download page was fetched to verify the renamed download links.
