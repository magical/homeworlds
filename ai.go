package homeworlds

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

type Position struct {
	bank   Bank
	stars  []Dwarf
	player uint8
	turn   uint8
}

type Dwarf struct {
	pieces Bank
	ships  [2]Bank
}

func PositionFromGame(g *Game) Position {
	var pos Position
	pos.player = uint8(g.CurrentPlayer)
	for p, n := range g.Bank {
		pos.bank.Set(p, n)
	}
	stars := g.sortedStars()
	pos.stars = make([]Dwarf, 2, len(g.Stars))
	pos.stars[North] = dwarfFromStar(g.Homeworlds[North])
	pos.stars[South] = dwarfFromStar(g.Homeworlds[South])
	for _, name := range stars {
		s := g.Stars[name]
		if s.IsHomeworld {
			continue
		}
		pos.stars = append(pos.stars, dwarfFromStar(s))
	}
	return pos
}

func (pos *Position) CurrentPlayer() Player {
	return Player(pos.player)
}

func dwarfFromStar(s *Star) Dwarf {
	var r Dwarf
	if s == nil {
		return r
	}
	for _, p := range s.Pieces {
		r.pieces.Put(p)
	}
	for _, p := range s.Ships[North] {
		r.ships[North].Put(p)
	}
	for _, p := range s.Ships[South] {
		r.ships[South].Put(p)
	}
	return r
}

func (s *Dwarf) Ships(pl Player) Bank {
	return s.ships[pl&1]
}

func (s *Dwarf) OtherShips(pl Player) Bank {
	return s.ships[^pl&1]
}

// Action represents a basic action in homeworlds.
// The zero action is a valid action (Pass).
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
		Pass:       "Pass",
		Build:      "Build",
		Move:       "Move",
		Discover:   "Discover",
		Trade:      "Trade",
		Attack:     "Attack",
		Catastrope: "Catastrope",
		Sacrifice:  "Sacrifice",
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
	case Build, Attack:
		if a.arg == 0 {
			arg = ""
		}
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
	return PositionFromGame(g).BasicActions()
}

func (g Position) BasicActions() []Action {
	var actions []Action
	actions = append(actions, Action{typ: uint8(Pass)})

	for id, s := range g.stars {
		ships := s.Ships(g.CurrentPlayer())
		powers := s.pieces
		powers.add(ships)

		if powers.HasColor(Green) {
			for c := Color(0); c < Color(4); c++ {
				if ships.HasColor(c) && g.bank.HasColor(c) {
					q := piece(g.bank.SmallestOfColor(c), c)
					actions = append(actions, mkaction(Build, q, id, 0))
				}
			}
		}

		if powers.HasColor(Blue) {
			for it := ships.Iter(); !it.Done(); it.Next() {
				if it.Count() > 0 {
					p := it.Piece()
					for c := Color(0); c < Color(4); c++ {
						q := piece(p.Size(), c)
						if c != p.Color() && g.bank.Has(q) {
							actions = append(actions, mkaction(Trade, p, id, q))
						}
					}
				}
			}
		}

		if powers.HasColor(Red) {
			size := ships.Largest()
			ships := s.OtherShips(g.CurrentPlayer())
			for it := ships.Iter(); !it.Done(); it.Next() {
				if it.Count() > 0 {
					q := it.Piece()
					if q.Size() <= size {
						actions = append(actions, mkaction(Attack, q, id, 0))
					}
				}
			}
		}

		if powers.HasColor(Yellow) {
			for rid, r := range g.stars {
				if s.Connects(&r) {
					for it := ships.Iter(); !it.Done(); it.Next() {
						if p := it.Piece(); it.Count() > 0 {
							actions = append(actions, mkmove(p, id, rid))
						}
					}
				}
			}

			for it := g.bank.Iter(); !it.Done(); it.Next() {
				if it.Count() > 0 {
					q := it.Piece()
					if s.WouldConnect(q) {
						for it := ships.Iter(); !it.Done(); it.Next() {
							if p := it.Piece(); it.Count() > 0 {
								actions = append(actions, mkaction(Discover, p, id, q))
							}
						}
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

func (s *Dwarf) Connects(r *Dwarf) bool {
	return s.pieces.sizes()&r.pieces.sizes() == 0
}

func (s *Dwarf) WouldConnect(p Piece) bool {
	return s.pieces.sizes()&(1<<((p.Size()-1)*2)) == 0
}

func (b Bank) sizes() uint {
	x := uint(b.bits)
	x |= x >> 12
	x |= x >> 6
	x = (x & 0x15) | (x >> 1 & 0x15)
	return x
}

type SacrificeAction struct {
	Ship    Piece
	System  uint8
	Actions []Action
}

func (g *Game) SacrificeActions() []SacrificeAction {
	pos := PositionFromGame(g)
	return pos.SacrificeActions()
}

func (pos Position) SacrificeActions() []SacrificeAction {
	var actions []SacrificeAction
	for id, s := range pos.stars {
		ships := s.Ships(pos.CurrentPlayer())
		for it := ships.Iter(); !it.Done(); it.Next() {
			if it.Count() > 0 {
				n := int(it.Piece().Size())
				a := SacrificeAction{Ship: it.Piece(), System: uint8(id)}
				tmp := pos.sacrifice(it.Piece(), id)
				actions = sacrifice(&tmp, actions, a, n)
			}
		}
	}
	return actions
}

func sacrifice(g *Position, actions []SacrificeAction, sa SacrificeAction, n int) []SacrificeAction {
	//if !g.sanityCheck() {
	//	return actions
	//}
	switch sa.Ship.Color() {
	case Red:
		for id, s := range g.stars {
			ships := s.Ships(g.CurrentPlayer())
			enemy := s.OtherShips(g.CurrentPlayer())
			if !ships.IsEmpty() {
				size := ships.Largest()
				for it := enemy.Iter(); !it.Done(); it.Next() {
					if it.Count() > 0 && it.Piece().Size() <= size {
						a := mkaction(Attack, it.Piece(), id, 0)
						actions = appendSacrifice(actions, g, sa, a, n)
					}
				}
			}
		}

	case Yellow:
		for id, s := range g.stars {
			ships := s.Ships(g.CurrentPlayer())
			if !ships.IsEmpty() {
				for rid, r := range g.stars {
					if s.Connects(&r) {
						for it := ships.Iter(); !it.Done(); it.Next() {
							if it.Count() > 0 {
								a := mkmove(it.Piece(), id, rid)
								actions = appendSacrifice(actions, g, sa, a, n)
							}
						}
					}
				}

				for it := g.bank.Iter(); !it.Done(); it.Next() {
					q := it.Piece()
					if it.Count() > 0 && s.WouldConnect(q) {
						for it := ships.Iter(); !it.Done(); it.Next() {
							p := it.Piece()
							if it.Count() > 0 {
								a := mkaction(Discover, p, id, q)
								actions = appendSacrifice(actions, g, sa, a, n)
							}
						}
					}
				}
			}
		}

	case Green:
		for id, s := range g.stars {
			ships := s.Ships(g.CurrentPlayer())
			if !ships.IsEmpty() {
				for c := Color(0); c < Color(4); c++ {
					if ships.HasColor(c) && g.bank.HasColor(c) {
						q := piece(g.bank.SmallestOfColor(c), c)
						if !g.bank.Has(q) {
							log.Println(q, ships, g.bank)
							panic("oops")
						}
						a := mkaction(Build, q, id, 0)
						actions = appendSacrifice(actions, g, sa, a, n)
					}
				}
			}
		}
	}

	return actions
}

func appendSacrifice(actions []SacrificeAction, pos *Position, sa SacrificeAction, a Action, n int) []SacrificeAction {
	sa = sa.append(a)
	actions = append(actions, sa)
	if n > 1 {
		tmp, err := do(*pos, a)
		if err != nil {
			panic(err)
		}
		actions = sacrifice(&tmp, actions, sa, n-1)
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

func do(pos Position, a Action) (Position, error) {
	switch a.Type() {
	case Pass:
		return pos, nil
	case Build:
		return pos.build(a.Ship(), a.System()), nil
	case Trade:
		return pos.trade(a.Ship(), a.NewShip(), a.System()), nil
	case Move:
		if a.ToSystem() >= len(pos.stars) {
			return pos, fmt.Errorf("no such system %s", a.ToSystem())
		}
		return pos.move(a.Ship(), a.System(), a.ToSystem()), nil
	case Attack:
		return pos.attack(a.Ship(), a.System()), nil
	case Discover:
		return pos.discover(a.Ship(), a.System(), a.NewSystem()), nil
	}
	return pos, fmt.Errorf("unknown action %s", a.Type())
}

func (pos Position) build(p Piece, s int) Position {
	pos = pos.copy()
	pos.bank.Take(p)
	pl := pos.player & 1
	pos.stars[s].ships[pl].Put(p)
	pos.gc(s, true)
	return pos
}

func (pos Position) copy() Position {
	oldstars := pos.stars
	pos.stars = make([]Dwarf, len(oldstars))
	copy(pos.stars, oldstars)
	return pos
}

func (pos Position) trade(p, q Piece, s int) Position {
	pos = pos.copy()
	pos.bank.Put(p)
	pos.bank.Take(q)
	pl := pos.player & 1
	pos.stars[s].ships[pl].Take(p)
	pos.stars[s].ships[pl].Put(q)
	pos.gc(s, true)
	return pos
}

func (pos Position) move(p Piece, s, r int) Position {
	pos = pos.copy()
	pl := pos.player & 1
	pos.stars[s].ships[pl].Take(p)
	pos.stars[r].ships[pl].Put(p)
	pos.gcmove(s, r)
	return pos
}

func (pos Position) attack(p Piece, s int) Position {
	pos = pos.copy()
	pl := pos.player & 1
	pos.stars[s].ships[pl].Put(p)
	pos.stars[s].ships[pl^1].Take(p)
	// can't result in catastrophe
	return pos
}

func (pos Position) discover(p Piece, s int, q Piece) Position {
	r := len(pos.stars)
	oldstars := pos.stars
	pos.stars = make([]Dwarf, len(pos.stars)+1)
	copy(pos.stars, oldstars)
	pos.stars[r].pieces.Put(q)
	pos.bank.Take(q)
	pl := pos.player & 1
	pos.stars[s].ships[pl].Take(p)
	pos.stars[r].ships[pl].Put(p)
	pos.gcmove(s, r)
	return pos
}

// delete star if it is empty
// if catastrophe is true, check for catastrophes first
func (pos *Position) gc(id int, catastrophe bool) {
	star := &pos.stars[id]
	if catastrophe {
		// check for catastrophe
		pieces := star.pieces
		pieces.add(star.ships[0])
		pieces.add(star.ships[1])
		var mask uint32
		for c := Color(0); c < Color(4); c++ {
			if pieces.ColorCount(c) >= 4 {
				mask |= uint32(63) << (c * 6)
			}
		}
		if mask != 0 {
			// XXX ugly
			// delete affected star pieces
			pos.bank.bits += star.pieces.bits & mask
			star.pieces.bits &^= mask
			// delete affected ships
			pos.bank.bits += star.ships[0].bits & mask
			pos.bank.bits += star.ships[1].bits & mask
			star.ships[0].bits &^= mask
			star.ships[1].bits &^= mask
			if star.pieces.IsEmpty() {
				pos.bank.add(star.ships[North])
				pos.bank.add(star.ships[South])
				goto delete
			}
		}
	}

	// delete if empty
	if star.ships[North].IsEmpty() && star.ships[South].IsEmpty() {
		pos.bank.add(star.pieces)
		goto delete
	}
	return

delete:
	if id >= 2 {
		pos.stars = append(pos.stars[:id], pos.stars[id+1:]...)
	}
}

func (pos *Position) gcmove(s, r int) {
	// delete stars in reverse order to prevent
	// indices from changing
	if s < r {
		pos.gc(r, true)
		pos.gc(s, false)
	} else {
		pos.gc(s, false)
		pos.gc(r, true)
	}
}

// add the contents of another bank to this one
// if this would cause overflow, the result is undefined
func (b *Bank) add(other Bank) {
	b.bits += other.bits
}

func (pos Position) sacrifice(p Piece, s int) Position {
	pos = pos.copy()
	pos.bank.Put(p)
	pl := pos.player & 1
	pos.stars[s].ships[pl].Take(p)
	pos.gc(s, false)
	return pos
}

func (pos Position) sanityCheck() bool {
	b := make(map[Piece]int)
	for i := 0; i < 12; i++ {
		b[Piece(i)] = 3
	}
	for _, s := range pos.stars {
		for it := s.pieces.Iter(); !it.Done(); it.Next() {
			b[it.Piece()] -= it.Count()
		}
		for it := s.ships[North].Iter(); !it.Done(); it.Next() {
			b[it.Piece()] -= it.Count()
		}
		for it := s.ships[South].Iter(); !it.Done(); it.Next() {
			b[it.Piece()] -= it.Count()
		}
	}

	ok := true
	for i := 0; i < 12; i++ {
		if pos.bank.Get(Piece(i)) != b[Piece(i)] {
			ok = false
			log.Printf("bank: have %d %s, expected %d", pos.bank.Get(Piece(i)), Piece(i), b[Piece(i)])
		}
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "%v\n", pos)
		//panic("sanity check failed")
	}
	return ok
}

func (b Bank) String() string {
	var pieces []string
	for it := b.Iter(); !it.Done(); it.Next() {
		for i := 0; i < it.Count(); i++ {
			pieces = append(pieces, it.Piece().String())
		}
	}
	return "{" + strings.Join(pieces, ", ") + "}"
}

func (pos Position) String() string {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "Bank:", pos.bank.String())
	fmt.Fprintln(&buf, "Stars:")
	for _, s := range pos.stars {
		fmt.Fprintln(&buf, "- Pieces:", s.pieces.String())
		fmt.Fprintln(&buf, "  North:", s.ships[North].String())
		fmt.Fprintln(&buf, "  South:", s.ships[South].String())
	}
	return buf.String()
}

type AI struct {
	r     *rand.Rand
	depth int
	trace bool

	// stats
	evaluated int64
	visited   int64
}

func NewAI() *AI {
	return &AI{
		r:     rand.New(rand.NewSource(1)),
		depth: 5,
		trace: false,
	}
}

func (ai *AI) Minimax(pos Position, last Action) (Action, float64) {
	acts := pos.BasicActions()
	shuffle(acts, ai.r)
	var maxact Action
	max := -5.0
	depth := ai.depth
	t := time.Now()
	ai.visited = 0
	ai.evaluated = 0
	for _, a := range acts {
		tmp, err := do(pos, a)
		if err != nil {
			panic(err)
		}
		if a.Type() == Attack && a == last {
			fmt.Println("Action returns to an earlier state:", a)
			continue
		}
		//tmp.catastrophes()
		tmp = tmp.endturn()
		v := -ai.minimax(tmp, pos, depth-1, -max)
		//fmt.Printf("%d %c %f\n", depth, "+-"[pos.player], v)
		if ai.trace {
			log.Printf("%*s player=%d depth=%d v=%f min= max=%f move=%s", ai.depth-depth, "", pos.CurrentPlayer(), depth, v, max, a)
		}
		if v > max {
			max = v
			maxact = a
		}
	}
	d := time.Since(t)
	ms := float64(d) / float64(time.Millisecond)
	log.Printf("visited=%d (%.1f/ms) evaluated=%d (%.1f/ms) in %s",
		ai.visited, float64(ai.visited)/ms,
		ai.evaluated, float64(ai.evaluated)/ms,
		d)
	return maxact, max
}

func (ai *AI) minimax(pos, last Position, depth int, min float64) float64 {
	ai.visited++
	if pos.over() {
		ai.evaluated++
		return pos.score() * float64(depth+1)
	}
	if depth <= 0 {
		ai.evaluated++
		return pos.score()
	}
	max := -5.0
	acts := pos.BasicActions()
	shuffle(acts, ai.r)
	for _, a := range acts {
		tmp, err := do(pos, a)
		if err != nil {
			panic(err)
		}
		if a.Type() == Attack && tmp.Equal(last) {
			if depth == 4 {
				fmt.Println("action returns to an earlier state:", a)
			}
			continue
		}
		//tmp.catastrophes()
		tmp = tmp.endturn()
		v := -ai.minimax(tmp, pos, depth-1, -max)
		if ai.trace {
			log.Printf("%*s player=%d depth=%d v=%f min=%f max=%f move=%s", ai.depth-depth, "", pos.CurrentPlayer(), depth, v, min, max, a)
		}
		//fmt.Printf("%d %f\n", depth, v)
		if v > max {
			max = v
		}
		if max >= min {
			break
		}
	}
	return max
}

func (pos Position) over() bool {
	return pos.stars[North].ships[North].IsEmpty() || pos.stars[South].ships[South].IsEmpty()
}

func (pos Position) endturn() Position {
	pos.player ^= 1
	return pos
}

var points = []int{0, 1, 3, 9}

func (pos Position) score() float64 {
	if pos.stars[North].ships[North].IsEmpty() {
		return float64(pos.player)*2 - 1
	}
	if pos.stars[South].ships[South].IsEmpty() {
		return -float64(pos.player)*2 + 1
	}

	v := 0
	w := 0

	// +5 points for being the current player
	if pos.CurrentPlayer() == North {
		v += 5
	} else {
		w += 5
	}

	// +10 points for having a large at homeworld
	if pos.stars[North].ships[North].Largest() == Large {
		v += 10
	}
	if pos.stars[South].ships[South].Largest() == Large {
		w += 10
	}

	// +50 points for occupying the opponent's homeworld
	if !pos.stars[South].ships[North].IsEmpty() {
		v += 10
	}
	if !pos.stars[North].ships[South].IsEmpty() {
		w += 10
	}

	// +1 point for each small ship
	// +3 points for each medium ship
	// +9 points for each large ship
	var north Bank
	var south Bank
	for _, s := range pos.stars {
		north.add(s.ships[North])
		south.add(s.ships[South])
	}
	for it := north.Iter(); !it.Done(); it.Next() {
		v += points[it.Piece().Size()] * it.Count()
	}
	for it := south.Iter(); !it.Done(); it.Next() {
		w += points[it.Piece().Size()] * it.Count()
	}

	// +30 points for monopolizing a color
	for c := Color(0); c < Color(4); c++ {
		if north.HasColor(c) && !south.HasColor(c) {
			v += 30
		}
		if south.HasColor(c) && !north.HasColor(c) {
			w += 30
		}
	}

	// TODO:
	// +points for being few hops from opponent's homeworld
	// +points for controlling a star
	// +50 points for still having both stars in your homeworld

	const max = 5 + 10 + 50 + (1+3+9)*12 + 30
	score := float64(v-w) / (max + 1)
	if pos.player == 1 {
		score = -score
	}
	return score
}

func shuffle(acts []Action, r *rand.Rand) {
	for i := 0; 1 < len(acts)-i; i++ {
		j := i + r.Intn(len(acts)-i)
		acts[i], acts[j] = acts[j], acts[i]
	}
	//fmt.Println(acts)
}

func (pos *Position) catastrophes() {
	for id := 0; id < len(pos.stars); id++ {
		star := &pos.stars[id]
		pieces := star.pieces
		pieces.add(star.ships[0])
		pieces.add(star.ships[1])
		var mask uint32
		for c := Color(0); c < Color(4); c++ {
			if pieces.ColorCount(c) >= 4 {
				mask |= uint32(63) << (c * 6)
			}
		}
		if mask != 0 {
			// XXX ugly
			pos.bank.bits += star.pieces.bits & mask
			pos.bank.bits += star.ships[0].bits & mask
			pos.bank.bits += star.ships[1].bits & mask
			star.pieces.bits &^= mask
			star.ships[0].bits &^= mask
			star.ships[1].bits &^= mask

			if star.pieces.IsEmpty() {
				// bin the whole star
				pos.bank.add(star.ships[0])
				pos.bank.add(star.ships[1])
				star.ships[0].bits = 0
				star.ships[1].bits = 0
				if id >= 2 {
					pos.stars = append(pos.stars[:id], pos.stars[id+1:]...)
					id--
				}
			} else if star.ships[0].IsEmpty() && star.ships[1].IsEmpty() {
				if id >= 2 {
					pos.bank.add(star.pieces)
					pos.stars = append(pos.stars[:id], pos.stars[id+1:]...)
					id--
				}
			}
		}
	}
}

func (b Bank) ColorCount(c Color) int {
	x := b.bits >> (c * 6) & 63
	return int(x>>4) + int(x>>2&3) + int(x&3)
}

func (pos Position) Equal(other Position) bool {
	if pos.bank != other.bank {
		return false
	}
	if len(pos.stars) != len(other.stars) {
		return false
	}
	for id := range pos.stars {
		if pos.stars[id] != other.stars[id] {
			return false
		}
	}
	return true
}
