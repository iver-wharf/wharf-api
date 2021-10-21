package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCaseStrings struct {
	name string
	// left hand side of the set operator
	left Strings
	// right hand side of the set operator
	right Strings
	want  Strings
}

func TestStringsDifference(t *testing.T) {
	testCases := []testCaseStrings{
		{
			name:  "both empty",
			left:  NewStrings(),
			right: NewStrings(),
			want:  NewStrings(),
		},
		{
			name:  "empty right",
			left:  NewStrings("a", "b", "c"),
			right: NewStrings(),
			want:  NewStrings("a", "b", "c"),
		},
		{
			name:  "empty left",
			left:  NewStrings(),
			right: NewStrings("a", "b", "c"),
			want:  NewStrings(),
		},
		{
			name:  "ignores missing",
			left:  NewStrings("a", "b", "c"),
			right: NewStrings("d", "e"),
			want:  NewStrings("a", "b", "c"),
		},
		{
			name:  "removes",
			left:  NewStrings("a", "b", "c"),
			right: NewStrings("a"),
			want:  NewStrings("b", "c"),
		},
	}
	runTestCaseStringsSlice(t, testCases, func(left, right Strings) Strings {
		return left.Difference(right)
	})
}

func TestStringsUnion(t *testing.T) {
	testCases := []testCaseStrings{
		{
			name:  "both empty",
			left:  NewStrings(),
			right: NewStrings(),
			want:  NewStrings(),
		},
		{
			name:  "empty right",
			left:  NewStrings("a", "b", "c"),
			right: NewStrings(),
			want:  NewStrings("a", "b", "c"),
		},
		{
			name:  "empty left",
			left:  NewStrings(),
			right: NewStrings("a", "b", "c"),
			want:  NewStrings("a", "b", "c"),
		},
		{
			name:  "ignores collisions",
			left:  NewStrings("a", "b", "c"),
			right: NewStrings("a", "b", "c"),
			want:  NewStrings("a", "b", "c"),
		},
		{
			name:  "unions",
			left:  NewStrings("a", "b"),
			right: NewStrings("c", "d"),
			want:  NewStrings("a", "b", "c", "d"),
		},
	}
	runTestCaseStringsSlice(t, testCases, func(left, right Strings) Strings {
		return left.Union(right)
	})
}

func TestStringsIntersect(t *testing.T) {
	testCases := []testCaseStrings{
		{
			name:  "both empty",
			left:  NewStrings(),
			right: NewStrings(),
			want:  NewStrings(),
		},
		{
			name:  "empty right",
			left:  NewStrings("a", "b", "c"),
			right: NewStrings(),
			want:  NewStrings(),
		},
		{
			name:  "empty left",
			left:  NewStrings(),
			right: NewStrings("a", "b", "c"),
			want:  NewStrings(),
		},
		{
			name:  "all",
			left:  NewStrings("a", "b", "c"),
			right: NewStrings("a", "b", "c"),
			want:  NewStrings("a", "b", "c"),
		},
		{
			name:  "intersect",
			left:  NewStrings("a", "b"),
			right: NewStrings("b", "c"),
			want:  NewStrings("b"),
		},
	}
	runTestCaseStringsSlice(t, testCases, func(left, right Strings) Strings {
		return left.Intersect(right)
	})
}

func runTestCaseStringsSlice(t *testing.T, testCases []testCaseStrings, f func(left, right Strings) Strings) {
	for _, tc := range testCases {
		runTestCaseStrings(t, tc, f)
	}
}

func runTestCaseStrings(t *testing.T, tc testCaseStrings, f func(left, right Strings) Strings) {
	t.Run(tc.name, func(t *testing.T) {
		leftLen := len(tc.left)
		rightLen := len(tc.right)

		got := f(tc.left, tc.right)
		assert.Equal(t, tc.want, got)

		assert.Len(t, tc.left, leftLen, "LEFT was changed")
		assert.Len(t, tc.right, rightLen, "RIGHT was changed")
	})
}
