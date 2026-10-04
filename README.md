# Office Pong

Table tennis (pingis) challenges, live scoring and a leaderboard for the office. Challenge a colleague, pick 1, 3, 5 or 7 sets, and tap your own score on your phone during the match.

## How to play

1. Sign up with a username and password.
2. Tap the swords next to a player and pick the number of sets. They accept, and you both land on the game screen.
3. Tap your own (green) button when you win a point. Their (red) button shows their score.
4. The winner and the new ratings are saved automatically. See the leaderboard and your profile for stats.

**Rules:** a set goes to 11 points with a 2-point lead (so 10-10 continues until someone leads by 2). A match is 1, 3, 5 or 7 sets and the first to win the majority takes it. There is one shared rating (Elo, starts at 1500). Longer matches move it more: K = 24 / 32 / 40 / 48 for 1 / 3 / 5 / 7 sets.

## Deploy (Portainer + Cloudflare Tunnel)

`compose.yml` runs everything: Postgres, the Go API, the Nuxt web app, a daily DB backup and a Caddy proxy on `127.0.0.1:8090`. It assumes `cloudflared` already runs on the host.

1. **Tunnel:** in your Cloudflare tunnel, add a published application: hostname `pingis.emiljo.com`, service `HTTP` → `localhost:8090`.
2. **Backup folder** (one time, on the server):
   `mkdir -p /tank/shared/nas-shared/backups/pingis && chown -R 999:999 /tank/shared/nas-shared/backups/pingis`
3. **Portainer:** Stacks → Add stack → **Repository**. Repository URL `https://github.com/MrEjonzzon/office-pong`, reference `refs/heads/main`, compose path `compose.yml`.
4. **Environment variables:** `POSTGRES_PASSWORD` and `BETTER_AUTH_SECRET` (see `stack.env.example`; generate with `openssl rand -hex 16` and `openssl rand -hex 32`). Don't change `POSTGRES_PASSWORD` after the first deploy.
5. **Deploy.** Tables are created automatically on first start. To update later: push, then "Pull and redeploy".

Live games are kept in memory, so redeploying during a game loses that game.

## Admin

- **Backups:** the database is dumped daily to `/tank/shared/nas-shared/backups/pingis` (7 daily, 4 weekly, 6 monthly, plus `last/`). Back up now: `docker exec <db-backup container> /backup.sh`. Restore into an empty DB: `gunzip -c <file>.sql.gz | docker exec -i officepong_postgres psql -U postgres officepong`. The dumps live on the same machine, so copy them elsewhere to survive losing the server.
- **Reset a password** (on the server, needs `python3` and `docker`): `./scripts/reset-password.sh <username> <new-password>`. It also logs the user out everywhere.

## Local development

1. `docker compose -f compose.dev.yml up -d` (Postgres on `localhost:5434`).
2. API: in `api/`, run `go run . -dbport 5434 -dbpass postgres`.
3. Web: in `web/`, copy `.env.example` to `.env`, then `npm install` and `npm run dev`.

## Project structure

- `api/`: Go backend (REST + game WebSocket, ratings, DB setup).
- `web/`: Nuxt 3 frontend (Vue 3, Tailwind) with better-auth login.
- `app/`: Flutter app.
- `deploy/`: Caddy reverse proxy. `scripts/`: admin scripts.
- `compose.yml`: production stack. `compose.dev.yml`: local Postgres only.

## Credits

Office Pong was written by [@EliottCarvalhal](https://github.com/EliottCarvalhal): the API, the web app and the Flutter app. I got a local copy of his code as a zip and committed it as the first import (`7490731`), so git shows that commit under my name even though the code is his.

My additions on top, in later commits, are small: hosting (Docker, Portainer, Cloudflare Tunnel, backups), username login, and the scoring rules (win by 2, best-of-N sets).

This repository has no license file. Rights to the original code remain with its author.
