# Native Lua-logg — 6 november 2026

Status: **genomfört**, inklusive en upprepning med rätt målfil och verifierad återställning. Ingen nätåtkomst eller GitHub-synk användes i detta test.

## Avgränsning

Testa endast om `C_Log.LogMessage` finns för en addon och om **en** unik, ofarlig markör kan nå en lokal loggfil snabbt. Första hypotesen var `Logs/DeveloperLog.log`; faktisk destination visade sig vara `Logs/General.log`. Ingen spelarkontext, AI, chattöverföring, RAM-läsning, utfyllnad eller upprepad sändning ingår. Ögonblicksbild och AI-retur påbörjas inte.

Förstapartsmodulen `apps/bridge-lab/native-log-probe.lua` laddas i en befintlig experimentinbox; ingen klientbinär eller TOC ändras. Laddning och `/wispnative status` läser bara funktionstyp och de tre berörda CVar-värdena.

Ett uttryckligt `/wispnative test <run-id>`:

1. Kontrollerar funktion och inställnings-/timer-API. Vid saknad funktion avbryts försöket utan ändringar.
2. Sparar exakta ursprungsvärden för `developerLogFilterNormal`, `developerLogWriteToFile` och `developerLog`. Saknade eller oväntade värden gör att det avbryts före ändring.
3. Aktiverar bara värden som inte redan är `1` och verifierar återläsningen. Vid fel görs återställning.
4. Anropar `C_Log.LogMessage` exakt en gång med `WISPNATIVE1:<run-id>:<uptime>:<epoch>`. Samma ID kan inte återspelas.
5. Återställer ändrade värden efter 60 sekunder, vid `/wispnative stop` eller på utloggnings-/reload-event. Återläsning verifieras och eventuella återställningsfel redovisas; vid fel finns manuellt återförsök. Ursprungsvärden och kvittenser stannar i Wisp Bridge Labs lokala SavedVariables.

Mätperioden avslutas innan reload görs för att spara kvittensen. En markör som först syns efter återställning, reload eller klientavslut ska särskiljas från snabb leverans medan testet var aktivt.

## Oberoende filmätning

Vår befintliga read-only-mätare har fått ett separat `--protocol native`. Chatloggens tidigare protokoll är oförändrat. Första försöket följde endast `DeveloperLog.log`; den korrigerade upprepningen följde endast `General.log`. Mätaren utgår från filens ursprungliga slutposition, hanterar nyskapande/append/upptäckt trunkering och lagrar enbart exakt rätt markörs numeriska tider. Annan utvecklarloggtext sparas inte av mätaren. Originalfilen skrivs aldrig av mätaren.

Nominellt pollintervall: 50 ms. Faktiskt största intervall, diskobservationens vägg-/monotona tid och eventuell native loggtidsstämpel registreras. Spelklockans förhållande till mätarklockan måste beaktas vid latensberäkning; heltalssekunder från Lua är inte millisekundprecision. Utfallet gäller denna klient, denna markör och denna trafikvolym.

Körning: `5bf81eee6ba7`. Privata råkvittenser och filtrerade observationer finns i ignorerade `.cache/native-log-test/5bf81eee6ba7/`. Testet innehåller ingen allmän logginsamling och inga råa klientfiler publiceras.

## Lokala förkontroller

- Lua-tester: inaktiv laddning/status, enda markör, blockering av omsändning, timeout-återställning, kastat loggfel, setter som ändrar innan fel, nekad setter, saknad funktion och saknad CVar.
- Isolerat Python-fixturtest: rätt native-markör upptäcks, numeriska tider parsas och orelaterad loggtext hamnar inte i evidensfilen.
- Go-test kördes med `GOPROXY=off` och `GOSUMDB=off`, med redan befintliga lokala beroenden. Inga externa paket eller tredjepartslösningar lades till.

## Första försöket: funktion och destination verifierade

Klienten visade `C_Log.LogMessage=function`. Ursprungsvärden: `developerLog=0`, `developerLogWriteToFile=1`, `developerLogFilterNormal=1`. Endast `developerLog` ändrades. Kvittensen visar exakt ett accepterat anrop och automatisk återställning av `developerLog` till `0` efter 60 sekunder, utan återställningsfel. Användaren kontrollerade även att samtliga värden återgått till ursprungsläget före reload.

Markören återfanns efter klientavslut som en `[N][Lua]`-rad i **`General.log`**, inte i `DeveloperLog.log`. Den första kontinuerliga mätaren följde därför fel destination. En riktad genomsökning av klientens loggfiler tidigare under försöket såg ingen markör, men den ersätter inte kontinuerlig mätning av rätt fil. **Det går inte att fastställa första diskögonblick eller avfärda realtid från det första försöket.** DeveloperLog-filens fortsatta tomhet är inte ett negativt resultat för skrivningen till General.log.

## Upprepning med rätt fil

Körning `d2b6fb185c8d` använder samma förstapartsprobe, samma enmarkörsbegränsning och samma återställning, men filmätaren följer `General.log` från före klientstart. Den första markörens ID återanvändes inte. Evidens ligger separat under `.cache/native-log-test/d2b6fb185c8d/`.

Den riktiga Lua-kvittensen visar:

- klient 1.60.1 / build 70790 / Interface 16001;
- exakt en markör skickad via ett accepterat anrop;
- endast `developerLog` ändrades från `0` till `1`;
- `developerLogWriteToFile` och `developerLogFilterNormal` var redan `1` och ändrades inte;
- automatisk återställning av `developerLog` till `0` efter cirka 60 sekunder, verifierad återläsning och inga återställningsfel;
- användaren verifierade samma slutvärden innan reload.

| Kontroll | Resultat |
| --- | --- |
| Funktionen tillgänglig i vanlig addon-Lua | Ja |
| Ett märkt `C_Log.LogMessage`-anrop accepterat | Ja |
| Markör på disk under 60-sekundersfönstret | Nej |
| Markör på disk efter återställning och reload | Nej; General.log fortfarande 0 byte |
| Markör vid normalt klientavslut | Ja; exakt en matchande rad i General.log |
| Uppmätt native-loggtid → första diskobservation | **188,556 sekunder** |

Första diskobservation: **13:46:40,686 CET**. Den återfunna loggradens egen tidsstämpel var 13:43:32,130. Native-tiden och spel-uptime stämmer mot mätarens vägg-/monotona klocka inom cirka 0,7 ms. Nominell pollperiod var 50 ms; största uppmätta mellanrum var 104 ms. Pollosäkerheten är alltså liten i förhållande till den observerade fördröjningen.

Fördröjningen är ett **manuellt, avslutsutlöst resultat**, inte normal medel- eller p95-latens. Inga utfyllnadsmeddelanden skickades. Den slutliga General.log var 39 138 byte, mindre än 64 KiB; det bevisar ingen buffertgräns. Det är fortfarande oprövat om en begränsad lokal volym eller en tillgänglig flushmekanism kan ge snabbare skrivning.

## Slutsats och kvarvarande osäkerhet

Vi har nu verifierat en **helt lokal utgående skrivväg från vanlig addon-Lua till General.log**. Den kräver inte självviskningar, AI, processminnesåtkomst eller pixlar. Men en liten enstaka markör gav **inte realtid** i den uppmätta körningen. Det är därför för tidigt att använda vägen som livekoppling eller kalla den omöjlig under alla andra villkor.

Båda försöken aktiverade `developerLog` tillfälligt. Eftersom destinationen visade sig vara General.log är det inte bevisat att den inställningen behövs för `C_Log.LogMessage`; den kan ha varit överflödig. Ingen sådan variant har testats.

## Avslut och återställning

Båda mätarprocesserna är stoppade. Den tillfälliga inbox 3 är återställd från sin exakta förtestkopia med WoW stängt. Originalets loggfiler har inte raderats eller skrivits om av våra verktyg. Lua-kvittenserna exporterades med vår data-only-parser, inte genom att exekvera SavedVariables. Filtrerad evidens och `summary.json` för upprepningen ligger endast i ignorerade cachekataloger. Ingen publicering eller nätåtkomst gjordes.

Steg 2 och 3 i dagens plan har inte påbörjats. Eventuell nästa undersökning av buffertvolym är ett separat deltest av utgående transport och kräver en ny avgränsning.
