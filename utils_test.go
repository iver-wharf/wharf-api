package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProjectGroupFromGitURL(t *testing.T) {
	type testCase struct {
		gitURL      string
		projectName string
	}
	tests := []struct {
		args testCase
		want string
	}{
		{
			args: testCase{
				gitURL:      "git@gitlab.somedomain:default/marek-test-proj.git",
				projectName: "marek-test-proj",
			},
			want: "default",
		},
		{
			args: testCase{
				gitURL:      "git@gitlab.somedomain:default/nestedgroup/projname.git",
				projectName: "projname",
			},
			want: "default/nestedgroup",
		},
		{
			args: testCase{
				gitURL:      "git@gitlab.somedomain:default/nestedgroup/projname.git",
				projectName: "Projname",
			},
			want: "default/nestedgroup",
		},
		{
			args: testCase{
				gitURL:      "git@gitlab.somedomain:default/other/nestedgroup/name.with.dots.git",
				projectName: "name.with.dots",
			},
			want: "default/other/nestedgroup",
		},
		{
			args: testCase{
				gitURL:      "git@gitlab.somedomain:default/Project.git",
				projectName: "Project",
			},
			want: "default",
		},
		{
			args: testCase{
				gitURL:      "git@gitlab.somedomain:default/group/project-with-dash.git",
				projectName: "project-with-dash",
			},
			want: "default/group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.args.gitURL, func(t *testing.T) {
			got := getProjectGroupFromGitURL(tt.args.gitURL, tt.args.projectName)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
		Branch{Name: "b1"},
		Branch{Name: "b2"},
		Branch{Name: "b3"},
		Branch{Name: "b4"},
	}

	_, ok := findDefaultBranch(branches)

	assert.False(t, ok)
}
