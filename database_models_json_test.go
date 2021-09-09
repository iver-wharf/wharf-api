package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectBuildDefMarshalJSON(t *testing.T) {
	var testCases = []struct {
		name         string
		inputProject interface{}
		wantBuildDef interface{}
	}{
		{
			name:         "no build def",
			inputProject: Project{},
			wantBuildDef: nil,
		},
		{
			name:         "with build def/value",
			inputProject: Project{BuildDefinition: "myStage: moo"},
			wantBuildDef: map[string]interface{}{
				"myStage": "moo",
			},
		},
		{
			name:         "with build def/ref",
			inputProject: &Project{BuildDefinition: "myStage: moo"},
			wantBuildDef: map[string]interface{}{
				"myStage": "moo",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.inputProject)
			require.NoError(t, err)

			var projectMap map[string]interface{}
			err = json.Unmarshal(b, &projectMap)

			require.NoError(t, err)
			assert.Equal(t, tc.wantBuildDef, projectMap["build"])
		})
	}
}

func TestBuildStatusMarshalJSON(t *testing.T) {
	var testCases = []struct {
		name       string
		inputBuild interface{}
		wantStatus string
	}{
		{
			name:       "invalid status ID",
			inputBuild: Build{StatusID: -15},
			wantStatus: "-15",
		},
		{
			name:       "zero status ID",
			inputBuild: Build{StatusID: 0},
			wantStatus: "Scheduling",
		},
		{
			name:       "non-zero status ID/value",
			inputBuild: Build{StatusID: BuildRunning},
			wantStatus: "Running",
		},
		{
			name:       "non-zero status ID/ref",
			inputBuild: &Build{StatusID: BuildRunning},
			wantStatus: "Running",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.inputBuild)
			require.NoError(t, err)

			var buildMap map[string]interface{}
			err = json.Unmarshal(b, &buildMap)

			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, buildMap["status"])
		})
	}
}
