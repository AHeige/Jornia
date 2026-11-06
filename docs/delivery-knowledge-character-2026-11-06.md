# Wisp – världskunskap, karaktärsfrågor och testpaket

Beställt 2026-11-06. Arbeta i ordning och bocka endast av verifierade resultat.

- [x] 1. Inventera nuvarande datakällor, databas, VM-resurser och befintliga scheman.
- [x] 2. Uppdatera tillgänglig Forever-världskunskap och förbättra fortsatt insamling utan att blanda spelversioner eller tillskriva okända källor återanvändningsrätt.
- [x] 3. Installera/verifiera avgränsade schemalagda serverkörningar, återupptagning, aktualitet, rensning och driftstatus.
- [x] 4. Verifiera och vid behov komplettera AI-frågor om vald karaktär, utrustning, aktiva quests och lokal journey med korrekt aktualitet och karaktärsisolering.
- [x] 5. Kör en bred frågematris: värld, Forever-skillnader, karaktär/journey, saknade fakta, konflikter och fel. Redovisa verkliga AI-körningar separat från deterministiska tester.
- [x] 6. Bygg och verifiera nedladdningspaket med dokumenterade funktioner och begränsningar. Publicera inom tidigare godkänd GitHub-distribution om befintligt releaseflöde stöder det.
- [x] 7. Kontrollera installation/drift, uppdatera dokumentation och rapportera resultat samt kvarvarande dataluckor.

## Avgränsning

”All fakta ever” är ambitionsnivån för täckning, inte en mätbar färdigstatus. Inga källor bevisar komplett kunskap om hela spelet. Importera brett från tillåtna källor, visa faktisk täckning och behåll explicit okänt där underlag saknas. Spelarjournaler förblir lokala; privata karaktärsdata ska inte bli centrala världsfakta. Paketering ska inte påstå att den experimentella återanvändbara Lua-returen redan är integrerad.

## Resultat per steg

Påbörjat. Mätvärden, körningar och artefakter förs in efter verifiering.

### 1. Inventering

VM har 11 GiB ledig disk och cirka 5,3 GiB tillgängligt RAM. Wisp-databasen är 72 MB. Befintlig halvtimmesimport fungerar redan: 30 329 entiteter, 6 888 rå-/detaljdokument, 24 215 köade jobb; senaste åtta batcher lyckades. Detta är inte fullständiga quest-/NPC-fakta. Aktiv API-version är `static-1.60.1_70399-forever-eu`, äldre än testklientens 70790. Övriga containrar inventerades före ändringar. ForeverDB kräver godkänd utvecklaråtkomst; QuestieDB:s historiska rättigheter är fortfarande oklara. Ingen sådan databas kopieras utan klarlagd åtkomst.

### 2–3. Insamling och drift

En ny 500-anropsbatch slutfördes 15:54:49 UTC, följd av ombyggnad av 2 000 giltiga cachedokument till projektion v2. Fakta som API:t uttryckligen ger om krav, belöningar, kvalitet, föremålsklass och tooltips kan nu användas. Cacheombyggnad behåller ursprunglig hämtningstid, expiry och uppdateringsdeadline. Ingen questberättelse, spelgrafik eller separat communitydatabas bulkimporterades.

Befintlig cronrad ersattes: var 15:e minut, högst 500 anrop, minst en sekund mellan anrop, 12 minuters arbetsdeadline. Daglig credential-oberoende rensning 03:17 enligt VM:ns tidszon. `flock` och databaslås förhindrar överlapp. Refresh avstår under 3 GiB ledig disk eller 512 MiB tillgängligt RAM; rensning tillåts även då. Ingester har fortsatt 192 MiB RAM/0,5 CPU. Inga ändringar i andra projekts containrar, proxy eller brandvägg. Separat importer-image gör att importern kan uppdateras utan API-/modell-/databasomstart.

Rensningen gav `purged`, runner gav exit 0, och cron startade nästa verkliga batch 16:00:01 UTC. Jobb återupptas från PostgreSQL. `ingestion-last.log`, föregående logg och atomisk `ingestion-status.json` är begränsad driftdiagnostik; `ingestion_runs` och `/api/knowledge/coverage` visar faktisk körstatus/täckning. Saknad källa eller 404/429 skapar inga nya fakta. Fullständighet redovisas fortfarande som falsk.

### 4. Personliga frågor

Vanliga frågor om nivå, klass och utrustning fungerar lokalt/offline när respektive källa finns. AI kan få en uttryckligen granskad projektion av journal + daterad Armory, även med endast profil och tom journal. Namn, GUID, realm, porträtt, anteckningar och konversationshistorik lämnas utanför. Profiländring/expiry gör gammalt godkännande ogiltigt. Karaktärsbyte och bekräftat svar nollställer delning. Källa `character:profile` binds till datum/expiry; journalens nivå hålls åtskild från den äldre profilen.

Aktiva quests, väskor, guld och talanger finns inte i denna releases underlag. Frågor om dessa ska avstå från att gissa; de är inte implementerade som liveinsamling. Detta har uttryckliga negativa tester. Webbläsartester visar förhandsgranskning och samtycke på desktop/mobil och med reducerad rörelse.

### 5. Frågetester

Slutlig verifiering hittills: 46/46 befintliga acceptansfall mot VM, 50/50 nya deterministiska fall (representanter från 18 API-datatyper på två språk + okända fakta/spelversionsgränser), 40/40 lokala frågor i den faktiskt uppackade Mac-binären och 18/18 verkliga AI-frågor mot Berget. AI-skriptets 32/32 inkluderar dessutom 14 negativa icke-modellfall som redan finns i den breda matrisen; de dubbelräknas inte. Totalt 154 frågefall innan separat HTTPS-paketprov. Detta är en avgränsad regression, inte ett löfte om att alla möjliga frågor blir rätt.

Testningen hittade och rättade: API-katalogpost med samma namn som en granskad quest skymde relationer; slutparentes gav missad SQL-träff; för stor världskontext blockerade annars besvarbara personliga frågor; modellen behövde tydligare åtskillnad mellan profil och journal. Första AI-omgången var 29/32, omgången efter rättning 32/32. Cache-/promptändringar prövades med nya avsiktliga testanrop, inga osäkra betalda anrop spelades om automatiskt.

Go-tester och vet passerar, liksom PostgreSQL-integrationsfall i separata tillfälliga scheman och fyra webbläsarsviter. Både utvecklingsbinären och uppackade Mac-paketet har genomgått test av förhandsgranskning, profildelning, identitetsmaskning, offlinehjälp, mobilvy och reducerad rörelse. Modellsvar kontrollerades för status, förväntade fakta och tillåtna källhänvisningar; de personliga svaren granskades också manuellt. Slutlig AI-matris hade uppskattad modellkostnad 0,011606 EUR, första omgången 0,010887 EUR. API-importerna gör inga modell-anrop.

### 6. Paket

0.6.0 för Windows amd64 och Mac arm64 är byggt. SHA-256:

- Windows: `4fd19b1f617f6a8a1ba541718c95e5fe470254c7b4e7183661ce36250ea3a42a`
- Mac: `91be60f95dcc2f57c7b0c76271b91cb4af067d25b38836e53cc6093420ca92ba`

Arkivintegritet, arkitektur (PE32+ x86-64 / Mach-O arm64) och kontroll mot privata filer/faktiskt konfigurerade hemligheter passerade. Uppackade Mac-binären testades i nytt datalager. 40 offlinefrågor + fyra verkliga HTTPS-frågor genom gateway → Go → LangChain → Berget passerade med syntetisk profil/journal. Bland annat blev svaret korrekt att journalen registrerat nivå 16 medan den äldre profilen visar 15. Den kortlivade testkoden återkallades och tokenfilen togs bort. Kompisens kvot/kod ändrades inte.

Paketet innehåller produktaddonen och desktopchatten. OCR/blockretur ingår inte. Windows-binären är korskompilerad, inte körd på kompisens dator; båda binärerna är osignerade. Den privata källkoden hålls separat från den publika webb-/release-repon.

### 7. Samlad täckning och gränser

Efter manuella och schemalagda uppdateringar: 30,578 serverbara entiteter, 7,886 giltiga API-dokument, 23,467 kvarvarande jobb. Vid start var det 30 329 entiteter och 6 888 dokument. Antalet rå-/detaljdokument ökade med 998; två klassindexprober ersatte befintliga dokument. Databasen är 87 MB och VM:n har cirka 11 GiB ledigt. Aktiv namespace är fortfarande `static-1.60.1_70399-forever-eu`: ingen ny klientbuild har hittats eller påståtts vara verifierad.

Detta omfattar bland annat 7,051 questposter, 17,537 föremål, 877 NPC-poster och 1,895 abilities. Antal är inte fullständiga mål, dropptabeller eller spawnpositioner. Nya batcher fortsätter automatiskt, men kan aldrig garanteras ge ”all fakta ever” när källan saknar uppgifter. ForeverDB-access är nästa kompletterande källberoende; ingen nyckel eller generell återpubliceringsrätt har antagits.

Alla 158 slutliga frågefall passerade (154 + fyra HTTPS-paketfall). Testning kan inte bevisa korrekthet för godtyckliga frågor. Båda ZIP-filerna är osignerade och Windows har inte körts i denna leverans. Realtids-/blockkanalen är fortfarande separat forskning. Inga personliga spelhändelser från användaren behövde skickas för verifieringen.

Publicerad release: [v0.6.0](https://github.com/AHeige/wisp-site/releases/tag/v0.6.0). Båda ZIP-filer laddades därefter ned anonymt och matchade exakt de lokalt testade SHA-256-värdena. Nedladdningssidans uppdatering finns i publicerade commit `1213876`; Pages-driftsättning kontrolleras separat innan slutrapport.

Slutkontroll: GitHub Pages rapporterar `built`, workflow `success`, och den verkliga sidan visar 0.6.0 samt rätt Windows-/Mac-länkar. Publicerad webbcommit: `1213876`. Slutlig API-image `20261106-knowledge5` är healthy; separat ingester `20261106-knowledge2`, modell/gateway/databas är kvar. Ogiltig användning av Forever-profil med Retail-scope avvisades med HTTP 400 före modellsteget. Tidszonen är Europe/Stockholm, alltså daglig rensning 03:17 svensk VM-tid. Alla inventerade andra projekts containrar behöll sina ID:n under arbetet.

[Nedladdning](https://aheige.github.io/wisp-site/) · [Release och SHA-256](https://github.com/AHeige/wisp-site/releases/tag/v0.6.0).

Checklistan är slutförd för den verifierade leveransen. Komplett spelkunskap, godkänd ForeverDB-access, liveinsamling av aktiva quests och integrerad ingame-retur kvarstår som produktarbete, inte som påstått färdiga funktioner.
