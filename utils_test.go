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

			ymlToObj := convert(ymlInterface)

			b, err := json.Marshal(ymlToObj)
			require.Nil(t, err)
			assert.Equal(t, tc.wantJSON, string(b))
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
