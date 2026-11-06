# Dator ↔ WoW-klient: lokal fördjupning, 6 november 2026

## Slutsats

**Aktuell riktning efter användarens styrning: SpeakText och rösthjälpprocessen först.** Endast leverans under ett oavbrutet spelpass är relevant. Reload, utloggning och klientavslut ska inte användas som transport, positivkontroll eller nödvändigt avslut på kommande tester. Loggvolymspåret nedan är en teknisk hypotes, inte det valda nästa testet.

Ett nytt, separat spår är **röstsystemets faktiska hjälpprocess och lokala transport**. Det är ett relevant fynd, men ännu inte en kanal Wisp kan läsa. Det ska inte marknadsföras som en färdig localhost-API.

Ingen internetåtkomst eller GitHub-synk användes. Klienten var stängd under analysen. Inga spelinställningar, klientfiler, nätverksanslutningar eller systemskydd ändrades. Ett separat kontrollprogram kördes enbart mot egna temporära datafiler.

## Vad som undersöktes

- Den installerade, universella Mach-O-binären för Forever 1.60.1.70790, båda arkitekturernas strängar, arm64-sektioner och importerade systemfunktioner.
- `WowVoiceProxy` och dess lokala hjälpprocessfiler; ingen hjälpprocess startades eller ersattes.
- Sambandet mellan Lua-funktioner, lokala filformat, loggar och interna klienttjänster.
- Våra faktiska chatlogg-/native-loggmätningar och en ny kontroll av C-filbuffring i **vår egen process**.

Utgångspunkten är att skilja en funktion som finns i klienten från en funktion som en addon kan anropa, och båda dessa från en transport som vår externa process kan läsa.

## 1. Den bevisade filvägen

```text
Addonens Lua
  → C_Log.LogMessage
  → klientens loggsystem / eventuell intern kö
  → användarprocessens filbuffert
  → operativsystemets filsystem
  → Wisp läser rätt markör
```

Den exakta fördelningen mellan klientkö och C-filbuffert är **inte fastställd**. Kedjan visar möjliga buffringslager, inte en återvunnen fullständig anropsgraf.

Det tidigare testet visade:

- en accepterad markör i vanlig addon-Lua;
- faktisk destination `General.log`;
- 0 byte på disk under mätfönstret och efter reload;
- markören synlig först vid klientavslut, efter cirka 189 sekunder;
- slutlig filstorlek 39 138 byte.

Det utesluter snabb leverans för den lilla prövade payloaden. Det utesluter inte en volymtröskel.

### Nytt kontrollprogram: vad extern fsync kan och inte kan göra

Ett eget litet C-program använde `fopen`/`fwrite`. En separat Python-process kontrollerade filstorleken och körde `fsync` på en separat öppnad filbeskrivare. Testet använde dels standardbuffring, dels en uttrycklig 64 KiB-buffert. Inget kördes i WoW-processen.

| Egen kontroll | Liten markör på disk | Extern fsync gjorde markören synlig | Mer data gjorde markören synlig |
| --- | --- | --- | --- |
| Standardbuffring på samma hemfilsystem som WoW | Nej | Nej | Ja, i ungefär 4 KiB-block |
| Uttrycklig 64 KiB-buffert på samma filsystem | Nej | Nej | Ja, när bufferten passerades |
| Standardbuffring på externa arbetsvolymen | Nej | Nej | Inte vid de prövade cirka 349 KiB; vid stängning |
| Uttrycklig 64 KiB-buffert på externa volymen | Nej | Nej | Ja, när bufferten passerades |

Filsystemens rapporterade `st_blksize` skilde sig: 4 096 byte hemma och 1 048 576 byte på externa volymen. Detta är observerad metadata och kontrollprogrammets beteende, inte bevis för vilka inställningar WoW använder. Kontrolldata på hemfilsystemet skapades i en egen temporär katalog under Library/Caches och togs bort efter testet; inga spelar- eller spelfiler användes som kontrollfiler.

**Slutsats från kontrollen:** en extern disk-flush når inte data som fortfarande ligger i en annan process användarminnesbuffert. En filbevakare, `fsync` eller en alternativ filändelse löser inte i sig detta. Däremot kan mer lokalt producerad data i vissa buffertmodeller få tidigare data att lämna processen.

WoW:s General.log höll kvar cirka 39 KiB trots att vanlig C-standardbuffring på samma filsystem i kontrollen redan skrev efter cirka 4 KiB. Därför räcker inte antagandet ”vanlig 4 KiB-stdio” som förklaring för General.log. Större buffert, intern kö, ytterligare villkor eller annan loggimplementation återstår som möjliga förklaringar.

### Oprövad hypotes A: en begränsad lokal volymtrappa — inte nästa valda test

**Inte genomfört i WoW.**

- Återanvänd endast den verifierade `C_Log.LogMessage`-vägen och en oberoende mätare av General.log.
- Börja med en kort unik markör som kontroll.
- Öka därefter inskickad, syntetisk lokal data stegvis, exempelvis kumulativt 1, 4, 16, 64 och högst 256 KiB. Totalbudget, inklusive kontrollmarkörer, ska vara uttrycklig; aldrig en obunden loop.
- Sprid små skrivningar över flera bildrutor för att kunna avbryta och undvika att blockera spel-UI:t. Registrera faktiskt inskickade byte/anrop, eventuella fel och eventuell truncering. Anta inte att loggern bevarar hela stora meddelanden.
- Mät vid vilken fas den ursprungliga markören först blir synlig medan klienten är öppen. Skilj inskickad mängd från faktiskt skrivna loggbyte.
- Håll övriga villkor jämförbara med föregående försök och bevara/återställ berörda inställningar.

Om markören börjar komma ut innan klientavslut har vi en möjlig lokal transport att karakterisera vidare. Då krävs upprepning, latens/jitter, CPU/diskmätning, avbrott, loggrotation och bedömning av kostnaden per meddelande innan produktbruk. Om inget kommer vid budgetgränsen stoppar försöket; det bevisar inte omöjlighet för alla tänkbara volymer.

Att `developerLog` behöver aktiveras för att C_Log ska skriva till General.log är fortfarande oprövat. Ändra inte samtidigt den variabeln i volymtestet; det skulle blanda två hypoteser.

## 2. Nytt fynd: röstens hjälpprocess

Den installerade klienten har `World of Warcraft.app/Contents/Helpers/WowVoiceProxy.app`, med separat körbar fil och ljudbibliotek. I hjälpprocessens binär finns tillsammans:

- loopback-adressen `127.0.0.1` och WebSocket-relaterad kod;
- texter om att proxyn lyssnar, att WoW-klienten ansluter och att sista klientanslutningen stängts;
- tjänsteregistret `BlizzServiceRegistry` och transportbeteckningen `v0.blink.transport`;
- `VoiceService.rpc.cpp` samt typnamn för request/response-hantering;
- anrop för `SpeakText`, TTS-status och lokala TTS-hanterare;
- filen `Voice.log` och ett diagnostiskt meddelandeformat där TTS-text kan ingå.

I **spelklientens egen binär** finns också:

```lua
C_VoiceChat.SpeakText(voiceID, text, rate, volume [, overlap])
```

Det här stöder hypotesen att Lua-text kan skickas från spelet till en annan lokal process genom klientens egna röstfunktioner. Det är en mer konkret IPC-ledtråd än en allmän förekomst av ordet `socket`.

### Begränsningar

- Aktiv endpoint, port, autentisering, kryptering och exakt meddelandeformat är inte verifierade live.
- En andra klientanslutning får inte automatiskt den första anslutningens data. Wisp har inte fått någon läsrätt till rösttransporten genom detta fynd.
- Diagnostikformatet bevisar inte att texten skrivs i aktuell konfiguration, hamnar i just Voice.log eller skrivs snabbt.
- Röstprovider och tillgänglig TTS kan påverka vilken kodväg som används. Lokal TTS och fjärr-TTS ska inte förväxlas.
- Att ersätta röstproxyn, ändra tjänsteregistret eller fånga all voice-/inloggningstrafik är inte ett rimligt första test. Inget sådant gjordes.

### Föreslaget nästa experiment B: endast realtid under öppet spel

Börja med en läsande inventering av lokala TTS-funktioner och röster. Om lokal TTS finns: använd en enda tydligt syntetisk testfras och observera endast vår markör i röstsystemets egen diagnostik, med oberoende tidmätning. Inget mikrofonpåslag, ingen anslutning till en röstkanal och ingen insamling av andra spelares tal eller chatt behövs för den föreslagna kontrollen. Om det inte går att bekräfta en lokal väg ska testet avbrytas, inte tyst falla tillbaka på fjärr-TTS.

Det kan bevisa en annan utgående väg, men tillgänglig data för den externa Wisp-processen och eventuell buffring behöver fortfarande visas. Ren endpoint-metadata från en pågående hjälpprocess kan undersökas separat utan att läsa payloads; det bevisar inte att en avlyssningsbar textkanal finns.

## 3. Vägar som inte gav en färdig lösning

| Väg | Lokal evidens | Bedömning |
| --- | --- | --- |
| Generisk localhost-HTTP för Lua | Inget verifierat generellt anrop hittat | Klientens egna HTTP-bibliotek ger inte addonen ett HTTP-API |
| Inbyggda browser-/localhost-URL:er | Träffarna låg bland butiks-/checkout-/SSO-kod | Inte en addon-HTTP-brygga; inga URL:er öppnades |
| Nätverkets PacketPipe | Socket-, signalerings- och anslutningskod | Intern klienttransport, inte en Lua-fil-/socketfunktion |
| `mmap`/`shm_open`/Mach-meddelanden | Systemimporter finns | Visar intern kapacitet, inte tillgång från Lua eller rätt att läsa speldata externt |
| SavedVariables | Verifierad diskdata vid save/reload/logout | Robust journal, fortfarande inte en livekanal |
| Tangentbindnings-/konfigurationsfiler | SaveBindings, SetBinding och cachefilnamn | Riskerar spelarinställningar; spar- och combatvillkor oprövade, lägre prioritet |
| Textur-snapshot till fil | Metodnamn TestPrintToFile | Inte ett verifierat text-API; pixlar är dessutom bortvalt av användaren |
| Extern fsync eller filbevakning | Negativt i eget kontrollprogram för oserialiserad data | Kan inte tömma WoW:s användarminnesbuffert |
| Read-only spelminne | Föregående task-port-test nekades av macOS | Inga nya försök eller skyddsändringar gjordes |
| Retur via oladdade addon-inboxar | Två verkliga leveranser bevisade tidigare | Begränsad inkommande kanal, inte en utgående lösning |

## Evidens och vad vi inte vet

Råa lokala utdrag och egna kontrollresultat ligger under ignorerade `.cache/client-file-audit/`, bland annat `voice-proxy-strings.txt` och `own-process-buffer-control-filesystems.json`. Inga spelbinärer, bibliotek eller loggar vendoreras eller publiceras. Det egna kontrollprogrammet använder bara C-standardbiblioteket och Python-standardbiblioteket.

En fullständig tillförlitlig anropsgraf/buffertstorlek fastställdes inte genom statisk disassemblering. Förekomst av `fwrite`, `fflush` och `setvbuf` i binärens importer räcker inte för att knyta en viss buffertstorlek till General.log. Ingen analys här ersätter ett korrekt live-test.

**Aktuell ordning enligt användaren:** prioritera B. A utförs inte som del av detta arbete. Ingen ny live-probe har startats. Steg 2 och 3 i dagens plan är orörda.

## Tillägg: SpeakText-loggens faktiska bevisläge

Hjälpprocessens lokala binär innehåller ett format för lyckad TTS som kan bära `Text`, röst, destination och status. SDK:t har separata destinationsnamn för lokal uppspelning respektive fjärröverföring. Det motiverar att undersöka **lokal** SpeakText, men kopplingen från det diagnostikformatet till en viss fil och dess flushbeteende är inte verifierad. Någon särskild fil med namnet SpeakText.log har inte hittats.

Det lokala API-underlaget innehåller `C_VoiceChat.GetTtsVoices()` med `voiceID` och `name`. Read-only-kontrollen av funktionen och en icke-tom lokal röstlista måste komma före taltestet. SDK:t innehåller också fel för saknad eller ej stödd TTS-motor. Att signaturen finns i klienten garanterar därför inte att TTS fungerar på den här Macen.

Nästa avgränsning bör vara: inventera lokal röstfunktion → en enda syntetisk fras, om lokal uppspelning kan bekräftas → bevaka enbart frasens unika markör i rätt klient-/hjälpprocessloggar medan spelet förblir öppet. Bestäm och redovisa en kort mättidsgräns. Om texten inte syns inom den ska utfallet vara negativt eller ofullständigt för realtid; inget avslut/reload används för att rädda testet. Aktivering av mikrofon, röstkanal, fjärr-TTS, service-registry-ändring eller ersättning av hjälpprocessen ingår inte.
