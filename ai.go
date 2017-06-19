package homeworlds

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
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
func (b BasicAction) Action() SacrificeAction {
	return SacrificeAction{typ: b.typ, ship: b.ship, system: b.system, arg: b.arg}
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

type SacrificeAction struct {
	typ     uint8
	ship    uint8
	system  uint8
	arg     uint8
	actions [3]BasicAction
}

func (sa SacrificeAction) Type() ActionType { return ActionType(sa.typ) }
func (sa SacrificeAction) Ship() Piece      { return Piece(sa.ship) }
func (sa SacrificeAction) System() int      { return int(sa.system) }
func (sa SacrificeAction) N() int           { return int(sa.arg) }

func mksacrifice(ship Piece, system int) SacrificeAction {
	return SacrificeAction{typ: uint8(Sacrifice), ship: uint8(ship), system: uint8(system)}
}

func (g *Game) SacrificeActions() []SacrificeAction {
	pos := PositionFromGame(g)
	return pos.SacrificeActions()
}

func (pos Position) SacrificeActions() []SacrificeAction {
	var sg sacrificeGenerator
	actions := sg.Generate(pos)
	return actions
}

type sacrificeGenerator struct {
	acts  []SacrificeAction
	poses []Position
}

func (sg *sacrificeGenerator) Generate(pos Position) []SacrificeAction {
	for id, s := range pos.stars {
		ships := s.Ships(pos.CurrentPlayer())
		for it := ships.Iter(); !it.Done(); it.Next() {
			if it.Count() > 0 {
				n := int(it.Piece().Size())
				if it.Piece() == Y3 {
					// XXX large yellows can result
					// in millions of potential actions
					// which is more than we can handle
					n = 3
				}
				a := mksacrifice(it.Piece(), id)
				tmp := pos.sacrifice(it.Piece(), id)
				sg.gen(a, &tmp, n)
			}
		}
	}
	if len(sg.acts) <= 100 {
		return sg.acts
	}
	// deduplicate
	sort.Sort(&sortActions{sg.acts, sg.poses})
	n := 1
	for i := 1; i < len(sg.poses); i++ {
		if !sg.poses[i-1].Equal(sg.poses[i]) {
			sg.acts[n] = sg.acts[i]
			n++
		}
	}
	//log.Printf("reduced %d actions to %d", len(sg.acts), n)
	return sg.acts[:n]
}

func (sg *sacrificeGenerator) gen(sa SacrificeAction, pos *Position, n int) {
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

func (sg *sacrificeGenerator) emit(pos *Position, sa SacrificeAction, b BasicAction, n int) {
	sa = sa.append(b)
	tmp := pos.do(b)
	sg.acts = append(sg.acts, sa)
	sg.poses = append(sg.poses, tmp)
	if n > 1 && !pos.over() {
		sg.gen(sa, &tmp, n-1)
	}
}

func (sa SacrificeAction) append(b BasicAction) SacrificeAction {
	sa.actions[sa.arg] = b
	sa.arg++
	return sa
}

type sortActions struct {
	a []SacrificeAction
	p []Position
}

func (sa *sortActions) Len() int { return len(sa.a) }
func (sa *sortActions) Swap(i, j int) {
	sa.a[i], sa.a[j] = sa.a[j], sa.a[i]
	sa.p[i], sa.p[j] = sa.p[j], sa.p[i]
}
func (sa *sortActions) Less(i, j int) bool {
	if sa.p[i].bank.less(sa.p[j].bank) {
		return true
	}
	if len(sa.p[i].stars) < len(sa.p[j].stars) {
		return true
	}
	for k := range sa.p[i].stars {
		s := &sa.p[i].stars[k]
		r := &sa.p[j].stars[k]
		if s.pieces.less(r.pieces) {
			return true
		}
		if s.ships[0].less(r.ships[0]) {
			return true
		}
		if s.ships[1].less(r.ships[1]) {
			return true
		}
	}
	return false
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
	return buf.String()
}

type AI struct {
	r     *rand.Rand
	depth int
	trace int  // trace up to depth
	debug bool // enable sanity checks

	// stats
	evaluated int64
	visited   int64
}

func NewAI() *AI {
	return &AI{
		r:     rand.New(rand.NewSource(1)),
		depth: 3,
		trace: 0,
	}
}

func (ai *AI) Minimax(pos Position, last BasicAction) (SacrificeAction, float64) {
	t := time.Now()
	ai.visited = 0
	ai.evaluated = 0

	var maxact SacrificeAction
	max := -5.0
	depth := ai.depth
	ply := 1

	acts := pos.BasicActions()
	shuffle(acts, ai.r)
	log.Printf("%d basic actions to examine", len(acts))
	for _, a := range acts {
		tmp := pos.do(a)
		if a.Type() == Attack && a == last {
			fmt.Println("Action returns to an earlier state:", a)
			continue
		}
		//tmp.catastrophes()
		if ai.debug && !tmp.sanityCheck() {
			fmt.Println("last action:", a)
			continue
		}
		tmp.endturn()
		v := -ai.minimax(tmp, pos, ply+1, depth-1, -max)
		//fmt.Printf("%d %c %f\n", depth, "+-"[pos.player], v)
		if ply <= ai.trace {
			log.Printf("%*s player=%d ply=%d depth=%d v=%f min= max=%f move=%s", ply, "", pos.CurrentPlayer(), ply, depth, v, max, a)
		}
		if v > max {
			max = v
			maxact = a.Action()
		}
	}

	var sg sacrificeGenerator
	sacts := sg.Generate(pos)
	sshuffle(sacts, ai.r)
	log.Printf("%d sacrifice actions to examine", len(sacts))
	log.Printf("reduced %d actions to %d", len(sg.acts), len(sacts))
	for _, a := range sacts {
		tmp := pos.do(a.Basic())
		for i := 0; i < a.N(); i++ {
			tmp = tmp.do(a.actions[i])
		}
		tmp.endturn()
		v := -ai.minimax(tmp, pos, ply+1, depth-1, -max)
		if ply <= ai.trace {
			log.Printf("%*s player=%d ply=%d depth=%d v=%f min= max=%f move=%s", ply, "", pos.CurrentPlayer(), ply, depth, v, max, a.Basic())
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

func (ai *AI) minimax(pos, last Position, ply, depth int, min float64) float64 {
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
	// basic actions
	acts := pos.BasicActions()
	shuffle(acts, ai.r)
	for _, a := range acts {
		tmp := pos.do(a)
		if a.Type() == Attack && tmp.Equal(last) {
			if ply == 2 {
				fmt.Println("action returns to an earlier state:", a)
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
		v := -ai.minimax(tmp, pos, ply+1, depth-1, -max)
		if ply <= ai.trace {
			log.Printf("%*s player=%d ply=%d depth=%d v=%f min=%f max=%f move=%s", ply, "", pos.CurrentPlayer(), ply, depth, v, min, max, a)
		}
		if v > max {
			max = v
		}
		if max >= min {
			return max
		}
	}
	// sacrifice actions
	sacts := pos.SacrificeActions()
	//sshuffle(sacts, ai.r)
	for _, sa := range sacts {
		tmp := pos.do(sa.Basic())
		for i := 0; i < sa.N(); i++ {
			tmp = tmp.do(sa.actions[i])
		}
		tmp.endturn()
		v := -ai.minimax(tmp, pos, ply+1, depth-1, -max)
		if ply <= ai.trace {
			log.Printf("%*s player=%d ply=%d depth=%d v=%f min=%f max=%f move=%s", ply, "", pos.CurrentPlayer(), ply, depth, v, min, max, sa.Basic())
		}
		if v > max {
			max = v
		}
		if max >= min {
			return max
		}
	}
	return max
}

func (pos Position) over() bool {
	return pos.stars[North].ships[North].IsEmpty() || pos.stars[South].ships[South].IsEmpty()
}

func (pos *Position) endturn() {
	pos.player ^= 1
}

func (sa SacrificeAction) Basic() BasicAction {
	return BasicAction{typ: sa.typ, ship: sa.ship, system: sa.system, arg: sa.arg}
}

func (sa SacrificeAction) Action(i int) BasicAction {
	return sa.actions[i]
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

func sshuffle(acts []SacrificeAction, r *rand.Rand) {
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

func (sa SacrificeAction) String() string {
	s := ""
	s += sa.Basic().String()
	if sa.Type() == Sacrifice {
		for i := 0; i < sa.N(); i++ {
			s += ", " + sa.Action(i).String()
		}
	}
	return s
}
