package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/namsral/flag"

	mdb "office-pong/database"
	mw "office-pong/middleware"

	gamews "office-pong/game"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq" // added import for postgres driver
)

const (
	defaultPort = "8080"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	var (
		port       = flag.String("p", defaultPort, "Server listen port")
		dbHost     = flag.String("dbhost", "localhost", "Postgres host")
		dbPort     = flag.String("dbport", "5432", "Postgres port")
		dbName     = flag.String("dbname", "officepong", "Postgres database name")
		dbUser     = flag.String("dbuser", "postgres", "Postgres user name")
		dbPassword = flag.String("dbpass", "", "Postgres password")
		dbSSLMode  = flag.String("dbsslmode", "", "Postgres sslmode (default: disable for localhost, require otherwise)")
	)
	flag.Parse()

	sslMode := *dbSSLMode
	if sslMode == "" {
		sslMode = "require"
		if *dbHost == "localhost" || *dbHost == "127.0.0.1" {
			sslMode = "disable"
		}
	}
	postgresConnStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", *dbUser, url.QueryEscape(*dbPassword), *dbHost, *dbPort, *dbName, sslMode)
	fmt.Printf("Connecting to Postgres at %s:%s/%s (sslmode=%s)\n", *dbHost, *dbPort, *dbName, sslMode)

	connStr := postgresConnStr
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("error opening connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}

	//TODO: look over this, might be fine if we're doing websocket auth via param instead of header
	// only applies middleware to protected routes, and not the websocket who handles its own auth
	r := mux.NewRouter()
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(mw.AuthMiddleware((db)))

	// better-auth tables (same as web/better-auth_migrations), so a fresh DB works without `npm run migrate`
	authUser := `CREATE TABLE IF NOT EXISTS "user" ("id" text not null primary key, "name" text not null, "email" text not null unique, "emailVerified" boolean not null, "image" text, "createdAt" timestamp not null, "updatedAt" timestamp not null);`
	authSession := `CREATE TABLE IF NOT EXISTS "session" ("id" text not null primary key, "expiresAt" timestamp not null, "token" text not null unique, "createdAt" timestamp not null, "updatedAt" timestamp not null, "ipAddress" text, "userAgent" text, "userId" text not null references "user" ("id"));`
	authAccount := `CREATE TABLE IF NOT EXISTS "account" ("id" text not null primary key, "accountId" text not null, "providerId" text not null, "userId" text not null references "user" ("id"), "accessToken" text, "refreshToken" text, "idToken" text, "accessTokenExpiresAt" timestamp, "refreshTokenExpiresAt" timestamp, "scope" text, "password" text, "createdAt" timestamp not null, "updatedAt" timestamp not null);`
	authVerification := `CREATE TABLE IF NOT EXISTS "verification" ("id" text not null primary key, "identifier" text not null, "value" text not null, "expiresAt" timestamp not null, "createdAt" timestamp, "updatedAt" timestamp);`

	enumCheck := `DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'challenge_state') THEN
        CREATE TYPE challenge_state AS ENUM ('pending', 'active', 'denied', 'complete');
    END IF;
END$$;`

	challenges := `CREATE TABLE IF NOT EXISTS "challenges" (
    "id" TEXT NOT NULL PRIMARY KEY, 
		"challenger" TEXT NOT NULL REFERENCES "user" ("id"),
		"challengee" TEXT NOT NULL REFERENCES "user" ("id"),
    "state" challenge_state NOT NULL, 
    "createdBy" TEXT NOT NULL REFERENCES "user" ("id"),
    "createdAt" TIMESTAMP NOT NULL
);`

	games := `CREATE TABLE IF NOT EXISTS "games" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "challengeId" TEXT NOT NULL REFERENCES "challenges" ("id"),
    "state" TEXT NOT NULL,
    "createdAt" TIMESTAMP NOT NULL
);`

	userStats := `CREATE TABLE IF NOT EXISTS "user_stats"(
		"userId" TEXT NOT NULL PRIMARY KEY REFERENCES "user" ("id"),
		"mmr" INTEGER NOT NULL DEFAULT 1500,
		"wins" INTEGER NOT NULL DEFAULT 0,
		"losses" INTEGER NOT NULL DEFAULT 0,
		"updatedAt" TIMESTAMP NOT NULL DEFAULT NOW()
	)`

	for _, query := range []string{authUser, authSession, authAccount, authVerification, enumCheck, challenges, games, userStats} {
		_, err := db.Exec(query)
		if err != nil {
			log.Fatalf("error creating table: %v", err)
		}
	}

	protected.HandleFunc("/v1/users", getUsersHandler(db)).Methods("GET")
	protected.HandleFunc("/v1/user/{id}/challenges", getChallengesHandler(db)).Methods("GET")
	protected.HandleFunc("/v1/user/{id}/challenge/{id2}", sendChallengeHandler(db)).Methods("POST")

	protected.HandleFunc("/v1/challenge/{id}/accept", acceptChallengeHandler(db)).Methods("POST")
	protected.HandleFunc("/v1/challenge/{id}/decline", declineChallengeHandler(db)).Methods("POST")

	protected.HandleFunc("/v1/user/{id}/stats", getUserStatsHandler(db)).Methods("GET")
	protected.HandleFunc("/v1/leaderboard", getLeaderboardHandler(db)).Methods("GET")

	// Game-specific websocket endpoint, clients join by game id
	r.HandleFunc("/api/v1/game/{id}/ws", gamews.WSHandler(db)).Methods("GET")

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodOptions,
	})

	fmt.Printf("Server running on port http://localhost:%s \n", *port)

	http.Handle("/", r)

	http.ListenAndServe(":"+*port, handlers.CORS(originsOk, headersOk, methodsOk)(r))
}

/* User */
type GetUsersResponseV1 struct {
	Data  []mdb.UserV2 `json:"data"`
	Error error        `json:"error,omitempty"`
}

func (r GetUsersResponseV1) error() error {
	return r.Error
}

func getUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := mdb.GetUsers(db)
		encodeResponse(w, GetUsersResponseV1{Data: users, Error: err}, http.StatusOK)
	}
}

/* Challenges */
type GetChallengesResponse struct {
	Data  []mdb.Challenge `json:"data"`
	Error error           `json:"error,omitempty"`
}

func (r GetChallengesResponse) error() error {
	return r.Error
}

func getChallengesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]

		if id == "" {
			encodeError(fmt.Errorf("invalid request"), w, http.StatusBadRequest)
			return
		}

		challenges, err := mdb.GetChallenges(db, id)
		encodeResponse(w, GetChallengesResponse{Data: challenges, Error: err}, http.StatusOK)
	}
}

type SendChallengeResponse struct {
	Data  string `json:"data"`
	Error error  `json:"error,omitempty"`
}

func (r SendChallengeResponse) error() error {
	return r.Error
}

func sendChallengeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		challenger := params["id"]
		challengee := params["id2"]
		if challenger == "" || challengee == "" {
			encodeError(fmt.Errorf("invalid request"), w, http.StatusBadRequest)
			return
		}

		id, err := mdb.CreateChallenge(db, challenger, challengee)
		encodeResponse(w, SendChallengeResponse{Data: id, Error: err}, http.StatusCreated)
	}
}

type AcceptChallengeResponseV1 struct {
	Data  string `json:"data"`
	Error error  `json:"error,omitempty"`
}

func acceptChallengeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		params := mux.Vars(r)
		id := params["id"]
		if id == "" {
			encodeError(fmt.Errorf("invalid request"), w, http.StatusBadRequest)
			return
		}

		gameID, err := mdb.AcceptChallenge(db, id)
		if err != nil {
			encodeError(err, w, http.StatusInternalServerError)
			return
		}

		// Create new game hub in memory so clients can connect immediately
		_ = gamews.CreateHub(gameID, db)

		// Respond with the game id
		encodeResponse(w, AcceptChallengeResponseV1{Data: gameID, Error: err}, http.StatusAccepted)
	}
}

func declineChallengeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		params := mux.Vars(r)
		id := params["id"]
		if id == "" {
			encodeError(fmt.Errorf("invalid request"), w, http.StatusBadRequest)
			return
		}

		err := mdb.DeclineChallenge(db, id)
		if err != nil {
			encodeError(err, w, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

type GetUserStatsResponse struct {
	Data  *mdb.UserStats `json:"data"`
	Error error          `json:"error,omitempty"`
}

func (r GetUserStatsResponse) error() error {
	return r.Error
}

func getUserStatsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		id := params["id"]
		if id == "" {
			encodeError(fmt.Errorf("invalid request"), w, http.StatusBadRequest)
			return
		}
		stats, err := mdb.GetUserStats(db, id)
		if err != nil {
			encodeError(err, w, http.StatusInternalServerError)
			return
		}
		encodeResponse(w, stats, http.StatusOK)
	}
}

type GetLeaderboardResponse struct {
	Data  []mdb.LeaderboardEntry `json:"data"`
	Error error                  `json:"error,omitempty"`
}

func (r GetLeaderboardResponse) error() error {
	return r.Error
}

func getLeaderboardHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := mdb.GetLeaderboard(db)
		if err != nil {
			encodeError(err, w, http.StatusInternalServerError)
			return
		}
		encodeResponse(w, GetLeaderboardResponse{Data: entries}, http.StatusOK)
	}
}

/* General Shit */
type errorer interface {
	error() error
}

func encodeError(err error, w http.ResponseWriter, status int) {
	fmt.Println("error : ", err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func encodeResponse(w http.ResponseWriter, resp interface{}, status int) error {
	if e, ok := resp.(errorer); ok && e.error() != nil {
		encodeError(e.error(), w, status)
		return nil
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(resp)
}
