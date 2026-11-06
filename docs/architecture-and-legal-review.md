# Wisp: arkitektur och granskning av regler

Granskat 2026-11-05 med Sverige/EU som utgångspunkt. Granskningen föregick companion-implementationen; aktuell testversion och återstående spelverifiering beskrivs i [testguiden](wow-first-test.md) och [verifieringsrapporten](verification.md). Detta är ett tekniskt beslutsunderlag utifrån publicerade regler, inte ett juridiskt utlåtande eller ett godkännande från Blizzard. Faktisk funktion, avtalspart, klientversion, distribution och finansiering måste bedömas tillsammans före publik lansering.

## Arkitektur vi utgår från

- Central kunskapsgraf för offentlig världskunskap och källor.
- Lokal SQLite-databas i companionen för spelarens händelser, historik och preferenser.
- Inget obligatoriskt Wisp-konto och ingen automatisk synk av spelarhistorik till en central databas.
- Central modellproxy håller Berget-nyckeln och tar emot bara det sammanhang som behövs för ett begärt svar.
- Första companionversionen fungerar som en separat chatt med manuell inmatning/import. Automatisk spelkoppling är en separat fråga som ännu inte är godkänd eller implementerad.

SQLite är avsett för lokal applikationslagring och kräver ingen separat databasserver eller användaradministration. SQLite-kärnan anges vara public domain; licenserna för vald drivrutin, eventuella tillägg och installationspaket måste granskas separat. Se [användningsområden](https://www.sqlite.org/whentouse.html) och [licensbesked](https://www.sqlite.org/copyright.html).

## Blizzard: avtalsregler och spelintegration

Blizzards [officiella addonpolicy](https://us.forums.blizzard.com/en/wow/t/ui-add-on-development-policy/24534/1) kräver gratis distribution och offentligt läsbar kod. Policyn förbjuder reklam och donationsuppmaningar i addonens spelgränssnitt, förbjuder skadlig påverkan på spel/andra spelare och hänvisar vidare till EULA och användarvillkor. Den omfattar också avgifter för addonrelaterade tjänster. En separat companion är inte automatiskt ett undantag från de reglerna.

**Konsekvens för Wisp:** håll addon och kopplade funktioner gratis under prototypen; bygg ingen betald AI-funktion, obligatorisk egen betalnyckel eller abonnemangslösning kopplad till addon utan en konkret avtalsbedömning. Donationer på en extern webbplats har ett uttryckligt utrymme i addonpolicyn, men ger inte generellt tillstånd för alla finansieringsmodeller.

Blizzards [EULA](https://www.blizzard.com/en-gb/legal/08b946df-660a-40e4-a072-1fbde65173b1/blizzard-end-user-license-agreement) begränsar bland annat otillåten insamling av speldata, obehöriga kopplingar och tredjepartsprogram. [Anti-cheating-avtalet](https://www.blizzard.com/en-us/legal/simple/cd5930c0-2784-420c-a23d-1e0d6ff8599b/anti-cheating-agreement) behandlar otillåten förändring av spelupplevelsen och insamling av information. Ett avtalsbrott och ett lagbrott är olika frågor; avtalsbrott kan ändå medföra att spelarkonton eller funktioner påverkas.

**Teknisk gräns för prototypen:** ingen processinjektion, minnesläsning, paketavlyssning, kringgående av skydd, modifiering av klientfiler eller automatiska spelhandlingar. Att ett program bara läser eller inte används i strid är inte ensamt ett bevis på tillåtlighet.

**Öppet:** det finns inget generellt godkännande i dessa dokument för vår kombination av Lua-addon, extern companion, SavedVariables-import och AI-kontext. Ett dokumenterat addon-API eller en tekniskt läsbar fil bevisar inte att all sekundär användning är tillåten. Utvärdera den exakta metoden, särskilt eventuell tvåvägs-/realtidsbrygga, innan den distribueras. En separat manuell chatt undviker många av de integrationsfrågorna men är inte ett generellt frikort för tredjepartsprogram.

Addonpolicyn är en äldre, fortfarande publicerad allmän policy. Granskningen fann inget Forever-specifikt godkännande för Wisp. EULA/API-sidor gav HTTP 403 vid direkt hämtning; relevanta stycken granskades via indexerade utdrag från Blizzards egna sidor. Kontrollera avtalen som faktiskt gäller aktuell EU-användare och klient vid en lanseringsgranskning.

## Världskunskap och rättigheter

Warcraft Wiki anger [CC BY-SA 4.0 för material som wikin kan licensiera](https://warcraft.wiki.gg/wiki/Warcraft_Wiki:Copyrights). Det ger inte wikin rätt att vidarelicensiera all underliggande Blizzard-egendom. Wikins [egen upphovsrättspolicy](https://warcraft.wiki.gg/wiki/Warcraft_Wiki:Copyright_policy) beskriver också beroendet av rättigheter till spelmaterial. Amerikansk fair use är inte ett generellt svenskt/EU-tillstånd.

[CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/) kräver bland annat attribution, licenslänk och ändringsangivelse samt delning enligt samma/kompatibel licens av distribuerat bearbetat material. Det innebär inte automatiskt att hela applikationskoden får samma licens. Gränsen för en större blandad databas och dess eventuella databasskydd behöver granskas innan bulkimport eller publicering.

Det nuvarande lilla Ashenvale-urvalet har attribution, licenslänkar, ändringsbeskrivning och granskningsbegränsningar i [datameddelandet](licenses/ashenvale-data.md). Detta är dokumenterade licensåtgärder, inte en fullständig rättighetsklarering. `source_checked` avser faktakälla och kompatibilitetsgranskning; det är inget juridiskt klartecken.

**Fortsatt princip:** korta strukturerade uppgifter, dokumenterad proveniens och egna beskrivningar. Kopiera inte hela questtexter, böcker, kartor, bilder, musik eller stora databasuttag utan verifierad rätt. Egen formulering och liten volym minskar vissa risker men avgör inte ensamma tillåtligheten. Import av QuestieDB, Wowhead och andra källor kräver separat granskning.

Om [Blizzards Developer API](https://www.blizzard.com/en-us/legal/a2989b50-5f16-43b1-abec-2ae17cc09dd6/blizzard-developer-api-terms-of-use) används tillkommer dess villkor, bland annat kring avgifter, hantering/vidaredistribution av data, API-nycklar, attribution och integritet. En offentligt tillgänglig API-post är inte automatiskt fritt återdistribuerbar. Håll API-material åtskilt från material som licensieras som ett öppet kunskapspaket. Ingen Blizzard-API-integration finns i nuvarande prototyp.

Använd egen grafisk identitet och en tydlig upplysning om att Wisp är en oberoende tjänst. Kontrollera namnet Wisp separat före offentlig distribution; denna granskning är ingen varumärkessökning eller registrering.

## Lokal historik och GDPR

Lokal lagring minskar vår centrala datamängd. Inget konto betyder däremot inte automatiskt anonymitet. IP-adresser och identifierbara användaruppgifter kan vara personuppgifter enligt [IMY](https://www.imy.se/verksamhet/dataskydd/det-har-galler-enligt-gdpr/introduktion-till-gdpr/personuppgifter/). Karaktärsnamn, meddelanden och händelsekombinationer kan också identifiera människor. Behandling kan ske även utan beständig serverlagring.

När Wisp bestämmer ändamål och medel för sin serverbehandling behöver Wisp-operatören normalt bedömas som personuppgiftsansvarig för den behandlingen. [IMY:s rollbeskrivning](https://www.imy.se/verksamhet/dataskydd/det-har-galler-enligt-gdpr/personuppgiftsansvariga-och-personuppgiftsbitraden/) hjälper att skilja detta från en leverantörs biträdesroll. Rollerna för ren lokal behandling behöver bedömas mot den faktiska implementationen.

**Byggkrav före användartest med persondata:** definiera operatör och rättslig grund per ändamål; ge begriplig integritetsinformation; minimera data och mottagare; använd TLS och begränsad loggning; ange lagringstid och hantera rättigheter samt incidenter. Ett val att aktivera AI-kontext är en produktinställning och ersätter inte hela GDPR-bedömningen. Om samtycke används måste det uppfylla kraven för giltigt samtycke.

Lagra bara användarens avsedda spelhistorik. Samla inte rutinmässigt in guildchatt, whispers, andra spelares meddelanden eller Battle.net-uppgifter. Ta aldrig emot Blizzard-lösenord. Utforma export, radering och val för lokal retention. Lokal databaslagring är inte i sig kryptering eller skydd mot andra processer; använd OS-behörigheter och bedöm ytterligare skydd efter datainnehållet. Hantering av barn och åldersgräns behöver bestämmas före offentlig distribution.

[PTS beskriver](https://pts.se/internet-och-telefoni/kakor-cookies) reglerna om åtkomst/lagring på terminalutrustning och undantaget för funktioner som är nödvändiga för en begärd tjänst. Enbart ordet ”lokal” avgör inte samtyckesfrågan. Bedöm den konkreta appfunktionen och eventuell telemetri; anta inte att en vanlig lokal SQLite-fil kräver en generell kakbanner, och anta inte heller att all spårning i en desktopapp är undantagen.

## Berget

Bergets nuvarande [villkor, avsnitt 7](https://berget.ai/en/terms) uppger att promptinnehåll och modellutdata inte lagras, men medger tekniska loggar och metadata. Uttrycket zero retention innebär därför inte att inga uppgifter behandlas eller att inga metadata finns. Villkoren riktar sig till verksamheter och professionella kunder, vilket passar ett operatörsägt backendkonto bättre än ett krav att varje spelare skaffar ett privat Berget-konto.

Bergets [DPA](https://berget.ai/en/dpa), direkt hämtat med uppdateringsdatum 2026-10-22, reglerar behandling av personuppgifter och listar underbiträden. Säkerställ att det gäller vårt konto och avsedda användning, och stäm av underbiträden, geografisk behandling, säkerhet och faktisk loggretention innan personhistorik skickas. En allmän [webbplatsintegritetspolicy](https://berget.ai/en/privacy) är inte ensam ett fullständigt avtal för vår inferensbehandling. Aktuella huvudvillkor hämtades med uppdateringsdatum 2026-10-16; vissa sökindexutdrag visade äldre datum.

Nuvarande API behandlar frågan och offentlig grafkontext; inget personligt minne är implementerat. Applikationsloggarna skriver status, version, modell, tokenantal och kostnad men inte frågetext eller svarstext. Det är en kodobservation, inte en granskning av framtida proxy-, CDN-, APM- eller produktionsloggar. Sådana lager måste ingå innan vi lovar en viss integritetsnivå.

## AI-information och installation

EU-kommissionens [vägledning om artikel 50](https://digital-strategy.ec.europa.eu/en/faqs/transparency-obligations-under-article-50-ai-act) anger att transparenskraven gäller från 2026-08-02. Gör det tydligt för spelaren att Wisp är AI. Granska även vilka övriga skyldigheter som gäller oss som systemleverantör och vilka som hanteras av modellleverantören; en AI-etikett är inte en fullständig AI Act-bedömning.

Installationen kan vara en vanlig användarinstallerad companion: appen skapar en SQLite-fil i användarprofilen och behöver ingen lokal PostgreSQL-server, Docker-installation eller databasadministration. Rekommenderad Windows-plats är `%LOCALAPPDATA%/Wisp/`. Det är en implementationsrekommendation, inte ett juridiskt krav. Addoninstallation kan behöva separat vald sökväg och filbehörighet.

Installern bör förklara filåtkomst, lokal lagring, AI-anrop och avinstallation. Signerade paket och verifierade uppdateringar är rekommenderade säkerhetsåtgärder; de är inte ett bevis på Blizzard-godkännande och beskrivs här inte som generella lagkrav. Kontrollera licensmeddelanden för alla levererade komponenter. Gör ingen generell antivirus-/anti-cheat- eller administratörsdispens till installationskrav.

## Vad granskningen betyder för nästa steg

Vi kan fortsätta prototypa den centrala grafen och en separat companion med lokal historik, på testdata eller avsiktligt tillförda personliga anteckningar. Det är min riskbedömning, inte en garanti att produkten är rättsligt godkänd.

Före spelkoppling: dokumentera och stäm av exakt addon- och exportmetod. Före publik tjänst: klarlägg personuppgiftsansvar, Berget-avtal, integritetsinformation, data-/varumärkesrättigheter och distributionsvillkor. Före eventuell betalmodell kopplad till addon: separat avtalsgranskning. Lanseringsgranskningen bör göras av jurist med EU/Sverige- och spel/IP-erfarenhet när implementationen är konkret.

### Förberett underlag till Blizzard – inte skickat

Vi vill beskriva en gratis, läsbar Lua-addon och en separat Windows-companion utan spelautomatisering. Följande behöver klarläggas för just Forever:

1. Får addonens dokumenterade API:er användas för att registrera spelarens egna zonbyten och questhändelser i SavedVariables?
2. Får användaren därefter importera denna sparade data till en lokal companion för privat historik och AI-samtal?
3. Vilka begränsningar gäller om användaren väljer att sända ett begränsat utdrag till en extern inferenstjänst?
4. Finns någon uttryckligen tillåten metod att föra companion-svar tillbaka till addonens UI? Vi förutsätter inte att det finns en sådan realtidsbrygga.
5. Hur tillämpas addonpolicyn på den tillhörande gratis companionen och extern finansiering?

Addonpolicyn anger WoWUI@blizzard.com som kontaktväg. Ingen kontakt har tagits i denna granskning. Ett eventuellt svar måste läsas utifrån sin omfattning; frånvaro av svar är inte ett godkännande.
