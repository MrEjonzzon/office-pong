package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mdb "office-pong/database"
	"office-pong/dbtest"
	mw "office-pong/middleware"

	"github.com/gorilla/mux"
)

func TestConfirmHandlerUsesTokenUser(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.SeedUser(t, db, "luke", "Luke")
	dbtest.SeedUser(t, db, "vader", "Vader")
	gameID := dbtest.SeedGame(t, db, "luke", "vader", 1)
	if err := mdb.ProposeResult(db, gameID, "luke", "luke", ""); err != nil {
		t.Fatalf("propose: %v", err)
	}

	r := mux.NewRouter()
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(mw.AuthMiddleware(db))
	protected.HandleFunc("/v1/game/{id}/result/confirm", gameActionHandler(db, mdb.ConfirmResult)).Methods("POST")

	confirm := func(token string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/game/"+gameID+"/result/confirm", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := confirm("token-luke"); code != http.StatusConflict {
		t.Fatalf("luke confirm = %d, want 409", code)
	}
	if code := confirm("token-vader"); code != http.StatusOK {
		t.Fatalf("vader confirm = %d, want 200", code)
	}
	g, err := mdb.GetGame(db, gameID)
	if err != nil || g.State != "completed" {
		t.Fatalf("game = %+v, err %v", g, err)
	}
}
