package modelconv

import (
	"testing"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/stretchr/testify/assert"
)

func TestDBProjectToResponseBuildDefParsing(t *testing.T) {
	var testCases = []struct {
		name      string
		dbProject database.Project
		want      interface{}
	}{
		{
			name:      "no build def",
			dbProject: database.Project{},
			want:      nil,
		},
		{
			name:      "with build def",
			dbProject: database.Project{BuildDefinition: "myStage: moo"},
			want: map[string]interface{}{
				"myStage": "moo",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resProject := DBProjectToResponse(tc.dbProject)
			assert.Equal(t, tc.want, resProject.ParsedBuildDefinition)
		})
	}
}
