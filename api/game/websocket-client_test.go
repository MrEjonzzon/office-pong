package game

import (
	"database/sql"
	"testing"

	mdb "office-pong/database"
	"office-pong/dbtest"
)

func setupGame(t *testing.T) (*sql.DB, string) {
	db := dbtest.Open(t)
	dbtest.SeedUser(t, db, "luke", "Luke")
	dbtest.SeedUser(t, db, "vader", "Vader")
	gameID := dbtest.SeedGame(t, db, "luke", "vader", 1)
	return db, gameID
}

func joinBoth(h *Hub) {
	h.ensurePlayer(Player{UID: "luke", Name: "Luke"})
	h.ensurePlayer(Player{UID: "vader", Name: "Vader"})
}

func TestMatchWinCreatesProposal(t *testing.T) {
	db, gameID := setupGame(t)
	h := CreateHub(gameID, db)
	joinBoth(h)

	for i := 0; i < 11; i++ {
		h.applyScoreUpdate("luke", 1)
	}

	g, err := mdb.GetGame(db, gameID)
	if err != nil {
		t.Fatalf("get game: %v", err)
	}
	if g.State != "awaiting_confirmation" || deref(g.ProposedBy) != "luke" || deref(g.WinnerID) != "luke" || deref(g.Score) != "11-0" {
		t.Fatalf("game = %s proposedBy=%q winner=%q score=%q", g.State, deref(g.ProposedBy), deref(g.WinnerID), deref(g.Score))
	}
	if s := h.snapshot().Status; s != "awaiting_confirmation" {
		t.Fatalf("hub status = %q", s)
	}
	var mmr int
	if err := db.QueryRow(`SELECT "mmr" FROM "user_stats" WHERE "userId"='luke'`).Scan(&mmr); err != nil || mmr != 1500 {
		t.Fatalf("luke mmr = %d, err %v", mmr, err)
	}
}

func TestHubLoadsWaitingProposal(t *testing.T) {
	db, gameID := setupGame(t)
	if err := mdb.ProposeResult(db, gameID, "vader", "vader", ""); err != nil {
		t.Fatalf("propose: %v", err)
	}

	h := CreateHub(gameID, db)
	joinBoth(h)

	s := h.snapshot()
	if s.Status != "awaiting_confirmation" || s.ProposedBy != "vader" || s.Meta["winner"] != "vader" {
		t.Fatalf("snapshot = status %q proposedBy %q winner %q", s.Status, s.ProposedBy, s.Meta["winner"])
	}
	h.applyScoreUpdate("luke", 1)
	if got := h.snapshot().Players["luke"].Score; got != 0 {
		t.Fatalf("luke score = %d, want 0", got)
	}
}

func TestRefreshAfterRejectRestartsScore(t *testing.T) {
	db, gameID := setupGame(t)
	h := CreateHub(gameID, db)
	joinBoth(h)
	for i := 0; i < 5; i++ {
		h.applyScoreUpdate("luke", 1)
	}
	if err := mdb.ProposeResult(db, gameID, "luke", "luke", ""); err != nil {
		t.Fatalf("propose: %v", err)
	}
	if err := mdb.RejectResult(db, gameID, "vader"); err != nil {
		t.Fatalf("reject: %v", err)
	}

	RefreshHub(db, gameID)

	s := h.snapshot()
	if s.Status != "ongoing" || len(s.Sets) != 0 {
		t.Fatalf("snapshot = status %q sets %v", s.Status, s.Sets)
	}
	for uid, p := range s.Players {
		if p.Score != 0 || p.SetsWon != 0 {
			t.Fatalf("%s score %d setsWon %d, want 0", uid, p.Score, p.SetsWon)
		}
	}
}
