# Companion window modes — 6 November 2026

The website's “In your game” pictures were produced from our own executable Lua prototype, not from a running Forever client. `apps/addon-preview/WispPanel.lua` now has two modes of that same window:

- **Full:** default 860 × 540 UI units, character sidebar, artwork, journey and chat. Minimum 760 × 500.
- **Compact:** default 420 × 340, just the character identity, conversation, composer, state and expand/close controls. Minimum 340 × 260. Drag the lower-right grip to resize; drag the background/header to move. Each mode retains its own dimensions for the lifetime of this UI.

The same chat frames and model are reused rather than creating two independent conversations. Switching mode keeps messages, input draft, selected full-view tab and pending request. Compact mode hides the sidebar, banner and journey. Mode changes and resizing are immediate; a static treatment keeps text and drag feedback predictable. No new animation framework or third-party product logic is used.

Drafts and conversation state are scoped by character. One pending transport operation disables sending across both views. `OnSend(question, characterID)` must return a confirmed operation ID; `Resolve(operationID, characterID, text, isError)` rejects mismatched replies. Transport errors preserve the draft; neither toggling mode, closing nor resizing resends a question. These hooks are a UI contract, not a newly connected live transport. Size/draft persistence across addon reload is not implemented by this prototype.

## Verification

- Lua 5.1 execution test covers mode switching, positive composer width at minimum size, resize begin/end, independent character drafts/messages, single send during pending, stale/wrong-character response rejection, completion on another character and draft retention after failure. Companion package tests pass.
- Actual Lua executed in the existing developer simulator. Browser walkthrough: entered a draft, expanded to the full view, returned to compact with draft/message preservation, dragged the corner from 420 × 340 to approximately 570 × 404, moved the window, then shrank it again. No Lua errors. Preview adapter now forwards cursor coordinates to the renderer's move/resize primitive.
- Screenshots: `lua-companion-full.png` and `lua-companion-compact.png`. Artwork/font rendering in the simulator is not proof of exact Forever rendering. Text shown is explicitly synthetic; AI remains disconnected in this preview.

## Delivery boundary

This source is still separate from both the released addon and the simpler OCR lab. It was not installed into the running game, published as a release or presented as a new live AI test. Next integration work is to use this view as the product shell, package its game-compatible artwork and connect its send/resolve contract to the verified transport while keeping OCR geometry independent of the user's resize. In particular, the fixed outbox/capture band in today's lab cannot simply be stretched along with this window. The current finite return-pool limit also remains.
