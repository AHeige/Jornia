# Chatloggens fördröjning — 6 november 2026

Status: **genomfört** på Forever 1.60.1, build 70790 / Interface 16001, Apple Silicon Mac, med Cone Thrower. Steg 2 och 3 i dagens plan har inte påbörjats.

## Upplägg

Förstapartsproben `apps/bridge-lab/chat-probe.lua` har installerats i den befintliga experimentplatsen `WispBridgeInbox03/Inbox.lua`. Det kräver ingen ny TOC och kan laddas genom `/wispbridge inbox 3`. Proben är inaktiv vid laddning. Den skickar endast fasta testmarkörer när spelaren uttryckligen kör `/wispchat a` eller `/wispchat b`; ingen AI används. Ingen annan chatt sparas av proben.

`/wispchat status` frågar efter aktuellt loggningsläge utan att ändra det. Start sparar tidigare loggningsläge, karaktärens GUID, klientbuild och tider. Stop, utloggning och en tio minuters timeout återställer läget. Byte av GUID stoppar testet vid nästa sändförsök. WoW:s inbyggda chatloggning kan skriva annan chatt lokalt under intervallet; vår mätare behandlar endast exakt matchande markörer för vårt körnings-ID. Inget skickas till AI eller externa analystjänster.

- **A:** vanlig chatloggning; inget extra efter mottagandet.
- **B:** 0,1 sekunder efter `CHAT_MSG_WHISPER` för vår exakta markör kör proben `LoggingChat(false)` följt av `LoggingChat(true)`. Väntan skiljer växlingen från det ursprungliga eventanropet. Både faktisk eventtid och före/efter växling registreras.

Självviskningen går via WoW-servern. Ett lyckat sändanrop räknas inte som mottagande: proben väntar på `CHAT_MSG_WHISPER`, skiljt från det utgående `CHAT_MSG_WHISPER_INFORM`.

## Mätning

`watch_chatlog.py` läser enbart från `Logs/WoWChatLog.txt`, från filens ursprungliga slutposition. Avläsningsintervallet är 50 ms; faktiskt största intervall registreras. Den följer nyskapad fil, append samt upptäckt byte/trunkering. Den skriver aldrig till spelets loggfil och lagrar inte omatchande chatrader.

För varje match sparas variant, sekvens, sändarens spel-uptime/heltalstid, första diskobservationens väggklocka och monotona tid. Om loggraden innehåller en native tidsstämpel sparas endast den tidsstämpeln, inte resten av raden. Beräkningen native-tid → diskobservation antar att WoW:s loggtid använder Macens lokala tidszon; detta måste kontrolleras mot de faktiska raderna innan siffrorna används. Sekundupplösning från Lua får inte beskrivas som millisekundprecision. Spelets relativa sänd-/mottagnings-/växlingstider kan jämföras direkt med varandra.

A-försök som inte synts före B-växlingen är censurerade vid den tidpunkten. Om B tömmer äldre A-markörer ska de beskrivas som växlingsutlösta observationer, inte snabb vanlig loggning. Reload/utloggning görs först efter avslutad mätperiod och återställt läge för att spara Lua-resultaten.

Råa testresultat ligger lokalt under ignorerade `.cache/chatlog-test/`; inga ursprungliga loggar raderas eller publiceras.

## Förkontroller

- Lua-test verifierar att laddning/status inte slår på loggning eller skickar meddelanden, att B triggas av mottagandet och inte den utgående ekohändelsen, samt att både ursprungligt `false` och `true` återställs. Försenade timeranrop efter stop gör inget.
- Ett isolerat Python-fixturtest verifierar append-detektion, tidsstämpelparsning och att orelaterad chatt inte hamnar i resultatfilen.
- Ett första `/run`-försök mötte WoW:s generella custom-script-varning innan kommandot kördes. Det är inte ett resultat för chattransporten. Testflödet använder addonkommandon i stället; normal Wisp-användning ska inte kräva inklistrade `/run`-skript.

## Resultat

Körnings-ID: `8624d79bb0a3`. Alla åtta unika meddelanden mottogs i spelet. Den extra A-sändningen bevarades som ett eget försök, inte som ett fel eller en dublett. Sändanrop → mottaget event tog **241–260 ms**. Det var inte serverresan som orsakade den stora diskfördröjningen.

| Försök | Antal | Mottaget i spelet | På disk före klientavslut | Resultat |
| --- | ---: | --- | --- | --- |
| A1–A4, vanlig loggning | 4 | 4/4 | 0/4 | Ingen snabb leverans |
| B5–B7, direkt av/på | 3 | 3/3 | 0/3 | Växlingen tömde inte bufferten |
| B8, paus över flera bildrutor | 1 | 1/1 | 0/1 | Verklig av-tid 270 ms; ingen snabb leverans |

Den sista kontrollen tillkom för att inte förväxla en växling i samma Lua-anrop med en faktisk av-period. `chat-gap-probe.lua` registrerade `logging_while_off=false`, därefter `logging_after_toggle=true`. Direktvarianten registrerade också återaktiverat läge för alla tre försök. Alla event- och växlingstider kommer från sparade Lua-kvittenser.

Filen var 0 byte under A-fasen, efter båda typerna av B, efter `/wispchat stop` och efter `/reload`. Vid normalt avslut av hela WoW-klienten blev filen **3 810 byte**. Den oberoende mätaren såg **16 matchande rader för åtta unika ID:n**, samtliga i samma diskobservation **12:37:14,292 CET**. Två rader per ID beror på de två loggade sidorna av självviskningen; det är inte sexton försök.

Fördröjningen från respektive native loggtidsstämpel till den gemensamma diskobservationen var cirka **228–620 sekunder**. Det beskriver detta manuellt genomförda, avslutsutlösta test — inte en uppskattning av normal medel- eller p95-latens. Native tidsstämplar och spelets högupplösta mottagningstider stämmer mot mätarens vägg-/monotona klocka inom cirka 2 ms. Mätaren avläste nominellt var 50 ms; största observerade pollintervall var 141 ms. Den begränsningen kan inte förklara flera minuters fördröjning.

**Slutsats:** i den här klienten och vid den här trafikvolymen är chatloggen inte en tillräcklig realtidstransport. Hypotesen att de prövade av/på-växlingarna skulle ge omedelbar flush fick inget stöd. Positivkontrollen vid klientavslut visar att payloadarna faktiskt loggades och att filväg/mätare var rätt.

Detta bevisar inte att varje möjlig chatloggmetod är omöjlig. Filen blev mindre än 4 KiB; en volymstyrd buffertgräns eller andra explicit prövade flushmekanismer är fortfarande möjliga hypoteser. Ingen sådan gräns mättes, och vi skickade inte fyllnadsmeddelanden för att pressa fram den. Ett sådant försök skulle vara ett separat, överenskommet deltest av steg 1.

## Återställning och evidens

Ursprungligt loggningsläge var `false`. Testet stoppades manuellt före reload och avslut, med återställning till ursprungsläget. Båda mätarprocesserna är stoppade. De tillfälligt använda inboxplatserna 3 och 4 återställdes från sina exakta förtestkopior med WoW stängt; de är åter inaktiva testinboxar. Ursprungliga chatloggar är bevarade.

Filtrerade observationer, operatörsnoteringar, sparade addon-kvittenser och maskinläsbar sammanställning ligger endast i `.cache/chatlog-test/8624d79bb0a3/`. De publiceras inte. Lua-filen lästes med vår begränsade data-parser, aldrig genom Lua-exekvering. `summarize_chatlog.py` återskapar sammanställningen från dessa filtrerade filer.

**Beslut:** chatloggning i den testade formen används inte som livekoppling. Vi har avslutat steg 1:s A/B-test och börjar inte automatiskt med ögonblicksbild eller AI-retur.

Nästa beslut gäller endast om den utgående loggvägen är snabb och upprepningsbar nog att fortsätta undersöka.
