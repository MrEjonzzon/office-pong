# Result confirmation and head-to-head stats

Date: 2026-11-21. Status: approved.

## Intent

A player can give themselves a win today. Each player taps their own score, the server ends the match when the score says so, and the ratings change at once. Nobody confirms the result.

This change has two parts:

1. **Result confirmation.** A match ends with a proposed result. The ratings change only when both players agree. Players can still track the score point by point, or they can pick the winner directly.
2. **Head-to-head stats.** Players can see their record against each other player, and a K/D value (wins ÷ losses).

Build part 1 first. The stats count saved games, so they are only correct when the results are confirmed.

## Goals

- A match result is saved, and the ratings change, only when both players agree.
- A player can pick the winner without tracking the score.
- A player can still track the score. When the score ends the match, the result becomes a proposal. It is not saved directly.
- A player who leaves the game screen can confirm or reject later, from the player list on the home page.
- Either player can cancel a match that is not completed. A cancelled match has no result, and the two players can challenge each other again.
- The home player list shows your record against each player, for example `5–3`.
- The profile shows your overall K/D and one row per opponent: wins, losses and K/D.

## Non-goals

- Proposals do not expire. An unconfirmed proposal stays until the opponent confirms or rejects it.
- No history of rejected proposals.
- No profile pages for other players. The profile shows only your own stats.
- No K/D on the leaderboard.
- No points-based ratio. Matches where the winner is picked have no points.
- No fix for the existing gap where accept and decline do not check that the caller is the challengee. Note it as a known gap.

## Anti-goals

- A player can confirm their own proposal, or confirm for the opponent. This would make the feature useless.
- The ratings change twice for one match, for example after a double tap or a retry.
- A redeploy loses a proposal that waits for confirmation.

## How it works today (relevant parts)

- **Challenge flow:** challenge → accept → `createGame` inserts a `games` row with state `ongoing` → both players open `/game/<id>` and join the websocket hub.
- **Live game state** is kept only in API memory (`game.Hub`). A redeploy loses it.
- **Scoring:** the client sends `score:update` over the websocket. `applyScoreUpdate` counts sets with `SetWinner` and `SetsToWin`. When the match is won, it starts a goroutine that calls `UpdateMMRAfterGame` and `CompleteGame`.
- **Auth:** `AuthMiddleware` checks that the bearer token is valid, but it does not pass the user ID to the handler. The handlers trust the user ID in the URL path. The websocket handler gets the user ID from the `token` query parameter.
- **Cheat gap in scoring:** `score:update` takes a `userId` from the client and uses the sender's ID only when it is empty. A client can therefore change the opponent's score (delta −1 or +1).
- **Data that exists:** `games` has `winnerId`, `score` (text, for example `"11-7, 9-11"`) and `state`. `challenges` has `challenger`, `challengee` and `bestOf`. `user_stats` has `mmr`, `wins` and `losses`.

## Approaches

1. **Proposal saved on the `games` row, confirmed through REST routes. Chosen.** The database is the only source of truth. The game screen and the home page use the same routes. The websocket only tells the screen that something changed. This is the most work, but it survives a redeploy and has one path for each action.
2. **Everything over the websocket, plus one REST route for the home page.** Two paths for the same action, and live state is lost on a redeploy. Rejected.
3. **A separate `results` table with one row per proposal.** It gives a history that nobody asked for. Rejected (see Non-goals).

For the stats, count confirmed games from the data that exists. The challenges route already returns all games of a user, so the frontend counts them and no new route is necessary. Rejected: running counters per player pair, because counters can get out of step with the games. Also rejected for now: a separate SQL count route, because the challenges route already has the data.

Cuts after review (to keep it small):

- No `proposedWinnerId` or `proposedScore` columns. The proposal uses `winnerId` and `score`, and `state` tells if they are confirmed.
- No "undo the last point" after a reject. The score starts again at 0–0.
- No head-to-head route. The frontend counts from the challenges route.

## Design: result confirmation

### Data

Add one column to `games`, with `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, the same way as `winnerId` and `score`:

- `proposedBy` text, references `user`

The proposal uses the existing `winnerId` and `score` columns. `games.state` tells if they are confirmed:

| `state` | `winnerId`, `score` |
|---|---|
| `ongoing` | empty |
| `awaiting_confirmation` | the proposed result (`score` empty when the winner was picked) |
| `completed` | the confirmed result |
| `cancelled` | empty |

The column is plain text, so no enum change is necessary. Stats count only `completed`.

The challenge stays `active` until the result is confirmed. Then it becomes `complete`, as today.

### Who the caller is

The new routes must get the caller's user ID from the bearer token, not from the URL or the body. Change `AuthMiddleware` so that it puts the user ID from `VerifyUserToken` into the request context, and add a small helper to read it. The existing routes can keep using the path, which is outside this scope.

Also fix the scoring cheat gap: `score:update` always uses the sender's user ID. Ignore `userId` in the payload.

### Rules

- **Propose:** only a player in the game can propose, and only when the game is `ongoing`. The winner must be one of the two players. The proposer can name themselves or the opponent.
- **The proposer counts as having agreed.** Only the other player can confirm or reject.
- **Confirm:** in one database transaction:
  1. Update the game from `awaiting_confirmation` to `completed`, but only if the caller is the other player.
  2. Update the ratings (`UpdateMMRAfterGame`, changed so that it runs inside the given transaction).
  3. Set the challenge to `complete`.

  If step 1 changes no row, return 409 and change nothing. A double tap or a retry can therefore not apply the ratings twice.
- **Reject:** only the other player. The game goes back to `ongoing`, and `winnerId`, `score` and `proposedBy` are cleared.
- **Score-based finish:** when `applyScoreUpdate` finds that the match is won, it creates a proposal. It does not save the result. The player whose point ended the match is the proposer. `score` is the set scores from the winner's side, as today. Scoring is blocked while the game is `awaiting_confirmation`.
- **Cancel:** either player, when the game is `ongoing` or `awaiting_confirmation`. In one transaction: the game becomes `cancelled` (clear `winnerId`, `score`, `proposedBy`), and the challenge becomes `complete`. The ratings do not change. Reuse `complete` so that the `challenge_state` enum does not change. `CreateChallenge` then lets the two players challenge each other again. This also ends disputed matches (players who keep rejecting) and abandoned matches.
- **After a reject, the score starts again at 0–0.** This is the same for a tracked and a picked result, and the same as after a redeploy. The players then track again or pick the winner. A disputed score usually needs a replay anyway.

### API

All routes are under `/api/v1`, use the auth middleware, and return the existing `{ data, error }` format.

| Route | Body | Result |
|---|---|---|
| `POST /v1/game/{id}/result` | `{ "winnerId": "..." }` | Creates a proposal. 409 if the game is not `ongoing`. 403 if the caller is not in the game. 400 if the winner is not a player. |
| `POST /v1/game/{id}/result/confirm` | none | Confirms. 409 if there is nothing to confirm, or if the caller is the proposer. |
| `POST /v1/game/{id}/result/reject` | none | Rejects. Same errors as confirm. |
| `POST /v1/game/{id}/cancel` | none | Cancels. 403 if the caller is not in the game. 409 if the game is `completed` or `cancelled`. |

After a successful call, the handler updates the hub if it is in memory (`GetHub`) and sends the new state to all connected clients.

### Hub and websocket

- `GameState` gets: `status` value `awaiting_confirmation`, plus `proposedBy` and `score` (the same text as `games.score`). The proposed winner goes in the existing `meta.winner`. Keep the existing hub status names `playing` and `finished` (decision: they map to `ongoing` and `completed` in the database).
- `CreateHub` reads the game row (extend the query that already reads `bestOf`). It does not rebuild the points per player from the `score` text. The screen shows the `score` text as it is.
- If the game is `awaiting_confirmation`, `completed` or `cancelled`, the hub starts in that state with `winnerId`, `score` and `proposedBy`. A reconnecting client then sees the proposal or the result, and scoring stays blocked.
- Remove the goroutine in `applyScoreUpdate` that updates the ratings and completes the game. Only the confirm route does that now.

### Game screen (`web/pages/game/[gameID].vue`)

- The score buttons work as today.
- A **"Pick winner"** button opens a choice: "I won" or "<opponent> won". It calls the propose route.
- When the game is `awaiting_confirmation`:
  - The proposer sees "Waiting for <opponent> to confirm".
  - The other player sees "<proposer> says <winner> won (score, if any)" with **Confirm** and **Reject**.
- When the game is `completed`, the existing "You Win / You Lose" overlay shows.
- A **"Cancel match"** button, with a confirm prompt, calls the cancel route. When the game is cancelled, both screens show "Match cancelled" with "Back to Home". The hub status for this is `cancelled`.

### Home player list (`web/components/views/user-list.vue`)

- `GET /v1/user/{id}/challenges` also returns the game's `state`, `winnerId` and `proposedBy` (through the existing join to `games`). The head-to-head stats use the same fields.
- For a challenge whose game is `awaiting_confirmation`:
  - If you are the other player, show **Confirm** and **Reject**, with a short text such as "Vader says he won".
  - If you are the proposer, show a waiting icon.
- The list watches for `active` challenges and sends the challenger to `/game/<id>`. Skip that redirect when the game is `awaiting_confirmation`, or the challenger never sees the confirm buttons on the home page.

## Design: head-to-head stats

### Data

No new route. `GET /v1/user/{id}/challenges` already returns every challenge of the user, with the opponent. With the game `state` and `winnerId` added (see the home player list above), the frontend counts wins and losses per opponent from the challenges whose game is `completed`. The home list already loads this data. The profile loads it too, plus `/v1/users` for names and images.

Completed games from before this change also count, because they were real results at that time.

`ponytail:` the frontend counts over all challenges of the user. Add a SQL count route if one user gets thousands of games.

### K/D

K/D = wins ÷ losses, shown with 2 decimals. If losses is 0, K/D = wins. This is the normal convention in games, and it avoids division by zero. Calculate it in the frontend.

### UI

- **Home player list:** show your record next to each player, for example `5–3`. Show nothing when you have no games against that player.
- **Profile:** add the overall K/D under the existing wins and losses. Calculate it from the same `user_stats` wins and losses, so the three numbers agree. Add a list with one row per opponent: name, `wins–losses`, K/D.
- The rows per opponent can add up to less than the totals. `delete-user.sh` deletes a user's games but keeps the other players' stats. This is expected.

## Constraints

- Postgres 15. Create the schema in `api/main.go` with `IF NOT EXISTS` statements, as today. No migration tool.
- One API replica. Live game state stays in memory. Only the proposal is in the database.
- The frontend is Nuxt 3 with Vue 3, Tailwind and the components in `web/components/ui`.
- Tests: Go tests next to the code (see `api/game/rules_test.go`). There is no frontend test setup.

## Left to the builder

- The exact layout and text in the UI, within the existing style.
- The context key and helper names for the user ID.
- How to split the database functions.
