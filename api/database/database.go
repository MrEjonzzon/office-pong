package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

//TODO : split the different domains in to separate files

/* USERS */
func GetUsers(db *sql.DB) ([]UserV2, error) {
	rows, err := db.Query(`SELECT "id", "name", COALESCE("image", '') FROM "user"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserV2
	for rows.Next() {
		var u UserV2
		err := rows.Scan(&u.ID, &u.Name, &u.Image)
		if err != nil {
			log.Println(err)
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

func GetUserByID(db *sql.DB, id string) (*UserV2, error) {
	var u UserV2
	err := db.QueryRow(`SELECT "id", "name", COALESCE("image", '') FROM "user" WHERE "id" = $1`, id).Scan(&u.ID, &u.Name, &u.Image)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

/* CHALLENGES  */

func GetChallenges(db *sql.DB, id string) ([]Challenge, error) {
	//TODO: allow for querying maybe in the future
	query := `
        SELECT c."id", c."challenger", c."challengee", c."state", c."createdBy", c."createdAt", g."id"
        FROM "challenges" c
        LEFT JOIN "games" g ON g."challengeId" = c."id"
        WHERE (c.challenger = $1 OR c.challengee = $1)
    `

	rows, err := db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var challenges []Challenge
	for rows.Next() {
		var c Challenge
		if err := rows.Scan(&c.ID, &c.Challenger, &c.Challengee, &c.State, &c.CreatedBy, &c.CreatedAt, &c.GameID); err != nil {
			log.Println("scan error:", err)
			continue
		}
		challenges = append(challenges, c)
	}
	return challenges, nil
}

func CreateChallenge(db *sql.DB, challenger, challengee string) (string, error) {
	//check challenge and challengee don't already have a challenge
	var challengeExists bool
	//TODO : the state === "active" case should probably be tightened
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM "challenges" WHERE "challenger" = $1 AND "challengee" = $2 AND "state" = 'active')`,
		challenger, challengee).Scan(&challengeExists)
	if err != nil {
		return "", fmt.Errorf("error checking challenge existence: %v", err)
	}

	if challengeExists {
		return "", fmt.Errorf("challenge already exists between %s and %s", challenger, challengee)
	}

	// check challenger exists
	var challengerExists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM "user" WHERE "id" = $1)`, challenger).Scan(&challengerExists)
	if err != nil {
		return "", fmt.Errorf("error checking challenger: %v", err)
	}
	if !challengerExists {
		return "", fmt.Errorf("challenger with ID %s does not exist", challenger)
	}

	// check challengee exists
	var challengeeExists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM "user" WHERE "id" = $1)`, challengee).Scan(&challengeeExists)
	if err != nil {
		return "", fmt.Errorf("error checking challengee: %v", err)
	}
	if !challengeeExists {
		return "", fmt.Errorf("challengee with ID %s does not exist", challengee)
	}

	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	// create challenge
	challengeID := uuid.New().String()
	query := `
        INSERT INTO "challenges" ("id", "state", "createdBy", "challenger", "challengee", "createdAt")
        VALUES ($1, 'pending', $2, $2, $3, NOW())
    `
	_, err = tx.Exec(query, challengeID, challenger, challengee)
	if err != nil {
		return "", fmt.Errorf("error creating challenge: %v", err)
	}

	return challengeID, tx.Commit()
}

func AcceptChallenge(db *sql.DB, challengeID string) (string, error) {
	// Auth token check

	// Check if the challenge exists and is in 'pending' state
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM "challenges" WHERE "id" = $1)`, challengeID).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("error checking challenge existence: %v", err)
	}
	if !exists {
		return "", fmt.Errorf("challenge with ID %s does not exist", challengeID)
	}

	// Internal "create game" function
	gameID, err := createGame(db, challengeID)

	// Update the challenge state to 'active'
	query := `
		UPDATE "challenges"
		SET "state" = 'active'
		WHERE "id" = $1;
	`
	result, err := db.Exec(query, challengeID)
	if err != nil {
		return "", fmt.Errorf("error accepting challenge: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("error getting rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return "", fmt.Errorf("no challenge was updated, check if you are the challenger or challengee")
	}

	return gameID, nil
}

func createGame(db *sql.DB, challengeID string) (string, error) {
	// Create a new game associated with the challenge
	gameID := uuid.New().String()

	// Succeed with websocket setup
	query := `
		INSERT INTO "games" ("id", "challengeId", "state", "createdAt")
		VALUES ($1, $2, 'ongoing', NOW());
	`
	_, err := db.Exec(query, gameID, challengeID)
	if err != nil {
		return "", fmt.Errorf("error creating game: %v", err)
	}
	return gameID, nil
}

func DeclineChallenge(db *sql.DB, challengeID string) error {

	// Check if the challenge exists and is in 'pending' state
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM "challenges" WHERE "id" = $1)`, challengeID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking challenge existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("challenge with ID %s does not exist", challengeID)
	}

	// Set challenge state to 'denied'
	query := `
		UPDATE "challenges"
		SET "state" = 'denied'
		WHERE "id" = $1;
	`
	result, err := db.Exec(query, challengeID)
	if err != nil {
		return fmt.Errorf("error declining challenge: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no challenge was updated, check if you are the challenger or challengee")
	}

	return nil
}

/* USER STATS (mmr etc) */

func GetLeaderboard(db *sql.DB) ([]LeaderboardEntry, error) {
	rows, err := db.Query(`
		SELECT u."id", u."name", COALESCE(u."image", ''), s."mmr", s."wins", s."losses"
		FROM "user_stats" s
		JOIN "user" u ON u."id" = s."userId"
		ORDER BY s."mmr" DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.Name, &e.Image, &e.MMR, &e.Wins, &e.Losses); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func EnsureUserStats(db *sql.DB, userID string) error {
	_, err := db.Exec(`
        INSERT INTO "user_stats" ("userId", "mmr", "wins", "losses", "updatedAt")
        VALUES ($1, 1500, 0, 0, NOW())
        ON CONFLICT ("userId") DO NOTHING
    `, userID)
	return err
}

func GetUserStats(db *sql.DB, userID string) (*UserStats, error) {
	if err := EnsureUserStats(db, userID); err != nil {
		return nil, fmt.Errorf("error ensuring user stats: %v", err)
	}

	var stats UserStats
	err := db.QueryRow(`
        SELECT "userId", "mmr", "wins", "losses", "updatedAt"
        FROM "user_stats"
        WHERE "userId" = $1
    `, userID).Scan(&stats.UserID, &stats.MMR, &stats.Wins, &stats.Losses, &stats.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error getting user stats: %v", err)
	}
	return &stats, nil
}

func GetUserMMR(db *sql.DB, userID string) (int, error) {
	var mmr int
	err := db.QueryRow(`SELECT "mmr" FROM "user_stats" WHERE "userId" = $1`, userID).Scan(&mmr)
	if err == sql.ErrNoRows {
		// Auto-create with default
		if err := EnsureUserStats(db, userID); err != nil {
			return 0, err
		}
		return 1500, nil
	}
	return mmr, err
}

func UpdateMMRAfterGame(db *sql.DB, winnerID, loserID string) (int, int, error) {
	winnerMMR, err := GetUserMMR(db, winnerID)
	if err != nil {
		return 0, 0, fmt.Errorf("error getting winner MMR: %v", err)
	}
	loserMMR, err := GetUserMMR(db, loserID)
	if err != nil {
		return 0, 0, fmt.Errorf("error getting loser MMR: %v", err)
	}

	winnerDelta, loserDelta := GetRatingChange(winnerMMR, loserMMR, true)

	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
        UPDATE "user_stats" SET "mmr" = "mmr" + $1, "wins" = "wins" + 1, "updatedAt" = NOW()
        WHERE "userId" = $2
    `, winnerDelta, winnerID)
	if err != nil {
		return 0, 0, err
	}

	_, err = tx.Exec(`
        UPDATE "user_stats" SET "mmr" = "mmr" + $1, "losses" = "losses" + 1, "updatedAt" = NOW()
        WHERE "userId" = $2
    `, loserDelta, loserID)
	if err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return winnerMMR + winnerDelta, loserMMR + loserDelta, nil
}

func CompleteGame(db *sql.DB, gameID, winnerID string) error {
	_, err := db.Exec(`UPDATE "games" SET "state" = 'completed' WHERE "id" = $1`, gameID)
	if err != nil {
		return fmt.Errorf("error completing game: %v", err)
	}

	_, err = db.Exec(`
        UPDATE "challenges" SET "state" = 'complete'
        WHERE "id" = (SELECT "challengeId" FROM "games" WHERE "id" = $1)
    `, gameID)
	return err
}

/* AUTH */

func VerifyUserToken(db *sql.DB, token string) (string, error) {
	var userID string
	var expiresAt sql.NullTime

	query := `
		SELECT "userId", "expiresAt"
		FROM session
		WHERE session."token" = $1
	`

	err := db.QueryRow(query, token).Scan(&userID, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("invalid token")
		}
		return "", err
	}

	// Check if the token has expired
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return "", fmt.Errorf("token expired")
	}

	return userID, nil
}
