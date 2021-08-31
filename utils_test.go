package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	type testCase struct {
		name     string
		buildDef string
		wantJSON string
		wantYML  string
	}

	tests := []testCase{
		{
			name:     "Simple build definition convert test",
			buildDef: "build:\n  foo:\n    docker:\n      file: \"test.txt\"",
			wantJSON: "{\"build\":{\"foo\":{\"docker\":{\"file\":\"test.txt\"}}}}",
			wantYML:  "map[build:map[foo:map[docker:map[file:test.txt]]]]",
		},
		{
			name:     "Build definition convert test",
			buildDef: "build:\n  foo.service:\n    docker:\n      file: src/Foo.Service/Dockerfile\n      tag: ${GIT_COMMIT},latest\n  foo.web:\n    docker:\n      file: src/Foo.Web/Dockerfile\n      tag: ${GIT_COMMIT},latest\n\ndeploy:\n  service:\n    helm:\n      name: foo-service\n      namespace: stage\n      chart: web-app\n      repo: ${CHART_REPO}/library\n      files: [\"deploy/stage-service.yml\"]\n\n  web:\n    helm:\n      name: foo-web\n      namespace: stage\n      chart: web-app\n      repo: ${CHART_REPO}/library\n      files: [\"deploy/stage-web.yml\"]\n\n  elastic:\n    kubectl:\n      namespace: stage\n      file: deploy/elastic.yml\n",
			wantJSON: "{\"build\":{\"foo.service\":{\"docker\":{\"file\":\"src/Foo.Service/Dockerfile\",\"tag\":\"${GIT_COMMIT},latest\"}},\"foo.web\":{\"docker\":{\"file\":\"src/Foo.Web/Dockerfile\",\"tag\":\"${GIT_COMMIT},latest\"}}},\"deploy\":{\"elastic\":{\"kubectl\":{\"file\":\"deploy/elastic.yml\",\"namespace\":\"stage\"}},\"service\":{\"helm\":{\"chart\":\"web-app\",\"files\":[\"deploy/stage-service.yml\"],\"name\":\"foo-service\",\"namespace\":\"stage\",\"repo\":\"${CHART_REPO}/library\"}},\"web\":{\"helm\":{\"chart\":\"web-app\",\"files\":[\"deploy/stage-web.yml\"],\"name\":\"foo-web\",\"namespace\":\"stage\",\"repo\":\"${CHART_REPO}/library\"}}}}",
			wantYML:  "map[build:map[foo.service:map[docker:map[file:src/Foo.Service/Dockerfile tag:${GIT_COMMIT},latest]] foo.web:map[docker:map[file:src/Foo.Web/Dockerfile tag:${GIT_COMMIT},latest]]] deploy:map[elastic:map[kubectl:map[file:deploy/elastic.yml namespace:stage]] service:map[helm:map[chart:web-app files:[deploy/stage-service.yml] name:foo-service namespace:stage repo:${CHART_REPO}/library]] web:map[helm:map[chart:web-app files:[deploy/stage-web.yml] name:foo-web namespace:stage repo:${CHART_REPO}/library]]]]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var ymlInterface interface{}
			err := yaml.Unmarshal([]byte(tc.buildDef), &ymlInterface)
			require.Nil(t, err)
			assert.Equal(t, tc.wantYML, fmt.Sprint(ymlInterface))

			ymlToObj := unmarshalledYAMLToMarshallableJSON(ymlInterface)

			b, err := json.Marshal(ymlToObj)
			require.Nil(t, err)
			assert.Equal(t, tc.wantJSON, string(b))
		})
	}
}

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
