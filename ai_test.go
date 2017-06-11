package homeworlds

import (
	"fmt"
	"os"
	"testing"
	"unsafe"
)

func TestBasicActions(t *testing.T) {
	for _, a := range game.BasicActions() {
		fmt.Println(a)
	}
}

func BenchmarkActions(b *testing.B) {
	pos := PositionFromGame(game)
	for i := 0; i < b.N; i++ {
		_ = pos.BasicActions()
	}
}

func TestSacrificeActions(t *testing.T) {
	Print(os.Stdout, game)
	for _, sa := range game.SacrificeActions() {
		fmt.Println(sa)
	}
	fmt.Println()

	tmp := game.Copy()
	tmp.EndTurn()
	acts := tmp.SacrificeActions()
	var size uintptr
	size = uintptr(len(acts)) * unsafe.Sizeof(acts[0])
	for _, sa := range acts {
		fmt.Println(sa)
		size += uintptr(len(sa.Actions)) * unsafe.Sizeof(sa.Actions[0])
	}
	fmt.Println("size: ", size)
}

func BenchmarkSacrificeActionsNorth(b *testing.B) {
	game := game.Copy()
	for i := 0; i < b.N; i++ {
		_ = game.SacrificeActions()
	}
}

func BenchmarkSacrificeActionsSouth(b *testing.B) {
	game := game.Copy()
	game.EndTurn()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = game.SacrificeActions()
	}
}
