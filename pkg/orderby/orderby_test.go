package orderby

import (
	"testing"

	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_errorOnNilMap(t *testing.T) {
	unwanted, err := Parse("foo asc", nil)
	assert.ErrorIs(t, err, ErrNilParseMap, "unexpected result:", unwanted)
}

func TestParse_ErrorPath(t *testing.T) {
	fieldToColumnNames := map[string]database.SafeSQLName{
		"buildId": "build_id",
	}
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "only space",
			input: "          ",
		},
		{
			name:  "only whitespace",
			input: "   \t\t   \t  \t ",
		},
		{
			name:  "separated by newline",
			input: "buildId\nasc",
		},
		{
			name:  "missing direction",
			input: "buildId ",
		},
		{
			name:  "uppercase direction",
			input: "buildId ASC",
		},
		{
			name:  "invalid direction",
			input: "buildId foo",
		},
		{
			name:  "unknown field name",
			input: "tractorBeam asc",
		},
		{
			name:  "invalid field name casing",
			input: "bUiLdiD asc",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			order, err := Parse(tc.input, fieldToColumnNames)
			assert.NotNil(t, err, "unexpected result: %q", order)
		})
	}
}

func TestParse_HappyPath(t *testing.T) {
	fieldToColumnNames := map[string]database.SafeSQLName{
		"buildId": "build_id",
	}
	testCases := []struct {
		name     string
		input    string
		namesMap map[string]database.SafeSQLName
		want     Column
	}{
		{
			name:     "valid mapped asc",
			input:    "buildId asc",
			namesMap: fieldToColumnNames,
			want:     Column{"build_id", Asc},
		},
		{
			name:     "valid mapped desc",
			input:    "buildId desc",
			namesMap: fieldToColumnNames,
			want:     Column{"build_id", Desc},
		},
		{
			name:     "excess whitespace",
			input:    "   \t\t  buildId \t  \tdesc  \t  ",
			namesMap: fieldToColumnNames,
			want:     Column{"build_id", Desc},
		},
		{
			name:     "excess values",
			input:    "buildId desc these values will be ignored",
			namesMap: fieldToColumnNames,
			want:     Column{"build_id", Desc},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input, tc.namesMap)
			require.Nil(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
