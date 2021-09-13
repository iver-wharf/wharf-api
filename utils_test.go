package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindDefaultGroupSuccess(t *testing.T) {
	var (
		main = Branch{Name: "main", Default: true}
		b1   = Branch{Name: "b1"}
		b2   = Branch{Name: "b2"}
		b3   = Branch{Name: "b3"}
		b4   = Branch{Name: "b4"}
	)
	tests := []struct {
		name     string
		branches []Branch
	}{
		{
			name:     "at the beginning",
			branches: []Branch{main, b1, b2, b3, b4},
		},
		{
			name:     "in the middle",
			branches: []Branch{b1, b2, main, b3, b4},
		},
		{
			name:     "at the end",
			branches: []Branch{b1, b2, b3, b4, main},
		},
		{
			name:     "multiple",
			branches: []Branch{b1, main, main, b4, main},
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
	branches := []Branch{
		{Name: "b1"},
		{Name: "b2"},
		{Name: "b3"},
		{Name: "b4"},
	}

	_, ok := findDefaultBranch(branches)

	assert.False(t, ok)
}
