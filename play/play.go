package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/magical/homeworlds"
)

func main() {
	g := newGame()
	s := bufio.NewScanner(os.Stdin)
	for !g.IsOver() {
		io.WriteString(os.Stdout, "> ")
		if !s.Scan() {
			break
		}
		line := s.Text()
		a, err := parseAction(line)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(a)
		err = do(g, a)
		if err != nil {
			fmt.Println(err)
			continue
		}
		g.CurrentPlayer = homeworlds.Player((int(g.CurrentPlayer) + 1) % g.NumPlayers)
		homeworlds.Print(os.Stdout, g)
	}
	if s.Err() != nil {
		fmt.Println(s.Err())
	}
	if g.IsOver() {
		fmt.Println("Winner:", g.Winner())
	}
}

type Action struct {
	Type      homeworlds.ActionType
	Ship      homeworlds.Piece
	System    string
	NewShip   homeworlds.Piece
	NewSystem string
}

const Homeworld homeworlds.ActionType = 99

var parseError = errors.New("parse error")

func parseAction(s string) (Action, error) {
	parts := strings.Fields(s)
	//    Homeworld star1 star2 ship
	//    Discover ship fromSystem newStar newName
	//    Move ship fromSystem toSystem
	//    Build ship inSystem
	//    Trade oldShip newShip inSystem
	//    Attack ship inSystem
	//    Sacrifice ship inSystem
	//    Catastrophe color inSystem
	var a Action
	var err error
	switch {
	case len(parts) == 3 && parts[0] == "build":
		a.Type = homeworlds.Build
		a.System = parts[2]
		a.Ship, err = parseShip(parts[1])
		if err != nil {
			goto fail
		}
		return a, nil
	case len(parts) == 4 && parts[0] == "trade":
		a.Type = homeworlds.Trade
		a.System = parts[3]
		a.Ship, err = parseShip(parts[1])
		if err != nil {
			goto fail
		}
		a.NewShip, err = parseShip(parts[2])
		if err != nil {
			goto fail
		}
		return a, nil
	case len(parts) == 4 && parts[0] == "move":
		a.Type = homeworlds.Move
		a.System = parts[2]
		a.NewSystem = parts[3]
		a.Ship, err = parseShip(parts[1])
		if err != nil {
			goto fail
		}
		return a, nil
	case len(parts) == 3 && parts[0] == "attack":
		a.Type = homeworlds.Attack
		a.System = parts[2]
		a.Ship, err = parseShip(parts[1])
		if err != nil {
			goto fail
		}
		return a, nil
	case len(parts) == 5 && parts[0] == "discover":
		a.Type = homeworlds.Discover
		a.System = parts[2]
		a.NewSystem = parts[4]
		a.Ship, err = parseShip(parts[1])
		if err != nil {
			goto fail
		}
		a.NewShip, err = parseShip(parts[3])
		if err != nil {
			goto fail
		}
		return a, nil
	}

fail:
	return Action{}, parseError
}

func parseShip(s string) (homeworlds.Piece, error) {
	var pieces = map[string]homeworlds.Piece{
		"R1": homeworlds.R1, "R2": homeworlds.R2, "R3": homeworlds.R3,
		"Y1": homeworlds.Y1, "Y2": homeworlds.Y2, "Y3": homeworlds.Y3,
		"G1": homeworlds.G1, "G2": homeworlds.G2, "G3": homeworlds.G3,
		"B1": homeworlds.B1, "B2": homeworlds.B2, "B3": homeworlds.B3,
	}
	p, ok := pieces[s]
	if !ok {
		return homeworlds.Piece(0), parseError
	}
	return p, nil
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

func do(g *homeworlds.Game, a Action) error {
	star, ok := g.Stars[a.System]
	if !ok {
		return fmt.Errorf("no such system %s", a.System)
	}
	switch a.Type {
	case homeworlds.Build:
		return g.Build(a.Ship, star)
	case homeworlds.Trade:
		return g.Trade(a.Ship, star, a.NewShip)
	case homeworlds.Move:
		toStar, ok := g.Stars[a.NewSystem]
		if !ok {
			return fmt.Errorf("no such system %s", a.NewSystem)
		}
		return g.Move(a.Ship, star, toStar)
	case homeworlds.Attack:
		target := homeworlds.North
		if g.CurrentPlayer == homeworlds.North {
			target = homeworlds.South
		}
		return g.Attack(a.Ship, star, target)
	case homeworlds.Discover:
		return g.Discover(a.Ship, star, a.NewShip, a.NewSystem)
	}
	return nil
}
