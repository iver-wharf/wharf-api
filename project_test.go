package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

const wharfInstanceID = "test"

type parseBuildParamsTestCase struct {
	name     string
	buildID  uint
	buildDef []byte
	params   []byte
	want     []BuildParam
}

func TestParseBuildParams(t *testing.T) {
	defaultBuildDef := []byte(`
inputs:
- name: message
  type: string
  default: default message string
`)
	buildID := uint(1)

	tests := []parseBuildParamsTestCase{
		{
			name:     "Parse message from input",
			buildID:  buildID,
			buildDef: defaultBuildDef,
			params:   []byte(`{"message":"test"}`),
			want: []BuildParam{{
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
			want: []BuildParam{{
				BuildParamID: 0,
				BuildID:      buildID,
				Name:         "message",
				Value:        "default message string",
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseBuildParams(tc.buildID, tc.buildDef, tc.params)
			require.Nil(t, err)
			assert.Equal(t, len(tc.want), len(got))

			for i, param := range tc.want {
				assert.Equal(t, param.Value, got[i].Value)
			}
		})
	}
}

func TestParseBuildParamsErrorsWhenInvalidInput(t *testing.T) {
	invalidBuildDef := []byte(`
inputs:
- name: tabs instead of spaces are invalid
	type: string
	,invalid
`)
	validBuildDef := []byte(`
inputs:
- name: message
  type: string
  default: default message string
`)
	invalidParams := []byte(`{ var_name_without_quotes: "value" }`)
	validParams := []byte(`{ "var_name_with_quotes": "value" }`)
	buildID := uint(1)

	tests := []parseBuildParamsTestCase{
		{
			name:     "Error when invalid build definition",
			buildID:  buildID,
			buildDef: invalidBuildDef,
			params:   validParams,
			want:     []BuildParam{},
		},
		{
			name:     "Error when invalid params",
			buildID:  buildID,
			buildDef: validBuildDef,
			params:   invalidParams,
			want:     []BuildParam{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseBuildParams(tc.buildID, tc.buildDef, tc.params)
			require.NotNil(t, err)
			assert.Equal(t, len(tc.want), len(got))
		})
	}
}

func TestGetParamsWithOptionalEnvironment(t *testing.T) {
	type testCase struct {
		name        string
		environment null.String
		want        bool
	}

	project := Project{}
	vars := []BuildParam{}

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
			build := Build{
				Environment: tc.environment,
			}

			params, err := getParams(project, build, vars, wharfInstanceID)
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
