package set_test

import (
	"fmt"

	"github.com/iver-wharf/wharf-api/internal/set"
)

func ExampleString_loop() {
	s := set.NewString("a", "b", "c")

	for value := range s {
		fmt.Println("Value:", value)
	}

	// Unordered output:
	// Value: a
	// Value: b
	// Value: c
}
