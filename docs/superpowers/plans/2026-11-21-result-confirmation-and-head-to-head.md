# Result Confirmation and Head-to-Head Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ratings change only when both players confirm the result. Players can pick the winner or track the score, either player can cancel a match, and players see head-to-head records and K/D.

**Architecture:** The proposal is kept on the `games` row (`state`, `winnerId`, `score`, and the new `proposedBy`). Four REST routes (propose, confirm, reject, cancel) change it, and they get the caller from the token. The in-memory hub reloads from the row and pushes the new state over the websocket. The frontend counts head-to-head from the challenges route.

**Tech Stack:** Go 1.22 (gorilla/mux, gorilla/websocket, lib/pq), Postgres 15, Nuxt 3 / Vue 3 / Tailwind.

**Spec:** `docs/superpowers/specs/2026-11-21-result-confirmation-and-head-to-head-design.md`

## Global Constraints

- Schema changes only as `IF NOT EXISTS` statements, run at API start. No migration tool.
- One API replica. Live game state stays in memory. Only the proposal is in the database.
- `games.state` values: `ongoing`, `awaiting_confirmation`, `completed`, `cancelled`. Hub status values: `playing`, `awaiting_confirmation`, `finished`, `cancelled`.
- The `challenge_state` enum does not change. A cancelled match sets its challenge to `complete`.
- The new routes get the caller's user ID from the token (request context), never from the URL or the body.
- Test fixtures: Star Wars names (`luke`, `vader`, `yoda`), emails on `@example.com`.
- Go tests that need Postgres read `TEST_DATABASE_URL` and skip when it is not set.
- Commits: `type(scope): subject` (no Jira key, private repo), body explains why, end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Comment rule (pass on word for word): **Comments** must be plain, everyday language — max 1–2 lines. This applies in ALL modes (auto-accept, subagents, etc.). Only write a comment when the WHY is not obvious from the code itself. Never describe what the code does; only explain hidden constraints, workarounds, or non-obvious reasons.
  - No jargon words: "idempotent", "instantiate", "heuristics", "stateless", "delegates", "composes", "mirrors", "drift", "round-trip", "innermost/outermost", "pre-created", "rehearsal", "swallows", "deltas", "absolutes", "special-case", "sub-phase". Say it the way you'd explain it out loud: "passed on to", "stacks the wrappers", "stays in sync", "one call", "closest to the real client / on the outside", "created up front", "dry run", "logs and keeps going", "before/after values", "work around".
  - Same rule for user-facing strings that read like comments (metric help texts, log messages).
  - **ALWAYS follow this, every time, no exceptions.** It covers every file type (code, tests, YAML, Makefile, SQL, scripts, Dockerfile) and every comment you write or change.
  - Doc comments on functions and types may say what the thing does (1–2 lines, starting with the name). The why-only rule is for comments inside function bodies.
  - **No captain-obvious comments.** A doc comment must tell the reader something the name, package and signature don't already say. `// Config holds the seat-boost connection settings.` on `seatboost.Config` adds nothing, so leave it out. Never add a doc comment just to fill a gap.
  - A comment that no longer matches the code gets fixed or deleted in the same change.
  - **Never point at code by line number** (`ad_sync.go:1150`) in comments, `CLAUDE.md` files or docs — the next edit makes it wrong. Name the function, type or file instead.
  - When you hand coding work to a subagent, pass this rule on to it word for word.
  - **Check before you finish:** re-read your diff for any comment over 2 lines, any jargon, and any step-by-step narration, and fix them before reporting done.

## Review Focus

1. **Double confirm** (two taps, or both devices at once): ratings change once. Second confirm returns 409. Test: `TestConfirmTwiceChangesRatingsOnce` (Task 2).
2. **A player who is not in the game** calls confirm, reject or cancel: 403, nothing changes. Test: `TestOutsiderCannotConfirmOrCancel` (Task 2).
3. **Both players press "Pick winner" at the same time:** the second propose returns 409. Test: `TestProposeTwiceConflicts` (Task 2).
4. **Redeploy while a proposal waits:** a new hub shows the proposal and ignores score taps. Test: `TestHubLoadsWaitingProposal` (Task 3).
5. **Cancel after the result is confirmed:** 409, ratings stay. Test: `TestCancelCompletedConflicts` (Task 2).

## One-time test setup

```bash
docker compose -f compose.dev.yml up -d
docker exec officepong_postgres_dev createdb -U postgres officepong_test
export TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5434/officepong_test?sslmode=disable'
```

All `go test` commands run in `api/`.

---

### Task 1: Schema in one place, test helper, caller from token, sender-only scoring

**Files:**
- Create: `api/database/schema.go` (move the `CREATE`/`ALTER` statements out of `main()` in `api/main.go`)
- Modify: `api/main.go` (call `mdb.CreateSchema`)
- Create: `api/dbtest/dbtest.go`
- Modify: `api/middleware/auth.go`
- Create: `api/middleware/auth_test.go`
- Modify: `api/game/websocket-client.go` (move the message switch out of `WSHandler`)
- Create: `api/game/websocket-client_test.go`

**Interfaces:**
- Produces:
  - `database.CreateSchema(db *sql.DB) error`. Runs every statement from `main()` plus `ALTER TABLE "games" ADD COLUMN IF NOT EXISTS "proposedBy" text REFERENCES "user" ("id");`
  - `dbtest.Open(t *testing.T) *sql.DB`. Skips when `TEST_DATABASE_URL` is empty. Runs `CreateSchema`, then `TRUNCATE "user", "session", "account", "challenges", "games", "user_stats" CASCADE`.
  - `dbtest.SeedUser(t *testing.T, db *sql.DB, id, name string)`. Inserts a `user` row (email `<id>@example.com`), a `user_stats` row (1500, 0, 0) and a `session` row with token `token-<id>` that expires in one hour.
  - `dbtest.SeedGame(t *testing.T, db *sql.DB, challenger, challengee string, bestOf int) string`. Calls `CreateChallenge` and `AcceptChallenge`, and returns the game ID.
  - `middleware.UserIDFromContext(ctx context.Context) (string, bool)`. `AuthMiddleware` stores the ID from `VerifyUserToken` in the context. Use an unexported key type.
  - `middleware.WithUserID(ctx context.Context, uid string) context.Context`. Handler tests use it.
  - `game.(*Hub).handleMessage(uid string, env Envelope)`. The switch from `WSHandler`. `score:update` always sets `upd.UserID = uid`.

- [ ] **Step 1: Write the failing tests**

```go
// api/middleware/auth_test.go
func TestAuthMiddlewarePutsUserInContext(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.SeedUser(t, db, "luke", "Luke")
	var got string
	h := AuthMiddleware(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = UserIDFromContext(r.Context())
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token-luke")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got != "luke" { t.Fatalf("user in context = %q, want luke", got) }
}

// api/game/websocket-client_test.go (no database needed)
func TestScoreUpdateUsesSender(t *testing.T) {
	h := newHub("g1", nil, 1)
	h.ensurePlayer(Player{UID: "luke"})
	h.ensurePlayer(Player{UID: "vader"})
	go func() { for range h.broadcast {} }()
	h.handleMessage("luke", Envelope{Type: "score:update", Payload: raw(ScoreUpdate{UserID: "vader", Delta: 1})})
	s := h.snapshot()
	if s.Players["luke"].Score != 1 || s.Players["vader"].Score != 0 {
		t.Fatalf("luke=%d vader=%d, want 1 and 0", s.Players["luke"].Score, s.Players["vader"].Score)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./middleware/ ./game/`
Expected: FAIL. The build fails on `UserIDFromContext`, `handleMessage` and `dbtest` (they do not exist).

- [ ] **Step 3: Implement the interfaces above.** `main()` keeps only `if err := mdb.CreateSchema(db); err != nil { log.Fatalf(...) }`. `WSHandler` calls `h.handleMessage(uid, env)` in its read loop.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./...`
Expected: PASS (including the existing `TestSetWinner` and `TestSetsToWin`).

- [ ] **Step 5: Commit**

```bash
git add api/
git commit -m "refactor(api): get the caller from the token and score only for the sender"
```

---

### Task 2: Result and cancel rules in the database layer

**Files:**
- Create: `api/database/results.go`
- Modify: `api/database/database.go` (remove `UpdateMMRAfterGame`, `CompleteGame`, `GetGameBestOf`; extend `GetChallenges`)
- Modify: `api/database/models.go` (`Challenge` fields, new `Game`)
- Create: `api/database/results_test.go` (package `database_test`)

**Interfaces:**
- Consumes: `dbtest.Open`, `dbtest.SeedUser`, `dbtest.SeedGame` (Task 1).
- Produces:
  - `type Game struct { ID, ChallengeID, State, Challenger, Challengee string; BestOf int; WinnerID, Score, ProposedBy *string }`
  - `GetGame(db *sql.DB, gameID string) (*Game, error)`. Joins `challenges` for the players and `bestOf`.
  - `var ErrNotPlayer, ErrWrongState, ErrBadWinner = errors.New(...)`. The routes map them to 403, 409 and 400.
  - `ProposeResult(db *sql.DB, gameID, proposerID, winnerID, score string) error`. `score == ""` stores NULL. Only from `ongoing`.
  - `ConfirmResult(db *sql.DB, gameID, userID string) error`. One transaction:
    1. `UPDATE games SET state='completed' WHERE id=$1 AND state='awaiting_confirmation' AND "proposedBy" <> $2`. If no row changes, return `ErrWrongState`.
    2. Update the ratings with `KFactor` and `GetRatingChange`, reading MMR through the transaction. Call `EnsureUserStats` for both players before the transaction.
    3. Set the challenge to `complete`.
  - `RejectResult(db *sql.DB, gameID, userID string) error`. Only the non-proposer, only from `awaiting_confirmation`. Back to `ongoing` with `winnerId`, `score` and `proposedBy` NULL.
  - `CancelGame(db *sql.DB, gameID, userID string) error`. From `ongoing` or `awaiting_confirmation`. The game becomes `cancelled` with fields NULL, and the challenge becomes `complete`, in one transaction.
  - All four return `ErrNotPlayer` when `userID` is not the challenger or the challengee. Check this with `GetGame` before any update.
  - `Challenge` gets `GameState *string json:"gameState,omitempty"`, `WinnerID *string json:"winnerId,omitempty"`, `ProposedBy *string json:"proposedBy,omitempty"`. `GetChallenges` selects them from the existing join.

- [ ] **Step 1: Write the failing tests.** Each test seeds `luke` and `vader` (plus `yoda` where needed) and a best-of-1 game with `luke` as challenger. Equal ratings at best-of-1 give +12 / −12 (K = 24).

```go
func TestConfirmTwiceChangesRatingsOnce(t *testing.T) {
	// ProposeResult(game, "luke", "luke", "11-7") → nil
	// ConfirmResult(game, "luke") → ErrWrongState (proposer cannot confirm)
	// ConfirmResult(game, "vader") → nil
	// ConfirmResult(game, "vader") → ErrWrongState
	// user_stats: luke mmr 1512 wins 1; vader mmr 1488 losses 1
	// games row: state completed, winnerId luke, score "11-7"; challenge state complete
}
func TestRejectClearsProposal(t *testing.T) {
	// propose by luke, RejectResult(game, "luke") → ErrWrongState, RejectResult(game, "vader") → nil
	// GetGame: State "ongoing", WinnerID nil, Score nil, ProposedBy nil; ratings unchanged
}
func TestProposeTwiceConflicts(t *testing.T) {
	// ProposeResult by luke → nil; ProposeResult by vader → ErrWrongState
}
func TestProposeRejectsBadWinner(t *testing.T) {
	// ProposeResult(game, "luke", "yoda", "") → ErrBadWinner
}
func TestOutsiderCannotConfirmOrCancel(t *testing.T) {
	// propose by luke; ConfirmResult/RejectResult/CancelGame with "yoda" → ErrNotPlayer; state still awaiting_confirmation
}
func TestCancelFreesChallenge(t *testing.T) {
	// CancelGame(game, "vader") → nil; game state cancelled, challenge complete, ratings unchanged
	// CreateChallenge(db, "luke", "vader", 1) → nil error
}
func TestCancelCompletedConflicts(t *testing.T) {
	// propose + confirm, then CancelGame(game, "luke") → ErrWrongState; luke mmr still 1512
}
func TestGetChallengesReturnsGameFields(t *testing.T) {
	// propose by luke; GetChallenges(db, "vader")[0]: GameState "awaiting_confirmation", WinnerID "luke", ProposedBy "luke"
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./database/`
Expected: FAIL. The build fails on `ProposeResult`, `ConfirmResult`, `RejectResult`, `CancelGame`, `GetGame` and the `Err*` values.

- [ ] **Step 3: Implement `api/database/results.go` and the `GetChallenges`/`models.go` changes as listed in Interfaces.** Remove `UpdateMMRAfterGame`, `CompleteGame` and `GetGameBestOf`. Their callers in `api/game/websocket-client.go` will not compile; Task 3 replaces them. To keep this commit building, make `CreateHub` call `GetGame` and use `g.BestOf`, and delete the rating goroutine in `applyScoreUpdate` (Task 3 adds the proposal there).

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add api/
git commit -m "feat(api): save results only after both players confirm"
```

---

### Task 3: Routes and hub on top of the rules

**Files:**
- Modify: `api/main.go` (4 routes and handlers)
- Modify: `api/game/websocket-client.go`
- Modify: `api/game/websocket-client_test.go`

**Interfaces:**
- Consumes: Task 2 functions and errors, `middleware.UserIDFromContext` and `middleware.WithUserID` (Task 1).
- Produces:
  - Routes on `protected`: `POST /v1/game/{id}/result` (body `{"winnerId": "..."}`), `POST /v1/game/{id}/result/confirm`, `POST /v1/game/{id}/result/reject`, `POST /v1/game/{id}/cancel`. Errors: `ErrNotPlayer` → 403, `ErrWrongState` → 409, `ErrBadWinner` → 400, other errors → 500. Success → 200 with `{ "data": "ok" }` through `encodeResponse`. No user ID in context → 401.
  - `GameState` gets `ProposedBy string json:"proposedBy,omitempty"` and `Score string json:"score,omitempty"`. The winner goes in `Meta["winner"]`.
  - `game.(*Hub).loadFromGame(g *mdb.Game)`. Maps `ongoing`→`playing`, `awaiting_confirmation`→`awaiting_confirmation`, `completed`→`finished`, `cancelled`→`cancelled`. Copies `WinnerID`, `Score` and `ProposedBy`. For `ongoing`, it sets all points, `SetsWon` and `Sets` to zero (the score restarts after a reject). Bumps `Version`.
  - `game.RefreshHub(db *sql.DB, gameID string)`. If the hub is in memory: `GetGame`, `loadFromGame`, broadcast `pack("state", snapshot)`. Every handler calls it after success.
  - `CreateHub` calls `loadFromGame` on the new hub.
  - `applyScoreUpdate` ignores updates unless the status is `playing`. On a match win it calls `mdb.ProposeResult(h.db, h.ID, scorerUID, winnerUID, score)`. The scorer is the sender of the winning point, which is the same as the winner here. Only if that succeeds does it set status `awaiting_confirmation`, `ProposedBy`, `Score` and `Meta["winner"]`. On error it logs and keeps `playing`.

- [ ] **Step 1: Write the failing tests** in `api/game/websocket-client_test.go` (they need the database):

```go
func TestMatchWinCreatesProposal(t *testing.T) {
	// seed luke, vader, best-of-1 game; h := CreateHub(game, db); ensurePlayer both
	// 11 × handleMessage("luke", score:update +1)
	// GetGame: State "awaiting_confirmation", ProposedBy "luke", WinnerID "luke", Score "11-0"
	// snapshot().Status == "awaiting_confirmation"; luke mmr still 1500
}
func TestHubLoadsWaitingProposal(t *testing.T) {
	// seed game; ProposeResult(game, "vader", "vader", "")
	// h := CreateHub(game, db) (fresh hub, as after a redeploy); ensurePlayer both
	// snapshot(): Status "awaiting_confirmation", ProposedBy "vader", Meta["winner"] "vader"
	// handleMessage("luke", score:update +1) → luke score stays 0
}
func TestRefreshAfterRejectRestartsScore(t *testing.T) {
	// seed game; hub; luke scores 5; ProposeResult by luke; RejectResult by vader; RefreshHub(db, game)
	// snapshot(): Status "playing", all scores 0, Sets empty
}
```

Plus one handler test in `api/main_test.go`:

```go
func TestConfirmHandlerUsesTokenUser(t *testing.T) {
	// seed game, ProposeResult by luke
	// request with ctx WithUserID("luke") and mux.SetURLVars{"id": game} → 409
	// same request with WithUserID("vader") → 200; GetGame state "completed"
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./...`
Expected: FAIL. The build fails on `RefreshHub` and the handler names, and `TestMatchWinCreatesProposal` sees state `ongoing`.

- [ ] **Step 3: Implement the routes, handlers and hub changes as listed in Interfaces.** Handler names: `proposeResultHandler`, `confirmResultHandler`, `rejectResultHandler`, `cancelGameHandler`, each `func(db *sql.DB) http.HandlerFunc`.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./... && go vet ./...`
Expected: PASS, no vet findings.

- [ ] **Step 5: Commit**

```bash
git add api/
git commit -m "feat(api): add propose, confirm, reject and cancel routes"
```

---

### Task 4: Game screen

**Files:**
- Modify: `web/composables/game-state.ts`
- Modify: `web/pages/game/[gameID].vue`
- Modify: `README.md` ("How to play")

**Interfaces:**
- Consumes: Task 3 routes and `GameState` fields (`status`, `proposedBy`, `score`, `meta.winner`).
- Produces: `useGameState` also returns `proposeWinner(winnerId: string)`, `confirmResult()`, `rejectResult()`, `cancelMatch()`. Each one is `$fetch` POST with `Authorization: Bearer <token>` to the Task 3 route, with a destructive toast on error. Local `GameState` gets `proposedBy?: string` and `score?: string`. `status` is `'playing' | 'awaiting_confirmation' | 'finished' | 'cancelled'`.

- [ ] **Step 1: Add the composable functions and fields.**
- [ ] **Step 2: Update the page.**
  - A "Pick winner" button opens two choices: "I won" and "<opponent name> won".
  - A "Cancel match" button asks `confirm('Cancel this match? No result is saved.')`.
  - While `awaiting_confirmation`: an overlay. The proposer sees `Waiting for <opponent> to confirm`. The other player sees `<proposer> says <winner> won` plus `score` when set, with Confirm and Reject buttons.
  - While `cancelled`: an overlay `Match cancelled` with the existing "Back to Home" link.
  - Disable the score buttons unless `status === 'playing'`.
- [ ] **Step 3: Update "How to play" in `README.md`.** Describe pick winner or tap score, opponent confirms, reject restarts the score at 0–0, and cancel.
- [ ] **Step 4: Verify.**
  - Run `cd web && npm ci && npm run build`. Expected: build succeeds.
  - Then run the API and web as described in README "Local development", with two browsers logged in as two users:
    1. Pick winner → the other browser shows Confirm.
    2. Confirm → both see the result, and the profile ratings change.
    3. Reject → score 0–0.
    4. Cancel → both see "Match cancelled".
- [ ] **Step 5: Commit**

```bash
git add web/composables/game-state.ts web/pages/game/[gameID].vue README.md
git commit -m "feat(web): pick a winner, confirm or reject, and cancel on the game screen"
```

---

### Task 5: Home player list: confirm from home and head-to-head

**Files:**
- Modify: `web/types/api.ts`
- Create: `web/utils/head-to-head.ts`
- Modify: `web/components/views/user-list.vue`

**Interfaces:**
- Consumes: `gameState`, `winnerId` and `proposedBy` on challenges (Task 2), and the confirm/reject routes (Task 3).
- Produces:
  - `APIChallenge` gets `gameState?: 'ongoing' | 'awaiting_confirmation' | 'completed' | 'cancelled'`, `winnerId?: string`, `proposedBy?: string`.
  - `headToHead(challenges: APIChallenge[], myId: string): Record<string, { wins: number; losses: number }>`. Keyed by opponent ID. It counts only `gameState === 'completed'` with `winnerId` set.
  - `kd(wins: number, losses: number): string`. `losses === 0 ? wins.toFixed(2) : (wins / losses).toFixed(2)`.
  - Nuxt auto-imports both from `web/utils`.

- [ ] **Step 1: Add the types and `web/utils/head-to-head.ts`.**
- [ ] **Step 2: Update `user-list.vue`.**
  - The redirect watcher also requires `c.gameState !== 'awaiting_confirmation'`.
  - `PlayerRow.status` gets `'awaiting'`, used when the found challenge's `gameState === 'awaiting_confirmation'`.
  - For `'awaiting'`: if `challenge.proposedBy === myId`, a disabled clock button. Otherwise the text `<proposer name> says <winner name> won` with Confirm and Reject buttons. They POST to the Task 3 routes, then call `executeGetChallenges()`.
  - Next to each name, show `wins–losses` from `headToHead` when that player has at least one completed game.
- [ ] **Step 3: Verify.**
  - Run `npm run build`. Expected: succeeds.
  - Manual test:
    1. Propose a result, then leave the game screen. The opponent's home list shows Confirm/Reject, and the proposer's list shows the clock.
    2. The challenger is not sent to the game.
    3. After confirm, the record appears, for example `1–0`.
- [ ] **Step 4: Commit**

```bash
git add web/types/api.ts web/utils/head-to-head.ts web/components/views/user-list.vue
git commit -m "feat(web): confirm results and show head-to-head on the home list"
```

---

### Task 6: Profile: K/D and per-opponent records

**Files:**
- Modify: `web/pages/profile.vue`
- Modify: `README.md` ("How to play": where the stats are)

**Interfaces:**
- Consumes: `headToHead` and `kd` (Task 5), `GET /v1/user/{id}/challenges` and `GET /v1/users`.

- [ ] **Step 1: Update the profile.**
  - Under Wins and Losses, add `K/D : {{ kd(stats.wins, stats.losses) }}`, from `user_stats`.
  - Below that, add a list from `headToHead(challenges, myId)`. Each row: opponent image and name (from `/v1/users`), `wins–losses`, K/D. Sort by most games first. Show "No matches yet" when the list is empty.
- [ ] **Step 2: Update `README.md` with one line saying where the head-to-head and K/D appear.**
- [ ] **Step 3: Verify.** Run `npm run build`, expect it to succeed. Manually check that the profile shows K/D and one row per opponent that matches the home list.
- [ ] **Step 4: Commit**

```bash
git add web/pages/profile.vue README.md
git commit -m "feat(web): show K/D and head-to-head on the profile"
```
