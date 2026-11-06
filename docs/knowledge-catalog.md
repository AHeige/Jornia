# Wisp: kategori- och källregister

Uppdaterat 6 november 2026. Målet är användbara svar i tydliga områden, inte största möjliga räknare.

- [x] Skapa ett stabilt register för 25 kategorier.
- [x] Visa faktisk postmängd och aktiva API-detaljdokument utan att kalla dem fälttäckning.
- [x] Skilj råfakta från ännu inte implementerade planerare/rekommendationer.
- [x] Gör registret åtkomligt från chatten utan modellkostnad.
- [x] Ansökan om ForeverDB-utvecklaråtkomst inskickad av användaren.
- [ ] Få godkänd ForeverDB-utvecklarnyckel.
- [ ] Verifiera autentiserade svar, faktisk schema/build/härkomst och villkor innan import aktiveras.
- [ ] Koppla in ForeverDB som kompletterande källa och testa uppdateringar/attribution.
- [ ] Bygg och verifiera talentbyggen, gearkombinationer och questrutter från tillräckliga fakta.
- [ ] Lägg till egna källbelagda områdesguider och tydligt subjektiva upplevelsebeskrivningar.

## Registret som körs

`GET /api/knowledge/coverage` på det interna Wisp-API:t innehåller `categories` och `category_counts_overlap`. Det är ett serverregister, inte en ny navigationssida i klienten. Befintlig desktopklient kan använda frågorna nedan efter serveruppdateringen. Ingen ny klientversion behövs.

| Grupp | Kategorier |
|---|---|
| Uppdrag och utforskning | Quests, zones/places, NPCs, travel |
| Karaktär och combat | Classes/races, specs/roles, abilities, talent trees |
| Föremål och ekonomi | Items, gear/sets, professions, recipes, gathering, reputation |
| Innehåll och aktiviteter | Dungeons, raids, bosses, PvP, events, collections |
| Berättelse och upplevelse | Lore/storylines, places/experiences |
| Härledda förslag | Character builds, gear combinations, quest routes |

Varje rad anger nödvändiga fält/relationer och föreslagna nästa källor. `partial` betyder en del poster, `missing` inga egna poster i kategorin och `not_implemented` att planeraren inte finns. `field_coverage: not_measured` är avsiktligt: många namngivna quests innebär inte kompletta mål, krav, belöningar eller kedjor. Items och gear överlappar; kategorisummorna får inte adderas till ett nytt totalantal. Detaljdokumenträknaren filtreras på aktiv API-namespace och giltig expiry.

Frågor som besvaras direkt från registerstatus, även med AI valt:

- `Vad kan du hjälpa mig med?`
- `Vad vet du om quests?`
- `Vad vet du om talanger?`
- `What do you know about professions?`
- `What do you know about quest routes?`
- `Vad vet du om upplevelser?`

Svar får route `knowledge_catalog`, daterad förstapartskälla och noll modellkostnad. De redovisar luckor; de lägger inte till spelfakta från modellminnet. Namngivna frågor som `What do you know about Astranaar?` går vidare till det vanliga kunskapsflödet. Personliga journaler ingår aldrig i kategoriräknarna.

## Vald snabb väg: Blizzard + ForeverDB

Blizzard-importen fortsätter var 15:e minut. ForeverDB är ett rekommenderat komplement för questrelationer/positioner, talanger, professioner och skillnader mellan builds — inte en ersättning av vårt minne, vår grafmodell eller vår produktlogik.

Enligt [API-dokumentationen](https://foreverdb.net/developers) krävs godkänd utvecklaråtkomst och synlig attribution med länk till ForeverDB.net. API:t beskriver 60 anrop/minut och 10 000/dag. Det är ingen generell öppen datalicens. Varje fälts Classic-/Forever-/observationshärkomst ska behållas; kedjelänkar, krav och alternativa krav är olika relationer. En förekomst i klientdata bevisar inte live-tillgänglighet.

Efter godkännande läggs `FOREVERDB_API_KEY` i operatörens ignorerade `.env`, aldrig i klienten eller chatten. Första steget blir en begränsad meta-/schema-/buildkontroll. Därefter prioriteras:

1. Zones och quests: mål, givare/mottagare, positioner och typade kedjerelationer.
2. Classes/talents: ranks, tiers och beroenden.
3. Professions/recipes: material, tillverkat föremål och hur recept lärs.
4. Dungeons/gear: instanser, bossar, föremål och setbonusar.
5. Upplevelser och härledda planerare, när nödvändiga fakta finns.

Inte först en fullständig ny itembulkimport: det området finns redan brett hos Blizzard och skulle förbruka kompletteringskvoten innan de viktigaste luckorna fylls. Ingen ny scheduler eller tom importer aktiveras innan autentiserade svar och återanvändningsvillkor faktiskt verifierats. Lore och subjektiva omdömen behöver separata, tydliga underlag.

## Eget världskorpus

Användaren vill gärna samla datan själv och äga kunskapslagret. Det passar den valda arkitekturen: tillåtna API-listor och detaljer importeras sidvis till vår centrala, normaliserade graf. Spelarens vanliga fråga läser vår databas; den ska inte behöva ett nytt ForeverDB-anrop för varje svar. Källversioner, attribution, utgångstider och respektive återanvändningsvillkor följer med. Godkänd åtkomst är en förutsättning, inte en obegränsad rätt att spegla eller återpublicera deras databas.

Egen insamling genom frivilliga spelobservationer är en kompletterande framtida källa. Den nuvarande journalen laddas inte automatiskt upp och byter inte syfte till datainsamling. Central världsobservation kräver separat opt-in och ett eget schema utan privat karaktärshistorik. Områdesguider och subjektiva upplevelser kan författas och granskas separat.

Driftverifiering: 25 kategorier och 31 riktiga registerfrågor mot VM passerade, inklusive saknade talanger/lore/upplevelser och ännu inte implementerade planerare. Ingen modell anropades. Go-tester, vet och PostgreSQL-integrationsfall i isolerade scheman passerade. API-image `wisp-private-api:20261106-catalog1` är healthy; endast Wisp-API:t återskapades. Ingesterns befintliga 15-minutersschema och övriga containrar behölls. Återgångsunderlag: `.env.pre-catalog-20261106` på VM:n. Någon ForeverDB-import har ännu inte aktiverats.
