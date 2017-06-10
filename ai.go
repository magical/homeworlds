package homeworlds

import (
	"fmt"
	"sort"
)

type AI struct {
}

type Action struct {
	typ    uint8
	system uint8
	ship   uint8
	arg    uint8
}

type ActionType int

const (
	Pass ActionType = iota
	Build
	Move
	Discover
	Trade
	Attack
	Catastrope
)

func (a Action) Type() ActionType { return ActionType(a.typ) }
func (a Action) System() int      { return int(a.system) }
func (a Action) Ship() Piece      { return Piece(a.ship) }
func (a Action) NewShip() Piece   { return Piece(a.arg) }
func (a Action) NewSystem() Piece { return Piece(a.arg) }
func (a Action) ToSystem() int    { return int(a.arg) }

func (t ActionType) String() string {
	return map[ActionType]string{
		Catastrope: "Catastrope",
		Build:      "Build",
		Move:       "Move",
		Discover:   "Discover",
		Attack:     "Attack",
		Trade:      "Trade",
		Pass:       "Pass",
	}[t]
}

func (a Action) String() string {
	var arg interface{} = a.arg
	switch a.Type() {
	case Discover:
		arg = a.NewSystem()
	case Move:
		arg = a.ToSystem()
	case Trade:
		arg = a.NewShip()
	}
	return fmt.Sprintf("%s %d %s %v", a.Type(), a.System(), a.Ship(), arg)
}

func mkaction(typ ActionType, ship Piece, system int, arg Piece) Action {
	return Action{typ: uint8(typ), system: uint8(system), ship: uint8(ship), arg: uint8(arg)}
}
func mkmove(ship Piece, system, tosystem int) Action {
	return Action{typ: uint8(Move), system: uint8(system), ship: uint8(ship), arg: uint8(tosystem)}
}

func (g *Game) BasicActions() []Action {
	var actions []Action
	actions = append(actions, Action{typ: uint8(Pass)})

	stars := g.sortedStars()
	for _, name := range stars {
		s := g.Stars[name]
		systemId := sort.SearchStrings(stars, name)

		var powers uint
		for _, p := range s.Pieces {
			powers |= 1 << p.Color()
		}
		for _, p := range s.Ships[g.CurrentPlayer] {
			powers |= 1 << p.Color()
		}

		if powers&(1<<Green) != 0 {
			var colors uint
			for _, p := range s.Ships[g.CurrentPlayer] {
				colors |= 1 << p.Color()
			}
			for c := Color(0); c < Color(4); c++ {
				if colors&(1<<c) != 0 {
					q, ok := g.smallest(c)
					if ok {
						actions = append(actions, mkaction(Build, q, systemId, 0))
					}
				}
			}
		}

		if powers&(1<<Blue) != 0 {
			for _, p := range s.Ships[g.CurrentPlayer] {
				for c := Color(0); c < Color(4); c++ {
					q := piece(p.Size(), c)
					if c != p.Color() && g.available(q) {
						actions = append(actions, mkaction(Trade, p, systemId, q))
					}
				}
			}
		}

		if powers&(1<<Red) != 0 {
			size := s.largest(g.CurrentPlayer)
			for pl, ships := range s.Ships {
				if pl != g.CurrentPlayer {
					for _, q := range ships {
						if q.Size() <= size {
							actions = append(actions, mkaction(Attack, q, systemId, 0))
						}
					}
				}
			}
		}

		if powers&(1<<Yellow) != 0 {
			for _, name := range stars {
				r := g.Stars[name]
				if s.connects(r) {
					for _, p := range s.Ships[g.CurrentPlayer] {
						id := sort.SearchStrings(stars, name)
						actions = append(actions, mkmove(p, systemId, id))
					}
				}
			}

			for q, _ := range g.Bank {
				if g.available(q) && s.wouldConnect(q) {
					for _, p := range s.Ships[g.CurrentPlayer] {
						actions = append(actions, mkaction(Discover, p, systemId, Piece(q)))
					}
				}
			}
		}
	}

	return actions
}

// Smallest return the smallest available piece of a given color, or false
func (g *Game) smallest(c Color) (Piece, bool) {
	for size := Small; size <= Large; size++ {
		p := piece(size, c)
		if g.available(p) {
			return p, true
		}
	}
	return 0, false
}

func (s *Star) wouldConnect(p Piece) bool {
	for _, q := range s.Pieces {
		if q.Size() == p.Size() {
			return false
		}
	}
	return true
}
