package main

import (
	"fmt"
	"os"

	"github.com/magical/homeworlds"
)

var systemid = 2

func main() {
	g := newGame()
	ai := homeworlds.NewAI()
	turn := 1
	var last homeworlds.Action
	for !g.IsOver() {
		fmt.Println("\nTurn number", turn)
		homeworlds.Print(os.Stdout, g)
		//actions := g.BasicActions()
		//n := rand.Intn(len(actions))
		//a := actions[n]
		pos := homeworlds.PositionFromGame(g)
		a, v := ai.Minimax(pos, last.Basic())
		fmt.Println("Action:", a.Basic(), "Score:", v)
		do(g, a)
		last = a
		turn++
	}
	if g.IsOver() {
		fmt.Println("Winner:", g.Winner())
	}
}

func starMap(g *homeworlds.Game) []*homeworlds.Star {
	stars := g.SortedStars()
	m := make([]*homeworlds.Star, 2, len(g.Stars))
	m[0] = g.Homeworlds[homeworlds.North]
	m[1] = g.Homeworlds[homeworlds.South]
	for _, name := range stars {
		s := g.Stars[name]
		if s.IsHomeworld {
			continue
		}
		m = append(m, s)
	}
	return m
}

func newGame() *homeworlds.Game {
	north := &homeworlds.Star{
		Name:        "north",
		IsHomeworld: true,
		Pieces:      []homeworlds.Piece{homeworlds.G3, homeworlds.Y1},
		Ships: map[homeworlds.Player][]homeworlds.Piece{
			homeworlds.North: {homeworlds.B3},
		},
	}
	south := &homeworlds.Star{
		Name:        "south",
		IsHomeworld: true,
		Pieces:      []homeworlds.Piece{homeworlds.Y3, homeworlds.B2},
		Ships: map[homeworlds.Player][]homeworlds.Piece{
			homeworlds.South: {homeworlds.G3},
		},
	}
	game := &homeworlds.Game{
		Phase:         0,
		NumPlayers:    2,
		CurrentPlayer: homeworlds.North,
		Bank:          make(map[homeworlds.Piece]int),
		Homeworlds: map[homeworlds.Player]*homeworlds.Star{
			homeworlds.North: north,
			homeworlds.South: south,
		},
		Stars: map[string]*homeworlds.Star{
			"north": north,
			"south": south,
		},
	}
	game.ResetBank()
	return game
}

func do(g *homeworlds.Game, a homeworlds.Action) error {
	err := doBasic(g, a.Basic())
	if err != nil {
		return err
	}
	catastrophe(g)

	if a.Type() == homeworlds.Sacrifice {
		for i := 0; i < a.N(); i++ {
			err := doBasic(g, a.Action(i))
			if err != nil {
				return err
			}
			catastrophe(g)
		}
	}

	g.EndTurn()
	return nil
}

func doBasic(g *homeworlds.Game, a homeworlds.BasicAction) error {
	stars := starMap(g)
	if a.System() >= len(stars) {
		return fmt.Errorf("no such system %d", a.System())
	}
	star := stars[a.System()]
	switch a.Type() {
	case homeworlds.Pass:
		return nil
	case homeworlds.Build:
		return g.Build(a.Ship(), star)
	case homeworlds.Trade:
		return g.Trade(a.Ship(), star, a.NewShip())
	case homeworlds.Move:
		if a.ToSystem() >= len(stars) {
			return fmt.Errorf("no such system %d", a.ToSystem())
		}
		toStar := stars[a.ToSystem()]
		return g.Move(a.Ship(), star, toStar)
	case homeworlds.Attack:
		target := homeworlds.North
		if g.CurrentPlayer == homeworlds.North {
			target = homeworlds.South
		}
		return g.Attack(a.Ship(), star, target)
	case homeworlds.Discover:
		name := fmt.Sprint(systemid)
		systemid++
		return g.Discover(a.Ship(), star, a.NewShip(), name)
	case homeworlds.Sacrifice:
		return g.Sacrifice(a.Ship(), star)
	}
	return nil
}

func catastrophe(g *homeworlds.Game) {
	for c := homeworlds.Color(0); c < homeworlds.Color(4); c++ {
		for _, s := range g.Stars {
			g.Catastrophe(c, s)
		}
	}
}
