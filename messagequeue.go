package main

import (
	"github.com/iver-wharf/messagebus-go"
)

// GetMQConnection returns the connection, or nil if it either had an error on
// creation together with said error, or nil on both connection and error if
// it was disabled through configuration.
func GetMQConnection(config MQConfig) (*messagebus.MQConnection, error) {
	if !config.Enabled {
		return nil, nil
	}

	return messagebus.NewConnection(
		config.Host,
		config.Port,
		config.Username,
		config.Password,
		config.QueueName,
		config.VHost,
		config.DisableSSL,
		config.ConnAttempts,
	)
}
