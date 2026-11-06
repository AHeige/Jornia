# Aktuell spelstatus och beständigt minne

Undersökt 2026-11-05. Realtidskoppling är ett nytt transportproblem; den är inte implementerad eller installerad i den nuvarande Windows-klienten.

## Det konkreta problemet

Efter en questinlämning kan karaktären bli nivå 16 medan senast sparade journal säger nivå 15. Journalen får då inte behandlas som aktuell nivå för zonråd. En tidsstämpel förklarar historiken men gör inte en gammal rekommendation korrekt.

Addonen reagerar redan på spelhändelser i minnet. SavedVariables skickar inte dessa händelser löpande till desktopappen. En osynlig addon gör inte filen kontinuerligt uppdaterad. Automatisk UI-reload är en riktig omladdning, inte en tyst flush.

## Tekniskt kandidatspår

[WoW Context Bridge](https://github.com/nukeador/wow-context-bridge) är ett MIT-licensierat experiment som använder en synlig pixelremsa och extern bildavläsning. Projektet anger Retail-test, ej verifierad Forever-kompatibilitet, och väntande svar från Blizzard. Det är tekniskt referensmaterial, inte ett Blizzard-godkännande eller ett Wisp-speltest.

Windows har [Graphics Capture](https://learn.microsoft.com/en-us/windows/apps/develop/media-authoring-processing/screen-capture), där användaren väljer fönster via systemets dialog. En eventuell Wisp-prototyp kan begränsa lokal behandling till en explicit vald statusregion. Ingen bildström behöver skickas till Berget. [Blizzards EULA](https://www.blizzard.com/en-gb/legal/2c72a35c-bf1b-4ae6-99ec-80624e1b429c/blizzard-end-user-license-agreement) har breda begränsningar kring obehörig insamling; en normal OS-capturefunktion bevisar inte att en viss WoW-brygga är godkänd. Ingen sådan capture har aktiverats i användarens klient här.

## Avgränsning för en första prototyp

- Endast spelarens egen nivå och zon, senare valda questhändelser. Inga andra spelare, whispers, guildchatt eller skyddade/secret-värden.
- Eget synligt, uttryckligen aktiverat statusfält i addonen och separat samtycke till fönsteravläsning. Avstängd som standard. Inga minneshandtag, injektioner, paketavlyssning eller syntetiska spelhandlingar.
- Versionerat meddelande, karaktärsidentitet, sekvensnummer, checksumma och återkommande heartbeat. En fastfrusen bild får inte förnya färskheten bara för att den läses igen.
- Nuvarande status är ett tillfälligt lokalt tillstånd, skilt från journalens beständiga händelser. Vid förlorad signal blir aktuell status okänd efter en kort timeout; den gamla nivån lever inte vidare som aktuell.
- Källor och tider visas. Personlig kontext till Berget kräver förhandsvisning och ett eget val. Det flödet finns inte ännu.
- Om status ändras medan ett AI-svar genereras måste det framgå vilken snapshot svaret byggde på. Inget automatiskt nytt betalt anrop görs.

## Avgörande kontroller

Först syntetisk avsändare/mottagare: nivå 15 → questinlämning → nivå 16; fråga direkt; korrekt snapshot; fel checksumma; bildfrysning; avbruten capture; fönsterbyte, upplösning/UI-skalning och byte av karaktär. Därefter faktisk Forever-klient med uppmätt latens och påverkan. SavedVariables används fortsatt för beständig resa och återställning. Varken syntetisk avkodning eller ett fungerande annat projekt bevisar den faktiska Forever-kopplingen eller avtalsstatusen.
