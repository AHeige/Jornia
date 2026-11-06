# Start with Wisp 0.5.2

Private desktop alpha for Windows and Apple Silicon Macs. The in-game Lua chat is still a prototype; the downloadable addon records a local journal. Forever 1.60.x / Interface 16001 is the target, with real gameplay testing still required.

## Start

1. Unzip into its own folder. Close any older Wisp console. On Windows, double-click **Start-Wisp.cmd**; on Mac, launch the included `wisp` binary from Terminal. Wisp opens in your normal browser. No Electron, developer tools or separate database installation.
2. Choose **Connect your journal**. Wisp checks standard WoW install locations and the Windows installation registry. Confirm the client folder containing `WTF`, approve local reading, and choose **Import all local characters**. You can enter a custom path. Wisp discovers character directories and reads every existing Wisp journal in that folder. It does not parse other addons, account settings or chat logs.
3. Or choose **Explore a demo** without WoW. Its invented events stay in a separate, marked demo journal.
4. Use **Journey** to see your selected character, dated entries and recorded places. Filter quests, levels or notes. **Characters** and the persistent character selector switch the journal, conversations and drafts together.
5. Open **Companion** and ask **What was I doing last time?** or **My journey** for local, offline help. For AI, open **Settings & connection → AI connection**, enter your separate private invite code, and choose **Connect Wisp**. Choose **Use AI** when asking a world question.
6. To include personal context, review the exact journal excerpts and choose **Include journal excerpts** for that question. Consent resets after a confirmed answer. Names, notes and previous conversations are not included automatically.

The Windows app is unsigned; Windows may show a security warning. Mac users may need to approve the unsigned binary themselves. Signed installation and automatic updates are not part of this alpha. Windows 0.5.2 has been cross-built, not run on the remote Windows machine.

## Your actual character and equipment

Under **Settings & connection → Character portrait & gear**, enter the first name, surname and region of a public Forever character. Choose an existing matching character or **Add a character**, approve the lookup and click **Load character from Blizzard**. An invite code is required. One lookup uses one server request from the invite quota.

Wisp loads the official character render and equipped item list, with source/fetch times and the profile's last login. This is a published Armory snapshot, not a live 3D model or live gear. Some new/low-level characters or individual endpoints may be unavailable. Expired profile data is removed when Wisp is next used, with a maximum cache lifetime of 28 days. Profile data stays in the local database and is not inserted into shared world knowledge or AI journal excerpts. Portraits load from Blizzard's render host.

## Record new adventures

1. Exit WoW. Copy `AddOns/Wisp` from the ZIP to the client's `Interface/AddOns/Wisp`. The TOC belongs directly in that folder; do not overwrite an unrelated addon called Wisp.
2. Start WoW and enable Wisp. Read and accept the separate journal prompt if you want recording. It records sessions, areas, levels, completed quest IDs and your notes. It does not collect chat.
3. Play, then `/reload` or log out so WoW writes SavedVariables. The companion imports new saved entries automatically. Repeated reads do not duplicate them.
4. Switch characters in Wisp to see each separate journey. A character found only through directories can appear immediately, but has no invented level, gear or old adventures. Previously unrecorded history cannot be recovered.

`/wisp start` and `/wisp stop` control recording. `/wisp mark` adds a place observation; `/wisp note <text>` saves a note. `/wisp export` provides a manual packet for **Paste an addon export**.

## Your data

**Pause reading** stops local file reading; addon recording is separate. **History & privacy** exports local journals, conversations and unexpired profiles without the invite code. **Forget selected character** asks for confirmation, deletes that character's local data and prevents automatic rediscovery/reimport; the original addon file is untouched. An explicit manual journal import can restore it.

Windows storage: `%LOCALAPPDATA%/Wisp/player.db`. Mac: the user configuration directory under Wisp. The database is not encrypted by Wisp; protect your account and backups. Replacing the ZIP preserves this data directory.

Wisp's existing HTTPS API remains at `https://wisp.logosab.com`; this is infrastructure, not a Logos-owned repository. Source belongs to the personal `AHeige/Wisp` GitHub repository. No Berget/Blizzard developer secrets, SSH keys or invite codes are shipped in the ZIP. Initial invites last 30 days and allow 50 server requests per UTC day, shared by AI/world questions and explicit Armory lookups. Local memory is offline and does not use that quota. Uncertain remote outcomes are never automatically replayed.

## Character identity (0.5.2)

The addon also saves a small `WispCharacterDB` identity marker in its own per-character SavedVariables file. It contains the installation ID and character GUID; its file location links the real directory (including a surname/numeric realm) to the account journal. The companion does not guess from a first name or realm label. Restart WoW after updating the addon, then log in and `/reload` or log out once. Existing empty duplicate directory rows reconcile when that marker arrives; independent private content is preserved. Old 0.5.1 journals still import, but need the new addon marker to resolve mismatched directory names.
