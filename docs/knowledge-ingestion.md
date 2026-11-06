# Forever-data: insamling, versioner och täckning

Undersökt 2026-11-05. Målet är bred import av återanvändbar data och löpande uppdatering, med explicit redovisning av vad som saknas. Ingen källa som undersökts ger ett verifierat, komplett och fritt återpublicerbart Forever-korpus.

## Källval och faktisk åtkomst

**Officiella Blizzard Game Data API:** förstahandskällan för denna implementation. Den nya Forever-ytan rapporteras av utvecklaren bakom [blizzard_api 0.8.0](https://github.com/Pinta365/blizzard_api/releases/tag/0.8.0), med identifierade namespace och skillnader mot Classic/Retail. Den rapporten var ett upptäcktsspår; Wisp gjorde därefter egna autentiserade anrop och verifierade svarens namespace. Första levande körningen angav `static-1.60.1_70399-forever-eu`, region EU, språk en_GB. Det är API:ts uppgivna version, inte ett påstående om att den är samma som senaste spelklienten.

API-data används enligt [Blizzards Developer API Terms](https://www.blizzard.com/en-us/legal/a2989b50-5f16-43b1-abec-2ae17cc09dd6/blizzard-developer-api-terms-of-use). §18 anger max 30 dagars TTL; Wisp använder 28 dagar och schemalagd rensning. Detta är ingen öppen datalicens. Nycklar finns endast i operatörens ignorerade `.env`, på utvecklingsmaskinen och den privata VM:n. De skickas i OAuth/Authorization-header, aldrig som URL-parametrar eller i klientpaket. Ingen spelarprofil, guild, auktion eller kontodata hämtas av denna worker.

**QuestieDB:** [LICENSE.md](https://raw.githubusercontent.com/Questie/QuestieDB/master/LICENSE.md), [PROVENANCE.md](https://raw.githubusercontent.com/Questie/QuestieDB/master/PROVENANCE.md) och [Forever-beskrivningen](https://raw.githubusercontent.com/Questie/QuestieDB/master/docs/forever.md) kontrollerades igen. Historiska rättigheter är fortfarande uttryckligen oklara och Forever-underlaget beskrivs som ofullständigt. Ingen bulkimport görs på antagandet att allt är GPL.

**ForeverDB:** [utvecklar-API:t](https://foreverdb.net/developers) erbjuder den typ av kompletterande quests, givar-/mottagarrelationer, positioner och observationer som Blizzard-katalogerna inte bevisar. Det kräver en separat godkänd utvecklaråtkomst, attribution och efterlevnad av dess regler. Det är en möjlig komplettering, inte en genomförd import eller ett antaget fritt databasuttag. Klassisk data och observationer måste hållas åtskilda även där.

**Warcraft Wiki:** den befintliga smala handgranskade anpassningen behålls. Direkt bulkåtkomst returnerade 403; ingen sådan spärr har kringgåtts. **ForeverDiff/pfQuest, 60.tools och GearInsight** är kompletterande jämförelsekandidater. En Classic-baslinje eller en kodlicens innebär inte verifierad Forever-data eller en generell rätt att kopiera en annan webbplats databas.

## Vad workern gör

`apps/ingest` använder Go och den befintliga PostgreSQL-databasen. Den behöver ingen LLM, embeddings eller extra modellkostnad.

1. Hämta ett OAuth Client Credentials-token på servern.
2. Fråga Forever-klassindex för aktuell API-namespace. Kontrollera officiell värd och Forever-namespace i svars-/resurslänkar; inga Classic- eller Retail-fallbacks.
3. Följ tillåtna publika game-data-index och resurslänkar. Upptäckt omfattar bland annat klasser, raser, specialiseringar, yrken, quests/kategorier/områden, föremål, spells, creatures, factions, item sets, journal-encounters/instances, mounts och achievements där API:t faktiskt svarar.
4. För sök-API:n kan resultat kapas. Workern delar därför ett ID-intervall i mindre intervall tills resultatet kan pagineras inom gränserna. Ett kapat första resultat räknas aldrig som hela databasen. Sökintervallet är uttryckligen 0..2147483647; det bevisar inte att alla spelposter är publicerade i API:t.
5. Spara API-payload, checksumma, hämtningstid och expiry. Projektera identifierare, namn, utvalda numeriska värden och uttryckliga API-referenser till grafen. Interna markörer som `[DNT]` avvisas från den serverade grafen. Ett faktiskt internt testyrke i API-indexet fångades av detta filter.
6. Varje entitet och källa identifieras med namespace/build och språk. Samma numeriska ID i två builds blir olika grafposter. Servern använder den aktiva upptäckta API-builden; äldre rådata får finnas kvar endast inom sin TTL. Klassisk data och handgranskade communityuppgifter är egna källor, inte automatiska API-korrigeringar.
7. Spara händelser/referenser atomiskt per API-svar. Ny indexdata ersätter bara sin egen projektion. En indexreferens får inte skriva över ett redan hämtat detaljsvar. Ett tomt fält är okänt, inte noll; en API-länk får inte automatiskt bli questkrav eller dropchans.
8. Återuppta köade jobb efter avbrott, kontrollera nya källversioner, uppdatera resursdata normalt efter 24 timmar och rensa utgångna API-kopior. En ändrad projektion kan byggas om från giltig cache utan att förlänga dess ursprungliga expiry.

På VM:n har en batch nu högst 500 game-data-anrop var 15:e minut, minst en sekund mellan anrop och högst tolv minuters körtid. Fel/404/429 sparas som retry/not_available/rate_limited, inte som nya spelfakta. API-token och hemligheter loggas inte. Hårdvaruresurser begränsas separat i VM:s Compose-profil. Grafen, dokumenten och jobbstatus tillhör endast Wisp.

## Rådata, graf och täckning är olika saker

En indexpost med ett questnamn är inte ett komplett quest. Fulla tillgängliga JSON-svar behålls tidsbegränsat, men alla deras fält är inte färdigmodellerade eller tillgängliga i svarsroutningen. Grafprojektionen innehåller initialt valda katalogfakta och källreferenser. NPC-spawns, dropchanser, fulla mål, kedjor och behörighetsvillkor ska inte härledas ur att ett ID förekommer.

`GET /api/knowledge/coverage` anger därför alltid `complete: false`, antal serverbara poster per datatyp, aktiva API-namespace/språk, antal dokument, detaljdokument per resurstyp och köstatus. `record_counts_are_not_field_completeness` markerar uttryckligen att antal poster inte är ett mått på fullständiga fält. Totalmängden i själva spelet är okänd. 404, tomma index, kapade sökningar och API-lag får aldrig döljas som 100 procents täckning.

SQL-sökning med GIN-ordindex hittar ankare före en begränsad tvåhoppshämtning. Det gamla upplägget som bara laddade de första 100 posterna är ersatt i serverflödet. Go äger urval, källor och versioner; LangChain förblir ett tunt modellsteg. Modellkontext har en separat storleksbudget och ingester använder inga modell-anrop.

## Köra och granska

```sh
python3 scripts/ingest.py --max-requests 200
TEST_DATABASE_URL='postgres://wisp:wisp-local-only@127.0.0.1:55471/wisp?sslmode=disable' make check
```

På VM:n finns den separata Compose-profilen `ingestion`, med kommandot `wisp-ingest`. Scheduler och faktisk driftsättning redovisas i VM-dokumentationen. Den centrala modellen lagrar inga privata spelarjournaler.

Råkopior får inte arkiveras obegränsat i dumpfiler, loggar eller distributionspaket. Tillfällig överföringsdump tas bort efter inläsning. Framtida databasbackuper som innehåller API-data behöver samma eller kortare retention. Klienter som cachelagrar svar med `expires_at` rensar dem vid nästa användning; avstängda klienter kan inte utföra fysisk rensning. Det är en begränsning att hantera före publik distribution, inte ett löfte om ständig radering på en avstängd dator. Egna spelhändelser är separata och raderas inte av denna cachepolicy.

## Driftuppdatering 2026-11-06

Projektion v2 bevarar ytterligare uttryckliga API-fält (bland annat krav, belöningar, kvalitet, klasser och item-tooltips) som begränsad JSON i entitetens beskrivning. Länkar/media tas bort, fältstorlek begränsas och ingen relation ges starkare betydelse än källan visar. Questberättelser ingår inte i denna faktaprojektion. Saknade/utelämnade fält förblir okända. Varken ett länkat föremål eller en referens bevisar en dropchans eller att något finns i den aktuella spelvärlden.

Giltiga cachedokument med äldre projektion byggs om, normalt 200 per batch. `-reproject-limit 2000 -max-requests 1` gör en avgränsad större ombyggnad med en enda namespace-probe. Ursprunglig hämtningstid, expiry och nästa faktiska uppdateringsdeadline behålls. Den lokala CLI-standarden är 200 nätanrop; VM-runnern väljer 500. Ingen ökning av databasens eller andra tjänsters resursgränser.

Servern har separata Wisp-cronjobb för refresh var 15:e minut och `-purge-only` dagligen 03:17 (VM-tid). Lås, resursvakt och atomisk exitstatus finns i `infrastructure/vm/refresh-knowledge.sh`. Rensningen kontaktar inte en provider och behöver ingen giltig API-nyckel. För kopior i backuper/cache gäller samma TTL; statistik utan API-innehåll kan behållas för driftuppföljning. Se [leveranschecklistan](delivery-knowledge-character-2026-11-06.md) för verkliga resultat och frågetester.

ForeverDB:s dokumenterade API erbjuder kompletterande uppgifter men kräver en separat godkänd nyckel och synlig attribution. Ingen sådan nyckel har tillhandahållits i detta arbete. QuestieDB:s historiska rättigheter är fortsatt oklara; ingen bulkimport aktiverades. Originalkällornas status kontrollerades på nytt 6 november 2026.
