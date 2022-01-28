package main

import (
	"os"
	"strings"
	"testing"

	"github.com/iver-wharf/wharf-core/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_CIEngineFromMultipleSources(t *testing.T) {
	var configYAML = `
ci:
  engines:
    - name: My Jenkins
`
	os.Setenv("TEST_CI_ENGINES_0_TRIGGERTOKEN", "somesecret")
	defer os.Unsetenv("TEST_CI_ENGINES_0_TRIGGERTOKEN")

	var builder = config.NewBuilder(DefaultConfig)
	builder.AddConfigYAML(strings.NewReader(configYAML))
	builder.AddEnvironmentVariables("TEST")
	var config Config
	err := builder.Unmarshal(&config)
	require.NoError(t, err)

	assert.Len(t, config.CI.Engines, 1)
	assert.Equal(t, "My Jenkins", config.CI.Engines[0].Name)
	assert.Equal(t, "somesecret", config.CI.Engines[0].TriggerToken)
}
