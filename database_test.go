package main

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonToYaml(t *testing.T) {
	want := "name: test\n"
	b := []byte(`{"name":"test"}`)

	y, err := yaml.JSONToYAML(b)
	require.Nil(t, err)
	got := string(y)

	assert.Equal(t, want, got)
}
