package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/iver-wharf/wharf-core/pkg/config"
)

func buildTestConfig(configYAML string) (Config, error) {
	var builder = config.NewBuilder(DefaultConfig)
	builder.AddEnvironmentVariables("WHARF")
	builder.AddConfigYAML(strings.NewReader(configYAML))
	var config Config
	err := builder.Unmarshal(&config)
	return config, err
}

func ExampleConfig() {
	var configYAML = `
http:
  cors:
    allowAllOrigins: true

db:
  username: postgres
  password: secretpass
`

	// Prefix of WHARF_ must be prepended to all environment variables
	os.Setenv("WHARF_HTTP_BINDADDRESS", "0.0.0.0:8123")

	var config, err = buildTestConfig(configYAML)
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}

	fmt.Println("Allow any CORS?", config.HTTP.CORS.AllowAllOrigins)
	fmt.Println("HTTP bind address:", config.HTTP.BindAddress)
	fmt.Println("DB username:", config.DB.Username)
	fmt.Println("DB password:", config.DB.Password)

	// Output:
	// Allow any CORS? true
	// HTTP bind address: 0.0.0.0:8123
	// DB username: postgres
	// DB password: secretpass
}
