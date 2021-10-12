package set

import (
	"fmt"
	"strings"
)

type emptyStruct struct{}

type String map[string]emptyStruct

func (set String) Has(value string) bool {
	_, ok := set[value]
	return ok
}

func (set String) Add(values ...string) {
	for _, value := range values {
		set[value] = emptyStruct{}
	}
}

func (set String) Remove(value string) bool {
	if set.Has(value) {
		delete(set, value)
		return true
	}
	return false
}

func (set String) String() string {
	return fmt.Sprintf("{%s}", strings.Join(set.Slice(), ", "))
}

func (set String) Slice() []string {
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	return result
}

func (set String) Subtract(values String) String {
	result := String{}
	for value := range set {
		if !values.Has(value) {
			result.Add(value)
		}
	}
	return result
}
