package main

import (
	"bytes"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/magical/homeworlds"
)

var game *homeworlds.Game

func main() {
	host := flag.String("host", ":8080", "host and port to listen on")
	flag.Parse()

	game = newGame()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var b bytes.Buffer
		if err := homeworlds.Print(&b, game); err != nil {
			io.WriteString(w, err.Error())
			return
		}
		tmpl.Execute(w, b.String())
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
	var (
		north = homeworlds.Star{
			Name:        "north",
			IsHomeworld: true,
			Pieces:      []homeworlds.Piece{homeworlds.Y3, homeworlds.B2},
			Ships: map[homeworlds.Player][]homeworlds.Piece{
				homeworlds.North: {homeworlds.G3},
			},
		}

		south = homeworlds.Star{
			Name:        "south",
			IsHomeworld: true,
			Pieces:      []homeworlds.Piece{homeworlds.G3, homeworlds.R1},
			Ships: map[homeworlds.Player][]homeworlds.Piece{
				homeworlds.South: {homeworlds.Y3},
			},
		}
		game = homeworlds.Game{
			NumPlayers: 2,
			Homeworlds: map[homeworlds.Player]string{
				homeworlds.North: "north",
				homeworlds.South: "south",
			},
			Stars: map[string]*homeworlds.Star{
				"north": &north,
				"south": &south,
			},
		}
	)

	game.ResetBank()

	return &game
}
