package main

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

// Game represents the current state of a game.
type Game struct {
	// Phase records whether the game is
	// in the set-up phase,
	// in the middle, or has ended.
	Phase int

	// Players records the number of players.
	Players int

	// The player whose turn it is
	CurrentPlayer Player

	// Bank records which pieces are in the bank.
	Bank map[Piece]int

	// Homeworlds maps each player to their homeworld.
	// These systems are also present in the Stars slice.
	Homeworlds map[Player]*Star

	// Stars is a list of star systems that are currently occupied.
	Stars []*Star
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
	Ships []Ship
}

// Ship represents a piece controlled by a player.
type Ship struct {
	Piece  Piece
	Player Player
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
