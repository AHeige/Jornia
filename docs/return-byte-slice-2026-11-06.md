# Wisp: återanvändbar byte-/textretur, 6 november 2026

## Resultat

Egen kontrollerad text skickades från datorn till den verkliga Lua-rutan genom **samma återanvändbara filnamn**, med kvittens tillbaka genom lokal OCR. Ingen ny inbox laddades per byte eller meddelande. Samma redan körande mottagare tog emot flera texter. Ett medvetet skadat meddelande avvisades innan text visades.

| Test | Nyttolast | Totalt på ledningen | Tid från första byte till avläst slutkvittens | Utfall |
|---|---:|---:|---:|---|
| `Hi` | 2 byte | 9 byte | 4 611,7 ms | Text och slutkvittens verifierade i WoW/OCR |
| `Again` | 5 byte | 12 byte | 8 399,6 ms | Samma filer återanvändes; ny text verifierad |
| `BAD`, fel checksumma | 3 byte | 10 byte | 5 209,9 ms | `Checksum failed — no text delivered`; felkvittens avläst |
| `åäö` | 6 UTF-8-byte | 13 byte | 7 959,3 ms | Korrekt åäö verifierades visuellt och via slutkvittens |

Samtliga tester kördes på Cone Thrower i Forever 1.60.1.70790 / Interface 16001, i samma klientprocess PID 8007 som före denna slice. Ingen reload, utloggning, omstart eller spelhandling användes. Inga AI-/serveranrop gjordes. Ingen bild lämnade datorn.

Detta är **en fungerande liten textkanal med återanvändbara bärare**, inte en färdig chattprodukt. Dagens bytevisa kvittens är för långsam för längre AI-svar. Nyttolasttaket är medvetet 96 byte; flera minuter av godtycklig trafik, Windows och riktiga AI-svar via denna kanal är inte verifierade.

## Exakt vad som byggdes

- `apps/bridge-lab/return-byte-bus.lua`: självtest, åtta bitars mottagare, klockövergångar, två stabila läsningar, sekvens/prefixkvittens, ramkontroll, normal textvisning och tidsgräns. Inga egna nätverks- eller fil-API:er inne i Lua behövs.
- `scripts/experiments/return_bus/bus.py`: stdlib-sändare som ändrar filnärvaro och väntar på matchande kvittens genom vår native OCR-helper. Den laddar aldrig en addon eller skickar tangenttryckningar. Filerna återställs efter testet, med kontroll av ursprungliga SHA-256.
- Python-tester för bytegränser, UTF-8, skadad OCR-kvittens, återanvändning och återställning. Ett Go-test kör den faktiska Lua-koden i befintlig Lua 5.1-testmiljö, inklusive flera meddelanden, korruption, ljudbortfall och karaktärsbyte.

Ingen stor dependency eller annan produkts implementation infördes. Testkoden är vår egen och använder befintliga plattforms-/testprimitiver.

## Hur signalerna bär data

Åtta redan kända egna filnamn motsvarar åtta databitar. Filen finns = 1, filen saknas = 0. Ett ytterligare känt namn är en **alternerande klockbit**. Sändaren ställer först alla databitar och ändrar klockan sist. Den lämnar därefter alla bitar oförändrade tills rätt kvittens kommer.

Lua ignorerar databitar medan klockan står kvar. Vid en ny klocknivå läses data och kontroller två gånger, med nominellt 50 ms mellan körningarna. Mottagaren svarar med sekvensnummer, bytevärde och Adler-32 för det mottagna prefixet. Synlig text med ytterligare kontrollsumma läses tillbaka av Vision. Sändaren kräver rätt överföringsnonce, karaktärsbindning, sekvens, bytevärde och prefixdigest innan nästa byte skrivs.

Detta är inte en atomisk flerfilsoperation. Data-före-klocka, stabila prover och frysta data tills kvittens är vad som gör denna lilla testmodell kontrollerbar. Fördröjningar/filrace under andra förhållanden behöver fortfarande undersökas.

Ramen är `WP` + en längdbyte + nyttolast + fyra big-endian-byte med Adler-32 över header och nyttolast. Ett fullständigt godkänt paket blir `WISPDONE`; skadad checksumma ger `WISPFAIL`, felkod 3. Innan godkänd slutkontroll visas inte nyttolasten som ett svar. Kontrollsummor och synlig bindning är korruptions-/staleskydd, inte kryptering eller autentisering mot en lokal angripare.

Mottagaren kontrollerar också en permanent känd fil och en saknad kontroll. Ett otillgängligt ljud-API ska inte få alla databitar att tolkas som nollor. Karaktärsbyte stoppar den aktuella överföringen. En saknad/skadad kvittens leder till begränsad väntan och fel; ingen automatisk byte- eller AI-omsändning implementerades.

## Bärare och återställning i det verkliga testet

Den pågående klienten hade inga åtta dedikerade datafiler för denna nya modell. Vi använde därför åtta **egna labb-TOC-filer** som redan var kända: `WispOCRInbox081..088`, plus vår tidigare tomma `WispBridgeLab/wispreturn_pulse.wav` som klocka. Innehållet av dessa TOC-filer lästes inte som paket; endast närvaron observerades. Alla ursprungliga byte behölls i samma katalog via tillfälliga `.wispbus-hold`-namn och flyttades tillbaka. Ingen annan addons filer användes.

Detta är ett avgränsat sätt att undvika reload för forskning i ett pågående spelpass, **inte installationsdesignen**. En riktig Wisp-installation behöver separata, egna transportfiler och uppstarts-/uppdateringsregler. En updater och denna experimentella sändare ska inte samtidigt ändra bärarna.

Den gamla OCR/AI-läsaren pausades under testet för att isolera experimentet. Den lokala companion/servern och databasen ändrades inte. Inga vanliga frågor skickades eller upprepades.

Mottagarkoden laddades en gång genom den redan installerade, oanvända `WispBridgeInbox03`. Under sessionen laddades även en egen UI-uppdatering via inbox02 för stängning/progressvisning. Dessa var **bootstrap av testkod**, inte returdata per meddelande. Följande överföringar använde bara filnärvaro och `/wispbus` för att armera om samma protokoll. Kommandot nollställer våra mottagarvariabler; det laddar inte om WoW eller någon addon.

Efter testerna kontrollerades alla nio bärarnas innehåll, inga hållfiler lämnades, och de ursprungliga inbox02-/inbox03-filerna återställdes. Testmottagaren finns kvar i RAM i detta UI tills det naturligt upphör, men är inte beroende av att spara eller stänga klienten för att testet ska räknas som lyckat.

## Prestanda och vad siffrorna betyder

Den uppmätta nyttolasttakten var cirka **0,43–0,75 byte/s** för dessa mycket korta paket. Header och checksumma upptar sju byte, och en OCR-kvittens krävs för varje byte. Det är medvetet konservativt. Det är inte försvarbart att använda dessa testtider som ett löfte om snabb AI-chatt.

De registrerade enskilda OCR-läsningarna för bytekvittens låg i dessa körningar ungefär mellan **36 och 142 ms**. Totaltiden inkluderar dessutom skärmfångst, intervall mellan fångster, stabila Lua-läsningar, filoperationer och bytevis stop-and-wait. Initiering/Vision-uppvärmning och väntan före den första publicerade byten ingår inte i tabellens transporttid. Slutkvittensen lästes genom OCR; faktisk text och felmeddelande kontrollerades också i det verkliga spelet med CUA.

Lua visade noll API-fel och en diagnostikräknare med 9,1–13,5 ms summerat arbete i vissa samplingsgrenar. Tidiga returvägar/tomgång och hela spelets bildfrekvens ingår inte i den räknaren. **Ingen FPS- eller CPU-kostnadsgaranti följer av den siffran.**

Nästa hastighetsförsök bör minska antalet OCR-rundresor: flera byte per kontrollerat block, fler parallella databitar eller ett uttryckligt tidsstyrt fönster, med blocksekvens och checksumma samt begränsad omsändning. Valet ska föregås av mätning av fler `PlaySoundFile`-anrop och filuppdateringar; obegränsat många anrop per frame är inte motiverat av detta test.

## Verifiering och gränser

- Tre nya Python-tester passerar; companion-paketets tester passerar inklusive den faktiska Lua-mottagaren. Binärgränser `00/55/AA/FF`, tom nyttolast och UTF-8 täcks lokalt. De fyra livekörningarna är listade ovan; det är inte ett långt stresstest.
- Inga nya filnamn behövdes efter att mottagarkoden laddats. Samma bärarnamn återanvändes över meddelandena. Detta undanröjer labbens engångsinboxförbrukning **för just denna överföringsmetod**; det ändrar inte hur den gamla OCR/AI-prototypen fungerar.
- Handshake per byte är verifierad genom kvittenser som driver sändarens nästa steg. Återhämtning efter tappad kvittens, pausad skärmfångst, abrupt processdöd eller bytefel är fortfarande begränsad. Hållfiler kan återställas med den fasta `restore`-åtgärden; ett avbrutet meddelande blir inte automatiskt ett nytt meddelande.
- GUID-byte/sound-down provas i Lua-tester, inte genom karaktärsbyte eller ljudinställningsändring i det riktiga spelet. Ljudinställningar lämnades oförändrade.
- UI:t använder fasta textpositioner utan rörelse för att ge stabil OCR. Kontrollerade väntan-/slut-/feltexter är avsiktligt enkla. Detta ersätter inte Wisp-produktens journey/chattdesign.
- Ingen internetresearch, GitHub-push, VM-ändring, betalmodell eller offentlig release ingick i denna slice.

Råa lokala körloggar finns i ignorerad `.cache/return-bus-2026-11-06/{hi,again,corrupt,utf8}.jsonl`. Körning och försiktighetsregler finns i `scripts/experiments/return_bus/README.md`.

Efter verifieringen stoppades och doldes byte-labbpanelen med `/wispbus hide`, vilket också kontrollerades visuellt. Den vanliga OCR/AI-läsaren återstartades mot samma lokala companion. Ingen fråga skickades i samband med återstarten. Byte-sändarna och deras native capture-processer är avslutade; inga filändringar pågår i bakgrunden.
