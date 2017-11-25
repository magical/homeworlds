package homeworlds

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"
)

const maxPositions = 2e6

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
type BasicAction struct {
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

func (b BasicAction) Type() ActionType { return ActionType(b.typ) }
func (b BasicAction) System() int      { return int(b.system) }
func (b BasicAction) Ship() Piece      { return Piece(b.ship) }
func (b BasicAction) NewShip() Piece   { return Piece(b.arg) }
func (b BasicAction) NewSystem() Piece { return Piece(b.arg) }
func (b BasicAction) ToSystem() int    { return int(b.arg) }
func (b BasicAction) Action() Action {
	return Action{typ: b.typ, ship: b.ship, system: b.system, arg: b.arg}
}

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

func (b BasicAction) String() string {
	var arg interface{} = b.arg
	switch b.Type() {
	case Discover:
		arg = b.NewSystem()
	case Move:
		arg = b.ToSystem()
	case Trade:
		arg = b.NewShip()
	case Build, Attack:
		if b.arg == 0 {
			arg = ""
		}
	}
	s := fmt.Sprintf("%s %d %s %v", b.Type(), b.System(), b.Ship(), arg)
	return strings.TrimRight(s, " ")
}

func mkbasic(typ ActionType, ship Piece, system int, arg Piece) BasicAction {
	return BasicAction{typ: uint8(typ), system: uint8(system), ship: uint8(ship), arg: uint8(arg)}
}
func mkmove(ship Piece, system, tosystem int) BasicAction {
	return BasicAction{typ: uint8(Move), system: uint8(system), ship: uint8(ship), arg: uint8(tosystem)}
}

func (g *Game) BasicActions() []BasicAction {
	return PositionFromGame(g).BasicActions()
}

func (pos Position) BasicActions() []BasicAction {
	var actions []BasicAction
	actions = append(actions, BasicAction{typ: uint8(Pass)})

	for id, s := range pos.stars {
		ships := s.Ships(pos.CurrentPlayer())
		powers := s.pieces
		powers.add(ships)

		if powers.HasColor(Green) {
			for c := Color(0); c < Color(4); c++ {
				if ships.HasColor(c) && pos.bank.HasColor(c) {
					q := piece(pos.bank.SmallestOfColor(c), c)
					actions = append(actions, mkbasic(Build, q, id, 0))
				}
			}
		}

		if powers.HasColor(Blue) {
			for it := ships.Iter(); !it.Done(); it.Next() {
				if it.Count() > 0 {
					p := it.Piece()
					for c := Color(0); c < Color(4); c++ {
						q := piece(p.Size(), c)
						if c != p.Color() && pos.bank.Has(q) {
							actions = append(actions, mkbasic(Trade, p, id, q))
						}
					}
				}
			}
		}

		if powers.HasColor(Red) {
			size := ships.Largest()
			ships := s.OtherShips(pos.CurrentPlayer())
			for it := ships.Iter(); !it.Done(); it.Next() {
				if it.Count() > 0 {
					q := it.Piece()
					if q.Size() <= size {
						actions = append(actions, mkbasic(Attack, q, id, 0))
					}
				}
			}
		}

		if powers.HasColor(Yellow) {
			for rid, r := range pos.stars {
				if s.Connects(&r) {
					for it := ships.Iter(); !it.Done(); it.Next() {
						if p := it.Piece(); it.Count() > 0 {
							actions = append(actions, mkmove(p, id, rid))
						}
					}
				}
			}

			for it := pos.bank.Iter(); !it.Done(); it.Next() {
				if it.Count() > 0 {
					q := it.Piece()
					if s.WouldConnect(q) {
						for it := ships.Iter(); !it.Done(); it.Next() {
							if p := it.Piece(); it.Count() > 0 {
								actions = append(actions, mkbasic(Discover, p, id, q))
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

type Action struct {
	typ     uint8
	ship    uint8
	system  uint8
	arg     uint8
	actions [3]BasicAction
}

func (a Action) Type() ActionType { return ActionType(a.typ) }
func (a Action) Ship() Piece      { return Piece(a.ship) }
func (a Action) System() int      { return int(a.system) }
func (a Action) NewShip() Piece   { return Piece(a.arg) }
func (a Action) NewSystem() Piece { return Piece(a.arg) }
func (a Action) ToSystem() int    { return int(a.arg) }
func (a Action) N() int           { return int(a.arg) }

func mksacrifice(ship Piece, system int) Action {
	return Action{typ: uint8(Sacrifice), ship: uint8(ship), system: uint8(system)}
}

func (g *Game) SacrificeActions() []Action {
	pos := PositionFromGame(g)
	return pos.SacrificeActions()
}

func (pos Position) SacrificeActions() []Action {
	var sg sacrificeGenerator
	actions := sg.Generate(pos)
	return actions
}

type sacrificeGenerator struct {
	acts []Action
	//poses []Position
	val []float64
}

func (sg *sacrificeGenerator) Generate(pos Position) []Action {
	for id, s := range pos.stars {
		ships := s.Ships(pos.CurrentPlayer())
		for it := ships.Iter(); !it.Done(); it.Next() {
			if it.Count() > 0 {
				n := int(it.Piece().Size())
				if it.Piece() == Y3 {
					// XXX large yellows can result
					// in millions of potential actions
					// which is more than we can handle
					n = 2
				}
				if n > 2 {
					n = 2
				}
				a := mksacrifice(it.Piece(), id)
				tmp := pos.sacrifice(it.Piece(), id)
				sg.gen(a, &tmp, n)
			}
		}
	}
	//sort.Sort(sg)
	return sg.acts
}

func (sg *sacrificeGenerator) gen(sa Action, pos *Position, n int) {
	//if !pos.sanityCheck() {
	//	fmt.Printf("last action: %v\n", sa)
	//	return actions
	//}
	switch sa.Ship().Color() {
	case Red:
		for id, s := range pos.stars {
			ships := s.Ships(pos.CurrentPlayer())
			enemy := s.OtherShips(pos.CurrentPlayer())
			if !ships.IsEmpty() {
				size := ships.Largest()
				for it := enemy.Iter(); !it.Done(); it.Next() {
					if it.Count() > 0 && it.Piece().Size() <= size {
						b := mkbasic(Attack, it.Piece(), id, 0)
						sg.emit(pos, sa, b, n)
					}
				}
			}
		}

	case Green:
		for id, s := range pos.stars {
			ships := s.Ships(pos.CurrentPlayer())
			if !ships.IsEmpty() {
				for c := Color(0); c < Color(4); c++ {
					if ships.HasColor(c) && pos.bank.HasColor(c) {
						q := piece(pos.bank.SmallestOfColor(c), c)
						b := mkbasic(Build, q, id, 0)
						sg.emit(pos, sa, b, n)
					}
				}
			}
		}

	case Blue:
		for id, s := range pos.stars {
			ships := s.Ships(pos.CurrentPlayer())
			for it := ships.Iter(); !it.Done(); it.Next() {
				if it.Count() > 0 {
					p := it.Piece()
					for c := Color(0); c < Color(4); c++ {
						q := piece(p.Size(), c)
						if c != p.Color() && pos.bank.Has(q) {
							b := mkbasic(Trade, p, id, q)
							sg.emit(pos, sa, b, n)
						}
					}
				}
			}
		}

	case Yellow:
		for id, s := range pos.stars {
			ships := s.Ships(pos.CurrentPlayer())
			if !ships.IsEmpty() {
				for rid, r := range pos.stars {
					if s.Connects(&r) {
						for it := ships.Iter(); !it.Done(); it.Next() {
							if it.Count() > 0 {
								b := mkmove(it.Piece(), id, rid)
								sg.emit(pos, sa, b, n)
							}
						}
					}
				}

				for it := pos.bank.Iter(); !it.Done(); it.Next() {
					q := it.Piece()
					if it.Count() > 0 && s.WouldConnect(q) {
						for it := ships.Iter(); !it.Done(); it.Next() {
							p := it.Piece()
							if it.Count() > 0 {
								b := mkbasic(Discover, p, id, q)
								sg.emit(pos, sa, b, n)
							}
						}
					}
				}
			}
		}
	}
}

func (sg *sacrificeGenerator) emit(pos *Position, sa Action, b BasicAction, n int) {
	sa = sa.append(b)
	sg.acts = append(sg.acts, sa)
	//tmp := pos.do(b)
	//sg.val = append(sg.val, tmp.score())
	if n > 1 && !pos.over() {
		tmp := pos.do(b)
		sg.gen(sa, &tmp, n-1)
	}
}

// sort.Interface
func (sg *sacrificeGenerator) Len() int           { return len(sg.acts) }
func (sg *sacrificeGenerator) Less(i, j int) bool { return sg.val[i] < sg.val[j] }
func (sg *sacrificeGenerator) Swap(i, j int) {
	sg.acts[i], sg.acts[j] = sg.acts[j], sg.acts[i]
	sg.val[i], sg.val[j] = sg.val[j], sg.val[i]
}

func (a Action) append(b BasicAction) Action {
	a.actions[a.arg] = b
	a.arg++
	return a
}

func validAction(pos Position, a Action) bool {
	if a.Type() == Pass {
		return true
	}
	if a.System() >= len(pos.stars) {
		return false
	}
	star := pos.stars[a.System()]
	ships := star.ships[pos.player]
	powers := star.pieces
	powers.add(ships)
	switch a.Type() {
	case Build:
		return powers.HasColor(Green) &&
			pos.bank.Has(a.Ship()) &&
			ships.HasColor(a.Ship().Color()) &&
			pos.bank.SmallestOfColor(a.Ship().Color()) == a.Ship().Size()
	case Trade:
		return powers.HasColor(Blue) &&
			//b.Ship().Size() == b.NewShip().Size() &&
			pos.bank.Has(a.NewShip()) &&
			ships.Has(a.Ship())
	case Attack:
		return powers.HasColor(Red) &&
			!ships.IsEmpty() &&
			ships.Largest() >= a.Ship().Size() &&
			star.ships[pos.player^1].Has(a.Ship())
	case Move:
		if a.ToSystem() >= len(pos.stars) {
			return false
		}
		to := &pos.stars[a.ToSystem()]
		return powers.HasColor(Yellow) &&
			ships.Has(a.Ship()) &&
			star.Connects(to)
	case Discover:
		return powers.HasColor(Yellow) &&
			ships.Has(a.Ship()) &&
			pos.bank.Has(a.NewSystem()) &&
			star.WouldConnect(a.NewSystem())
	case Sacrifice:
		if !ships.Has(a.Ship()) {
			return false
		}
		power := a.Ship().Color()
		tmp := pos.do(a.Basic())
		for i := 0; i < a.N(); i++ {
			if !validSacrifice(tmp, a.Action(i), power) {
				return false
			}
			tmp = tmp.do(a.Action(i))
		}
		return true
	default:
		panic("unknown action " + a.String())
	}
}

func validSacrifice(pos Position, b BasicAction, power Color) bool {
	star := pos.stars[b.System()]
	ships := star.ships[pos.player]
	switch b.Type() {
	case Pass, Sacrifice:
		return false
	case Build:
		return power == Green &&
			pos.bank.Has(b.Ship()) &&
			ships.HasColor(b.Ship().Color()) &&
			pos.bank.SmallestOfColor(b.Ship().Color()) == b.Ship().Size()
	case Trade:
		return power == Blue &&
			//b.Ship().Size() == b.NewShip().Size() &&
			pos.bank.Has(b.NewShip()) &&
			ships.Has(b.Ship())
	case Attack:
		return power == Red &&
			!ships.IsEmpty() &&
			ships.Largest() >= b.Ship().Size() &&
			star.ships[pos.player^1].Has(b.Ship())
	case Move:
		if b.ToSystem() >= len(pos.stars) {
			return false
		}
		to := &pos.stars[b.ToSystem()]
		return power == Yellow &&
			ships.Has(b.Ship()) &&
			star.Connects(to)
	case Discover:
		return power == Yellow &&
			ships.Has(b.Ship()) &&
			pos.bank.Has(b.NewSystem()) &&
			star.WouldConnect(b.NewSystem())
	}
	panic("unknown action " + b.String())
}

func (pos Position) do(b BasicAction) Position {
	switch b.Type() {
	case Pass:
		return pos
	case Build:
		return pos.build(b.Ship(), b.System())
	case Trade:
		return pos.trade(b.Ship(), b.NewShip(), b.System())
	case Move:
		if b.System() >= len(pos.stars) {
			log.Println(pos)
			panic(fmt.Sprintf("no such system: %d", b.System()))
		}
		if b.ToSystem() >= len(pos.stars) {
			panic(fmt.Sprintf("no such system: %d", b.ToSystem()))
		}
		return pos.move(b.Ship(), b.System(), b.ToSystem())
	case Attack:
		return pos.attack(b.Ship(), b.System())
	case Discover:
		return pos.discover(b.Ship(), b.System(), b.NewSystem())
	case Sacrifice:
		return pos.sacrifice(b.Ship(), b.System())
	}
	panic(fmt.Sprintf("unknown action: %s", b.Type()))
}

func (pos Position) doXXX(a Action) Position {
	tmp := pos.do(a.Basic())
	if a.Type() == Sacrifice {
		for i := 0; i < a.N(); i++ {
			tmp = tmp.do(a.Action(i))
		}
	}
	tmp.endturn()
	return tmp
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
	pos.bank.Take(q)
	pos.stars[r].pieces.Put(q)
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
			if star.pieces.IsEmpty() {
				pos.bank.add(star.ships[North])
				pos.bank.add(star.ships[South])
				star.ships[0].bits = 0
				star.ships[1].bits = 0
				goto delete
			}
			// delete affected ships
			pos.bank.bits += star.ships[0].bits & mask
			pos.bank.bits += star.ships[1].bits & mask
			star.ships[0].bits &^= mask
			star.ships[1].bits &^= mask
		}
	}

	// delete if empty
	if star.ships[North].IsEmpty() && star.ships[South].IsEmpty() {
		goto delete
	}
	return

delete:
	if id >= 2 {
		pos.bank.add(star.pieces)
		pos.stars = append(pos.stars[:id], pos.stars[id+1:]...)
	}
}

func (pos *Position) gcmove(s, r int) {
	// delete stars in reverse order to prevent
	// indices from changing
	if s < r {
		pos.gc(r, true)
		pos.gc(s, false)
	} else if s > r {
		pos.gc(s, false)
		pos.gc(r, true)
	} else {
		panic("s == r")
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
	fmt.Fprintln(&buf, "Player:", pos.CurrentPlayer())
	return buf.String()
}

type AI struct {
	r *rand.Rand

	// config
	depth int  // depth of search
	trace int  // trace up to depth
	debug bool // enable sanity checks

	// state
	best      []Action
	cancelled bool

	// stats
	evaluated int64
	visited   int64
	positions int64
	unique    map[string]float64
}

func NewAI() *AI {
	return &AI{
		r:     rand.New(rand.NewSource(1)),
		depth: 3,
		debug: true,
		trace: 0,
	}
}

func (ai *AI) Reset() {
	ai.cancelled = false
	ai.visited = 0
	ai.evaluated = 0
	ai.positions = 0
	ai.unique = make(map[string]float64)
	ai.best = make([]Action, ai.depth)
}

func (ai *AI) Minimax(pos Position, last BasicAction) (Action, float64) {
	t := time.Now()
	ai.Reset()

	ply := 1

	var a Action
	var v float64
	for depth := 1; depth <= ai.depth; depth++ {
		tt := time.Now()
		ai.visited = 0
		ai.evaluated = 0
		ai.positions = 0
		ai.cancelled = false
		a, v = ai.minimax0(pos, last, ply, depth)
		dd := time.Since(tt)
		ms := float64(dd) / float64(time.Millisecond)
		log.Printf("maxdepth=%d positions=%d unique=%d visited=%d evaluated=%d (%.1f/ms) in %s",
			depth, ai.positions, len(ai.unique), ai.visited, ai.evaluated,
			float64(ai.evaluated)/ms,
			dd)
		if ai.cancelled {
			log.Print("(search aborted)")
		}
		if math.Abs(v) >= 1.0 {
			v += math.Copysign(float64(ai.depth-depth), v)
			break
		}
	}

	log.Println(pos)

	_ = t
	/*
		d := time.Since(t)
		ms := float64(d) / float64(time.Millisecond)
		log.Printf("positions=%d unique=%d visited=%d evaluated=%d (%.1f/ms) in %s",
			ai.positions, len(ai.unique), ai.visited, ai.evaluated,
			float64(ai.evaluated)/ms,
			d)
	*/

	return a, v
}

func (ai *AI) minimax0(pos Position, last BasicAction, ply, depth int) (Action, float64) {
	min := 5.0 // TODO: rename to hi, lo
	max := -5.0

	acts := pos.BasicActions()
	shuffle(acts, ai.r)
	log.Printf("%d basic actions to examine", len(acts))
	for _, a := range acts {
		if ai.debug && !validAction(pos, a.Action()) {
			log.Println(pos)
			log.Println("invalid action:", a)
		}
		tmp := pos.do(a)
		if a.Type() == Attack && a == last {
			if ply <= ai.trace {
				log.Println("%*s player=%d ply=%d action returns to an earlier state: %s",
					ply, "", pos.CurrentPlayer(), ply, a)
			}
			continue
		}
		//tmp.catastrophes()
		if ai.debug && !tmp.sanityCheck() {
			fmt.Println("last action:", a)
			continue
		}
		tmp.endturn()

		v := -ai.minimax(tmp, pos, ply+1, depth-1, -max, -min)
		//fmt.Printf("%d %c %f\n", depth, "+-"[pos.player], v)
		if ai.cancelled {
			break
		}
		if ply <= ai.trace {
			log.Printf("%*s player=%d ply=%d depth=%d v=%f min= max=%f move=%s", ply, "", pos.CurrentPlayer(), ply, depth, v, max, a)
		}
		if v > max {
			max = v
			ai.best[0] = a.Action()
		}
	}

	sacts := pos.SacrificeActions()
	sshuffle(sacts, ai.r)
	log.Printf("%d sacrifice actions to examine", len(sacts))
	for _, a := range sacts {
		if ai.debug && !validAction(pos, a) {
			log.Println(pos)
			log.Println("invalid action:", a)
		}
		tmp := pos.do(a.Basic())
		for i := 0; i < a.N(); i++ {
			tmp = tmp.do(a.actions[i])
		}
		if ai.debug && !tmp.sanityCheck() {
			fmt.Println("last action:", a)
			continue
		}
		tmp.endturn()
		v := -ai.minimax(tmp, pos, ply+1, depth-1, -max, -min)
		if ai.cancelled {
			break
		}
		if ply <= ai.trace {
			log.Printf("%*s player=%d ply=%d depth=%d v=%f min= max=%f move=%s", ply, "", pos.CurrentPlayer(), ply, depth, v, max, a.Basic())
		}
		if v > max {
			max = v
			ai.best[0] = a
		}
	}

	return ai.best[0], max
}

func (ai *AI) minimax(pos, last Position, ply, depth int, min, max float64) float64 {
	if ai.positions > maxPositions {
		ai.cancelled = true
		return 0 // abort
	}

	ai.positions++

	if pos.over() {
		ai.evaluated++
		return pos.score() * float64(depth+1)
	}

	h := ""
	/*
		h := pos.hash()
		if x, ok := ai.unique[h]; ok {
			if math.Abs(x) < 1 {
				return x
			}
		}
	*/

	if depth <= 0 {
		ai.evaluated++
		v := pos.score()
		//ai.unique[h] = v
		return v
	}

	ai.visited++

	if a := ai.best[ply-1]; a.Type() != Pass && validAction(pos, a) {
		tmp := pos.do(a.Basic())
		if a.Type() == Sacrifice {
			for i := 0; i < a.N(); i++ {
				tmp = tmp.do(a.Action(i))
			}
		}
		tmp.endturn()
		if !(a.Type() == Attack && tmp.Equal(last)) {
			v := -ai.minimax(tmp, pos, ply+1, depth-1, -max, -min)
			if ai.cancelled {
				return 0
			}
			if v > max {
				max = v
				if max >= min {
					return ai.record(h, max)
				}
			}
		}
	}

	// basic actions
	acts := pos.BasicActions()
	//shuffle(acts, ai.r) // run out r
	//sortActions(pos, acts)
	for _, a := range acts {
		if ai.debug {
			if !validAction(pos, a.Action()) {
				log.Println(pos)
				log.Println("invalid action", a)
			}
		}
		tmp := pos.do(a)
		if a.Type() == Attack && tmp.Equal(last) {
			if ply <= ai.trace {
				log.Println("%*s player=%d ply=%d action returns to an earlier state: %s",
					ply, "", pos.CurrentPlayer(), ply, a)
			}
			continue
		}
		if ai.debug && !tmp.sanityCheck() {
			fmt.Println("pos:", tmp)
			fmt.Println("last action:", a)
			continue
		}
		//tmp.catastrophes()
		tmp.endturn()
		v := -ai.minimax(tmp, pos, ply+1, depth-1, -max, -min)
		if ai.cancelled {
			break
		}
		if ply <= ai.trace {
			log.Printf("%*s player=%d ply=%d depth=%d v=%f min=%f max=%f move=%s", ply, "", pos.CurrentPlayer(), ply, depth, v, min, max, a)
		}
		if v > max {
			max = v
		}
		if max >= min {
			ai.best[ply-1] = a.Action()
			return ai.record(h, max)
		}
	}

	// sacrifice actions
	sacts := pos.SacrificeActions()
	//sshuffle(sacts, ai.r)
	for _, sa := range sacts {
		if ai.debug {
			if !validAction(pos, sa) {
				log.Println(pos)
				log.Println("invalid action:", sa)
			}
		}
		tmp := pos.do(sa.Basic())
		for i := 0; i < sa.N(); i++ {
			tmp = tmp.do(sa.actions[i])
		}
		tmp.endturn()
		v := -ai.minimax(tmp, pos, ply+1, depth-1, -max, -min)
		if ai.cancelled {
			break
		}
		if ply <= ai.trace {
			log.Printf("%*s player=%d ply=%d depth=%d v=%f min=%f max=%f move=%s", ply, "", pos.CurrentPlayer(), ply, depth, v, min, max, sa.Basic())
		}
		if v > max {
			max = v
		}
		if max >= min {
			break
		}
	}
	return ai.record(h, max)
}

func (ai *AI) record(h string, max float64) float64 {
	//ai.unique[h] = max
	return max
}

func sortActions(pos Position, acts []BasicAction) {
	var val []float64
	for _, a := range acts {
		val = append(val, pos.do(a).score())
	}
	sort.Sort(byValue{acts, val})
}

type byValue struct {
	acts []BasicAction
	val  []float64
}

func (x byValue) Len() int           { return len(x.acts) }
func (x byValue) Less(i, j int) bool { return x.val[i] > x.val[j] }
func (x byValue) Swap(i, j int) {
	x.acts[i], x.acts[j] = x.acts[j], x.acts[i]
	x.val[i], x.val[j] = x.val[j], x.val[i]
}

func (pos Position) over() bool {
	return pos.stars[North].ships[North].IsEmpty() || pos.stars[South].ships[South].IsEmpty()
}

func (pos *Position) endturn() {
	pos.player ^= 1
}

func (a Action) Basic() BasicAction {
	return BasicAction{typ: a.typ, ship: a.ship, system: a.system, arg: a.arg}
}

func (a Action) Action(i int) BasicAction {
	return a.actions[i]
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

func sshuffle(acts []Action, r *rand.Rand) {
	for i := 0; 1 < len(acts)-i; i++ {
		j := i + r.Intn(len(acts)-i)
		acts[i], acts[j] = acts[j], acts[i]
	}
}

func shuffle(acts []BasicAction, r *rand.Rand) {
	for i := 0; 1 < len(acts)-i; i++ {
		j := i + r.Intn(len(acts)-i)
		acts[i], acts[j] = acts[j], acts[i]
	}
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

func (a Action) String() string {
	s := ""
	s += a.Basic().String()
	if a.Type() == Sacrifice {
		for i := 0; i < a.N(); i++ {
			s += ", " + a.Action(i).String()
		}
	}
	return s
}

func (pos Position) hash() string {
	var b = make([]byte, 1+12*len(pos.stars))
	b[0] = pos.player
	for i, s := range pos.stars {
		b[1+i*12+0] = uint8(s.pieces.bits)
		b[1+i*12+1] = uint8(s.pieces.bits >> 8)
		b[1+i*12+2] = uint8(s.pieces.bits >> 16)
		b[1+i*12+3] = uint8(s.pieces.bits >> 24)
		b[1+i*12+4] = uint8(s.ships[0].bits)
		b[1+i*12+5] = uint8(s.ships[0].bits >> 8)
		b[1+i*12+6] = uint8(s.ships[0].bits >> 16)
		b[1+i*12+7] = uint8(s.ships[0].bits >> 24)
		b[1+i*12+8] = uint8(s.ships[1].bits)
		b[1+i*12+9] = uint8(s.ships[1].bits >> 8)
		b[1+i*12+10] = uint8(s.ships[1].bits >> 16)
		b[1+i*12+11] = uint8(s.ships[1].bits >> 24)
	}
	return string(b)
}

func (pos Position) winner() Player {
	if pos.stars[North].ships[North].IsEmpty() {
		return South
	}
	if pos.stars[South].ships[South].IsEmpty() {
		return North
	}
	return 3
}
