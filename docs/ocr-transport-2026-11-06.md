# Wisp: visuell OCR-transport, 6 november 2026

## Användarens nya riktning

En egen Lua-chatt är öppen inne i WoW. När spelaren trycker Enter ska frågan kunna läsas av från rutan och skickas till Wisp → backend → Berget → backend → ett svar i addonen. Detta prioriteras nu före SpeakText. Inga reload-, utloggnings- eller klientavslutssteg ska behövas för transporten.

## Första lokala beviset

Vi körde egen Objective-C-kod mot macOS inbyggda Vision-API på en av användarens redan bifogade WoW-bilder. Inga nya skärmbilder togs, inga tangenttryckningar övervakades, inga OS-behörigheter ändrades och ingen bild eller OCR-text skickades till en server.

- Nio textrader upptäcktes.
- `C_Log.LogMessage=function` och de synliga inställningsvärdena gick att läsa.
- Första anropet tog 7,673 sekunder; två efterföljande körningar tog 0,210 respektive 0,185 sekunder.
- Ett långt funktionsnamn fick ett extra mellanslag, trots rapporterad confidence 1. Confidence är inte ett bevis på korrekt text.

Detta är en **filbaserad OCR-kontroll**, inte ett verifierat liveflöde. Bilden är en beskuren faktisk WoW-bild, inte en fullskärmsbenchmark eller ett test av en färdig Wisp-utkorg. Ingen precision/latens för den färdiga produkten kan utlovas från dessa tre körningar.

Koden finns i `scripts/experiments/read_image_ocr.m`. Den använder bara plattformens Foundation/Vision för OCR-primitiven; all produktlogik ägs av Wisp. Ingen extern OCR-distribution eller färdig companion-lösning installerades. Swift-varianten stötte på ett lokalt compiler/SDK-moduleproblem; det fungerande beviset använder Objective-C. Systemets utvecklarverktyg ändrades inte.

## Föreslaget Enter-flöde

1. Lua hanterar Enter i sitt eget textfält och skapar ett stabilt, läsbart **utkorgskort** med frågan, ett unikt request-ID och karaktärskoppling. Kortet ligger kvar; det är ingen enbildsblinkning som en bildläsare lätt missar.
2. Companion hittar Wisp-panelens avgränsning i det uttryckligt valda WoW-fönstret. För själva läsningen används panelregionen. Vanlig chatt, andra appar och hela skärmen ska inte skickas till backend.
3. Lokal OCR läser och validerar kortet. Identifierare, avgränsare och en kontrollsumma bör vara tåliga att läsa och ge ett säkert avslag vid fel. Flera identiska läsningar kan hjälpa men ersätter inte validering. Samma request-ID får aldrig orsaka upprepade betalda anrop.
4. Endast den validerade frågetexten och den uttryckligt valda kontexten går till befintligt Wisp-API. Ingen generell tangentlogger behövs för att upptäcka en stabil synlig utkorg. Bilden behöver inte lämna datorn för denna lösning.
5. Lua visar begriplig status för väntande/sänd/besvarad fråga. Status ska motsvara en riktig kvittens, inte bara ett OCR-försök. Karaktärsbyte eller ett gammalt kort får inte kunna koppla ett svar till fel karaktär.

Live skärmfångst på Mac kräver separat hantering av macOS skärmåtkomst. Ingen sådan åtkomst begärdes i filtestet. Exakt fönsteridentifiering, lokalisering efter flytt/skalning, textstorlek, radbrytning och läsfel behöver verifieras i den riktiga klienten. En förberedd OCR-läsbar layout är en del av lösningen, inte kosmetik.

## Returvägen är fortfarande separat

OCR löser endast utgående data. Två svar genom lokala, ännu oladdade addon-inboxar är tidigare verifierade. Dessa har ett ändligt antal platser och en plats laddas en gång per UI-livstid. Hur addonen får veta att svaret är redo innan en plats används, samt uthållighet för många turer, är ännu inte löst.

Det första e2e-testet kan pröva ett enskilt korrelerat svar med en känd inbox. Detta ska inte presenteras som obegränsad eller färdig automatisk chatt. Ingen minnesskrivning, osynlig tangentinmatning eller ny otestad returmekanism införs som en underförstådd del av OCR-förslaget.

## Nästa avgränsade test

**En fråga, ingen AI ännu:** visa en kontrollerad fråga i en egen Lua-utkorg; tryck Enter; läs den lokalt från den pågående bilden medan spelet fortsätter. Kräv rätt fråga, karaktär och request-ID, ingen dubbelsändning och uppmätt Enter → validerad läsning. Prova felaktig/avklippt text som avslag. Först därefter prövas samma request genom AI och retur-inbox.

Status: lokal bild-OCR verifierad. Livefångst, utkorgs-UI, Enter-detektion och hela AI-returen är ännu inte implementerade/verifierade i detta experiment. Ingen nätåtkomst eller GitHub-push användes.

## Implementerad e2e-labb, samma dag

Den senare användarinstruktionen prioriterade hela e2e-kedjan. Därför finns nu en separat `Wisp OCR Lab`, egen Python-controller (standardbibliotek), native Objective-C-fångst med ScreenCaptureKit/Vision och ett uttryckligt `-ocr-lab`-läge i befintlig Go-companion. Ingen ny stor dependency infördes. Se `scripts/experiments/ocr/README.md` för körning, datagränser och begränsningar.

Returhypotesen ovan är nu implementerad med 96 separata förinstallerade LoD-platser, inte en enstaka manuellt utlöst inbox. Addonen förbrukar nästa plats var 600 ms medan en fråga väntar. Controllern skriver sänd-/mottagen-/slutstatus till samtliga egna inboxfiler när status faktiskt ändras. Detta kostar ett ändligt antal platser och bevisar inte obegränsad chatt.

Utgående kontroll kräver två identiska, kompletta läsningar och kontrollsumma över request-ID, karaktärsbindning och fråga. Frågan är begränsad till 160 ASCII-tecken i labben. Ingen journal skickas automatiskt. Faktiska journalutdrag kräver fortsatt separat granskning. Det gamla `Nästa avgränsade test`-avsnittet ovan beskriver tidigare plan, inte kvarvarande implementeringsstatus.

### Första körningen i verklig WoW-klient

Användaren startade WoW normalt efter installation och skrev `What can you tell me about Ashenvale?` på Cone Thrower. Request `272478521793971663000001`:

- Sista validerande läsningen: **59,3 ms OCR**, **207,0 ms fångst + lokalisering + OCR**.
- Två matchande läsningar krävdes; detta är inte ett mått på hela Enter→fråga-mottagen.
- Native Lua-status `Question received — Wisp is thinking…` observerades visuellt i det pågående spelet. Därmed fungerade utgående OCR → lokal companion → kvittens via automatisk fil-inbox.
- AI-steget misslyckades: API-loggen på VM:n visade `inference returned HTTP 502`. Controllerns API-/filsteg tog 5 275 ms innan fel publicerades.
- Även felet kom tillbaka automatiskt och observerades i samma WoW-fönster: `Could not finish — your question is kept`. Ingen reload, utloggning, klientstängning eller manuell inboxladdning användes.
- En separat liten modellkontroll på VM:n lyckades därefter via befintlig LangChain/Berget. Grundorsaken till det föregående modellfelet är inte belagd. Den ursprungliga frågan skickades inte om automatiskt.

Tre Python-protokoll-/retur-/feltester samt companion-paketets Go-tester passerade, inklusive Lua 5.1-körning med korrelerade svar, fel karaktär, vänteläge och avslag på dubbelsändning. Ett syntetiskt crop/OCR-test läste och validerade en kontrollrad som Vision delade upp i två observationer; 149,6–214,4 ms i de lokala fixture-körningarna. Ingen browser-/internetresearch eller GitHub-push användes; server-/Berget-anrop avsåg det uttryckligt begärda e2e-testet.

Status vid denna anteckning: live OCR, API-kvittens och automatisk felretur verifierade; ett **riktigt AI-svar hela vägen tillbaka** återstår att bekräfta.

### Slutförd kedja: riktigt Berget-svar i Lua

Efter två ytterligare felkörningar (`…000002`: 81,1 ms OCR; `…000003`: 47,6 ms OCR) isolerades orsaken med säker feldiagnostik: **`LengthFinishReasonError`** i modelladaptern. Den gamla maxgränsen på 220 output-token kunde avbryta det strukturerade svaret. Felet var alltså inte bevisat tillfälligt eller ett OCR-fel. SDK-/modelladaptern omvandlade avbrottet till HTTP 502.

Åtgärd: Go-API:t och modelladaptern accepterar nu högst **512 output-token**. Go äger fortsatt instruktionen om kort svar: högst 60 ord och högst fyra direkt relevanta evidence-ID:n; välj ett smalare svar om fler belägg skulle krävas. Detta är en promptinstruktion, inte en garanterad ordgräns. Inga automatiska återförsök eller nya bibliotek infördes. Oavslutade/ogiltiga svar förblir fel, aldrig fabricerade lyckade svar.

**Request `272478521793972374000004` lyckades och observerades visuellt i samma oavbrutna WoW-session på Cone Thrower.** Det var samma Ashenvale-fråga. Svaret beskrev zonen och namngivna platser, med källor samt varningen om att source_checked inte bekräftar aktuellt spelstate.

| Mätning | Uppmätt värde |
|---|---:|
| OCR, sista av två matchande läsningar | 46,2 ms |
| Fångst + lokalisering + OCR, samma läsning | 212,4 ms |
| Backendens rapporterade svarstid | 4 382 ms |
| Lokal API-acceptans/pollning + publicering av svarsfiler | 4 631,5 ms |
| Enter → verklig Lua-callback, visat i spelet | **6,1 sekunder** |

Operationens route var `berget` och modellen `mistralai/Mistral-Small-3.2-24B-Instruct-2506`. Den bekräftade användningen var 5 823 prompt-token och **229 svarstoken**; rapporterad uppskattad kostnad €0,0018156. Det färdiga svaret behövde alltså fler än den gamla gränsens 220 svarstoken. `reply_files_written` användes inte ensamt som leveransbevis: den verkliga Lua-rutan visade `Reply received — 6.1 s total` och själva svaret. Mätvärdena överlappar och ska inte summeras som oberoende steg. Den fördröjning som återstår utöver backend omfattar bl.a. två OCR-läsningar, pollningsintervall och kvittensleverans.

Samtliga Go-pakettester (`go test ./...`), tre Python-bridge-tester och två modelladaptertester passerade. Modelltesterna körde nätverksisolerat med falsk provider, inklusive kontroll att frågetext/nycklar inte hamnar i felloggar. Native-helpern kompilerades mot lokal SDK. Riktig on-screen-väntan, fel och lyckat svar kontrollerades med CUA. Ingen reducerad-rörelseinställning behöver ändra labben: statisk text valdes uttryckligen för läsbarhet och OCR-stabilitet.

### Drift och kvarvarande begränsningar

- VM: modell `wisp-private-model:20261106-ocr2`, API `wisp-private-api:20261106-ocr1`. Endast Wisp modell/API återskapades, med befintliga resursgränser och portar. De **20 övriga** löpande containrarnas ID:n var oförändrade efter slutuppdateringen. Diagnostiksteget före detta ändrade endast modellcontainern och bevarade övriga 21 ID:n.
- Föregående images finns kvar. Ägarprivata `.env.pre-ocr-model-diagnostics` och `.env.pre-ocr-output-fix` finns i VM-projektet för återgång; inga nycklar skrivs i denna dokumentation. Ingen prune, databasändring eller brandväggsändring utfördes.
- Macens installerade companion kör tillfälligt med det uttryckliga `-ocr-lab`-läget, på 8788 och med användarens befintliga databas. Normala launcher ändrades inte. Läsaren kör bara medan dess lokala process lever. Offentliga releases och nedladdningssidan ändrades inte.
- Returpoolen är fortfarande **ändlig**. Vi har visat en fungerande E2E-tur och felreturer, inte hur många normala samtal som fungerar eller hur obegränsad retur ska lösas.
- Labbens svarsyta har fast höjd; långa källtitlar/timingtext kan klippas. Själva lyckade svarstexten och osäkerhetsnotisen var synliga. Scrollbar historik, beständig väntestatus, bred språk-/teckentäckning, fler upplösningar och karaktärsbytes-test i verkligt spel återstår.
- Detta test skickade **bara frågetext**, inte aktuell questlista eller journalutdrag. Personligt/live questunderlag är nästa separat avgränsade steg. Att chatttransporten fungerar bevisar inte komplett eller korrekt Forever-världsdata.

**Aktuell status: första riktiga OCR → AI → Lua-kedjan är verifierad utan reload, utloggning, klientavslut, minnesläsning eller global tangentövervakning.**
