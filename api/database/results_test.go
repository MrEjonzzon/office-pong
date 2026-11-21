package database_test

import (
	"database/sql"
	"errors"
	"testing"

	mdb "office-pong/database"
	"office-pong/dbtest"
)

func setup(t *testing.T) (*sql.DB, string) {
	db := dbtest.Open(t)
	dbtest.SeedUser(t, db, "luke", "Luke")
	dbtest.SeedUser(t, db, "vader", "Vader")
	dbtest.SeedUser(t, db, "yoda", "Yoda")
	return db, dbtest.SeedGame(t, db, "luke", "vader", 1)
}

func stats(t *testing.T, db *sql.DB, id string) (mmr, wins, losses int) {
	t.Helper()
	err := db.QueryRow(`SELECT "mmr","wins","losses" FROM "user_stats" WHERE "userId"=$1`, id).Scan(&mmr, &wins, &losses)
	if err != nil {
		t.Fatalf("read stats %s: %v", id, err)
	}
	return
}

func challengeState(t *testing.T, db *sql.DB, gameID string) string {
	t.Helper()
	var s string
	err := db.QueryRow(`SELECT c."state" FROM "challenges" c JOIN "games" g ON g."challengeId"=c."id" WHERE g."id"=$1`, gameID).Scan(&s)
	if err != nil {
		t.Fatalf("read challenge: %v", err)
	}
	return s
}

func mustGame(t *testing.T, db *sql.DB, gameID string) *mdb.Game {
	t.Helper()
	g, err := mdb.GetGame(db, gameID)
	if err != nil {
		t.Fatalf("GetGame: %v", err)
	}
	return g
}

func wantErr(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfirmTwiceChangesRatingsOnce(t *testing.T) {
	db, game := setup(t)
	noErr(t, mdb.ProposeResult(db, game, "luke", "luke", "11-7"))
	wantErr(t, mdb.ConfirmResult(db, game, "luke"), mdb.ErrWrongState)
	noErr(t, mdb.ConfirmResult(db, game, "vader"))
	wantErr(t, mdb.ConfirmResult(db, game, "vader"), mdb.ErrWrongState)

	if mmr, wins, _ := stats(t, db, "luke"); mmr != 1512 || wins != 1 {
		t.Fatalf("luke mmr %d wins %d", mmr, wins)
	}
	if mmr, _, losses := stats(t, db, "vader"); mmr != 1488 || losses != 1 {
		t.Fatalf("vader mmr %d losses %d", mmr, losses)
	}
	g := mustGame(t, db, game)
	if g.State != "completed" || g.WinnerID == nil || *g.WinnerID != "luke" || g.Score == nil || *g.Score != "11-7" {
		t.Fatalf("unexpected game %+v", g)
	}
	if s := challengeState(t, db, game); s != "complete" {
		t.Fatalf("challenge state %s", s)
	}
}

func TestRejectClearsProposal(t *testing.T) {
	db, game := setup(t)
	noErr(t, mdb.ProposeResult(db, game, "luke", "luke", "11-7"))
	wantErr(t, mdb.RejectResult(db, game, "luke"), mdb.ErrWrongState)
	noErr(t, mdb.RejectResult(db, game, "vader"))

	g := mustGame(t, db, game)
	if g.State != "ongoing" || g.WinnerID != nil || g.Score != nil || g.ProposedBy != nil {
		t.Fatalf("unexpected game %+v", g)
	}
	if mmr, _, _ := stats(t, db, "luke"); mmr != 1500 {
		t.Fatalf("luke mmr %d", mmr)
	}
	if mmr, _, _ := stats(t, db, "vader"); mmr != 1500 {
		t.Fatalf("vader mmr %d", mmr)
	}
}

func TestProposeTwiceConflicts(t *testing.T) {
	db, game := setup(t)
	noErr(t, mdb.ProposeResult(db, game, "luke", "luke", ""))
	wantErr(t, mdb.ProposeResult(db, game, "vader", "vader", ""), mdb.ErrWrongState)
}

func TestProposeRejectsBadWinner(t *testing.T) {
	db, game := setup(t)
	wantErr(t, mdb.ProposeResult(db, game, "luke", "yoda", ""), mdb.ErrBadWinner)
}

func TestOutsiderCannotConfirmOrCancel(t *testing.T) {
	db, game := setup(t)
	noErr(t, mdb.ProposeResult(db, game, "luke", "luke", "11-7"))
	wantErr(t, mdb.ConfirmResult(db, game, "yoda"), mdb.ErrNotPlayer)
	wantErr(t, mdb.RejectResult(db, game, "yoda"), mdb.ErrNotPlayer)
	wantErr(t, mdb.CancelGame(db, game, "yoda"), mdb.ErrNotPlayer)
	if s := mustGame(t, db, game).State; s != "awaiting_confirmation" {
		t.Fatalf("state %s", s)
	}
}

func TestCancelFreesChallenge(t *testing.T) {
	db, game := setup(t)
	noErr(t, mdb.CancelGame(db, game, "vader"))
	if s := mustGame(t, db, game).State; s != "cancelled" {
		t.Fatalf("state %s", s)
	}
	if s := challengeState(t, db, game); s != "complete" {
		t.Fatalf("challenge state %s", s)
	}
	if mmr, _, _ := stats(t, db, "luke"); mmr != 1500 {
		t.Fatalf("luke mmr %d", mmr)
	}
	if mmr, _, _ := stats(t, db, "vader"); mmr != 1500 {
		t.Fatalf("vader mmr %d", mmr)
	}
	if _, err := mdb.CreateChallenge(db, "luke", "vader", 1); err != nil {
		t.Fatalf("new challenge: %v", err)
	}
}

func TestCancelCompletedConflicts(t *testing.T) {
	db, game := setup(t)
	noErr(t, mdb.ProposeResult(db, game, "luke", "luke", "11-7"))
	noErr(t, mdb.ConfirmResult(db, game, "vader"))
	wantErr(t, mdb.CancelGame(db, game, "luke"), mdb.ErrWrongState)
	if mmr, _, _ := stats(t, db, "luke"); mmr != 1512 {
		t.Fatalf("luke mmr %d", mmr)
	}
}

func TestGetChallengesReturnsGameFields(t *testing.T) {
	db, game := setup(t)
	noErr(t, mdb.ProposeResult(db, game, "luke", "luke", "11-7"))
	cs, err := mdb.GetChallenges(db, "vader")
	noErr(t, err)
	if len(cs) != 1 {
		t.Fatalf("got %d challenges", len(cs))
	}
	c := cs[0]
	if c.GameState == nil || *c.GameState != "awaiting_confirmation" ||
		c.WinnerID == nil || *c.WinnerID != "luke" ||
		c.ProposedBy == nil || *c.ProposedBy != "luke" {
		t.Fatalf("unexpected challenge %+v", c)
	}
}
