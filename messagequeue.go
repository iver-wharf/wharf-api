package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/iver-wharf/messagebus-go"
)

const DefaultConnectionAttempts uint64 = 10

type MQConfig struct {
	Enabled      bool
	Host         string
	Port         string
	User         string
	Password     string
	QueueName    string
	VHost        string
	DisableSSL   bool
	ConnAttempts uint64
}

func getRabbitConfigFromEnvironment() (MQConfig, error) {
	var (
		conf = MQConfig{}
		ok   bool
		err  error
	)

	if conf.Enabled, err = lookupOptionalEnvBool("RABBITMQENABLED", false); err != nil {
		return conf, err
	}

	if !conf.Enabled {
		return conf, nil
	}

	if conf.User, ok = os.LookupEnv("RABBITMQUSER"); !ok {
		return conf, errors.New("RABBITMQUSER environment variable required but not set")
	}

	if conf.Password, ok = os.LookupEnv("RABBITMQPASS"); !ok {
		return conf, errors.New("RABBITMQPASS environment variable required but not set")
	}

	if conf.Host, ok = os.LookupEnv("RABBITMQHOST"); !ok {
		return conf, errors.New("RABBITMQHOST environment variable required but not set")
	}

	if conf.Port, ok = os.LookupEnv("RABBITMQPORT"); !ok {
		return conf, errors.New("RABBITMQPORT environment variable required but not set")
	}

	if conf.VHost, ok = os.LookupEnv("RABBITMQVHOST"); !ok {
		return conf, errors.New("RABBITMQVHOST environment variable required but not set")
	}

	if conf.QueueName, ok = os.LookupEnv("RABBITMQNAME"); !ok {
		return conf, errors.New("RABBITMQNAME environment variable required but not set")
	}

	if conf.DisableSSL, err = lookupOptionalEnvBool("RABBITMQDISABLESSL", false); err != nil {
		return conf, err
	}

	if conf.ConnAttempts, err = lookupOptionalEnvUInt64("RABBITMQCONNATTEMPTS", DefaultConnectionAttempts); err != nil {
		return conf, err
	}

	return conf, nil
}

// lookupOptionalEnvBool returns the parsed environment variable, or the
// fallback argument value if the environment variable was unset or empty.
// Returns an error if it failed to parse the environment variable value as a
// boolean value.
func lookupOptionalEnvBool(name string, fallback bool) (bool, error) {
	if envStr, ok := os.LookupEnv(name); ok {
		if envStr == "" {
			return fallback, nil
		} else if envBool, err := strconv.ParseBool(envStr); err != nil {
			return false, fmt.Errorf("env: %q: unable to parse bool: %q", name, envStr)
		} else {
			return envBool, nil
		}
	}
	return fallback, nil
}

// lookupOptionalEnvUInt64 returns the parsed environment variable, or the
// fallback argument value if the environment variable was unset or empty.
// Returns an error if it failed to parse the environment variable value as an
// unsigned integer.
func lookupOptionalEnvUInt64(name string, fallback uint64) (uint64, error) {
	if envStr, ok := os.LookupEnv(name); ok {
		if envStr == "" {
			return fallback, nil
		} else if envInt, err := strconv.ParseUint(envStr, 10, 64); err != nil {
			return 0, fmt.Errorf("env: %q: unable to parse uint64: %q", name, envStr)
		} else {
			return envInt, nil
		}
	}
	return fallback, nil
}

// GetMQConnection returns the connection, or nil if it either had an error on
// creation together with said error, or nil on both connection and error if
// it was disabled through configuration.
func GetMQConnection() (*messagebus.MQConnection, error) {
	conf, err := getRabbitConfigFromEnvironment()

	if err != nil {
		return nil, err
	}

	if !conf.Enabled {
		return nil, nil
	}

	return messagebus.NewConnection(conf.Host, conf.Port, conf.User,
		conf.Password, conf.QueueName, conf.VHost, conf.DisableSSL, conf.ConnAttempts)
}
