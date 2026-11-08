# Companion window modes — 6 November 2026

The website's “In your game” pictures were produced from our own executable Lua prototype, not from a running Forever client. `apps/addon/Jornia/Companion.lua` now has two modes of that same window:

- **Full:** default 860 × 540 UI units, character sidebar, artwork, journey and chat. Minimum 760 × 500.
- **Compact:** default 420 × 340, just the character identity, conversation, composer, state and expand/close controls. Minimum 340 × 260. Drag the lower-right grip to resize; drag the background/header to move. Each mode retains its own dimensions for the lifetime of this UI.

The same chat frames and model are reused rather than creating two independent conversations. Switching mode keeps messages, input draft, selected full-view tab and pending request. Compact mode hides the sidebar, banner and journey. Mode changes and resizing are immediate; a static treatment keeps text and drag feedback predictable. No new animation framework or third-party product logic is used.

Drafts and conversation state are scoped by character. One pending transport operation disables sending across both views. `OnSend(question, characterID)` must return a confirmed operation ID; `Resolve(operationID, characterID, text, isError)` rejects mismatched replies. Transport errors preserve the draft; neither toggling mode, closing nor resizing resends a question. These hooks are a UI contract, not a newly connected live transport. Size/draft persistence across addon reload is not implemented by this prototype.

## Verification

- Lua 5.1 execution test covers mode switching, positive composer width at minimum size, resize begin/end, independent character drafts/messages, single send during pending, stale/wrong-character response rejection, completion on another character and draft retention after failure. Companion package tests pass.
- Actual Lua executed in the existing developer simulator. Browser walkthrough: entered a draft, expanded to the full view, returned to compact with draft/message preservation, dragged the corner from 420 × 340 to approximately 570 × 404, moved the window, then shrank it again. No Lua errors. Preview adapter now forwards cursor coordinates to the renderer's move/resize primitive.
- Screenshots: `lua-companion-full.png` and `lua-companion-compact.png`. Artwork/font rendering in the simulator is not proof of exact Forever rendering. Text shown is explicitly synthetic; AI remains disconnected in this preview.

## Local installation, 6 November 2026

The user requested installation into WoW. The view is now part of the addon source (`Companion.lua` and `Bindings.lua`) and installed in the Mac's `_forever_/Interface/AddOns/Wisp`. The journal's existing `Wisp.lua` and WTF files were not modified by the installer. The previous addon directory was backed up under ignored `.cache/addon-ui-backup-*`.

- `/wisp` opens/closes the window; `/wisp compact` or `/wispmini` opens compact chat; `/wisp journey` and `/wisp chat` open the full view's respective tabs. Existing journal commands still delegate to the original journal handler.
- Native bindings read only the logged-in character's existing journal. Merely installing/opening the UI does not start recording or send anything to AI. Notes and pause use the original explicit journal commands. Only already open views refresh on game events.
- First-party `scripts/addon-art` converts our existing artwork into power-of-two 32-bit TGA with alpha using Go's standard image codecs. The native banner draws above the backdrop; both banner and avatar were visibly confirmed in Forever.
- Actual client testing revealed that its ScrollingMessageFrame font mixin requires an explicit flags argument. This was fixed, added to the Lua test contract and activated via previously unused `WispBridgeInbox01` during the running session. A dedicated title-row mouse hit region was activated through unused slot 02. Both carrier files were restored afterwards. No `/reload`, client restart or requested logout was used. A server disconnect during the initial entry was recovered in the same running client.
- The real character's name (including surname), level, area and existing journal entries were visibly confirmed. Compact/full switching, resize from 420×340 to approximately 560×390, title-row dragging and draft preservation were exercised in the native client. The test draft was cleared afterwards without sending it. No note, pause or quest action was invoked for the UI test.

The normal public 0.7.0 archives predate this installation and were not overwritten. Local addon metadata is 0.7.1-local. **AI chat remains disconnected in this new window**; its status says so explicitly. The separate OCR lab has not been rewired into this view. Its fixed capture band must remain independent of user resizing, and the finite return-pool limit is unchanged. This installation is a real local UI/journal integration, not a newly verified AI transport.

The screenshots above remain simulator examples. Native verification screenshots are kept in ignored `.cache/addon-install-20261106` locally and are not uploaded with private player context.
