package main

import (
	"testing"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

const wharfInstanceID = "test"

func TestParseBuildParams(t *testing.T) {
	type testCase struct {
		name     string
		buildID  uint
		buildDef []byte
		params   []byte
		want     []database.BuildParam
	}

	defaultBuildDef := []byte(`
inputs:
- name: message
  type: string
  default: default message string
`)
	buildID := uint(1)

	tests := []testCase{
		{
			name:     "Parse message from input",
			buildID:  buildID,
			buildDef: defaultBuildDef,
			params:   []byte(`{"message":"test"}`),
			want: []database.BuildParam{{
				BuildParamID: 0,
				BuildID:      buildID,
				Name:         "message",
				Value:        "test",
			}},
		},
		{
			name:     "Parse message from default value",
			buildID:  buildID,
			buildDef: defaultBuildDef,
			params:   []byte(`{}`),
			want: []database.BuildParam{{
				BuildParamID: 0,
				BuildID:      buildID,
				Name:         "message",
				Value:        "default message string",
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDBBuildParams(tc.buildID, tc.buildDef, tc.params)
			require.Nil(t, err)
			assert.Equal(t, len(tc.want), len(got))

			for i, param := range tc.want {
				assert.Equal(t, param.Value, got[i].Value)
			}
		})
	}
}

func TestGetParamsWithOptionalEnvironment(t *testing.T) {
	type testCase struct {
		name        string
		environment null.String
		want        bool
	}

	project := database.Project{}
	vars := []database.BuildParam{}

	tests := []testCase{
		{
			name:        "Parse message from input",
			environment: null.StringFrom("build"),
			want:        true,
		},
		{
			name:        "Parse message from default value",
			environment: null.String{},
			want:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			build := database.Build{
				Environment: tc.environment,
			}

			params, err := getDBJobParams(project, build, vars, wharfInstanceID)
			require.Nil(t, err)

			hasEnv := false
			for _, param := range params {
				if param.Name == "ENVIRONMENT" {
					hasEnv = true
					break
				}
			}

			assert.Equal(t, tc.want, hasEnv)
		})
	}
}
