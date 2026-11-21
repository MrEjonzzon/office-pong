package database

import (
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNotPlayer  = errors.New("user is not a player in this game")
	ErrWrongState = errors.New("game is not in the right state for this action")
	ErrBadWinner  = errors.New("winner is not a player in this game")
)

func GetGame(db *sql.DB, gameID string) (*Game, error) {
	var g Game
	err := db.QueryRow(`
        SELECT g."id", g."challengeId", g."state", c."challenger", c."challengee", c."bestOf",
               g."winnerId", g."score", g."proposedBy"
        FROM "games" g
        JOIN "challenges" c ON c."id" = g."challengeId"
        WHERE g."id" = $1
    `, gameID).Scan(&g.ID, &g.ChallengeID, &g.State, &g.Challenger, &g.Challengee, &g.BestOf,
		&g.WinnerID, &g.Score, &g.ProposedBy)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func playerGame(db *sql.DB, gameID, userID string) (*Game, error) {
	g, err := GetGame(db, gameID)
	if err != nil {
		return nil, err
	}
	if userID != g.Challenger && userID != g.Challengee {
		return nil, ErrNotPlayer
	}
	return g, nil
}

func ProposeResult(db *sql.DB, gameID, proposerID, winnerID, score string) error {
	g, err := playerGame(db, gameID, proposerID)
	if err != nil {
		return err
	}
	if winnerID != g.Challenger && winnerID != g.Challengee {
		return ErrBadWinner
	}
	res, err := db.Exec(`
        UPDATE "games" SET "state" = 'awaiting_confirmation', "winnerId" = $2, "score" = NULLIF($3, ''), "proposedBy" = $4
        WHERE "id" = $1 AND "state" = 'ongoing'
    `, gameID, winnerID, score, proposerID)
	return oneRow(res, err, ErrWrongState)
}

// ConfirmResult completes the game and updates both ratings. Only the player who did not propose can confirm.
func ConfirmResult(db *sql.DB, gameID, userID string) error {
	g, err := playerGame(db, gameID, userID)
	if err != nil {
		return err
	}
	for _, id := range []string{g.Challenger, g.Challengee} {
		if err := EnsureUserStats(db, id); err != nil {
			return err
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var winnerID string
	err = tx.QueryRow(`
        UPDATE "games" SET "state" = 'completed'
        WHERE "id" = $1 AND "state" = 'awaiting_confirmation' AND "proposedBy" <> $2
        RETURNING "winnerId"
    `, gameID, userID).Scan(&winnerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrWrongState
	}
	if err != nil {
		return err
	}

	loserID := g.Challenger
	if winnerID == g.Challenger {
		loserID = g.Challengee
	}
	var winnerMMR, loserMMR int
	if err := tx.QueryRow(`SELECT "mmr" FROM "user_stats" WHERE "userId" = $1`, winnerID).Scan(&winnerMMR); err != nil {
		return err
	}
	if err := tx.QueryRow(`SELECT "mmr" FROM "user_stats" WHERE "userId" = $1`, loserID).Scan(&loserMMR); err != nil {
		return err
	}
	winnerDelta, loserDelta := GetRatingChange(winnerMMR, loserMMR, true, KFactor(g.BestOf))

	if _, err := tx.Exec(`
        UPDATE "user_stats" SET "mmr" = "mmr" + $1, "wins" = "wins" + 1, "updatedAt" = NOW()
        WHERE "userId" = $2
    `, winnerDelta, winnerID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
        UPDATE "user_stats" SET "mmr" = "mmr" + $1, "losses" = "losses" + 1, "updatedAt" = NOW()
        WHERE "userId" = $2
    `, loserDelta, loserID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE "challenges" SET "state" = 'complete' WHERE "id" = $1`, g.ChallengeID); err != nil {
		return err
	}
	return tx.Commit()
}

func RejectResult(db *sql.DB, gameID, userID string) error {
	if _, err := playerGame(db, gameID, userID); err != nil {
		return err
	}
	res, err := db.Exec(`
        UPDATE "games" SET "state" = 'ongoing', "winnerId" = NULL, "score" = NULL, "proposedBy" = NULL
        WHERE "id" = $1 AND "state" = 'awaiting_confirmation' AND "proposedBy" <> $2
    `, gameID, userID)
	return oneRow(res, err, ErrWrongState)
}

// CancelGame cancels an unfinished game and frees the challenge so the players can challenge again.
func CancelGame(db *sql.DB, gameID, userID string) error {
	g, err := playerGame(db, gameID, userID)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
        UPDATE "games" SET "state" = 'cancelled', "winnerId" = NULL, "score" = NULL, "proposedBy" = NULL
        WHERE "id" = $1 AND "state" IN ('ongoing', 'awaiting_confirmation')
    `, gameID)
	if err := oneRow(res, err, ErrWrongState); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE "challenges" SET "state" = 'complete' WHERE "id" = $1`, g.ChallengeID); err != nil {
		return err
	}
	return tx.Commit()
}

func oneRow(res sql.Result, err error, none error) error {
	if err != nil {
		return fmt.Errorf("update game: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return none
	}
	return nil
}
