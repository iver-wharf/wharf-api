package orderby

import (
	"fmt"
)

type Direction byte

const (
	Asc Direction = iota + 1
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
