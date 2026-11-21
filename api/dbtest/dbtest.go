package dbtest

import (
	"database/sql"
	"os"
	"testing"
	"time"

	mdb "office-pong/database"

	_ "github.com/lib/pq"
)

// Open connects to TEST_DATABASE_URL, creates the schema and empties all tables. Skips the test when the variable is not set.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := mdb.CreateSchema(db); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(`TRUNCATE "user", "session", "account", "challenges", "games", "user_stats" CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

// SeedUser inserts a user with stats and a session whose token is "token-<id>".
func SeedUser(t *testing.T, db *sql.DB, id, name string) {
	t.Helper()
	now := time.Now()
	stmts := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO "user" ("id","name","email","emailVerified","createdAt","updatedAt") VALUES ($1,$2,$3,true,$4,$4)`, []any{id, name, id + "@example.com", now}},
		{`INSERT INTO "user_stats" ("userId","mmr","wins","losses") VALUES ($1,1500,0,0)`, []any{id}},
		{`INSERT INTO "session" ("id","expiresAt","token","createdAt","updatedAt","userId") VALUES ($1,$2,$3,$4,$4,$5)`, []any{"session-" + id, now.Add(time.Hour), "token-" + id, now, id}},
	}
	for _, s := range stmts {
		if _, err := db.Exec(s.q, s.args...); err != nil {
			t.Fatalf("seed user %s: %v", id, err)
		}
	}
}

// SeedGame creates an accepted challenge and returns the game ID.
func SeedGame(t *testing.T, db *sql.DB, challenger, challengee string, bestOf int) string {
	t.Helper()
	challengeID, err := mdb.CreateChallenge(db, challenger, challengee, bestOf)
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	gameID, err := mdb.AcceptChallenge(db, challengeID)
	if err != nil {
		t.Fatalf("accept challenge: %v", err)
	}
	return gameID
}
