# Privat kompisalpha: distribution och AI-anslutning

Driftsatt 2026-11-06: **https://wisp.logosab.com**. Aktuell klient är 0.4.2, ett portabelt ZIP för Windows amd64 och Mac arm64. Publicerade versionsfiler och SHA-256 behålls; nya byggen får nytt versionsnummer. Denna alpha har ingen signerad installer eller automatisk uppdatering. Ladda ned nästa ZIP och stäng den äldre processen före start; samma lokala datakatalog används.

## Spelarens flöde

Ladda ned, packa upp och starta `Start-Wisp.cmd`. Under AI-anslutning anges en separat privat testkod. Journalen kan provas med demo utan WoW och fungerar lokalt utan anslutning. Addoninstallationen och två uttryckliga samtycken till registrering respektive filinläsning beskrivs i START-HERE. AI-användning av journalutdrag är ett eget val per fråga, med förhandsvisning.

Testkoden är en 256-bitars slumphemlighet. Den skickas i Authorization-header över HTTPS, sparas bara i companionens lokala SQLite-inställningar och utesluts från historikexport och operationskvitton. Den publika ZIP-filen har inga testkoder, providerhemligheter eller privata SSH-nycklar. Servern lagrar endast kodens SHA-256, etikett, utgångstid och kvot. Den första kompisinbjudan gäller i 30 dagar med högst 50 serverfrågor per UTC-dygn; lokala minnesfrågor räknas inte. Inbjudan är personlig och ska skickas separat från den publika länken.

Återkalla en inbjudan genom att sätta `tester_access.revoked=true` för dess etikett/hash i Wisp-databasen. Nästa serveranrop avvisas, medan lokal journal finns kvar. Detta är privat teståtkomst, inte ett fullständigt kontosystem eller en betalprodukt.

## VM och avgränsning

- Egen Wisp-gateway med 96 MiB minnesgräns och 0,25 CPU; endast nedladdningsfiler, startsida, autentiserad hälsa och autentiserad ask-route exponeras.
- Befintlig Caddy hanterar TLS för en separat virtuell värd. Bara Wisp-gatewayen delar proxyns nätverk. Databasen och modellproxyn exponeras inte publikt.
- Wisp-API:t har fortsatt global gräns på sex modellförsök/minut och två samtidiga. Modellsteget är Python/LangChain → Berget med begränsad output och utan automatisk retry eller tracing.
- Kvot och anropsidentiteter sparas i PostgreSQL. Samma kod/anrops-ID godtas bara en gång, även efter omstart. En upprepning returnerar 409; inget nytt modellförsök sker automatiskt. Servern lagrar inte frågetext, journalutdrag eller modellsvaret för återspelning. Ett förlorat resultat kan därför fortfarande kräva uppmärksamhet.
- Privata journalkontexter används inte som världsfakta. Wisp-API:t loggar bara aggregerad anropsmetadata, inte innehållet. Ingen garanti om externa leverantörers retention följer av detta.

Caddyfile och Wisp-konfiguration/databas säkerhetskopierades före ändringen. Caddy 2.11.7 läste om den validerade filen med SIGUSR1 utan containeromstart. Det befintliga dokumentindexets HTTPS-hälsa var 200 före och efter. Samtliga 14 andra containers hade samma ID och körde fortsatt efter driftsättningen. Ingen annan applikation eller VM:n startades om.

## Senare signerad installation

MSIX/App Installer över HTTPS är ett möjligt nästa steg. Det kräver en betrodd signeringsidentitet, stabil paketidentitet och verifierad migrering av datakatalogen. Bygg/signera utanför VM:n och håll signeringshemligheter borta från releasemappen. Behåll manuell återställning och offlinejournal när uppdateringsservern inte kan nås.

Källor: [Microsofts uppdateringsinställningar](https://learn.microsoft.com/en-us/windows/msix/app-installer/update-settings), [signeringskrav](https://learn.microsoft.com/en-us/windows/msix/package/signing-package-overview), [paketerade appars filhantering](https://learn.microsoft.com/en-us/windows/msix/desktop/desktop-to-uwp-behind-the-scenes), [Caddys omladdningssignal](https://caddyserver.com/docs/command-line).
