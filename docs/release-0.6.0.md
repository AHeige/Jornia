# Jornia 0.6.0 — Character questions and broader Forever knowledge

Portable private alpha for Windows amd64 and Apple Silicon Mac. Local character journals, journeys and conversations remain separate per character.

- Ask common questions about your last recorded level, class and equipped items locally, including while offline. Class and gear require an unexpired Armory snapshot.
- Optionally include dated class, race, level and equipment alongside journal excerpts in an AI question. Review the exact payload first. Permission is off by default, resets after confirmation and on character switch; names, GUIDs, notes, portraits and previous conversations are excluded.
- Shared Forever imports run in bounded server batches every 15 minutes. The graph now exposes additional explicit item/quest/spell API fields, with build/source dates and limited retention. Server updates do not require reinstalling the client.
- Same-named API catalog entries no longer hide a separately sourced quest relationship. Retrieval also handles punctuation at the end of names.

This release does not collect a live quest log, bag inventory, gold or talents. Journal files update when WoW saves them; Armory can lag the game. World knowledge remains incomplete and the API build can be older than the client. Missing facts are not inferred from catalog counts. AI answers need an invite code and connection, can be wrong, and expose their evidence.

Desktop chat opens in your normal browser. Experimental OCR/block-return labs are not included and in-game AI chat is not part of this ZIP. No Electron or separate database install. Both binaries are unsigned; Windows was cross-built rather than executed on the friend's machine. Replace the old extracted client after closing it; keep the normal local data directory. Follow START-HERE.md for installation and privacy choices.
