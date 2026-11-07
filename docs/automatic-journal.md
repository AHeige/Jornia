# Lokal automatisk äventyrsjournal — 0.3.0

Wisp får ett minne per karaktär. Den centrala världsgrafen och Berget-anropen ändras inte av denna funktion. SQLite innehåller originalobservationer och en lokal, återbyggbar graf; ingen separat grafserver installeras hos spelaren.

## Första användningen

1. Addonen frågar vid första inloggningen innan den startar automatisk registrering. Godkännandet sparas för installationen; nya karaktärer börjar därefter registrera vid inloggning. Varje karaktär kan pausas med `/wisp stop` och återupptas med `/wisp start`.
2. I companionens **Automatisk äventyrsjournal**, välj den absoluta sökvägen till klientmappen som innehåller `WTF`. Godkänn lokal filinläsning och välj **Aktivera och läs in journaler**.
3. Companionen läser befintliga Wisp-journaler för samtliga karaktärer i dessa filer. Därefter kontrollerar den nya sparningar var tredje sekund medan programmet körs. Inställningen återupptas automatiskt efter omstart av companionen.
4. Välj karaktär och öppna **Din resa**, eller fråga **Visa min resa**. Tidsstämpeln visar när informationen observerades i spelet. Senaste filinläsning är ett annat klockslag och visas separat.

Detta installerar inte autostart vid Windows-inloggning. Companionen behöver vara igång för att läsa nya filer. Saknas Wisp-data behöver karaktären först loggas in med addonen. Vanliga WoW-inställningar kan inte återskapa en äldre resa. Andra addons behöver egna, uttryckligt avgränsade importadaptrar och läses inte här.

## Vad som registreras

Start/slut på registreringspass, zon/delområde, tidpunkt, nivå, nivåökningar, inlämnade quest-ID:n, egna anteckningar, karaktärens ID/namn/realm samt klientversion/build. Ett registreringspass börjar vid accepterad start/inloggning/UI-reload; det är inte en garanti för en sammanhängande verklig spelsession. Ingen XP-, inventory-, utrustnings-, talang-, aktiv-quest- eller chattinsamling är implementerad i denna version.

Addonen observerar händelser under spelandet, men SavedVariables skrivs normalt vid `/reload` eller utloggning. Companionen får inte löpande livepositioner genom denna mekanism. Den tvingar inte fram reload, läser inte spelprocessens minne och styr inte spelet. Funktion och lagring behöver fortfarande verifieras i användarens verkliga Forever-klient; Lua-simulering är inte ett sådant speltest.

## Lokal graf och prestanda

Grafen har karaktär → registreringspass/observationsgrupp → händelse → plats/quest. Äldre händelser utan sessions-ID grupperas efter UTC-datum som importerade observationer, inte som bevis för en verklig session. Alla noder och relationer avgränsas med karaktärens lokala identitet. Plats- och questnoder dedupliceras. Varje ny händelse och dess grafprojektion skrivs atomiskt tillsammans med importkvittot. Grafen kan byggas om från originalhändelserna efter en uppgradering.

Ingen LLM behövs för projektion eller resesammanfattning. Händelser registreras vid förändringar, inte per bildruta. Oförändrade filers storlek/tidsstämpel används för att undvika innehållsläsning; förändrat innehåll får checksumma. Stabila händelse-ID:n gör att upprepade sparningar inte skapar dubbletter. Motstridigt innehåll under samma ID bevarar tidigare fakta och ger ett fel i stället för att tyst skriva över dem.

Addonen håller högst 2 000 händelser per karaktär; äldre poster faller ur addonbufferten om de inte hunnit importeras. Manuellt kopierat exportpaket innehåller de senaste 100. Automatinläsning delar större journaler i paket om högst 100, med mindre adaptiva paket för detaljerat state/framsteg. Begränsningar: 8 MiB per fil, högst 100 karaktärer per fil, högst 100 journalfiler och 20 000 traverserade katalogposter per kontroll, begränsat katalogdjup. Gränsöverskridande ger synligt fel, inte ett påstående om komplett historik. Lokalt importerade observationer sparas tills användaren raderar dem. Tidslinjen visar de senaste 50; exporten innehåller hela den lokala händelse- och samtalshistoriken.

## Filåtkomst, paus och radering

Inläsaren söker enbart `Wisp.lua` i `SavedVariables` under den valda klientens `WTF/Account`. Andra filers innehåll läses inte. Den kör aldrig Lua från disk: endast en begränsad dataliteral `WispDB = {...}` accepteras. Kod, andra globala variabler, orimliga storlekar/djup och ofullständiga filer avvisas. En ofullständig fil provas igen vid senare kontroller; redan sparad historik behålls. Källfilen ändras inte.

**Pausa automatisk inläsning** stoppar companionens läsning och bevarar lokal historik. `/wisp stop` stoppar addonens registrering för aktuell karaktär. Det är två olika aktiviteter och båda förklaras i gränssnittet.

**Glöm vald karaktär** raderar även den lokala grafen och spärrar automatisk återimport av samma karaktär. Spärren använder ett lokalt hashvärde av identiteten; detta är ingen anonymitetsgaranti. Addonens originalfil ligger kvar. En senare uttrycklig manuell import häver spärren. Befintliga exporter/säkerhetskopior påverkas inte av raderingen.

Journal, graf och samtal stannar lokalt. Världsfrågor skickar fortfarande bara frågan och spelversionen till servern. Valda historiska journalutdrag kan skickas per AI-fråga efter granskning. Den experimentella ingame-bryggan kan bifoga en färsk snapshot, med eget val per fråga; detta är inte en kontinuerlig livefeed. Lokal lagring innebär inte kryptering.


## Rikt state, lokal prototyp 0.7.2

Addonen läser aktuellt state i Lua-minnet och sparar ett daterat checkpoint per karaktär: level/XP, zon/subzon, klass/ras-ID, upp till 50 aktiva quests med begränsade mål/räknare, utrustade item-ID:n och professioner när klientens API stöder dem. Journey får förändringar för quests, utrustning och död/återuppståndelse. Pausad inspelning stoppar bestående nya händelser/checkpoints. Inga chattar eller andra spelares uppgifter samlas.

SQLite skiljer checkpoint från historiska händelser. En äldre fil kan inte skriva över ett nyare checkpoint. Återimport ger samma händelse en gång. Tomma Lua-tabeller normaliseras bara för schemaangivna listor; SavedVariables körs aldrig som kod. Otillgängliga API:er och begränsade listor märks explicit. Komplett historik garanteras inte om addonbufferten eller filgränsen överskrids.

För en godkänd fråga läser addonen om state vid skicket. Den begränsade frågesnapshoten kan bära upp till fyra quests, målräknare, gear-ID:n och professioner. Saknade fält och trunkering följer med. Inläsning från SavedVariables får aldrig ensam skapa ett live-markerat AI-underlag.
