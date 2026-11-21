# Office Pong

Table tennis (pingis) challenges, live scoring and a leaderboard for the office. Challenge a colleague, pick 1, 3, 5 or 7 sets, and tap your own score on your phone during the match.

## How to play

1. Sign up with a username and password.
2. Tap the swords next to a player and pick the number of sets. They accept, and you both land on the game screen.
3. Tap your own (green) button when you win a point. Their (red) button shows their score.
4. To end the match early, tap "Pick winner" and choose who won. Your opponent must confirm.
5. The result is saved only when both players agree. Then the ratings change. See the leaderboard and your profile for stats.
6. If your opponent rejects the result, the score restarts at 0-0.
7. Tap "Cancel match" to stop the match. No result is saved.

**Rules:** a set goes to 11 points with a 2-point lead (so 10-10 continues until someone leads by 2). A match is 1, 3, 5 or 7 sets and the first to win the majority takes it. There is one shared rating (Elo, starts at 1500). Longer matches move it more: K = 24 / 32 / 40 / 48 for 1 / 3 / 5 / 7 sets.

## Deploy (Portainer + Cloudflare Tunnel)

`compose.yml` runs everything: Postgres, the Go API, the Nuxt web app, a daily DB backup and a Caddy proxy on `127.0.0.1:8090`. It needs `cloudflared` on the host network. On the home server, it runs as the `cloudflared` stack from the Home-Network-Conf repo.

1. **Tunnel:** in your Cloudflare tunnel, add a published application: hostname `pingis.emiljo.com`, service `HTTP` → `localhost:8090`.
2. **Backup folder** (one time, on the server as root): `./scripts/setup.sh`. It creates `/tank/Data/backups/pingis` (or `BACKUP_DIR`) owned by `999:999`. From another machine: `ssh root@omv 'bash -s' < scripts/setup.sh`.
3. **Portainer:** Stacks → Add stack → **Repository**. Repository URL `https://github.com/MrEjonzzon/office-pong`, reference `refs/heads/main`, compose path `compose.yml`.
4. **Environment variables:** `POSTGRES_PASSWORD`, `BETTER_AUTH_SECRET` and `ADMIN_PASSWORD` (see `stack.env.example`; generate with `openssl rand -hex 16`, `openssl rand -hex 32` and `openssl rand -base64 24 | tr -d '/+='`). `ADMIN_USERNAME` defaults to `admin`. Don't change `POSTGRES_PASSWORD` after the first deploy.
5. **Deploy.** Tables are created automatically on first start. To update later: push, then "Pull and redeploy".

Live games are kept in memory, so redeploying during a game loses that game.

## Admin

- **Admin page:** `/admin` (e.g. `https://pingis.emiljo.com/admin`) lists users and lets you create users, rename (display name), set a password and delete users. The browser asks for a login: `ADMIN_USERNAME` / `ADMIN_PASSWORD` from the stack variables. If `ADMIN_PASSWORD` is empty, the page is disabled. Deleting removes the user's games and challenges, but other players' ratings are not recalculated.
- **Backups:** the database is dumped daily to `/tank/Data/backups/pingis` (7 daily, 4 weekly, 6 monthly, plus `last/`). Back up now: `docker exec <db-backup container> /backup.sh`. Restore (on the server): `./scripts/restore-backup.sh [file.sql.gz]`. Without a file it uses `last/officepong-latest.sql.gz`. It asks for confirmation, saves the current data as `pre-restore-<time>.sql.gz` in the backup folder, stops api and web, replaces the database and starts them again. The dumps live on the same machine, so copy them elsewhere to survive losing the server.
- **Reset a password** (on the server, needs `python3` and `docker`): `./scripts/reset-password.sh <username> <new-password>`. It also logs the user out everywhere.
- **Delete users** (e.g. test accounts; on the server): `./scripts/delete-user.sh <username> [username...]`. It asks for confirmation and removes their games and challenges too. Other players' ratings are not recalculated.

## Local development

1. `docker compose -f compose.dev.yml up -d` (Postgres on `localhost:5434`).
2. API: in `api/`, run `go run . -dbport 5434 -dbpass postgres`.
3. Web: in `web/`, copy `.env.example` to `.env`, then `npm install` and `npm run dev`.

## Project structure

- `api/`: Go backend (REST + game WebSocket, ratings, DB setup).
- `web/`: Nuxt 3 frontend (Vue 3, Tailwind) with better-auth login.
- `app/`: Flutter app.
- `deploy/`: Caddy reverse proxy. `scripts/`: server setup and admin scripts.
- `compose.yml`: production stack. `compose.dev.yml`: local Postgres only.

## Credits

Office Pong was written by [@EliottCarvalhal](https://github.com/EliottCarvalhal): the API, the web app and the Flutter app. I got a local copy of his code as a zip and committed it as the first import (`7490731`), so git shows that commit under my name even though the code is his.

My additions on top, in later commits, are small: hosting (Docker, Portainer, Cloudflare Tunnel, backups), username login, and the scoring rules (win by 2, best-of-N sets).

This repository has no license file. Rights to the original code remain with its author.
