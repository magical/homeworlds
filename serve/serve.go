package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/magical/homeworlds"
)

const gamefilename = "game.json"

func main() {
	host := flag.String("host", ":8080", "host and port to listen on")
	flag.Parse()

	game, err := loadGame(gamefilename)
	if err != nil {
		log.Println(err)
		game = newGame()
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var b bytes.Buffer
		if err := homeworlds.Print(&b, game); err != nil {
			io.WriteString(w, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, b.String())
	})

	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		cmd := r.PostFormValue("command")
		g, err := play(game, cmd)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		game = g
		if err := saveGame(gamefilename, game); err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})

	log.Fatal(http.ListenAndServe(*host, nil))
}

var tmpl = template.Must(template.New("game").Parse(`<!doctype html>
<title>Homeworlds</title>
<pre>{{.}}</pre>
<hr>
<form action="play" method="POST">
  <textarea name="command" cols="40" rows="5"></textarea><br>
  <button>Play</button>
</form>
`))

func newGame() *homeworlds.Game {
	game := homeworlds.NewGame(2)
	if err := game.BuildHomeworld(homeworlds.Y3, homeworlds.B2, homeworlds.G3, "north"); err != nil {
		panic(err)
	}
	game.EndTurn()
	if err := game.BuildHomeworld(homeworlds.G3, homeworlds.R1, homeworlds.Y3, "south"); err != nil {
		panic(err)
	}
	game.EndTurn()
	return game
}

func loadGame(filename string) (*homeworlds.Game, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	g, err := homeworlds.Unmarshal(b)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func saveGame(filename string, g *homeworlds.Game) error {
	b, err := homeworlds.Marshal(g)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0644)
}

func play(g *homeworlds.Game, cmd string) (*homeworlds.Game, error) {
	actions, err := parseAction(cmd)
	if err != nil {
		return g, err
	}
	g = g.Copy()
	for _, a := range actions {
		if err := do(g, a); err != nil {
			return nil, err
		}
	}
	g.EndTurn()
	return g, nil

}

type Action struct {
	Type      homeworlds.ActionType
	Ship      homeworlds.Piece
	System    string
	NewShip   homeworlds.Piece
	NewSystem string
}

var parseError = errors.New("parse error")

func parseAction(cmd string) ([]Action, error) {
	var actions []Action
	s := bufio.NewScanner(strings.NewReader(cmd))
	for s.Scan() {
		line := s.Text()
		a, err := parseSingleAction(line)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	return actions, nil
}

func parseSingleAction(s string) (Action, error) {
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
