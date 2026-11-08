# Jornia 0.8.0 — quest-log companion slice

The product is now Jornia (Journey in Azeroth). Existing Wisp journal databases, addon identities and transport headers remain compatible. The source repository is AHeige/Jornia; existing infrastructure URLs remain valid.

## Backend

- ForeverDB ingestion and retrieval are paused, without deleting stored records.
- The current AI mode reasons from explicitly approved, recent quest objectives. It computes remaining counts and distinguishes missing context from an empty quest log.
- A new character can register locally on its first native question. Character name and GUID remain local.

## Downloads

These desktop bundles include the browser companion and journal addon. The Mac OCR/audio return bridge is still a separately installed experimental runtime: it is **not included** in these ZIPs. The Windows native chat bridge has not been validated. These bundles are unsigned; Mac notarization and Windows signing remain outstanding. Do not disable operating-system security protections to install them.

Start `Start-Jornia.cmd` on Windows or `Jornia.app` on Apple Silicon. Copy the bundled `AddOns/Wisp` folder using the first-test guide. The technical addon folder name preserves existing saved history; its visible name is Jornia.
