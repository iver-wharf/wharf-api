package main

import (
	"errors"
	"os"

	"github.com/iver-wharf/messagebus-go"
)

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

	if conf.Pass, ok = os.LookupEnv("RABBITMQPASS"); !ok {
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
		conf.Pass, conf.QueueName, conf.VHost, conf.DisableSSL, conf.ConnAttempts)
}
