#!/usr/bin/env bash
# Delete users and everything tied to them (run on the server):
#   ./delete-user.sh <username> [username...]
# Other players' ratings are NOT recalculated.
set -euo pipefail
[ $# -ge 1 ] || { echo "usage: $0 <username> [username...]" >&2; exit 1; }

# usernames are lowercase letters, digits, _ and . so a comma-separated list is safe
names=$(printf '%s\n' "$@" | tr 'A-Z' 'a-z' | paste -sd, -)
psql() { docker exec -i officepong_postgres psql -U postgres officepong -v ON_ERROR_STOP=1 -v unames="$names" "$@"; }

echo "Matching users:"
found=$(psql -t -A <<'SQL'
SELECT username FROM "user" WHERE username = ANY(string_to_array(:'unames', ','));
SQL
)
[ -n "$found" ] || { echo "none found"; exit 1; }
echo "$found"

read -r -p "Delete these users and their games/challenges? Type yes: " answer
[ "$answer" = "yes" ] || { echo "aborted"; exit 1; }

psql <<'SQL'
BEGIN;
CREATE TEMP TABLE del AS
  SELECT id FROM "user" WHERE username = ANY(string_to_array(:'unames', ','));

DELETE FROM games WHERE "winnerId" IN (SELECT id FROM del)
   OR "challengeId" IN (SELECT id FROM challenges
        WHERE challenger IN (SELECT id FROM del) OR challengee IN (SELECT id FROM del)
           OR "createdBy" IN (SELECT id FROM del));
DELETE FROM challenges WHERE challenger IN (SELECT id FROM del)
   OR challengee IN (SELECT id FROM del) OR "createdBy" IN (SELECT id FROM del);
DELETE FROM user_stats WHERE "userId" IN (SELECT id FROM del);
DELETE FROM session    WHERE "userId" IN (SELECT id FROM del);
DELETE FROM account    WHERE "userId" IN (SELECT id FROM del);
DELETE FROM "user"     WHERE id       IN (SELECT id FROM del);
COMMIT;
SQL
