package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestProjectMarshalJSON(t *testing.T) {
	type testCase struct {
		name        string
		environment null.String
		want        bool
	}

	project := Project{
		ProjectID:   1,
		Name:        "cool-project",
		GroupName:   "cool-group",
		Description: "Cool description.",
		AvatarURL:   "https://not-a-real-url.example/",
		TokenID:     2,
		ProviderID:  3,
		Provider: &Provider{
			ProviderID: 3,
			Name:       "github",
			URL:        "https://api.github.com",
			UploadURL:  "",
			TokenID:    2,
		},
		BuildDefinition: `build:
  api:
    docker:
      file: Dockerfile
      tag: ${GIT_COMMIT},${GIT_TAG},latest
      append-cert: false
      args:
        - BUILD_VERSION=${GIT_TAG}
        - BUILD_GIT_COMMIT=${GIT_COMMIT}
        - BUILD_REF=${BUILD_REF}`,
		Branches: []Branch{
			{
				BranchID:  1,
				ProjectID: 1,
				Project:   nil,
				Name:      "master",
				Default:   true,
				TokenID:   2,
				Token: Token{
					TokenID:  2,
					Token:    "token_string",
					UserName: "username",
				},
			},
		},
		GitURL: "git@github.somedomain:default/nestedgroup/projname.git",
	}
	want := `{
		"projectId": 1,
		"name": "cool-project",
		"groupName": "cool-group",
		"description": "Cool description.",
		"avatarUrl": "https://not-a-real-url.example/",
		"tokenId": 2,
		"providerId": 3,
		"provider": {
			"providerId": 3,
			"name": "github",
			"url": "https://api.github.com",
			"uploadUrl": "",
			"tokenId": 2
		},
		"buildDefinition": "build:\n  api:\n    docker:\n      file: Dockerfile\n      tag: ${GIT_COMMIT},${GIT_TAG},latest\n      append-cert: false\n      args:\n        - BUILD_VERSION=${GIT_TAG}\n        - BUILD_GIT_COMMIT=${GIT_COMMIT}\n        - BUILD_REF=${BUILD_REF}",
		"branches": [
			{
				"branchId": 1,
				"projectId": 1,
				"name": "master",
				"default": true,
				"tokenId": 2
			}
		],
		"gitUrl": "git@github.somedomain:default/nestedgroup/projname.git",
		"build": {
			"build": {
				"api": {
					"docker": {
						"append-cert": false,
						"args": [
							"BUILD_VERSION=${GIT_TAG}",
							"BUILD_GIT_COMMIT=${GIT_COMMIT}",
							"BUILD_REF=${BUILD_REF}"
						],
						"file": "Dockerfile",
						"tag": "${GIT_COMMIT},${GIT_TAG},latest"
					}
				}
			}
		}
	}`

	got, err := json.Marshal(&project)
	require.Nil(t, err)
	assert.JSONEq(t, want, string(got))
}
