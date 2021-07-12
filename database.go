package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ghodss/yaml"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	MaxIdleConns    int
	MaxOpenConns    int
	MaxConnLifetime time.Duration
	Log             bool
}

func (p *Project) MarshalJSON() ([]byte, error) {
	type Alias Project
	return json.Marshal(&struct {
		ParsedBuildDefinition interface{} `json:"build"`
		*Alias
	}{
		ParsedBuildDefinition: ParseBuildDefinition(p),
		Alias:                 (*Alias)(p),
	})
}

func ParseBuildDefinition(project *Project) interface{} {
	if project.BuildDefinition != "" {
		var t interface{}
		err := yaml.Unmarshal([]byte(project.BuildDefinition), &t)
		if err != nil {
			log.Error().
				WithError(err).
				WithUint("project", project.ProjectID).
				Message("Failed to parse build-definition.")
			return nil
		}

		return convert(t)
	}

	return nil
}

func (b *Build) MarshalJSON() ([]byte, error) {
	type Alias Build
	return json.Marshal(&struct {
		Status string `json:"status"`
		*Alias
	}{
		Status: b.StatusID.String(),
		Alias:  (*Alias)(b),
	})
}

func openDatabase(config DatabaseConfig) (*gorm.DB, error) {
	const retryDelay = 2 * time.Second
	const maxAttempts = 3
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		config.Host,
		config.Port,
		config.User,
		config.Password)

	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err = gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
		if err == nil {
			break
		}
		log.Error().
			WithError(err).
			WithInt("attempt", attempt).
			WithInt("maxAttempts", maxAttempts).
			WithDuration("retryAfter", retryDelay).
			Message("Error connecting to database.")
		time.Sleep(retryDelay)
	}
	if err != nil {
		return db, err
	}

	db.Exec(fmt.Sprintf("CREATE DATABASE %s;", config.Name))

	psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Name)

	db, err = gorm.Open(postgres.Open(psqlInfo), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: getLogger(config),
	})
	if err != nil {
		return db, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return db, err
	}

	log.Debug().
		WithInt("maxIdleConns", config.MaxIdleConns).
		WithInt("maxOpenConns", config.MaxOpenConns).
		WithDuration("maxConnLifetime", config.MaxConnLifetime).
		Message("Setting database config.")
	sqlDb.SetMaxIdleConns(config.MaxIdleConns)
	sqlDb.SetMaxOpenConns(config.MaxOpenConns)
	sqlDb.SetConnMaxLifetime(config.MaxConnLifetime)

	err = sqlDb.Ping()
	return db, err
}

func getLogger(config DatabaseConfig) logger.Interface {
	if config.Log {
		return logger.Default.LogMode(logger.Info)
	}
	return logger.Default.LogMode(logger.Silent)
}

func getDatabaseConfigFromEnvironment() (DatabaseConfig, error) {
	conf := DatabaseConfig{}
	var ok bool

	conf.Host, ok = os.LookupEnv("DBHOST")
	if !ok {
		return conf, errors.New("DBHOST environment variable required but not set")
	}

	conf.Port, ok = os.LookupEnv("DBPORT")
	if !ok {
		return conf, errors.New("DBPORT environment variable required but not set")
	}

	conf.User, ok = os.LookupEnv("DBUSER")
	if !ok {
		return conf, errors.New("DBUSER environment variable required but not set")
	}

	conf.Password, ok = os.LookupEnv("DBPASS")
	if !ok {
		return conf, errors.New("DBPASS environment variable required but not set")
	}

	conf.Name, ok = os.LookupEnv("DBNAME")
	if !ok {
		return conf, errors.New("DBNAME environment variable required but not set")
	}

	var err error
	maxIdle, ok := os.LookupEnv("DBMAXIDLECONNS")
	if ok {
		conf.MaxIdleConns, err = strconv.Atoi(maxIdle)
		if err != nil {
			return conf, err
		}
	} else {
		conf.MaxIdleConns = 2 // Current default in sql package according to docs https://golang.org/pkg/database/sql/#DB.SetMaxIdleConns
	}

	maxOpen, ok := os.LookupEnv("DBMAXOPENCONNS")
	if ok {
		conf.MaxOpenConns, err = strconv.Atoi(maxOpen)
		if err != nil {
			return conf, err
		}
	} else {
		conf.MaxOpenConns = 0 // Current default in sql package according to docs https://golang.org/pkg/database/sql/#DB.SetMaxOpenConns
	}

	maxLifetime, ok := os.LookupEnv("DBMAXCONNLIFETIME")
	if ok {
		conf.MaxConnLifetime, err = time.ParseDuration(maxLifetime)
		if err != nil {
			return conf, err
		}
	} else {
		conf.MaxConnLifetime = 20 * time.Minute
	}

	conf.Log, _ = strconv.ParseBool(os.Getenv("DBLOG"))

	return conf, nil
}
