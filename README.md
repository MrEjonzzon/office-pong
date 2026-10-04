# Office Pong

## Deploy with Portainer (Docker + Cloudflare Tunnel)

`compose.yml` runs everything: Postgres, the Go API, the Nuxt web app, and a Caddy reverse proxy published on `127.0.0.1:8090`. It assumes `cloudflared` already runs on the host.

1. In your existing Cloudflare tunnel, add a published application: hostname `pingis.emiljo.com`, service `HTTP` → `localhost:8090`.
2. In Portainer: Stacks → Add stack → **Repository**. Set the repo URL and compose path `compose.yml`.
3. Add the environment variables from `stack.env.example` (`POSTGRES_PASSWORD`, `BETTER_AUTH_SECRET`). `PUBLIC_URL` / `PUBLIC_WS_URL` default to `pingis.emiljo.com`.
4. Deploy. Tables are created automatically on first start.

## Backups

The `db-backup` service dumps the database daily (gzipped `pg_dump`) to `/tank/shared/nas-shared/backups/pingis` on the host (override with `BACKUP_DIR`). It keeps 7 daily, 4 weekly and 6 monthly dumps in `daily/`, `weekly/` and `monthly/`, plus `last/` with the newest one.

- One-time host setup, before the first deploy: `mkdir -p /tank/shared/nas-shared/backups/pingis && chown -R 999:999 /tank/shared/nas-shared/backups/pingis` (the container writes as UID 999).
- Back up now: `docker exec <db-backup container> /backup.sh`
- Restore into an empty DB: `gunzip -c <file>.sql.gz | docker exec -i officepong_postgres psql -U postgres officepong`
- The dumps are on the same machine as the database, so they don't protect against losing the server. Copy the folder elsewhere for that.
- Reset a user's password (run on the server, needs `python3` and `docker`): `./scripts/reset-password.sh <username> <new-password>`. It also logs the user out everywhere. Without the repo on the server: `curl -O https://raw.githubusercontent.com/MrEjonzzon/office-pong/main/scripts/reset-password.sh && chmod +x reset-password.sh` (public repo only).

## Rules

- A set is won at 11 points with a 2-point lead (at 10-10 play continues until someone leads by 2).
- A match is 1, 3, 5 or 7 sets (first to win the majority). The challenger picks the format when sending the challenge; the opponent agrees by accepting.
- One shared rating (MMR, Elo, starts at 1500). Longer matches move it more: K = 24 (1 set), 32 (best of 3), 40 (best of 5), 48 (best of 7).
- The winner and set scores of each finished game are stored in the `games` table.

## How to Start the Project (local development)

1. **Backend (Go API)**

   - Start Postgres: `docker compose -f compose.dev.yml up -d` (published on `localhost:5434`).
   - Start the Go API server from the `api/` directory: `go run . -dbport 5434 -dbpass postgres` (matches `compose.yml`; flags default to port 5432 and an empty password).

2. **Frontend (Nuxt/Vue)**
   - Navigate to the `web/` directory.
   - Copy `.env.example` to `.env` and adjust if needed.
   - Install dependencies: `npm install`
   - Start the dev server: `npm run dev`

3. **Database**
   - Postgres via `compose.yml`, or any Postgres instance configured through `web/.env` and the API `-db*` flags
## Project Structure

- `api/` — Go backend API, database models, and logic.
- `web/` — Nuxt 3 frontend app (Vue 3, Tailwind, etc.)
  - `components/` — Vue components (user list, challenge item, etc.)
  - `pages/` — Nuxt pages (e.g., `home.vue`)
  - `types/` — TypeScript types for API and users.
  - `assets/` — Static files and images.
  - `configs/` — App and PWA configuration.
- `compose.yml` — production stack for Portainer; `compose.dev.yml` — local PostgreSQL only.
- `README.md` — Project description.

## Database Information

- Uses **PostgreSQL** (see `compose.dev.yml` for local Docker setup).
- Main tables: `user`, `challenges`, `matches`, `players`.
- Challenge logic ensures only one pending challenge between two users at a time.
- Connection info and secrets are managed via `.env` and Nuxt runtime config.

## Common Pitfalls

- **Token Handling:** WebSocket authentication uses the same token as session auth; consider using a separate, temporary token for WS.
- **Challenge Logic:** You cannot send a challenge to a user if a pending challenge already exists (incoming or outgoing).
- **Polling:** The frontend polls users and challenges every 2 seconds; be mindful of API rate limits and performance.
- **Database Migrations:** Use the provided migration scripts in `better-auth_migrations/` and the `migrate` npm script.
- **Environment Variables:** Ensure `.env` is correctly set up for local development.
- **Frontend/Backend Sync:** Make sure both servers are running and connected to the same database instance.

## Credits

The code was written by [@EliottCarvalhal](https://github.com/EliottCarvalhal). Only minor changes were made on top of it for hosting (Docker/Portainer deployment).

---

For more details, see the code and comments in the respective directories.
