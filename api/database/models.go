package database

import (
	"math"
	"time"
)

type Challenge struct {
	ID         string    `json:"id"`
	Challenger string    `json:"challenger"`
	Challengee string    `json:"challengee"`
	State      string    `json:"state"`
	CreatedBy  string    `json:"createdBy"`
	CreatedAt  time.Time `json:"createdAt"`
	GameID     *string   `json:"gameId,omitempty"`
}
type UserV2 struct {
	ID            string    `json:"id"`
	Name          string    `json:"name,omitempty"`
	Email         string    `json:"email,omitempty"`
	EmailVerified bool      `json:"emailVerified,omitempty"`
	Image         string    `json:"image,omitempty"`
	CreatedAt     time.Time `json:"createdAt,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt,omitempty"`
}

type UserStats struct {
	UserID    string    `json:"userId"`
	MMR       int       `json:"mmr"`
	Wins      int       `json:"wins"`
	Losses    int       `json:"losses"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type LeaderboardEntry struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	MMR    int    `json:"mmr"`
	Wins   int    `json:"wins"`
	Losses int    `json:"losses"`
}

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Player struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	MMR  int    `json:"mmr"`
	Team string `json:"team,omitempty"`
}

//TODO : look into how to handle doubles , perhaps limit elo rating gain to top
// rated player in each doubles team

func CalcWinProbs(p1MMR, p2MMR int) (float64, float64) {
	scaleF := 400
	sigmaR := p1MMR - p2MMR

	exponent := float64(-sigmaR) / float64(scaleF)
	expectedP1 := 1 / (1 + math.Pow(10, exponent))

	return expectedP1, 1 - expectedP1
}

// p1win enough for now
// https://en.wikipedia.org/wiki/Elo_rating_system#Formal_derivation_for_win/loss_games
func GetRatingChange(p1MMR, p2MMR int, p1Win bool) (int, int) {
	expectedP1, expectedP2 := CalcWinProbs(p1MMR, p2MMR)

	// TODO : look into reasonable kfactor, perhaps parameterize it
	kFactor := 32

	sa := 1.0
	sb := 0.0
	if !p1Win {
		sa = 0.0
		sb = 1.0
	}
	delta1 := int(math.Round(float64(kFactor) * (sa - expectedP1)))
	delta2 := int(math.Round(float64(kFactor) * (sb - expectedP2)))

	return delta1, delta2
}
