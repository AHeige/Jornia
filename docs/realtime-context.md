# Current game state and durable memory

Investigated 5 November 2026. At this stage, real-time connectivity was a new transport problem, neither implemented nor installed in the Windows client.

## The concrete problem

Turning in a quest can raise a character to level 16 while the last saved journal still says level 15. The journal must not be treated as the current level for zone recommendations. A timestamp explains history but does not make an outdated recommendation correct.

The addon already reacts to game events in memory. SavedVariables does not continuously send these events to the desktop application. An invisible addon does not make the file update continuously. Automatic UI reload is a real reload, not a silent flush.

## Technical candidate

[WoW Context Bridge](https://github.com/nukeador/wow-context-bridge) is an MIT-licensed experiment using a visible pixel strip and external screen reading. The project reports Retail testing, unverified Forever compatibility and a pending Blizzard response. It is technical reference material, not Blizzard approval or a Wisp game test.

Windows provides [Graphics Capture](https://learn.microsoft.com/en-us/windows/apps/develop/media-authoring-processing/screen-capture), where the user chooses a window through a system dialog. A Wisp prototype could restrict local processing to an explicitly selected status region. No image stream needs to reach Berget. [Blizzard's EULA](https://www.blizzard.com/en-gb/legal/2c72a35c-bf1b-4ae6-99ec-80624e1b429c/blizzard-end-user-license-agreement) broadly restricts unauthorized collection; an ordinary OS capture API does not establish approval for a particular WoW bridge. No capture had been enabled in the user's client at this stage.

## Initial prototype boundaries

- Only the player's own level and zone, later selected quest events. No other players, whispers, guild chat or protected/secret values.
- Own visible, explicitly enabled addon status field and separate consent to window reading. Off by default. No memory handles, injection, packet interception or synthetic gameplay.
- Versioned messages, character identity, sequence, checksum and heartbeat. A frozen image must not refresh freshness merely because it is read again.
- Current status is temporary local state, separate from durable journal events. After signal loss it becomes unknown after a short timeout; an old level does not remain current.
- Display sources and times. Personal context sent to Berget requires preview and a separate choice. This flow did not yet exist.
- If state changes during answer generation, show which snapshot supported the answer. Do not automatically make another paid request.

## Decisive checks

First use synthetic sender/receiver tests: level 15 → quest turn-in → level 16; immediate question; correct snapshot; bad checksum; frozen image; interrupted capture; window switching, resolution/UI scaling and character switching. Then test a real Forever client with measured latency and impact. SavedVariables remains the durable journey/recovery path. Neither synthetic decoding nor another working project proves the actual Forever integration or contractual status.
