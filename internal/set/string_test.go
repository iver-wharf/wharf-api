package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCaseString struct {
	name string
	// left hand side of the set operator
	left String
	// right hand side of the set operator
	right String
	want  String
}

func TestStringDifference(t *testing.T) {
	testCases := []testCaseString{
		{
			name:  "both empty",
			left:  NewString(),
			right: NewString(),
			want:  NewString(),
		},
		{
			name:  "empty right",
			left:  NewString("a", "b", "c"),
			right: NewString(),
			want:  NewString("a", "b", "c"),
		},
		{
			name:  "empty left",
			left:  NewString(),
			right: NewString("a", "b", "c"),
			want:  NewString(),
		},
		{
			name:  "ignores missing",
			left:  NewString("a", "b", "c"),
			right: NewString("d", "e"),
			want:  NewString("a", "b", "c"),
		},
		{
			name:  "removes",
			left:  NewString("a", "b", "c"),
			right: NewString("a"),
			want:  NewString("b", "c"),
		},
	}
	runTestCaseStringSlice(t, testCases, func(left, right String) String {
		return left.Difference(right)
	})
}

func TestStringUnion(t *testing.T) {
	testCases := []testCaseString{
		{
			name:  "both empty",
			left:  NewString(),
			right: NewString(),
			want:  NewString(),
		},
		{
			name:  "empty right",
			left:  NewString("a", "b", "c"),
			right: NewString(),
			want:  NewString("a", "b", "c"),
		},
		{
			name:  "empty left",
			left:  NewString(),
			right: NewString("a", "b", "c"),
			want:  NewString("a", "b", "c"),
		},
		{
			name:  "ignores collisions",
			left:  NewString("a", "b", "c"),
			right: NewString("a", "b", "c"),
			want:  NewString("a", "b", "c"),
		},
		{
			name:  "unions",
			left:  NewString("a", "b"),
			right: NewString("c", "d"),
			want:  NewString("a", "b", "c", "d"),
		},
	}
	runTestCaseStringSlice(t, testCases, func(left, right String) String {
		return left.Union(right)
	})
}

func TestStringIntersect(t *testing.T) {
	testCases := []testCaseString{
		{
			name:  "both empty",
			left:  NewString(),
			right: NewString(),
			want:  NewString(),
		},
		{
			name:  "empty right",
			left:  NewString("a", "b", "c"),
			right: NewString(),
			want:  NewString(),
		},
		{
			name:  "empty left",
			left:  NewString(),
			right: NewString("a", "b", "c"),
			want:  NewString(),
		},
		{
			name:  "all",
			left:  NewString("a", "b", "c"),
			right: NewString("a", "b", "c"),
			want:  NewString("a", "b", "c"),
		},
		{
			name:  "intersect",
			left:  NewString("a", "b"),
			right: NewString("b", "c"),
			want:  NewString("b"),
		},
	}
	runTestCaseStringSlice(t, testCases, func(left, right String) String {
		return left.Intersect(right)
	})
}

func runTestCaseStringSlice(t *testing.T, testCases []testCaseString, f func(left, right String) String) {
	for _, tc := range testCases {
		runTestCaseString(t, tc, f)
	}
}

func runTestCaseString(t *testing.T, tc testCaseString, f func(left, right String) String) {
	t.Run(tc.name, func(t *testing.T) {
		leftLen := len(tc.left)
		rightLen := len(tc.right)

		got := f(tc.left, tc.right)
		assert.Equal(t, tc.want, got)

		assert.Len(t, tc.left, leftLen, "LEFT was changed")
		assert.Len(t, tc.right, rightLen, "RIGHT was changed")
	})
}
