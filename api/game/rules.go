package game

const (
	setPoints = 11 // points needed to win a set
	setLead   = 2  // ...with at least this lead (deuce at 10-10)
)

// SetWinner returns 1 if a won the set, 2 if b won, 0 if the set is still going.
func SetWinner(a, b int) int {
	switch {
	case a >= setPoints && a-b >= setLead:
		return 1
	case b >= setPoints && b-a >= setLead:
		return 2
	}
	return 0
}

// SetsToWin is the number of sets needed to win a best-of-N match.
func SetsToWin(bestOf int) int {
	return bestOf/2 + 1
}

// ValidBestOf reports whether n is an allowed match format.
func ValidBestOf(n int) bool {
	return n == 1 || n == 3 || n == 5 || n == 7
}
