# Knowledge and provider decisions

Recorded 2026-11-05.

Update: the original fixture is now accompanied by a source-reviewed community slice under `forever`. See [Ashenvale research and comparison](ashenvale-research.md) for its provenance, licence, and verification limits. The original boundary decisions below describe the first fixture milestone.

## Ashenvale data boundary

The first dataset is an original development fixture, identified as `fixture-ashenvale-v1`. It contains Kalimdor, Ashenvale, Astranaar, Splintertree Post, The Zoram Strand, and a clearly synthetic spoiler-test story. Names provide a familiar test vocabulary; the fixture is not a validated Forever dataset and makes no claim about current quests, NPCs, coordinates, levels, or routes.

Original fixture descriptions and relationships are dedicated under CC0-1.0. This does not license Blizzard names, trademarks, or third-party content. No bulk third-party dataset has been copied or redistributed. This is not a decision about the license for the repository's application code.

Entities and edges carry source references. Sources record revision, provenance URI, validation status, redistribution conditions, and recording time. Entities and edges are scoped to an explicit game version. At the fixture milestone, default `forever` requests returned unknown rather than falling back to test data. They now serve the separately reviewed community slice with explicit `source_checked` assurance, while unsupported facts still return unknown.

Blizzard's [Classic travel guide](https://worldofwarcraft.blizzard.com/en-us/news/23156366/wow-classic-getting-around-azeroth) is a candidate factual reference for later review, not a Forever verification or redistribution license. QuestieDB and Warcraft Wiki remain candidates requiring source-specific licensing, revision, and compatibility review before import. No generic content license is assumed.

## Berget baseline

The [Berget capability matrix](https://docs.berget.ai/models/capabilities) lists `mistralai/Mistral-Small-3.2-24B-Instruct-2506` with chat and JSON-mode support. The [quickstart](https://docs.berget.ai/quickstart) documents an OpenAI-compatible endpoint at `https://api.berget.ai/v1`. The adapter uses `/chat/completions` with a configurable model and JSON response mode.

The [model catalogue](https://berget.ai/en/models) listed Mistral Small 3.2 at EUR 0.30 per million input tokens and EUR 0.30 per million output tokens at research time. These are editable example rates, not a price guarantee. The preview decision model is deferred: it is unnecessary for a first grounded conversation and still needs endpoint and quality validation.

A live API response is required to verify account access and behavior. Automated tests use a fake provider so ordinary tests do not spend credits or require a key.

## Next data milestone

Verify the curated geography and quest slice against a specific Forever client build, recording build number, date and observations. Add stronger verification only to claims actually observed. Continue to review source licences and provenance before expanding coverage. The original fixture remains available for regression tests.
