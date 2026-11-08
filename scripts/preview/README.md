# Jornia Lua preview

This developer-only adapter executes the unmodified `apps/addon/Jornia/Wisp.lua` in [srrmke/wow-addons-sandbox](https://github.com/srrmke/wow-addons-sandbox), pinned to `55431c10e99148fe5f9c00e2de358550bcf1c449` for the captured preview.

The sandbox uses Wasmoon (Lua 5.4 with 5.1 compatibility shims), simulated WoW APIs and DOM rendering of frames. It is not a running Forever client. Stock frame art/fonts are stand-ins; no Blizzard assets or renderer implementation are distributed with Wisp. The upstream renderer checkout and dependencies stay in ignored `.cache/`.

The adapter loads a synthetic player/world shim, then the actual Jornia source. It triggers the real journal consent handler, a quest-turn-in event and `/wisp export`. The preview hides the simulator's default minimap only, to focus on Jornia's frame. The export content is synthetic. No Jornia addon source or release artifact is modified by this process.

To reproduce from repository root:

```sh
git clone https://github.com/srrmke/wow-addons-sandbox.git .cache/wow-addons-sandbox
git -C .cache/wow-addons-sandbox checkout 55431c10e99148fe5f9c00e2de358550bcf1c449
npm ci --prefix .cache/wow-addons-sandbox --ignore-scripts --no-audit --no-fund
cp scripts/preview/addon-preview.ts .cache/wow-addons-sandbox/src/wisp-preview.ts
cp scripts/preview/addon-preview.html .cache/wow-addons-sandbox/wisp-preview.html
npm run dev --prefix .cache/wow-addons-sandbox -- --host 127.0.0.1 --port 18979 --strictPort
```

Open `http://127.0.0.1:18979/wisp-preview.html`. `?scenario=consent` shows the first-use popup, but its native auto-layout is not fully reproduced by the simulator. The published image uses the export window. Read `#preview-errors` for Lua exceptions; the captured export reports `[]`.

The render is a development preview, not evidence of game compatibility, visual fidelity of Blizzard templates or actual event behavior in Forever. Existing GopherLua tests separately exercise the addon under Lua 5.1 semantics.

## In-game product direction

The adapter additionally loads the original, unreleased `apps/addon/Jornia/Companion.lua`. Open `?scenario=chat` or `?scenario=journey` to render the intended in-game interface. These panels are not in release 0.4.2. `PreviewWorld.lua` and the TypeScript fixture wire local pause/note handlers to the real Jornia journal and provide labelled sample conversation text. The prototype has no live AI sender: attempting to send reports that the bridge is disconnected. No model answer is fabricated as an actual response.

The local UI is now installed in the Mac client with real journal bindings; this adapter still intentionally uses synthetic data and no AI sender. Native installation and verification are documented in `docs/lua-window-modes.md`. The simulator requires the cursor-forwarding handler in our adapter to exercise resize.
