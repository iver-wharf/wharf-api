package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCaseString struct {
	name string
	lhs  String
	rhs  String
	want String
}

func TestStringDifference(t *testing.T) {
	testCases := []testCaseString{
		{
			name: "both empty",
			lhs:  NewString(),
			rhs:  NewString(),
			want: NewString(),
		},
		{
			name: "empty rhs",
			lhs:  NewString("a", "b", "c"),
			rhs:  NewString(),
			want: NewString("a", "b", "c"),
		},
		{
			name: "empty lhs",
			lhs:  NewString(),
			rhs:  NewString("a", "b", "c"),
			want: NewString(),
		},
		{
			name: "ignores missing",
			lhs:  NewString("a", "b", "c"),
			rhs:  NewString("d", "e"),
			want: NewString("a", "b", "c"),
		},
		{
			name: "removes",
			lhs:  NewString("a", "b", "c"),
			rhs:  NewString("a"),
			want: NewString("b", "c"),
		},
	}
	runTestCaseStringSlice(t, testCases, func(lhs, rhs String) String {
		return lhs.Difference(rhs)
	})
}

func TestStringUnion(t *testing.T) {
	testCases := []testCaseString{
		{
			name: "both empty",
			lhs:  NewString(),
			rhs:  NewString(),
			want: NewString(),
		},
		{
			name: "empty rhs",
			lhs:  NewString("a", "b", "c"),
			rhs:  NewString(),
			want: NewString("a", "b", "c"),
		},
		{
			name: "empty lhs",
			lhs:  NewString(),
			rhs:  NewString("a", "b", "c"),
			want: NewString("a", "b", "c"),
		},
		{
			name: "ignores collisions",
			lhs:  NewString("a", "b", "c"),
			rhs:  NewString("a", "b", "c"),
			want: NewString("a", "b", "c"),
		},
		{
			name: "unions",
			lhs:  NewString("a", "b"),
			rhs:  NewString("c", "d"),
			want: NewString("a", "b", "c", "d"),
		},
	}
	runTestCaseStringSlice(t, testCases, func(lhs, rhs String) String {
		return lhs.Union(rhs)
	})
}

func TestStringIntersect(t *testing.T) {
	testCases := []testCaseString{
		{
			name: "both empty",
			lhs:  NewString(),
			rhs:  NewString(),
			want: NewString(),
		},
		{
			name: "empty rhs",
			lhs:  NewString("a", "b", "c"),
			rhs:  NewString(),
			want: NewString(),
		},
		{
			name: "empty lhs",
			lhs:  NewString(),
			rhs:  NewString("a", "b", "c"),
			want: NewString(),
		},
		{
			name: "all",
			lhs:  NewString("a", "b", "c"),
			rhs:  NewString("a", "b", "c"),
			want: NewString("a", "b", "c"),
		},
		{
			name: "intersect",
			lhs:  NewString("a", "b"),
			rhs:  NewString("b", "c"),
			want: NewString("b"),
		},
	}
	runTestCaseStringSlice(t, testCases, func(lhs, rhs String) String {
		return lhs.Intersect(rhs)
	})
}

func runTestCaseStringSlice(t *testing.T, testCases []testCaseString, f func(lhs, rhs String) String) {
	for _, tc := range testCases {
		runTestCaseString(t, tc, f)
	}
}

func runTestCaseString(t *testing.T, tc testCaseString, f func(lhs, rhs String) String) {
	t.Run(tc.name, func(t *testing.T) {
		lhsLen := len(tc.lhs)
		rhsLen := len(tc.rhs)

		got := f(tc.lhs, tc.rhs)
		assert.Equal(t, tc.want, got)

		assert.Len(t, tc.lhs, lhsLen, "LHS was changed")
		assert.Len(t, tc.rhs, rhsLen, "RHS was changed")
	})
}
