package main

import (
	"os"
	"time"

	"github.com/iver-wharf/wharf-core/pkg/config"
	"github.com/iver-wharf/wharf-core/pkg/env"
)

// Config holds all configurable settings for wharf-api.
//
// The config is read in the following order:
//
// 1. File: /etc/iver-wharf/wharf-api/config.yml
//
// 2. File: ./wharf-api-config.yml
//
// 3. File from environment variable: WHARF_CONFIG
//
// 4. Environment variables, prefixed with WHARF_
//
// Each inner struct is represented as a deeper field in the different
// configurations. For YAML they represent deeper nested maps. For environment
// variables they are joined together by underscores.
//
// All environment variables must be uppercased, while YAML files are
// case-insensitive. Keeping camelCasing in YAML config files is recommended
// for consistency.
type Config struct {
	CI   CIConfig
	HTTP HTTPConfig
	CA   CertConfig
	DB   DBConfig
	MQ   MQConfig

	// InstanceID may be an arbitrary string that is used to identify different
	// Wharf installations from each other. Needed when you use multiple Wharf
	// installations in the same environment, such as the same Kubernetes
	// namespace or the same Jenkins instance, to let Wharf know which builds
	// belong to which Wharf installation.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable WHARF_INSTANCE, which was added in
	// v0.7.9, also sets this value.
	//
	// Added in v4.2.0.
	InstanceID string
}

// CIConfig holds settings for the continuous integration (CI).
type CIConfig struct {
	// TriggerURL is the full URL that wharf-api will send a POST request to
	// with all of the build metadata. For example to trigger a Jenkins job via
	// the "Generic Webhook Trigger":
	// https://plugins.jenkins.io/generic-webhook-trigger
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable CI_URL, which was added in v0.6.0,
	// also sets this value.
	//
	// Added in v4.2.0.
	TriggerURL string

	// TriggerToken is passed along as a credentials token via the "token" query
	// parameter. When using the Jenkins plugin "Generic Webhook Trigger"
	// (https://plugins.jenkins.io/generic-webhook-trigger) then this token is
	// configured in the webhook settings.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable CI_URL, which was added in v0.6.0,
	// also sets this value.
	//
	// Added in v4.2.0.
	TriggerToken string

	// MockTriggerResponse will, when set to true, hinder wharf-api from sending
	// a HTTP POST trigger request when starting a new build and will instead
	// silently act like the build has been successfully scheduled.
	//
	// Useful when running Wharf locally and you want to test the behavior of
	// starting a new build, without actually needing a local Jenkins set up.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable MOCK_LOCAL_CI_RESPONSE, which was
	// added in v0.6.0, also sets this value.
	//
	// Added in v4.2.0.
	MockTriggerResponse bool
}

// HTTPConfig holds settings for the HTTP server.
type HTTPConfig struct {
	CORS CORSConfig

	// BindAddress is the IP-address and port, separated by a colon, to bind
	// the HTTP server to. An IP-address of 0.0.0.0 will bind to all
	// IP-addresses.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable BIND_ADDRESS, which was added in
	// v4.1.0, also sets this value.
	//
	// Added in v4.2.0.
	BindAddress string

	// BasicAuth is a comma-separated list of username:password pairs.
	//
	// Example for user named "admin" with password "1234" and a user named
	// "john" with the password "secretpass":
	// 	BasicAuth="admin:1234,john:secretpass"
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable BASIC_AUTH, which was added in v0.5.5,
	// also sets this value.
	//
	// Added in v4.2.0.
	BasicAuth string
}

// CORSConfig holds settings for the HTTP server's CORS settings.
type CORSConfig struct {
	// AllowAllOrigins enables all
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable ALLOW_CORS, which was added in v0.5.5,
	// when set to "YES" will then set this value to true.
	//
	// Added in v4.2.0.
	AllowAllOrigins bool
}

// CertConfig holds settings for certificates verification used when talking
// to remote services over HTTPS.
type CertConfig struct {
	// CertsFile points to a file of one or more PEM-formatted certificates to
	// use in addition to the certificates from the system
	// (such as from /etc/ssl/certs/).
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable CA_CERTS, which was added in v0.7.5,
	// also sets this value.
	//
	// Added in v4.2.0.
	CertsFile string
}

// DBConfig holds settings for connecting to a database, such as credentials and
// hostnames.
type DBConfig struct {
	// Host is the network hostname wharf-api will connect to.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable DBHOST, which was added in v0.5.5,
	// also sets this value.
	//
	// Added in v4.2.0.
	Host string

	// Port is the network port wharf-api will connect to.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable DBPORT, which was added in v0.5.5,
	// also sets this value.
	//
	// Added in v4.2.0.
	Port int

	// Username is the username part of credentials used when connecting to the
	// database.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable DBUSER, which was added in v0.5.5,
	// also sets this value.
	//
	// Added in v4.2.0.
	Username string

	// Password is the username part of credentials used when connecting to the
	// database.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable DBPASS, which was added in v0.5.5,
	// also sets this value.
	//
	// Added in v4.2.0.
	Password string

	// Name is the database name that wharf-api will store its data in. Some
	// databases also call this the "schema" name.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable DBNAME, which was added in v0.5.5,
	// also sets this value.
	//
	// Added in v4.2.0.
	Name string

	// MaxIdleConns is the maximum number of idle connections that wharf-api
	// will keep alive.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable DBMAXIDLECONNS, which was added in
	// v0.5.5, also sets this value.
	//
	// Added in v4.2.0.
	MaxIdleConns int

	// MaxOpenConns is the maximum number of open connections that wharf-api
	// will use at a single point in time.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable DBMAXOPENCONNS, which was added in
	// v0.5.5, also sets this value.
	//
	// Added in v4.2.0.
	MaxOpenConns int

	// MaxConnLifetime is the maximum age for a given database connection. If
	// any connection exceeds this limit, while not in use, it will be
	// disconnected.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable DBMAXCONNLIFETIME, which was added in
	// v0.5.5, also sets this value.
	//
	// Added in v4.2.0.
	MaxConnLifetime time.Duration

	// Log enables/disables database SQL query logging.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable DBLOG, which was added in v0.5.5,
	// also sets this value.
	//
	// Added in v4.2.0.
	Log bool
}

// MQConfig holds settings for connecting to a message queue
// (ex: AMQP/RabbitMQ), such as credentials and hostnames.
type MQConfig struct {
	// Enabled controls whether the message queue integration is turned
	// on or off.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable RABBITMQENABLED, which was added in
	// v3.0.0, also sets this value.
	//
	// Added in v4.2.0
	Enabled bool

	// Host is the network hostname wharf-api will connect to.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable RABBITMQHOST, which was added in
	// v0.7.9, also sets this value.
	//
	// Added in v4.2.0
	Host string

	// Host is the network port wharf-api will connect to.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable RABBITMQPORT, which was added in
	// v0.7.9, also sets this value.
	//
	// Added in v4.2.0
	Port string

	// Username is the username part of credentials used when connecting to the
	// message queue instance (usually RabbitMQ).
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable RABBITMQUSER, which was added in
	// v0.7.9, also sets this value.
	//
	// Added in v4.2.0.
	Username string

	// Password is the password part of credentials used when connecting to the
	// message queue instance (usually RabbitMQ).
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable RABBITMQPASS, which was added in
	// v0.7.9, also sets this value.
	//
	// Added in v4.2.0.
	Password string

	// QueueName is the name of the AMQP message queue that wharf-api will use.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable RABBITMQNAME, which was added in
	// v0.7.9, also sets this value.
	//
	// Added in v4.2.0.
	QueueName string

	// VHost is the name of the AMQP virtual host that wharf-api will use.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable RABBITMQVHOST, which was added in
	// v0.7.9, also sets this value.
	//
	// Added in v4.2.0.
	VHost string

	// DisableSSL will make wharf-api connect to the message queue service via
	// AMQP when set to true, and AMQPS when set to false.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable RABBITMQDISABLESSL, which was added in
	// v0.7.9, also sets this value.
	//
	// Added in v4.2.0.
	DisableSSL bool

	// ConnAttempts sets the number of connection attempts when setting up the
	// initial AMQP connection. If all those attempts fail, then the wharf-api
	// application will exit.
	//
	// For backward compatibility, that may be removed in the next major release
	// (v5.0.0), the environment variable RABBITMQCONNATTEMPTS, which was added
	// in v0.7.9, also sets this value.
	//
	// Added in v4.2.0.
	ConnAttempts uint64
}

// DefaultConfig is the hard-coded default values for wharf-api's configs.
var DefaultConfig = Config{
	HTTP: HTTPConfig{
		BindAddress: "0.0.0.0:8080",
	},
	DB: DBConfig{
		// Current default in sql package according to docs
		// https://golang.org/pkg/database/sql/#DB.SetMaxIdleConns
		MaxIdleConns: 2,
		// Current default in sql package according to docs
		// https://golang.org/pkg/database/sql/#DB.SetMaxOpenConns
		MaxOpenConns:    0,
		MaxConnLifetime: 20 * time.Minute,
	},
	MQ: MQConfig{
		ConnAttempts: 10,
	},
}

func loadConfig() (Config, error) {
	cfgBuilder := config.NewBuilder(DefaultConfig)

	cfgBuilder.AddConfigYAMLFile("/etc/iver-wharf/wharf-api/config.yml")
	cfgBuilder.AddConfigYAMLFile("wharf-api-config.yml")
	if cfgFile, ok := os.LookupEnv("WHARF_CONFIG"); ok {
		cfgBuilder.AddConfigYAMLFile(cfgFile)
	}
	cfgBuilder.AddEnvironmentVariables("WHARF")

	var cfg Config
	err := cfgBuilder.Unmarshal(&cfg)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.addBackwardCompatibleConfigs(); err != nil {
		return Config{}, err
	}
	return cfg, err
}

func (cfg *Config) addBackwardCompatibleConfigs() error {
	if value, ok := os.LookupEnv("ALLOW_CORS"); ok && value == "YES" {
		cfg.HTTP.CORS.AllowAllOrigins = true
	}
	return env.BindMultiple(map[interface{}]string{
		&cfg.CI.MockTriggerResponse: "MOCK_LOCAL_CI_RESPONSE",
		&cfg.CI.TriggerToken:        "CI_TOKEN",
		&cfg.CI.TriggerURL:          "CI_URL",
		&cfg.HTTP.BasicAuth:         "BASIC_AUTH",
		&cfg.HTTP.BindAddress:       "BIND_ADDRESS",
		&cfg.CA.CertsFile:           "CA_CERTS",
		&cfg.DB.Host:                "DBHOST",
		&cfg.DB.Log:                 "DBLOG",
		&cfg.DB.MaxConnLifetime:     "DBMAXCONNLIFETIME",
		&cfg.DB.MaxIdleConns:        "DBMAXIDLECONNS",
		&cfg.DB.MaxOpenConns:        "DBMAXOPENCONNS",
		&cfg.DB.Name:                "DBNAME",
		&cfg.DB.Password:            "DBPASS",
		&cfg.DB.Port:                "DBPORT",
		&cfg.DB.Username:            "DBUSER",
		&cfg.MQ.ConnAttempts:        "RABBITMQCONNATTEMPTS",
		&cfg.MQ.DisableSSL:          "RABBITMQDISABLESSL",
		&cfg.MQ.Enabled:             "RABBITMQENABLED",
		&cfg.MQ.Host:                "RABBITMQHOST",
		&cfg.MQ.Password:            "RABBITMQPASS",
		&cfg.MQ.Port:                "RABBITMQPORT",
		&cfg.MQ.QueueName:           "RABBITMQNAME",
		&cfg.MQ.Username:            "RABBITMQUSER",
		&cfg.MQ.VHost:               "RABBITMQVHOST",
		&cfg.InstanceID:             "WHARF_INSTANCE",
	})
}
