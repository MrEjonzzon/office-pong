#!/usr/bin/env bash
# Reset a user's password (run on the server):  ./reset-password.sh <username> <new-password>
set -euo pipefail
user="${1:?usage: $0 <username> <new-password>}"
pass="${2:?usage: $0 <username> <new-password>}"

# Same format as better-auth: "<hex salt>:<hex scrypt key>"
hash=$(PASS="$pass" python3 -c '
import hashlib, os, secrets, unicodedata
salt = secrets.token_hex(16)
key = hashlib.scrypt(unicodedata.normalize("NFKC", os.environ["PASS"]).encode(),
                     salt=salt.encode(), n=16384, r=16, p=1, dklen=64, maxmem=64 * 1024 * 1024)
print(f"{salt}:{key.hex()}")')

docker exec -i officepong_postgres psql -U postgres officepong -v ON_ERROR_STOP=1 \
  -v hash="$hash" -v uname="${user,,}" <<'SQL'
UPDATE account SET password = :'hash', "updatedAt" = NOW()
WHERE "providerId" = 'credential' AND "userId" = (SELECT id FROM "user" WHERE username = :'uname');
DELETE FROM session WHERE "userId" = (SELECT id FROM "user" WHERE username = :'uname');
SQL
