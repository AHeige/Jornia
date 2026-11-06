# Privat VM-installation — 2026-11-05

Wisp kör på `ubuntu-dev`, användare `leon`, under `/home/leon/wisp-private`.
Anslutning från utvecklings-Macen:

```sh
ssh -i ~/.ssh/ubuntu_dev -p 2222 leon@178.174.244.160
```

## Befintlig miljö

Inventeringen hittade 11 aktiva containrar för Allokerat (webb, inbox, workers och Neo4j), Sefira document preview (index, Caddy och Neo4j), två Temporal-miljöer och en kind/Kubernetes-testmiljö. Portarna 80/443 används redan av Sefiras Caddy. Dessa projekt, nätverk, volymer, proxy och brandvägg ändrades inte. Efter Wisp-installationen var samtliga ursprungliga container-ID:n fortfarande igång med RestartCount 0.

VM:n har 6 vCPU och 15 GiB RAM, med cirka 8,7 GiB tillgängligt före installation. Rotdisken är 47 GiB; efter installation var 19 GiB ledigt. Ingen global Docker-prune, daemonuppdatering, tjänsteomstart eller diskändring utfördes.

## Isolering och drift

- Compose-projekt: `wisp-private`; definition i `infrastructure/vm/compose.yaml`.
- API: endast `127.0.0.1:18090` på VM:n. Databas och modellservice saknar publicerade portar.
- Egen PostgreSQL-volym och privata nätverk; befintliga databaser används inte.
- Minne begränsat till 512 MiB för PostgreSQL, 384 MiB för LangChain och 192 MiB för API. Init-jobbet har separat gräns 192 MiB. CPU-gränser 0,5 / 0,75 / 0,5 kärnor respektive.
- Vid uppmätt tomgång: API cirka 2 MiB, databas 22 MiB, modellservice 88 MiB. Detta är ingen garanti om belastning under större importer.
- Loggrotation: högst tre filer à 5 MB per tjänst.
- API/modell kör utan root, med skrivskyddat filsystem och tappade capabilities. Databasen kör som postgres.
- Berget-nyckel och separat slumpgenererat databaslösenord ligger i VM-projektets `.env` med mode 600. Inga hemligheter ingår i versionshanterade filer eller images.
- Go-binären korskompilerades på Mac. Containerbyggen på VM:n begränsades till en CPU och 512 MiB RAM, ett i taget. Inga systempaket installerades på VM-värden.

```sh
cd /home/leon/wisp-private
docker compose ps
curl -fsS http://127.0.0.1:18090/healthz
docker compose logs --tail=30 api model
```

Stoppa endast Wisp med `docker compose stop` i denna katalog. Detta behåller datavolymen. Använd inte `down -v` för normal drift.

## Verifiering

Healthcheck, upprepad idempotent bootstrap samt 30/30 API-acceptansfall passerade mot VM:n över en SSH-tunnel. Ett verkligt anrop körde VM API → LangChain → Berget/Mistral Small 3.2 och svarade med Astranaar samt källreferenser; 767 prompttoken, 61 svarstoken, uppskattad kostnad €0,0002484 och modellstegslatens 1551 ms.

Windows 0.2.0 ansluter nu genom en separat SSH-identitet på Windows och värdens befintliga VirtualBox-mappning på 127.0.0.1:2222. Endast Wisp-API:t på VM:ns 127.0.0.1:18090 tillåts som tunnelmål. Servernyckeln är pinnad; ingen publik API-port eller ändring av proxy/brandvägg har gjorts. En pg_dump i custom-format togs före uppdateringen och innehållsförteckningen kontrollerades med pg_restore. Automatisk backup och full återläsningsövning återstår.

## Windows-värdens lagring

Läsande PowerShell-inventering visade ASUS Z170-P, en Kingston A400 `SA400S37120G` (SATA, ungefär 112 GiB), samt C: med ungefär 39 GiB ledigt. Ingen ytterligare fysisk disk rapporterades. Exakt fysisk SATA-port, lediga strömkontakter och monteringsplats har inte verifierats. ASUS-manualen anger fyra SATA-portar och en M.2 Socket 3 för 2242/2260/2280, SATA/PCIe. För mer utvecklingsutrymme rekommenderas 2 TB som planeringsval; befintlig SSD behöver inte ersättas om en extra disk kan anslutas.


## Uppdatering 0.2.0

API-image: `wisp-private-api:20261105-2`; modell-image oförändrad. Initieringen är idempotent. 46/46 faktatester passerar på VM:n; grafen har 49 granskade entiteter och 67 relationer. Före uppdateringen sparades `.env.pre-0.2.0`, föregående image och `backups/pre-0.2.0.dump`. Övriga 14 inventerade containrar behöll sina container-ID:n och hade RestartCount 0 efter uppdateringen. Inga globala rensningar utfördes.

Windows startar med `Start-Wisp-VM.cmd` i Desktop/Wisp-0.2.0/wisp-0.2.0-windows-amd64. `Start-Wisp.cmd` använder samma lokala SQLite-journal utan VM-tunnel. Den privata Wisp-nyckeln stannar under `%LOCALAPPDATA%/Wisp/connection`, med rättigheter för användaren och SYSTEM. VM:ns authorized_keys innehåller en separat rad med `restrict`, tillåten port-forwarding, `permitopen="127.0.0.1:18090"`, `permitlisten="127.0.0.1:18090"` och `command="/bin/false"`. Vanlig SSH-shell, PTY, agent- och X11-forwarding ges inte. Detta är en konfiguration för denna privata testdator, inte en publik installationslösning.

## OCR e2e, 2026-11-06

Aktiva Wisp-images efter det verifierade Mac-testet: API `wisp-private-api:20261106-ocr1`, modell `wisp-private-model:20261106-ocr2`. Övriga tjänster inklusive gateway/databas återskapades inte; 20 övriga löpande container-ID:n var oförändrade efter slutuppdateringen. Inga nya nätverk, portar, bibliotek, migrationer eller resursökningar. Modell-/API-healthcheck passerade.

Modelladaptern loggar nu bara felklass och eventuell upstream HTTP-status vid fel; inga exception-strängar, frågor, svar eller provider-nycklar. `LengthFinishReasonError` visade att den tidigare gränsen 220 output-token avbröt svar. Gränsen är nu 512, med instruktion i Go om kort text och få relevanta belägg. Ingen automatisk betald retry. Go-pakettesterna och två nätverksisolerade modelltester passerade. Berget-svaret i den lyckade körningen använde 229 output-token och visades i WoW efter 6,1 s.

Privata miljöbackuper för återgång finns under projektet som `.env.pre-ocr-model-diagnostics` och `.env.pre-ocr-output-fix`; befintliga image-taggar behölls. Läs `docs/ocr-transport-2026-11-06.md` för mätning och kvarvarande begränsningar.
