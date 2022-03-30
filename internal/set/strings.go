package set

import (
	"fmt"
	"sort"
	"strings"
)

// Strings is a set of strings. The set is case sensitive.
type Strings map[string]emptyStruct

// NewStrings returns a new set of strings with all the given values added.
func NewStrings(values ...string) Strings {
	set := Strings{}
	set.Set(values...)
	return set
}

// Has returns true if the set has the given value.
func (set Strings) Has(value string) bool {
	_, ok := set[value]
	return ok
}

// Set all the given values to the set. Collisions are ignored.
func (set Strings) Set(values ...string) {
	for _, value := range values {
		set[value] = emptyStruct{}
	}
}

// Unset a given value from the set. Returns true if removed, or false if the
// value did not exist in the set.
func (set Strings) Unset(value string) bool {
	if set.Has(value) {
		delete(set, value)
		return true
	}
	return false
}

// String returns a string representation of the set.
func (set Strings) String() string {
	return fmt.Sprintf("{%s}", strings.Join(set.Slice(), ", "))
}

// GoString returns the string representation for debugging use cases, such as
// when printing the value in test results.
func (set Strings) GoString() string {
	values := set.Slice()
	sort.Strings(values)
	valuesInterfaces := make([]any, len(values))
	for i, v := range values {
		valuesInterfaces[i] = v
	}
	return goString(set, valuesInterfaces)
}

// Slice returns the set as a slice.
func (set Strings) Slice() []string {
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	return result
}

// Clone returns a copy of the set.
func (set Strings) Clone() Strings {
	result := Strings{}
	for value := range set {
		result.Set(value)
	}
	return result
}

// Difference returns a new set, where the new set has the values of the original
// set but without the values from the given set.
func (set Strings) Difference(values Strings) Strings {
	result := Strings{}
	for value := range set {
		if !values.Has(value) {
			result.Set(value)
		}
	}
	return result
}

// Union returns a new set, where all the values of the original set and the
// given values are included. Collisions are ignored.
func (set Strings) Union(values Strings) Strings {
	result := set.Clone()
	for value := range values {
		result.Set(value)
	}
	return result
}

// Intersect returns a new set, where only the values that exists in both the
// original set and the given values are incldued.
func (set Strings) Intersect(values Strings) Strings {
	result := Strings{}
	for value := range values {
		if set.Has(value) {
			result.Set(value)
		}
	}
	return result
}
