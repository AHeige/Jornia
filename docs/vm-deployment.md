# Private VM installation on 5 November 2026

Jornia runs on `ubuntu-dev` as `leon` under `/home/leon/wisp-private`. Development Mac connection:

```sh
ssh -i ~/.ssh/ubuntu_dev -p 2222 leon@178.174.244.160
```

## Existing environment

Inventory found 11 active containers for Allokerat (web/inbox/workers/Neo4j), Sefira document preview (index/Caddy/Neo4j), two Temporal environments and a kind/Kubernetes test environment. Ports 80/443 already belong to Sefira Caddy. Those projects/networks/volumes/proxy/firewall were unchanged. All original IDs remained running with RestartCount 0 after installation.

The VM has 6 vCPUs/15 GiB RAM, about 8.7 GiB available before installation. Root disk: 47 GiB; 19 GiB free afterward. No global Docker prune, daemon update, service restart or disk modification.

## Isolation and operations

- Compose project `wisp-private`; definition `infrastructure/vm/compose.yaml`.
- API binds only VM 127.0.0.1:18090. Database/model have no published ports.
- Own PostgreSQL volume/private networks, no reuse of other databases.
- Memory caps: PostgreSQL 512 MiB, LangChain 384 MiB, API/init 192 MiB each. CPU: 0.5/0.75/0.5 cores respectively.
- Observed idle memory: API about 2 MiB, database 22 MiB, model 88 MiB; not a large-import guarantee.
- Logs: at most three 5 MB files per service.
- API/model run nonroot with read-only filesystems and dropped capabilities; database runs as postgres.
- Berget key/random database password in VM `.env`, mode 600. No secrets in tracked files/images.
- Go cross-built on Mac. Sequential VM container builds capped at one CPU/512 MiB. No host system packages installed.

```sh
cd /home/leon/wisp-private
docker compose ps
curl -fsS http://127.0.0.1:18090/healthz
docker compose logs --tail=30 api model
```

Stop only Jornia using `docker compose stop` in this directory, preserving its volume. Do not use `down -v` in normal operations.

## Verification

Health, repeated idempotent bootstrap and 30/30 acceptance cases passed through SSH. One real VM API → LangChain → Berget/Mistral Small 3.2 call answered with Astranaar/citations: 767 input tokens, 61 output tokens, estimated EUR 0.0002484 and 1,551 ms model latency.

Windows 0.2.0 uses its own SSH identity and the host's existing VirtualBox mapping at 127.0.0.1:2222. Only VM 127.0.0.1:18090 is an allowed tunnel destination. Host key pinned; no public API port or proxy/firewall change. A custom-format pg_dump preceded the update and pg_restore inspected its contents. Automated backups/full restore drill remained outstanding.

## Windows host storage

Read-only PowerShell inventory reported ASUS Z170-P, Kingston A400 `SA400S37120G` (SATA, about 112 GiB), and C: with about 39 GiB free. No additional physical disk reported. Physical SATA port, spare power and mounting space were not verified. ASUS manual specifies four SATA ports and one M.2 Socket3 for 2242/2260/2280, SATA/PCIe. A 2 TB additional disk was the planning recommendation; replacing the existing SSD is unnecessary if another can connect.

## Version 0.2.0 update

API image `wisp-private-api:20261105-2`; model unchanged. Bootstrap idempotent; 46/46 VM fact tests pass, graph 49 reviewed entities/67 relations. Backups: `.env.pre-0.2.0`, previous image and `backups/pre-0.2.0.dump`. All 14 unrelated inventoried containers retained IDs/RestartCount 0. No global cleanup.

Windows launches `Start-Wisp-VM.cmd` in Desktop/Wisp-0.2.0/wisp-0.2.0-windows-amd64. `Start-Wisp.cmd` uses the same local journal without tunneling. Private Jornia key remains in `%LOCALAPPDATA%/Wisp/connection`, limited to user/SYSTEM. A separate authorized_keys entry uses `restrict`, permitted forwarding, `permitopen="127.0.0.1:18090"`, `permitlisten="127.0.0.1:18090"` and `command="/bin/false"`. No shell, PTY, agent or X11 forwarding. This is private-machine configuration, not public installation design.

## OCR E2E on 6 November 2026

After verified Mac testing: API `wisp-private-api:20261106-ocr1`, model `wisp-private-model:20261106-ocr2`. Gateway/database and other services were not recreated; 20 other running IDs remained unchanged. No networks/ports/libraries/migrations/resource increases. API/model health passed.

Model-adapter diagnostics now log error class/optional upstream status only, excluding exception text, questions, answers and keys. LengthFinishReasonError identified the old 220-token limit aborting answers. Limit now 512, with Go instructing concise text/few relevant citations. No automatic paid retry. Go and two isolated model tests pass. Successful reply used 229 output tokens and appeared in WoW after 6.1 s.

Private rollback files `.env.pre-ocr-model-diagnostics` and `.env.pre-ocr-output-fix` remain with prior images. See `docs/ocr-transport-2026-11-06.md` for measurements/limits.

## Knowledge and character questions 0.6.0 on 6 November 2026

Final API `wisp-private-api:20261106-knowledge5`; separate ingester `wisp-private-ingest:20261106-knowledge2`, built from the existing API base without new system packages. Model `20261106-ocr2`, gateway and database retained image/container identities. Only API was recreated for context/question rules. Before/after checks found no unrelated changes. No global cleanup, proxy/firewall/disk changes.

User cron for leon:

```cron
*/15 * * * * /home/leon/wisp-private/refresh-knowledge.sh refresh # wisp-knowledge-refresh
17 3 * * * /home/leon/wisp-private/refresh-knowledge.sh purge # wisp-knowledge-purge
```

Runner has locking, a 3 GiB free-disk/512 MiB available-RAM guard, two bounded log files and atomic `ingestion-status.json` with exit/times. Purge-only requires no credentials/network. Ingester caps: 192 MiB/0.5 CPU, 500 requests, one-second spacing, twelve-minute deadline. Cron uses VM time; logs use UTC. Batch 16:00:01–16:08:49 UTC completed 500 requests/200 reprojections. A separate bounded run reprojected 2,000 documents with one namespace probe, preserving expiry/deadlines.

Rollback files: `compose.yaml.pre-knowledge-20261106`, `.env.pre-knowledge-20261106`, `refresh-knowledge.sh.pre-knowledge-20261106`, `crontab.pre-knowledge-20261106`. Restore only Jornia entries if other projects changed later; never blindly replace a newer full crontab. Prior images remain. Projection v2 needs no destructive migration. Future cross-built uploads require `chmod 755` before Docker builds: scp/umask may produce mode 700.

Public ZIPs/site use personal `AHeige/wisp-site`, not Logos. The VM continues serving the configured protected API. Version 0.6.0 provides explicitly reviewed dated character context, without automatic journal uploads. See the [delivery checklist](delivery-knowledge-character-2026-11-06.md) for coverage/tests/packages/limits.
