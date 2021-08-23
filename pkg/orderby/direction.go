package orderby

import (
	"fmt"
)

// Direction tells if an ordering is in ascending order or descending order.
type Direction byte

const (
	// Asc means "ascending sorting order". For example the numbers 1, 2, 3, 4
	// are in ascending order, as well as the letters A, B, C, D.
	Asc Direction = iota + 1
	// Desc means "descending sorting order". For example the numbers 4, 3, 2, 1
	// are in descending order, as well as the letters D, C, B, A.
	Desc
)

// ParseDirection returns true if the direction is ascending ('asc') or false if
// it is descending ('desc'), or an error if neither.
// Valid input values are 'asc' and 'desc'.
func ParseDirection(direction string) (Direction, error) {
	switch direction {
	case "asc":
		return Asc, nil
	case "desc":
		return Desc, nil
	default:
		return Direction(0), fmt.Errorf("invalid direction, only 'asc' or 'desc' supported, but got: %q", direction)
	}
}

func (d Direction) String() string {
	switch d {
	case Asc:
		return "asc"
	case Desc:
		return "desc"
	default:
		panic(fmt.Sprintf("invalid %T value: %d", d, byte(d)))
	}
}
