package game

import "testing"

func TestSetWinner(t *testing.T) {
	cases := []struct{ a, b, want int }{
		{11, 9, 1}, {11, 10, 0}, {12, 10, 1}, {10, 12, 2},
		{15, 13, 1}, {10, 10, 0}, {0, 0, 0}, {9, 11, 2}, {10, 11, 0}, {11, 0, 1},
	}
	for _, c := range cases {
		if got := SetWinner(c.a, c.b); got != c.want {
			t.Errorf("SetWinner(%d,%d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestSetsToWin(t *testing.T) {
	for bestOf, want := range map[int]int{1: 1, 3: 2, 5: 3, 7: 4} {
		if got := SetsToWin(bestOf); got != want {
			t.Errorf("SetsToWin(%d) = %d, want %d", bestOf, got, want)
		}
	}
}
