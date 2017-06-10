package homeworlds

import (
	"fmt"
	"testing"
)

func TestActions(t *testing.T) {
	for _, a := range game.Actions() {
		fmt.Println(a)
	}
}
