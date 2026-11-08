# Companion dependency notices

The original Lua addon is licensed under MIT; see `AddOns/Wisp/LICENSE.txt`. No Blizzard game assets are bundled. The companion is an experimental Jornia build, not an official Blizzard product.

The companion binary embeds SQLite via modernc.org/sqlite (BSD-3-Clause wrapper; SQLite core public domain) and its Go dependencies. Their applicable licence files are included under `THIRD-PARTY/` in the test bundle. Go's runtime is supplied under the Go BSD licence. The bundled CSS is generated from Tailwind CSS and daisyUI, both MIT-licensed; their notices are also included.

Node.js, npm, Go, Python, LangChain and PostgreSQL are not required on the player's computer. The server stack runs separately. Server-side LangChain dependencies are locked in `apps/langchain/requirements.lock` and are not included in the companion binary.

The prototype stores its SQLite database unencrypted in the user's app-data directory. It binds its UI to the local loopback interface. No API keys, game assets, character histories or developer `.env` files are included in a release bundle.
