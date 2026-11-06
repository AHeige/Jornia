# Companion-harness och leverans

Beställt 6 november 2026. Första mål: fråga på spelarens språk → aktuell godkänd spelkontext → källbelagt nästa steg → valbar ”Visa vägen” → validerad markering i Lua.

- [x] Egen tunn Go-harness med verktygsregister, begränsad körning, spårning utan privata payloads och strukturerade resultat.
- [x] Nästa-steg-underlag från aktuell addon-snapshot; historia och Armory får inte utges för aktuell queststatus.
- [x] Strukturerade navigationshandlingar; inga modellgenererade Lua-skript eller koordinater.
- [x] Språkoberoende frågor/svar och UTF-8 genom OCR-labben, utan ASCII-språklista.
- [x] Valbar questkontext och klickstyrd waypoint i Lua; aktualitet, karaktär och aktivt quest kontrolleras igen vid klick.
- [ ] Tester av hela flödet, språk, fel, gamla svar, karaktärsbyte och avsaknad av native waypoint-stöd.
- [x] Uppdatera wisp.logosab.com och nedladdningsflödet inom befintlig isolering.
- [ ] Paketkontroller: privata filer/hemligheter, binäridentitet, SHA-256, installation och tillgänglig kodsignering/notarisation.
- [x] Dokumentera faktisk signerings-/antivirusstatus; inga garantier om externa antivirus och inga avstängda säkerhetsskydd.

Utgångsläge: desktopalpha 0.6.0 är publicerad. OCR-/returlabbar är separata. Den återanvändbara blockreturen är experimentell. Ingen reload/omstart av WoW får användas som transportsteg eller för att läsa in ny testkod. Nyckel till ForeverDB är ansökt om, ännu inte erhållen; vägvisning får därför utgå från stöd som faktiskt finns i klienten och tydligt avstå vid dataluckor.


## Verifiering och återstående grindar

- `go test ./...` passerar, inklusive körning av Lua i 5.1-miljö, samtycke, felaktiga handlingar, gamla snapshots, fel karaktär, combat, saknade API:er och saknad waypoint-bekräftelse.
- Fem protokolltester i Python passerar. Vision-fixturen läser en japansk UTF-8-fråga och en quest-snapshot genom hexprotokollet (358,6 ms i den syntetiska körningen). Detta är inte en ny live-latensmätning.
- Fem riktiga Berget-frågor med uttryckligen syntetiska aktuella snapshots passerar på franska, spanska, japanska, arabiska och svenska. Alla fick stödjande state-citat och en validerad handling. Se `harness-live-evaluation.json`. Den första iterationens brister sparas separat; inga modellfel återförsöks automatiskt.
- API `wisp-private-api:20261106-harness3` kör Go direkt mot Berget. Bara Wisp-API-containern byttes; övriga befintliga containers bevarades. Den gamla modellproxyn är stoppad; dess image och privata konfigurationsbackup är bevarade för rollback. Den ligger i Compose-profilen `legacy-model`.
- Mac-kompanjonen uppdaterades lokalt med bevarad databas. Båda ZIP-arkiven kontrollerades för CRC, SHA-256 och privata filnamn. Mac-appens executable serverade den inbäddade UI:n med separat testdatabas. Windows är korskompilerad och har inte körts i detta test.
- Livekontroll: C_QuestLog.GetNextWaypoint och C_Map.SetUserWaypoint finns i klienten. **Quests 376 och 3901 på Cone Thrower gav dock inga native-positioner.** Denna adapter ska då inte erbjuda en markering. Nästa navigationsslice behöver källbelagda positioner och ett separat validerat kartfallback för dessa quests.
- Nytt live OCR-flöde är **inte godkänt**: första kortet klipptes; två väntetester förbrukade returpoolen. Teckensnitt, fast texthöjd och crop rättades; den syntetiska Vision-kedjan passerar. Inga nya frågor nådde Berget i dessa två liveförsök. Ny kod lästes in genom oanvända slots 094/095, vars filer sedan återställdes. Ingen reload, utloggning eller omstart genomfördes av oss. Spelet lämnades igång. Finita returpoolen är nu i praktiken slut i denna UI-session.
- Mac-distribution är spärrad i nedladdningssidan tills Developer ID Application-certifikat och notarytool-Keychain-profil finns och Apple har accepterat paketet. Se `macos-signing.md`. Apple Developer-kontot är bekräftat, men ingen giltig signeringsidentitet finns installerad. Windows-signering saknas också.

**Nästa konkreta test:** när en ny normal spel-session finns, kör den korrigerade snapshot/OCR-rutan genom hela kedjan. Alternativt integrera den redan separat bevisade återanvändbara blockreturen innan ytterligare finite-session-test. Ingen reload/omstart får begäras för att komma förbi dagens gräns. Gör inte detta till en allmän vän-installation med påstått färdig in-game-navigering.

Publicering: [0.7.0 Windows developer alpha](https://github.com/AHeige/wisp-site/releases/tag/v0.7.0) är publicerad med SHA-256; den faktiskt nedladdade filens hash matchade lokalt. GitHub-sidan (commit `11efefb`) och wisp.logosab.coms hänvisning är uppdaterade. Mac-knappen är spärrad i väntan på signering. Endast publika releasetillgångar laddades upp, ingen källkod, nyckel eller spelarjournal. Se `release-0.7.0-download.png`.

Avslut: labbläsaren stoppades när returslotsen var slut och labbrutan doldes. Den uppdaterade vanliga companion-processen fortsätter på port 8788 med bevarad lokal data. Testets extra app-process och SSH-tunnel stoppades.
