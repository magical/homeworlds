package homeworlds

import (
	"errors"
)

type (
	Size   uint
	Color  uint
	Piece  uint
	Player uint
)

// Sizes of pieces
const (
	_ Size = iota
	Small
	Medium
	Large
)

// Colors of pieces
const (
	Red Color = iota
	Yellow
	Green
	Blue
)

// Pieces
const (
	R1 Piece = iota
	R2
	R3

	Y1
	Y2
	Y3

	G1
	G2
	G3

	B1
	B2
	B3
)

func (p Piece) Color() Color {
	return Color(p) / 3
}

func (p Piece) Size() Size {
	return Size(p)%3 + 1
}

// Players
// TODO: east, west.
const (
	North Player = iota
	South
)

// Game represents the current state of a game.
type Game struct {
	// Phase records whether the game is
	// in the set-up phase,
	// in the middle, or has ended.
	Phase int

	// NumPlayers records the number of players.
	NumPlayers int

	// CurrentPlayer records whose turn it is.
	CurrentPlayer Player

	// Bank records how many of each piece are in the bank.
	Bank map[Piece]int

	// Homeworlds maps each player to the name of their homeworld,
	// which can be looked up in the Stars map.
	Homeworlds map[Player]string

	// Stars is a map of star systems that are currently occupied.
	// It is keyed by the name of the system.
	Stars map[string]*Star
}

// Star represents an occupied star system.
type Star struct {
	// Name is the name given to the star system
	// by the player who discovered it.
	Name string

	// IsHomeworld records whether the star is a player's homeworld
	IsHomeworld bool

	// The piece (or pieces) which the star is made of.
	Pieces []Piece

	// The ships occupying the star.
	Ships map[Player][]Piece
}

// Actions:
//    Homeworld star1 star2 ship
//    Discover ship fromSystem newStar newName
//    Move ship fromSystem toSystem
//    Build ship inSystem
//    Trade oldShip newShip inSystem
//    Attack ship inSystem
//    Sacrifice ship inSystem
//    Catastrophe color inSystem
//    Pass

// Build places the given piece in the star system.
// Returns an error if the piece is unavailable,
// or if a smaller piece of the same color is available,
// or if the player does not control a ship of the same color.
//
// TODO: Make sure we have access to the grow power.
// Unless this is a sacrifice action...
func (g *Game) Build(p Piece, s *Star) error {
	if !s.ownsColor(g.CurrentPlayer, p.Color()) {
		return errors.New("Build: color not available")
	}
	if !g.available(p) {
		return errors.New("Build: piece not avalable")
	}
	// TODO: This loop is unclear.
	for i := 1; i < int(p.Size()); i++ {
		if g.available(p - Piece(i)) {
			return errors.New("Build: smaller piece available")
		}
	}
	g.take(p)
	s.add(g.CurrentPlayer, p)
	return nil
}

func (g *Game) available(p Piece) bool {
	return g.Bank[p] > 0
}

func (s *Star) ownsColor(pl Player, c Color) bool {
	for _, p := range s.Ships[pl] {
		if p.Color() == c {
			return true
		}
	}
	return false
}

func (s *Star) add(pl Player, p Piece) {
	s.Ships[pl] = append(s.Ships[pl], p)
}

func (g *Game) take(p Piece) {
	g.Bank[p]--
}

// Move moves a ship from one star to another.
// Returns an error if the current player
// does not control the ship at the specified system,
// or if the systems are not connected.
func (g *Game) Move(p Piece, s, dst *Star) error {
	if !s.connects(dst) {
		return errors.New("Move: system not connected")
	}
	ok := s.remove(g.CurrentPlayer, p)
	if !ok {
		return errors.New("Move: no such ship")
	}
	dst.add(g.CurrentPlayer, p)
	if s.empty() {
		g.destroy(s)
	}
	return nil
}

func (s *Star) connects(r *Star) bool {
	// Two systems are connected if they do not share
	// any pieces of the same size.
	for _, p := range s.Pieces {
		for _, q := range r.Pieces {
			if p.Size() == q.Size() {
				return false
			}
		}
	}
	return true
}

func (s *Star) remove(pl Player, p0 Piece) bool {
	for i, p := range s.Ships[pl] {
		if p == p0 {
			s.Ships[pl] = append(s.Ships[pl][:i], s.Ships[pl][i+1:]...)
			return true
		}
	}
	return false
}

// Empty reports whether any ships occupy the star.
func (s *Star) empty() bool {
	for _, ships := range s.Ships {
		if len(ships) > 0 {
			return false
		}
	}
	return true
}

// Destroy returns a star and all its ships to the bank.
func (g *Game) destroy(s *Star) {
	for _, ships := range s.Ships {
		for _, p := range ships {
			g.put(p)
		}
	}
	for _, p := range s.Pieces {
		g.put(p)
	}
	s.Ships = nil
	s.Pieces = nil
	delete(g.Stars, s.Name)
}

func (g *Game) put(p Piece) {
	g.Bank[p]++
}

// Attack takes control of a piece owned by the target player.
// Returns an error if the target does not own the target piece,
// or if the target piece is larger than the attacking player's largest ship.
func (g *Game) Attack(p Piece, s *Star, target Player) error {
	if target == g.CurrentPlayer {
		return errors.New("Attack: cannot attack yourself")
	}
	if !s.owns(target, p) {
		return errors.New("Attack: no such piece")
	}
	if p.Size() > s.largest(g.CurrentPlayer) {
		return errors.New("Attack: target piece too large")
	}
	s.remove(target, p)
	s.add(g.CurrentPlayer, p)
	return nil
}

func (s *Star) owns(pl Player, p0 Piece) bool {
	for _, p := range s.Ships[pl] {
		if p0 == p {
			return true
		}
	}
	return false
}

func (s *Star) largest(pl Player) Size {
	size := Size(0)
	for _, p := range s.Ships[pl] {
		if p.Size() > size {
			size = p.Size()
		}
	}
	return size
}

// Trade swaps a piece p for a piece q of the same size from the bank.
// Returns an error if the pieces are not the same size,
// or if the desired piece is not available,
// or if the player does not own the traded piece.
func (g *Game) Trade(p Piece, s *Star, q Piece) error {
	if p.Size() != q.Size() {
		return errors.New("Trade: size mismatch")
	}
	if !g.available(q) {
		return errors.New("Trade: piece not available")
	}
	if !s.owns(g.CurrentPlayer, p) {
		return errors.New("Trade: no such piece")
	}
	s.remove(g.CurrentPlayer, p)
	g.put(p)
	g.take(q)
	s.add(g.CurrentPlayer, q)
	return nil
}

// Sacrifice returns a piece to the bank.
// Typically this allows the player to take further actions,
// but this is not enforced.
// Returns an error if the player does not own the piece.
func (g *Game) Sacrifice(p Piece, s *Star) error {
	ok := s.remove(g.CurrentPlayer, p)
	if !ok {
		return errors.New("Sacrifice: no such piece")
	}
	g.put(p)
	// TODO: destroy star if empty
	return nil
}

// Catastrophe returns all pieces of the given color in the system to the bank,
// including all players' ships and the star itself.
// This may result in the complete destruction of the system.
// Returns an error if the color is not overpopulated.
func (g *Game) Catastrophe(c Color, s *Star) error {
	if s.population(c) < 4 {
		return errors.New("Catastrophe: not overpopulated")
	}
	for pl, ships := range s.Ships {
		s.Ships[pl] = g.filter(ships, c)
	}
	s.Pieces = g.filter(s.Pieces, c)
	if len(s.Pieces) == 0 || s.empty() {
		g.destroy(s)
	}
	return nil
}

func (s *Star) population(c Color) int {
	n := 0
	for _, ships := range s.Ships {
		for _, p := range ships {
			if p.Color() == c {
				n++
			}
		}
	}
	for _, p := range s.Pieces {
		if p.Color() == c {
			n++
		}
	}
	return n
}

func (g *Game) filter(pieces []Piece, c Color) []Piece {
	var q []Piece
	for _, p := range pieces {
		if p.Color() != c {
			q = append(q, p)
		} else {
			g.put(p)
		}
	}
	return q
}

// Discover constructs a new star named newName out of newPiece
// and moves the piece p to it.
// Returns an error if the requested piece is not available,
// or if the new system would not be connected to the old system,
// or if the active piece is not controlled by the player,
// or if the name is already taken.
func (g *Game) Discover(p Piece, s *Star, newPiece Piece, newName string) error {
	if !s.owns(g.CurrentPlayer, p) {
		return errors.New("Discover: no such piece")
	}
	if !g.available(newPiece) {
		return errors.New("Discover: piece not available")
	}
	if _, exists := g.Stars[newName]; exists {
		return errors.New("Discover: name already taken")
	}
	// TODO: don't allocate yet.
	newStar := &Star{
		Name:   newName,
		Pieces: []Piece{newPiece},
		Ships:  make(map[Player][]Piece),
	}
	if !s.connects(newStar) {
		return errors.New("Discover: system not connected")
	}
	g.take(newPiece)
	g.Stars[newName] = newStar
	s.remove(g.CurrentPlayer, p)
	newStar.add(g.CurrentPlayer, p)
	if s.empty() {
		g.destroy(s)
	}
	return nil
}

func (g *Game) ResetBank() {
	g.Bank = make(map[Piece]int)
	for p := 0; p < 12; p++ {
		g.Bank[Piece(p)] = 3
	}
	for _, s := range g.Stars {
		for _, p := range s.Pieces {
			g.Bank[p]--
		}
		for _, ships := range s.Ships {
			for _, p := range ships {
				g.Bank[p]--
			}
		}
	}
}

func (g *Game) EndTurn() {
	g.CurrentPlayer = Player((int(g.CurrentPlayer) + 1) % g.NumPlayers)
}

func (g *Game) IsOver() bool {
	for pl := Player(0); int(pl) < g.NumPlayers; pl++ {
		if len(g.Homeworld(pl).Ships[pl]) < 1 {
			return true
		}
	}
	return false
}

func (g *Game) Winner() Player {
	for pl := Player(0); int(pl) < g.NumPlayers; pl++ {
		if len(g.Homeworld(pl).Ships[pl]) > 0 {
			return pl
		}
	}
	return Player(100)
}

// Homeworld returns the Star that is the given player's homeworld.
func (g *Game) Homeworld(pl Player) *Star {
	return g.Stars[g.Homeworlds[pl]]
}

func (g0 *Game) Copy() *Game {
	g := *g0
	g.Bank = make(map[Piece]int)
	g.Stars = make(map[string]*Star)
	g.Homeworlds = make(map[Player]string)
	for p, n := range g0.Bank {
		g.Bank[p] = n
	}
	for k, s := range g0.Stars {
		g.Stars[k] = s.Copy()
	}
	for pl, s := range g0.Homeworlds {
		g.Homeworlds[pl] = s
	}
	return &g
}

func (s0 *Star) Copy() *Star {
	s := *s0
	s.Pieces = make([]Piece, len(s0.Pieces))
	copy(s.Pieces, s0.Pieces)
	s.Ships = make(map[Player][]Piece)
	for pl, ships := range s0.Ships {
		s.Ships[pl] = make([]Piece, len(ships))
		copy(s.Ships[pl], ships)
	}
	return &s
}
