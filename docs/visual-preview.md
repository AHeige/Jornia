# Wisp visual presentation — 2026-11-06

The existing VM-hosted page at https://wisp.logosab.com is retained. This change does not migrate hosting, modify the immutable 0.4.2 client packages, or change their access-code flow.

The presentation uses a deep forest palette, parchment typography, an original fantasy illustration and two product images. The visitor sees the promise, the three steps (play → saved local journal → selected AI context), the real companion, the addon preview, privacy and installation. Images and captions distinguish illustration, synthetic data and actual software. The download remains the primary action. No autoplay, parallax, tracking, additional fonts or client-side JavaScript is introduced. Native details provide optional FAQ depth; anchor motion respects reduced-motion preferences and content remains static for reading.

Assets:

- `apps/download/media/forest-original.png`: original generated artwork, retained locally in the repository. Not exposed by the public gateway.
- `apps/download/media/forest.jpg`: optimized website artwork, 1536×1024.
- `apps/download/media/addon-preview.jpg`: 960×540 capture from executed Wisp Lua in a sandbox, not a game screenshot.
- `apps/download/media/journey-preview.jpg`: actual packaged Wisp client showing synthetic demo events and its real Berget response. No tester code or personal history is visible.

Renderer provenance and reproduction: [scripts/preview/README.md](../scripts/preview/README.md). Sandbox source commit: `55431c10e99148fe5f9c00e2de358550bcf1c449`. The actual Wisp source executed without Lua exceptions; fonts, frames and game APIs remain emulated. The UI source contains the export frame, not an in-game AI chat or an avatar. The page describes the external chat accordingly.

## Original image generation

Built-in image generation tool, `stylized-concept` / website illustration. Final prompt:

> Create a premium editorial fantasy illustration for Wisp, a personal adventure journal and AI companion for an MMO. Wide landscape composition 1536x1024 or similar 3:2. A small lone cloaked traveler seen from behind on a winding stone path in an ancient dusky forest of very tall mossy trees, overgrown carved stone arches, a distant warmly lit tiny woodland settlement by still water. A single friendly floating pale mint luminous wisp beside the traveler's shoulder, subtly lighting their leather travel journal held at their side. Mood: quiet companionship, remembered journeys, curiosity, atmospheric painterly game concept art, restrained brushwork, beautiful depth, organic detailed foliage, deep forest greens, midnight teal, aged warm gold, desaturated parchment light, thin fog, soft sun shafts. Viewpoint slightly elevated, world dominates, small traveler at lower right third, foreground dark and cinematic. Original fictional environment, evocative of classic painted fantasy RPG landscapes without copying a game screenshot or identifiable character. No text, no typography, no logos, no UI, no frame, no HUD, no barcode, no excessive glowing particles. This is conceptual illustration, not a screenshot of a running game. Save an image suitable as a website hero asset.

The generated image was inspected before copying into the workspace; only JPEG format/quality optimization was applied. It is visibly captioned as AI-generated concept art. The original output remains under the tool's generated-images directory as well.

## User correction: the primary experience is inside WoW

The user clarified that both chat and journey should be in-game. The landing page now leads with the intended Lua interface, explicitly marked as an unreleased prototype; it describes 0.4.2 as an external technical alpha. The two additional images are `apps/download/media/ingame-chat-preview.jpg` and `apps/download/media/ingame-journey-preview.jpg`, captured from executed `apps/addon-preview/WispPanel.lua` with the renderer above. Sample AI text is supplied only by the fixture and is visibly labelled. It is not an actual in-game model response.

The renderer's tab and note actions were exercised. Sending a question without a bridge visibly reports that nothing was sent. The final screenshots, 960×540, retain the prototype/disconnected labels. See [the product decision and remaining transport work](ingame-experience.md).
