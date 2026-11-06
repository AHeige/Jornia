#!/bin/sh
# Installed in the private Wisp directory; invoked by the existing user's cron.
set -eu
cd /home/leon/wisp-private
umask 077
exec 9>.ingestion.lock
/usr/bin/flock -n 9 || exit 0
# Replace only this job's last-run log. Long-term aggregate history is in ingestion_runs.
exec /usr/bin/docker compose --profile ingestion run --rm --no-deps --name wisp-private-ingestion ingest > ingestion-last.log 2>&1
