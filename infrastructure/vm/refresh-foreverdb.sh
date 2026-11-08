#!/bin/sh
# Provider-specific bounded import. Separate lock and limits; no model calls.
set -eu
cd /home/leon/wisp-private
umask 077
# Explicit opt-in; the friend test must not depend on this provider.
if [ ! -f .foreverdb-enabled ]; then
 printf '{"status":"paused","reason":"operator_disabled"}\n' > foreverdb-last.log
 exit 0
fi
exec 9>.foreverdb.lock
/usr/bin/flock -n 9 || exit 0
available_kb=$(df -Pk . | awk 'NR==2 {print $4}')
memory_kb=$(awk '/^MemAvailable:/ {print $2}' /proc/meminfo)
if [ "$available_kb" -lt 3145728 ] || [ "$memory_kb" -lt 524288 ]; then
 printf '{"status":"resource_guard"}\n' > foreverdb-last.log
 exit 75
fi
if [ -f foreverdb-last.log ]; then cp foreverdb-last.log foreverdb-previous.log; fi
/usr/bin/docker compose --profile foreverdb run --rm --no-deps --name wisp-private-foreverdb foreverdb > foreverdb-last.log 2>&1
