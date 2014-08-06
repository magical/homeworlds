package homeworlds

import (
	"fmt"
	"io"
	"strings"
)

// Print formats a game as text.
//
// The output looks something like this:
//
//     The bank:
//       Red: 3 large, 3 medium, 3 small.
//       Green: none.
//       Blue: 2 large, 2 medium, 2 small.
//       Yellow: 1 large, 2 medium, 1 small.
//
//     Systems:
//       North's homeworld, a Y3/B2 star.
//       South's homeworld, a B2/Y1 star.
//       grover, a B3 star.
//       Orion, a Y3 star.
//       Virgo, a G3 star.
//
//     North's fleet:
//       At their homeworld: a large green and a medium green ship.
//       At Orion: a small green ship.
//       At Virgo: a small yellow ship.
//
//     South's fleet:
//       At their homeworld: a large green and a small green ship.
//       At grover: a large yellow ship, two medium green ships, and a small green ship.
//
//     It is North's turn.
//
func Print(w io.Writer, g *Game) error {
	// Print the contents of the bank.
	fmt.Fprintln(w, "The Bank:")
	fmtBank(w, g, Red)
	fmtBank(w, g, Green)
	fmtBank(w, g, Blue)
	fmtBank(w, g, Yellow)
	fmt.Fprintln(w)

	// Print the current systems.
	fmt.Fprintln(w, "Systems:")
	for i := 0; i < g.NumPlayers; i++ {
		pl := Player(i)
		s := g.Homeworlds[pl]
		fmt.Fprintf(w, "  %s's homeworld, a %s star.\n", pl, fmtStar(s.Pieces))
	}
	// BUG: Stars is a map, so this prints the stars in a random order.
	for _, s := range g.Stars {
		if s.IsHomeworld {
			continue
		}
		fmt.Fprintf(w, "  %s, a %s star.\n", s.Name, fmtStar(s.Pieces))
	}
	fmt.Fprintln(w)

	// Print the ships in each player's fleet.
	for i := 0; i < g.NumPlayers; i++ {
		pl := Player(i)
		fmt.Fprintf(w, "%s's fleet:\n", pl)
		for h, s := range g.Homeworlds {
			if len(s.Ships[pl]) == 0 {
				continue
			}
			if pl == h {
				fmt.Fprint(w, "  At their homeworld: ")
			} else {
				fmt.Fprintf(w, "  At %s's homeworld: ", h)
			}
			fmt.Fprint(w, fmtShips(s.Ships[pl]), ".\n")
		}
		for _, s := range g.Stars {
			if s.IsHomeworld {
				continue
			}
			if len(s.Ships[pl]) == 0 {
				continue
			}
			fmt.Fprintf(w, "  At %s: %s.\n", s.Name, fmtShips(s.Ships[pl]))
		}
		fmt.Fprintln(w)
	}

	// Print whose turn it is.
	fmt.Fprintf(w, "It is %s's turn.\n", g.CurrentPlayer)

	return nil
}

func fmtBank(w io.Writer, g *Game, c Color) {
	some := false
	fmt.Fprint(w, "  ", c, ": ")
	for size := Large; size >= Small; size-- {
		n := g.Bank[piece(size, c)]
		if n > 0 {
			if some {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprint(w, n, " ", strings.ToLower(size.String()))
			some = true
		}
	}
	if !some {
		fmt.Fprint(w, "none")
	}
	fmt.Fprint(w, ".\n")
}

func (c Color) String() string {
	switch c {
	case Red:
		return "Red"
	case Blue:
		return "Blue"
	case Green:
		return "Green"
	case Yellow:
		return "Yellow"
	default:
		return "Unknown color [BUG]"
	}
}

func (s Size) String() string {
	switch s {
	case Small:
		return "Small"
	case Medium:
		return "Medium"
	case Large:
		return "Large"
	default:
		return "Unknown size [BUG]"
	}
}

func piece(s Size, c Color) Piece {
	return Piece(c*3) + Piece(s-1)
}

func (pl Player) String() string {
	switch pl {
	case North:
		return "North"
	case South:
		return "South"
	default:
		return "Unknown player [BUG]"
	}
}

func fmtStar(p []Piece) string {
	if len(p) == 0 {
		return "BUG"
	}
	if len(p) == 1 {
		return p[0].String()
	}
	var s []string
	for _, p := range p {
		s = append(s, p.String())
	}
	return strings.Join(s, "/")
}

var pieceNamesLong = [...]string{
	R1: "small red",
	R2: "medium red",
	R3: "large red",

	Y1: "small yellow",
	Y2: "medium yellow",
	Y3: "large yellow",

	G1: "small green",
	G2: "medium green",
	G3: "large green",

	B1: "small blue",
	B2: "medium blue",
	B3: "large blue",
}

var pieceNamesShort = [...]string{
	R1: "R1",
	R2: "R2",
	R3: "R3",

	Y1: "Y1",
	Y2: "Y2",
	Y3: "Y3",

	G1: "G1",
	G2: "G2",
	G3: "G3",

	B1: "B1",
	B2: "B2",
	B3: "B3",
}

func (p Piece) String() string {
	return pieceNamesShort[p]
}

func (p Piece) name() string {
	return pieceNamesLong[p]
}

// fmtShips formats a list of pieces as natural text.
//
// Examples:
//    a large green ship
//    two small green ships
//    a small red, a small yellow, and a small blue ship
//    a large green ship and two small red ships
func fmtShips(ships []Piece) string {
	// TODO: This function is terrible. Sorry.
	if len(ships) == 0 {
		return "no ships [BUG]"
	}
	// Count how many of each piece we have.
	var num [12]int
	for _, p := range ships {
		num[p]++
	}
	// Make a new list in a fixed order (large to small, RGBY)
	// with identical pieces combined.
	type S struct {
		P Piece
		N int
	}
	var s []S
	for size := Large; size >= Small; size-- {
		for _, c := range []Color{Red, Green, Blue, Yellow} {
			p := piece(size, c)
			if num[p] > 0 {
				s = append(s, S{p, num[p]})
			}
		}
	}
	// Format each part as text
	// adding "ship" or "ships" where appropriate.
	var parts []string
	for i := range s {
		name := s[i].P.name()
		switch s[i].N {
		case 1:
			name = "a " + name
		case 2:
			name = "two " + name
		case 3:
			name = "three " + name
		default:
			name = "some " + name + " [BUG]"
		}
		if i == len(s)-1 || s[i].N != s[i+1].N {
			if s[i].N == 1 {
				name += " ship"
			} else {
				name += " ships"
			}
		}
		parts = append(parts, name)
	}
	// Join all the parts together
	return joinComma(parts)
}

func joinComma(s []string) string {
	switch len(s) {
	case 0:
		return ""
	case 1:
		return s[0]
	case 2:
		return s[0] + " and " + s[1]
	default:
		return strings.Join(s[:len(s)-1], ", ") + ", and " + s[len(s)-1]
	}
}
