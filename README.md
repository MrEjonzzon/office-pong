# Office Pong

## How to Start the Project

1. **Backend (Go API)**

   - Ensure PostgreSQL is running (see `compose.yml` for Docker setup).
   - Start the Go API server from the `api/` directory (`main.go`).
   - set -dbpass flag

2. **Frontend (Nuxt/Vue)**
   - Navigate to the `web/` directory.
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
- `compose.yml` — Docker Compose for PostgreSQL.
- `README.md` — Project description.

## Database Information

- Uses **PostgreSQL** (see `compose.yml` for Docker setup).
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

---

For more details, see the code and comments in the respective directories.
