package homeworlds

import (
	"os"
	"testing"
)

var (
	game = &Game{
		NumPlayers:    2,
		CurrentPlayer: North,
		Bank: map[Piece]int{
			R3: 3, R2: 3, R1: 3,
			G3: 0, G2: 0, G1: 0,
			B3: 2, B2: 2, B1: 2,
			Y3: 1, Y2: 2, Y1: 1,
		},
		Homeworlds: map[Player]*Star{
			North: north,
			South: south,
		},
		Stars: map[string]*Star{
			"North [BUG]": north,
			"South [BUG]": south,
			"grover":      grover,
			"orion":       orion,
			"virgo":       virgo,
		},
	}

	north = &Star{
		Name:        "North [BUG]",
		IsHomeworld: true,
		Pieces:      []Piece{Y3, B1},
		Ships:       map[Player][]Piece{North: {G3, G2}},
	}

	south = &Star{
		Name:        "South [BUG]",
		IsHomeworld: true,
		Pieces:      []Piece{B2, Y1},
		Ships:       map[Player][]Piece{South: {G3, G1}},
	}

	grover = &Star{
		Name:   "grover",
		Pieces: []Piece{B3},
		Ships:  map[Player][]Piece{South: {Y2, G1, G2, G2}},
	}

	orion = &Star{
		Name:   "Orion",
		Pieces: []Piece{Y3},
		Ships:  map[Player][]Piece{North: {G1}},
	}

	virgo = &Star{
		Name:   "Virgo",
		Pieces: []Piece{G3},
		Ships:  map[Player][]Piece{North: {Y1}},
	}
)

func ExamplePrint() {
	Print(os.Stdout, game)
/* Output:
The Bank:
  Red: 3 large, 3 medium, 3 small.
  Green: none.
  Blue: 2 large, 2 medium, 2 small.
  Yellow: 1 large, 2 medium, 1 small.

Systems:
  North's homeworld, a Y3/B1 star.
  South's homeworld, a B2/Y1 star.
  grover, a B3 star.
  Orion, a Y3 star.
  Virgo, a G3 star.

North's fleet:
  At their homeworld: a large green and a medium green ship.
  At Orion: a small green ship.
  At Virgo: a small yellow ship.

South's fleet:
  At their homeworld: a large green and a small green ship.
  At grover: two medium green ships, a medium yellow, and a small green ship.

It is North's turn.
*/
}

func TestFmtShips(T *testing.T) {
	tests := []struct {
		p    []Piece
		want string
	}{
		{[]Piece{G3}, "a large green ship"},
		{[]Piece{G1, G1}, "two small green ships"},
		{[]Piece{R1, Y1, B1}, "a small red, a small blue, and a small yellow ship"},
		{[]Piece{G3, R1, R1}, "a large green ship and two small red ships"},
	}
	for _, t := range tests {
		got := fmtShips(t.p)
		if got != t.want {
			T.Errorf("%v: got %q, want %q", t.p, got, t.want)
		}
	}
}
