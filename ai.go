package homeworlds

import "fmt"

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
	Sacrifice
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
	for systemId, name := range stars {
		s := g.Stars[name]

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
			for id, name := range stars {
				r := g.Stars[name]
				if s.connects(r) {
					for _, p := range s.Ships[g.CurrentPlayer] {
						actions = append(actions, mkmove(p, systemId, id))
					}
				}
			}

			for i := 0; i < 12; i++ {
				q := Piece(i)
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

type SacrificeAction struct {
	Ship    Piece
	System  uint8
	Actions []Action
}

func (g *Game) SacrificeActions() []SacrificeAction {
	var actions []SacrificeAction
	stars := g.sortedStars()
	for systemId, name := range stars {
		s := g.Stars[name]
		for _, p := range s.Ships[g.CurrentPlayer] {
			n := int(p.Size())
			a := SacrificeAction{Ship: p, System: uint8(systemId)}
			tmp := g.Copy()
			s = tmp.Stars[name]
			tmp.Sacrifice(p, s)
			actions = sacrifice(tmp, actions, a, n)
		}
	}
	return actions
}

func sacrifice(g *Game, actions []SacrificeAction, sa SacrificeAction, n int) []SacrificeAction {
	stars := g.sortedStars()
	switch sa.Ship.Color() {
	case Red:
		for systemId, name := range stars {
			s := g.Stars[name]
			if len(s.Ships[g.CurrentPlayer]) > 0 {
				size := s.largest(g.CurrentPlayer)
				for pl, ships := range s.Ships {
					if pl != g.CurrentPlayer {
						for _, q := range ships {
							if q.Size() <= size {
								a := mkaction(Attack, q, systemId, 0)
								actions = appendSacrifice(actions, g, stars, sa, a, n)
							}
						}
					}
				}
			}
		}

	case Yellow:
		for systemId, name := range stars {
			s := g.Stars[name]
			if len(s.Ships[g.CurrentPlayer]) > 0 {
				for id, name := range stars {
					r := g.Stars[name]
					if s.connects(r) {
						for _, p := range s.Ships[g.CurrentPlayer] {
							a := mkmove(p, systemId, id)
							actions = appendSacrifice(actions, g, stars, sa, a, n)
						}
					}
				}

				for i := 0; i < 12; i++ {
					q := Piece(i)
					if g.available(q) && s.wouldConnect(q) {
						for _, p := range s.Ships[g.CurrentPlayer] {
							a := mkaction(Discover, p, systemId, Piece(q))
							actions = appendSacrifice(actions, g, stars, sa, a, n)
						}
					}
				}
			}
		}

	case Green:
		for systemId, name := range stars {
			s := g.Stars[name]
			if len(s.Ships[g.CurrentPlayer]) > 0 {
				var colors uint
				for _, p := range s.Ships[g.CurrentPlayer] {
					colors |= 1 << p.Color()
				}
				for c := Color(0); c < Color(4); c++ {
					if colors&(1<<c) != 0 {
						q, ok := g.smallest(c)
						if ok {
							a := mkaction(Build, q, systemId, 0)
							actions = appendSacrifice(actions, g, stars, sa, a, n)
						}
					}
				}
			}
		}
	}

	return actions
}

func appendSacrifice(actions []SacrificeAction, g *Game, stars []string, sa SacrificeAction, a Action, n int) []SacrificeAction {
	sa = sa.append(a)
	actions = append(actions, sa)
	if n > 1 {
		tmp := g.Copy()
		do(tmp, stars, a)
		actions = sacrifice(tmp, actions, sa, n-1)
	}
	return actions
}

func (sa SacrificeAction) append(a Action) SacrificeAction {
	actions := sa.Actions
	sa.Actions = make([]Action, len(actions)+1)
	copy(sa.Actions, actions)
	sa.Actions[len(actions)] = a
	return sa
}

var newSystemId = 0

func do(g *Game, stars []string, a Action) error {
	star, ok := g.Stars[stars[a.System()]]
	if !ok {
		return fmt.Errorf("no such system %s", a.System())
	}
	switch a.Type() {
	case Build:
		return g.Build(a.Ship(), star)
	case Trade:
		return g.Trade(a.Ship(), star, a.NewShip())
	case Move:
		toStar, ok := g.Stars[stars[a.NewSystem()]]
		if !ok {
			return fmt.Errorf("no such system %s", a.NewSystem())
		}
		return g.Move(a.Ship(), star, toStar)
	case Attack:
		target := North
		if g.CurrentPlayer == North {
			target = South
		}
		return g.Attack(a.Ship(), star, target)
	case Discover:
		newSystemId++
		return g.Discover(a.Ship(), star, a.NewShip(), fmt.Sprint(newSystemId))
	}
	return nil
}
