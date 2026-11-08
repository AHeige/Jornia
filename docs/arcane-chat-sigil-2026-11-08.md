# In-chat sigil and translucent chat

The approved small in-chat QR tile is now framed as a Jornia sigil. The idle
state uses a first-party geometric emblem; accepted sends show the existing QR
inside the same frame. Captions distinguish sending, thinking, receiving, reply
received and retry. No extra transfer window or action is required to send.

Three palettes are available: Arcane (default), Grove and Ember. Right-click the
sigil to cycle them; Shift-right-click toggles reduced motion. Preferences are
written to existing local SavedVariables and persist when the game normally
saves them. No save/reload is required for immediate changes.

Only the outer rim pulses during an active operation: a gentle 2.6-second cycle.
Reduced motion uses a static border. QR modules, the four-module white quiet zone,
physical pixel alignment and protocol bytes are unchanged. The chat window's
background is 86% opaque; text, controls and the QR backing remain fully opaque.
The sigil's footprint is reserved beside messages. Compact-mode minimum height
accounts for it so it cannot overlap the context checkbox or input.

## Verification

- Lua syntax checks and the existing companion test suite passed.
- Installed in the running Mac client using unused LoD slots 093 and 094,
  restored afterward. No reload, logout, restart or gameplay control was used.
- Existing conversation and active transport were preserved; initial activation
  waited for the current operation to complete.
- Viewed full and compact modes, all three palettes, updated tooltip and reduced
  motion. Verified the normal and reduced-motion waiting states in game.
- Existing player questions after initial activation completed through the new
  sigil, including one while the theme changed. Reader metadata for requests
  ending `000006` and `000007` showed 17.1/22.5 ms decoding and 10.59/6.94 s total.
- Final compact/reduced-motion check, request `941724831794172822000008`:
  “Hur många Rattlecage Skeletons återstår?” returned “Du har 12 Rattlecage
  Skeletons kvar att döda.” The input cleared on Enter, the sigil showed Thinking
  and then Reply received, and one source was indicated. QR decoding took
  18.2 ms, return transport 5,323.5 ms and total delivery 8,075.8 ms.
- Restored Arcane with gentle motion enabled after the reduced-motion check.

These measurements cover this Mac session only. They do not establish Windows
readiness, a smaller QR encoding, instant-close delivery, or broad resolution
coverage. The QR remains visible while required by the existing transport;
the sigil decorates it rather than claiming an invisible channel.
