# Wisp: blockretur i pågående WoW-session

## Resultat

Åtta byte per kvittens fungerar mellan datorn och vår Lua-ruta. Samma 64 databärare och en klockfil återanvändes mellan överföringarna. Inga inboxar förbrukas per datablock eller svar. Inga AI-/nätverksanrop, reload, utloggning eller klientomstarter ingick.

| Liveprov | Nyttolast | Wire / block | Transporttid | Resultat |
|---|---:|---:|---:|---|
| Ny bytevis referens, `Again` | 5 byte | 12 / 12 bytekvittenser | 5 329,9 ms | Mottagen text |
| Block, `Again`, första versionen | 5 byte | 24 / 3 | 1 284,7 ms | Mottagen text |
| Block, `Again`, slutlig version | 5 byte | 24 / 3 | 1 438,7 ms | Mottagen text |
| Längre text inklusive åäö | 257 byte | 272 / 34 | 14 977,4 ms | Hela texten verifierad och visad |
| Avsiktligt skadad checksumma | 3 byte | 24 / 3 | 1 336,9 ms | Avvisad; ingen svarstext |
| Fel överföringsnonce | 3 byte | Första blocket av 3 | 423,9 ms | Avvisad direkt; ingen svarstext |

Den slutliga korta blockkörningen var cirka **3,7 gånger snabbare** än referensen. Detta är enstaka körningar, inte ett statistiskt benchmark. Det längre paketet gav **17,16 nyttolastbyte/s**. Det är fortfarande för långsamt för långa AI-svar med en snabb chattkänsla. Tiden börjar vid första publicerade blocket och slutar vid avläst, kontrollerad slutkvittens; uppvärmning och AI-generering ingår inte.

Referensen mättes om eftersom skärmläget ändrats sedan föregående slice. Samtliga nya prov kördes på Mac, Cone Thrower, Forever 1.60.1.70790 / Interface 16001, samma klientprocess PID 8007, med UI-bild 1920×1080. CUA användes för att visuellt kontrollera text, fel och prestandarader. Ingen bild skickades till en modell eller server.

## Implementation

- `apps/bridge-lab/return-block-bus.lua`: egen mottagare, 64 parallella bitar, två stabila prover, kontroller, aktuell karaktär och överföringsnonce, prefixkvittens, full paketkontroll och skrollbar svarsyta.
- `scripts/experiments/return_bus/block.py`: Python stdlib, frysta datablock tills korrekt OCR-kvittens, tidsgränser, felinjektion och återställning. Befintlig native Vision/ScreenCaptureKit-helper används lokalt.
- `bus.py` stöder två fasta bärarbanker: gamla åtta bitar och nya 64. Inga godtyckliga sökvägar från manifest används vid återställning.

Wire: `WB`, version 1, nonce uint32, karaktärsbindning uint16, nyttolastlängd uint16, nyttolast, Adler-32 uint32. Heltalsfält är big-endian. Paketet nollutfylls till åttabytesblock; databitar inom varje byte är little-endian på filbärarna. Taket är 1 024 nyttolastbyte, men det längsta verkliga provet var 257 byte. 1 024 täcks bara i lokalt protokolltest.

Datafilernas närvaro ställs först; klockfilens närvaro växlas sist. Lua läser vid en ny klocknivå och kräver två identiska dataprover med nominellt 50 ms intervall och giltiga kontrollfiler. Sändaren väntar på rätt nonce, bindning, sekvens, mottagen prefixlängd och prefixdigest innan nästa block publiceras. Slutlig text visas först när hela paketets checksumma och utfyllnad stämmer.

Kvittensen innehåller fasta fältlängder och egen kontrollsumma. OCR får utelämna mellanslagen mellan sifferfält, men får inte ändra siffror utan att kontrollsumman underkänns. Detta är skydd mot oavsiktlig korruption och gamla paket, inte kryptografisk autentisering.

## Problem hittade under liveproven

Första långa försöket kom inte till sändning: Vision slog ihop digest och kvittenskontroll till ett sifferfält. Läsaren korrigerades för fasta fältlängder utan att ta bort kontrollsumman. Ett regressionstest använder den faktiskt observerade OCR-raden.

Ett omförsök överlappade omarmering med startad sändare och tog en föregående READY. Det avbröts och filerna återställdes. **Armera först, starta sedan sändaren; armera inte om under en överföring.** Ingen automatisk omsändning görs.

Det första längre paketet fastnade därefter vid block 4: prefixdigest 2 673 674 982 passerade gränsen för ett signerat 32-bitars heltal och ingen ny kvittens visades. Efter byte från `%010d` till `%010.0f` fungerade hela paketet, inklusive slutdigest 3 876 742 076. Det pekar på WoW:s heltalsformatering; inget felstackspår hämtades. Lokala tester har nu en extra vakt mot att använda heltalsformat för digest, eftersom värdens Lua-testmiljö inte själv reproducerade denna begränsning.

Misslyckade prov räknas inte som levererade svar. Råloggarna bevaras tillsammans med lyckade prov i ignorerad `.cache/return-block-2026-11-06/`.

## Spelkostnad och begränsningar

Vid det lyckade 257-byteprovet visade Lua 6 019 prober, 190,3 ms summerad instrumenterad callbacktid, 1,75 ms maximal callback och noll API-undantag. FPS-raden visade baseline/min/medel 60/60/60. Räknaren omfattar även väntan efter armering, men inte självtestets startkostnad, metrics-rendering eller hela spelets CPU/GPU-arbete. FPS var begränsad till 60 i ett stillastående, kort prov; detta bevisar inte frånvaro av påverkan i strid eller på andra datorer.

Det finns ingen flerprocesslåsning, kö, automatisk återhämtning, omsändning, streaming eller produktkoppling till Berget i denna slice. Kör endast en sändare och ingen updater/annan OCR-läsare samtidigt. Förlorad kvittens ger stopp, inte ett nytt AI-anrop. Windows, andra ljudinställningar och klientversioner är oprövade. Bitfilernas synlighet är fortfarande en experimentell klientegenskap.

## Bootstrap och återställning

Den första mottagaren laddades från redan installerade, oanvända `WispBridgeInbox01`. Formateringskorrigeringen laddades från redan installerade `WispOCRInbox096`, efter en läsande kontroll att den var oladdad. Ett eget `/run C_AddOns.LoadAddOn(...)` användes för denna labbbootstrap; det är inget slutanvändarflöde. Därefter använde alla paket samma återanvändbara filer och `/wispblock` för omarmering, utan fler addonladdningar.

Bärarna är tillfälligt lånade **egna** `WispOCRInbox001..064/*_Camelot.toc` och vår tomma `WispBridgeLab/wispreturn_pulse.wav`. Dedikerade transportfiler krävs för produktinstallation. Alla 65 filer återställdes och innehållskontrollerades; inga `.wispbus-hold` finns kvar. Bootstrapfilerna 01 och 096 återställdes byte för byte från backup. Labbpanelen stoppades/doldes och den normala OCR/AI-läsaren återstartades till `Ready`, utan någon ny fråga.

Sex Python-tester samt hela `internal/companion` passerar. Tester täcker binärgränser, UTF-8, längdtak, återanvändning/återställning, korrupt och upprepad kvittens, skadad checksumma/utfyllnad, gammal nonce, karaktärsbyte och otillgänglig ljudkontroll. De senare två är simulerade, inte ändringar av spelarens riktiga karaktär eller ljudinställningar.

Nästa avgränsning är att minska OCR-kvittensens rundresor ytterligare och sedan koppla den återanvändbara returen till AI-flödet. Det gamla OCR/AI-flödet använder fortfarande sina ändliga engångsinboxar; det har inte tyst bytt transport genom denna labb.
