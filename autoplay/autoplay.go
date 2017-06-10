package main

import (
	"fmt"
	randpkg "math/rand"
	"os"

	"github.com/magical/homeworlds"
)

var systemid = 2

func main() {
	g := newGame()
	rand := randpkg.New(randpkg.NewSource(1))
	turn := 1
	for !g.IsOver() {
		fmt.Println("Turn number", turn)
		homeworlds.Print(os.Stdout, g)
		actions := g.BasicActions()
	retry:
		n := rand.Intn(len(actions))
		a := actions[n]
		fmt.Println(a)
		tmp := g.Copy()
		err := do(g, a)
		if err != nil {
			fmt.Println(err)
			break
		}
		catastrophe(g)
		if g.IsOver() && g.Winner() != g.CurrentPlayer {
			g = tmp
			goto retry
		}
		g.CurrentPlayer = homeworlds.Player((int(g.CurrentPlayer) + 1) % g.NumPlayers)
		turn++
	}
	if g.IsOver() {
		fmt.Println("Winner:", g.Winner())
	}
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
	stars := g.SortedStars()
	if a.System() >= len(stars) {
		return fmt.Errorf("no such system %d", a.System())
	}
	star := g.Stars[stars[a.System()]]
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
		toStar := g.Stars[stars[a.ToSystem()]]
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