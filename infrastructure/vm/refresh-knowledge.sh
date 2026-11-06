#!/bin/sh
# Wisp-only bounded ingestion. No system-wide cleanup, service restart or LLM calls.
set -eu
cd /home/leon/wisp-private
umask 077
action=${1:-refresh}
case "$action" in refresh|purge) ;; *) echo 'Expected refresh or purge' >&2; exit 2 ;; esac
exec 9>.ingestion.lock
/usr/bin/flock -n 9 || exit 0
started=$(date -u +%FT%TZ)
finish() {
    code=$?
    trap - EXIT
    printf '{"action":"%s","started_at":"%s","finished_at":"%s","exit_code":%s}\n' \
        "$action" "$started" "$(date -u +%FT%TZ)" "$code" > ingestion-status.json.tmp
    mv ingestion-status.json.tmp ingestion-status.json
    exit "$code"
}
trap finish EXIT
if [ "$action" = refresh ]; then
    # Preserve headroom on the shared VM; purge is deliberately allowed at low disk.
    available_kb=$(df -Pk . | awk 'NR==2 {print $4}')
    memory_kb=$(awk '/^MemAvailable:/ {print $2}' /proc/meminfo)
    if [ "$available_kb" -lt 3145728 ] || [ "$memory_kb" -lt 524288 ]; then
        printf '{"status":"resource_guard","disk_available_kb":%s,"memory_available_kb":%s}\n' "$available_kb" "$memory_kb" > ingestion-last.log
        exit 75
    fi
    set -- -max-requests 500
else
    set -- -purge-only
fi
# The Go worker has a 12-minute context; its container has fixed memory/CPU limits.
# One current and one previous aggregate log, never unbounded provider payload logs.
if [ -f ingestion-last.log ]; then cp ingestion-last.log ingestion-previous.log; fi
/usr/bin/docker compose --profile ingestion run --rm --no-deps --name wisp-private-ingestion ingest "$@" > ingestion-last.log 2>&1
