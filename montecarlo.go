package homeworlds

import "math/rand"

type MonteCarloAI struct {
	r *rand.Rand
}

func NewMonteCarloAI() *MonteCarloAI {
	return &MonteCarloAI{r: rand.New(rand.NewSource(1))}
}

func (ai *MonteCarloAI) Go(pos Position) (Action, float64) {
	const nmoves = 100
	const nsims = 500
	const maxdepth = 100
	var best []Action
	max := 0.0
	for i := 0; i < nmoves; i++ {
		acts := pos.BasicActions()
		sacts := pos.SacrificeActions()
		var a Action
		if ai.r.Intn(2) == 0 {
			n := ai.r.Intn(len(acts))
			a = acts[n].Action()
		} else {
			n := ai.r.Intn(len(sacts))
			a = sacts[n]
		}
		tmp := pos.doXXX(a)
		score := 0.0
		for j := 0; j < nsims; j++ {
			_, v := ai.playout(tmp, pos.CurrentPlayer(), maxdepth)
			if v >= 0 {
				score += v
			}
		}

		if score > max {
			max = score
			best = append(best[:0], a)
		}
	}
	return best[0], float64(max)
}

func (ai *MonteCarloAI) playout(pos Position, pl Player, depth int) ([]Action, float64) {
	var moves []Action
	for ply := 0; ply < depth; ply++ {
		if pos.over() {
			break
		}
		acts := pos.BasicActions()
		sacts := pos.SacrificeActions()

		randact := func() Action {
			if len(sacts) == 0 || ai.r.Intn(2) == 0 {
				i := ai.r.Intn(len(acts))
				a := acts[i]
				acts[i] = acts[len(acts)-1]
				acts = acts[:len(acts)-1]
				return a.Action()
			} else {
				i := ai.r.Intn(len(sacts))
				a := sacts[i]
				sacts[i] = sacts[len(sacts)-1]
				sacts = sacts[:len(sacts)-1]
				return a
			}
		}

		a := randact()
		tmp := pos.doXXX(a)
		// assume players are smart enough not to suicide
		for tmp.over() && tmp.winner() != pos.CurrentPlayer() {
			a = randact()
			tmp = pos.doXXX(a)
		}
		moves = append(moves, a)
		pos = tmp
	}
	v := pos.score()
	if pos.CurrentPlayer() != pl {
		v = -v
	}
	return moves, v
}

func (pos Position) allacts() []Action {
	bs := pos.BasicActions()
	ss := pos.SacrificeActions()
	var acts = make([]Action, 0, len(bs)+len(ss))
	for _, a := range bs {
		acts = append(acts, a.Action())
	}
	acts = append(acts, ss...)
	return acts
}
