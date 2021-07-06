package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/iver-wharf/wharf-core/pkg/config"
)

// Config holds all configurable settings for wharf-api.
type Config struct {
	HTTP HTTPConfig
	CA   CertConfig
	DB   DBConfig
	MQ   MQConfig
}

// HTTPConfig holds settings for the HTTP server.
type HTTPConfig struct {
	BindAddress string
	AllowCORS   bool
}

// CertConfig holds settings for certificates verification used when talking
// to remote services over HTTPS.
type CertConfig struct {
	Certs string
}

// DBConfig holds settings for connecting to a database, such as credentials and
// hostnames.
type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	MaxIdleConns    int
	MaxOpenConns    int
	MaxConnLifetime time.Duration
	Log             bool
}

// MQConfig holds settings for connecting to a message queue, such as
// credentials and hostnames.
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

// DefaultConfig is the hard-coded default values for wharf-api's configs.
var DefaultConfig = Config{
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

func readConfig() (Config, error) {
	cfgBuilder := config.NewBuilder(DefaultConfig)

	cfgBuilder.AddConfigYAMLFile("/etc/iver-wharf/wharf-api/config.yml")
	cfgBuilder.AddConfigYAMLFile("wharf-api-config.yml")
	if cfgFile, ok := os.LookupEnv("WHARF_CONFIG"); ok {
		cfgBuilder.AddConfigYAMLFile(cfgFile)
	}
	cfgBuilder.AddEnvironmentVariables("WHARF")

	var (
		cfg Config
		err = cfgBuilder.Unmarshal(&cfg)
	)
	if err == nil {
		err = cfg.addBackwardCompatibleConfigs()
	}
	return cfg, err
}

func (cfg *Config) addBackwardCompatibleConfigs() error {
	if err := cfg.HTTP.addOldHTTPConfigEnvVars(); err != nil {
		return err
	}
	if err := cfg.DB.addOldDBConfigEnvVars(); err != nil {
		return err
	}
	return cfg.MQ.addOldMQConfigEnvVars()
}

func (cfg *HTTPConfig) addOldHTTPConfigEnvVars() error {
	if value, ok := os.LookupEnv("BIND_ADDRESS"); ok {
		cfg.BindAddress = value
	}
	if value, ok := os.LookupEnv("ALLOW_CORS"); ok && value == "YES" {
		cfg.AllowCORS = true
	}
	return nil
}

func (cfg *DBConfig) addOldDBConfigEnvVars() error {
	var err error
	if value, ok := os.LookupEnv("DBHOST"); ok {
		cfg.Host = value
	}
	if cfg.Port, err = lookupOptionalEnvInt("DBPort", cfg.Port); err != nil {
		return err
	}
	if value, ok := os.LookupEnv("DBUSER"); ok {
		cfg.User = value
	}
	if value, ok := os.LookupEnv("DBPASS"); ok {
		cfg.Password = value
	}
	if value, ok := os.LookupEnv("DBNAME"); ok {
		cfg.Name = value
	}
	if cfg.MaxIdleConns, err = lookupOptionalEnvInt("DBMAXIDLECONNS", cfg.MaxIdleConns); err != nil {
		return err
	}
	if cfg.MaxOpenConns, err = lookupOptionalEnvInt("DBMAXOPENCONNS", cfg.MaxOpenConns); err != nil {
		return err
	}
	if cfg.MaxConnLifetime, err = lookupOptionalEnvDuration("DBMAXCONNLIFETIME", cfg.MaxConnLifetime); err != nil {
		return err
	}
	if cfg.Log, err = lookupOptionalEnvBool("DBLOG", cfg.Log); err != nil {
		return err
	}
	return nil
}

func (cfg *MQConfig) addOldMQConfigEnvVars() error {
	var err error
	if cfg.Enabled, err = lookupOptionalEnvBool("RABBITMQENABLED", cfg.Enabled); err != nil {
		return err
	}
	if value, ok := os.LookupEnv("RABBITMQUSER"); ok {
		cfg.User = value
	}
	if value, ok := os.LookupEnv("RABBITMQPASS"); ok {
		cfg.Password = value
	}
	if value, ok := os.LookupEnv("RABBITMQHOST"); ok {
		cfg.Host = value
	}
	if value, ok := os.LookupEnv("RABBITMQPORT"); ok {
		cfg.Port = value
	}
	if value, ok := os.LookupEnv("RABBITMQVHOST"); ok {
		cfg.VHost = value
	}
	if value, ok := os.LookupEnv("RABBITMQNAME"); ok {
		cfg.QueueName = value
	}
	if cfg.DisableSSL, err = lookupOptionalEnvBool("RABBITMQDISABLESSL", cfg.DisableSSL); err != nil {
		return err
	}
	if cfg.ConnAttempts, err = lookupOptionalEnvUInt64("RABBITMQCONNATTEMPTS", cfg.ConnAttempts); err != nil {
		return err
	}
	return nil
}

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

func lookupOptionalEnvInt(name string, fallback int) (int, error) {
	if envStr, ok := os.LookupEnv(name); ok {
		if envStr == "" {
			return fallback, nil
		} else if envInt, err := strconv.ParseUint(envStr, 10, strconv.IntSize); err != nil {
			return 0, fmt.Errorf("env: %q: unable to parse int: %q", name, envStr)
		} else {
			return int(envInt), nil
		}
	}
	return fallback, nil
}

func lookupOptionalEnvDuration(name string, fallback time.Duration) (time.Duration, error) {
	if envStr, ok := os.LookupEnv(name); ok {
		if envStr == "" {
			return fallback, nil
		} else if envDuration, err := time.ParseDuration(envStr); err != nil {
			return 0, fmt.Errorf("env: %q: unable to parse int: %q", name, envStr)
		} else {
			return envDuration, nil
		}
	}
	return fallback, nil
}
