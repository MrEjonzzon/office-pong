#!/usr/bin/env bash
# Replace the database with a backup dump (run on the server):
#   ./scripts/restore-backup.sh [file.sql.gz]
# Without a file it uses the newest dump. The current data is saved first, next to the backups.
set -euo pipefail
dir="${BACKUP_DIR:-/tank/Data/backups/pingis}"
file="${1:-$dir/last/officepong-latest.sql.gz}"
[ -f "$file" ] || { echo "no such file: $file" >&2; exit 1; }

psql() { docker exec -i officepong_postgres psql -U postgres -v ON_ERROR_STOP=1 "$@"; }
project=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' officepong_postgres)

read -r -p "Replace all office-pong data with $file? Type yes: " answer
[ "$answer" = "yes" ] || { echo "aborted"; exit 1; }

safety="$dir/pre-restore-$(date +%Y%m%d-%H%M%S).sql.gz"
docker exec officepong_postgres pg_dump -U postgres officepong | gzip > "$safety"
echo "current data saved to $safety"

# Stop api and web so nothing writes while the database is replaced.
apps=$(for service in api web; do
  docker ps -q --filter "label=com.docker.compose.project=$project" --filter "label=com.docker.compose.service=$service"
done)
if [ -n "$apps" ]; then
  docker stop $apps >/dev/null
  trap 'docker start $apps >/dev/null' EXIT
fi

psql -d postgres -c 'DROP DATABASE officepong WITH (FORCE)' -c 'CREATE DATABASE officepong'
gunzip -c "$file" | psql -d officepong -q >/dev/null
echo "restored $file"
