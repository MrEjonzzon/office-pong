#!/usr/bin/env bash
# Create the host folder the stack needs (run on the server as root, safe to run again):
#   ./scripts/setup.sh
set -euo pipefail
dir="${BACKUP_DIR:-/tank/Data/backups/pingis}"

mkdir -p "$dir"
# The db-backup container runs as the postgres user (999).
chown -R 999:999 "$dir"
