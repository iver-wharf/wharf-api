package orderby

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_ErrorPath(t *testing.T) {
	fieldToColumnNames := map[string]string{
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
	fieldToColumnNames := map[string]string{
		"buildId": "build_id",
	}
	testCases := []struct {
		name     string
		input    string
		namesMap map[string]string
		want     OrderBy
	}{
		{
			name:     "valid mapped asc",
			input:    "buildId asc",
			namesMap: fieldToColumnNames,
			want:     OrderBy{"build_id", Asc},
		},
		{
			name:     "valid mapped desc",
			input:    "buildId desc",
			namesMap: fieldToColumnNames,
			want:     OrderBy{"build_id", Desc},
		},
		{
			name:     "valid unmapped asc",
			input:    "buildId asc",
			namesMap: nil,
			want:     OrderBy{"buildId", Asc},
		},
		{
			name:     "valid unmapped desc",
			input:    "buildId desc",
			namesMap: nil,
			want:     OrderBy{"buildId", Desc},
		},
		{
			name:     "separated by tab",
			input:    "buildId\tdesc",
			namesMap: nil,
			want:     OrderBy{"buildId", Desc},
		},
		{
			name:     "excess whitespace",
			input:    "   \t\t  buildId \t  \tdesc  \t  ",
			namesMap: fieldToColumnNames,
			want:     OrderBy{"build_id", Desc},
		},
		{
			name:     "excess values",
			input:    "buildId desc these values will be ignored",
			namesMap: fieldToColumnNames,
			want:     OrderBy{"build_id", Desc},
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
