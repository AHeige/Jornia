# Wisp: ingame först

Bekräftat av användaren 2026-11-06: **AI-chatten och journey-hanteringen ska finnas inne i WoW, i Lua.** Desktop-companionen är bakgrundsinfrastruktur för lagring, anslutning och uppdatering. Den externa chatten i 0.4.2 är en teknisk alpha, inte den avsedda slutupplevelsen.

## Vyn

`apps/addon-preview/WispPanel.lua` är en körbar, opublicerad gränssnittsprototyp med flikarna Samtal och Min resa. Den läses av Lua-renderaren tillsammans med det befintliga journaladdonet. Fixturekod tillför en demokaraktär och tydligt märkta exempelsvar. Prototypen ingår inte i den publicerade 0.4.2-zippen och ändrar inte installerade WoW-filer.

Verifierat i simulatorn: flikbyte, visning av faktiska syntetiska journalposter från WispDB, tillägg av lokal anteckning och tydligt avvisad sändning när AI-bryggan saknas. Ingen modell körs av prototypen. Nya UI-funktioner är inte verifierade i Forever.

## Datans ansvar

- Addonet äger aktuellt tillstånd genom tillgängliga spel-API:er och sin journal i minnet. En fråga ska få en ny ögonblicksbild från klienten, inte desktopappens äldre SavedVariables-kopia.
- Companionen äger det beständiga lokala arkivet, begränsad privat kontext, autentiserad serveranslutning och uppdateringar. Journalen blir inte gemensam världskunskap.
- Servern äger urval av källbelagda världsfakta, versioner, kostnadsgränser och det tunna LangChain/Berget-steget.
- Svar måste bindas till fråga, karaktär och session. Återanslutning får inte återköra betalda frågor eller placera ett svar hos en annan karaktär.

## Återstående kärna: transport i båda riktningarna

Det går inte att ersätta bryggan genom att bara flytta ett chattfält till Lua. En automatisk, verifierad kanal behövs ut ur klienten och tillbaka. Användaren har invändningar mot pixelöverföring; den ska inte införas tyst som vald lösning.

Primär referens: [wow-ai](https://github.com/chelinho139/wow-ai) beskriver en Forever-lösning som får in svar via förberedda load-on-demand-addons och använder en visuell kanal ut. Detta är ett kandidatmönster för inkommande data, inte Wisp-verifiering, tillstånd från Blizzard eller bevis för en fungerande utgående loggkanal. Ingen tredjepartsbrygga har installerats.

Nästa verifiering måste använda en riktig Forever-klient: mät loggarnas skrivfördröjning och testa kontrollerad svarsinläsning. Skilj quest visad/accepterad/övergiven/inlämnad, testa flera snabba frågor, avbrott, spelomstart och karaktärsbyte. En rendering eller en fungerande extern AI-chatt bevisar inte ingame-kedjan.

Nedladdningssidan visar nu uttryckligen skillnaden mellan den avsedda ingame-upplevelsen, den körbara Lua-prototypen och den nedladdningsbara externa tekniska alphan.

## Two window modes, 6 November 2026

The Lua visual prototype now supports a full companion window and a movable/resizable compact chat, sharing drafts, messages and pending state. See [window modes and verification](lua-window-modes.md). This remains a separate UI prototype; it has not replaced the live OCR lab or been bundled in the public ZIP. Subsequent transport work supersedes the early investigation notes above: a finite OCR → Berget → Lua roundtrip and a separate reusable block-return experiment have been proven, while the newest snapshot/waypoint slice still has explicit live-test limits. Go now owns server orchestration and talks directly to Berget.

The subsequent user request installed the two-mode UI into the Mac client. Native character/journal/artwork and resizing have now been tested. The new view explicitly shows that its AI chat is not connected; the OCR lab remains separate. See `lua-window-modes.md` for installation commands, backups and the remaining transport boundary.
