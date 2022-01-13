package main

import (
	"testing"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/stretchr/testify/assert"
)

func TestFindDefaultGroupSuccess(t *testing.T) {
	var (
		main = database.Branch{Name: "main", Default: true}
		b1   = database.Branch{Name: "b1"}
		b2   = database.Branch{Name: "b2"}
		b3   = database.Branch{Name: "b3"}
		b4   = database.Branch{Name: "b4"}
	)
	tests := []struct {
		name     string
		branches []database.Branch
	}{
		{
			name:     "at the beginning",
			branches: []database.Branch{main, b1, b2, b3, b4},
		},
		{
			name:     "in the middle",
			branches: []database.Branch{b1, b2, main, b3, b4},
		},
		{
			name:     "at the end",
			branches: []database.Branch{b1, b2, b3, b4, main},
		},
		{
			name:     "multiple",
			branches: []database.Branch{b1, main, main, b4, main},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := findDefaultBranch(tt.branches)
			assert.True(t, ok)
			assert.Equal(t, main, got)
		})
	}
}

func TestFindDefaultGroupFail(t *testing.T) {
	branches := []database.Branch{
		{Name: "b1"},
		{Name: "b2"},
		{Name: "b3"},
		{Name: "b4"},
	}

	_, ok := findDefaultBranch(branches)

	assert.False(t, ok)
}

func TestNewLikeContainsValue(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "%",
		},
		{
			name:  "special chars",
			input: "foo%bar?moo_doo",
			want:  `%foo\%bar\?moo\_doo%`,
		},
		{
			name:  "escape",
			input: `foo\\bar`,
			want:  `%foo\\\\bar%`,
		},
		{
			name:  "escaped special char",
			input: `foo\%bar`,
			want:  `%foo\\\%bar%`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := newLikeContainsValue(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}
