package set_test

import (
	"fmt"

	"github.com/iver-wharf/wharf-api/internal/set"
)

func ExampleStrings_loop() {
	s := set.NewStrings("a", "b", "c")

	for value := range s {
		fmt.Println("Value:", value)
	}

	// Unordered output:
	// Value: a
	// Value: b
	// Value: c
}
