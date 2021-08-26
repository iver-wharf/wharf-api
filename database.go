package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ghodss/yaml"
	"github.com/iver-wharf/wharf-core/pkg/gormutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func (p *Project) marshalJSON() ([]byte, error) {
	type Alias Project
	return json.Marshal(&struct {
		ParsedBuildDefinition interface{} `json:"build"`
		*Alias
	}{
		ParsedBuildDefinition: parseBuildDefinition(p),
		Alias:                 (*Alias)(p),
	})
}

func parseBuildDefinition(project *Project) interface{} {
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

func (b *Build) marshalJSON() ([]byte, error) {
	type Alias Build
	return json.Marshal(&struct {
		Status string `json:"status"`
		*Alias
	}{
		Status: b.StatusID.String(),
		Alias:  (*Alias)(b),
	})
}

func openDatabase(config DBConfig) (*gorm.DB, error) {
	const retryDelay = 2 * time.Second
	const maxAttempts = 3
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		config.Host,
		config.Port,
		config.Username,
		config.Password)

	var gormConfig = gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: getLogger(config),
	}

	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err = gorm.Open(postgres.Open(psqlInfo), &gormConfig)
		if err == nil {
			break
		}
		if attempt < maxAttempts {
			log.Warn().
				WithError(err).
				WithInt("attempt", attempt).
				WithInt("maxAttempts", maxAttempts).
				WithDuration("retryAfter", retryDelay).
				Message("Failed attempt to reach database.")
			time.Sleep(retryDelay)
		} else {
			log.Warn().
				WithError(err).
				WithInt("maxAttempts", maxAttempts).
				Message("Failed all attempts to reach database.")
		}
	}
	if err != nil {
		return db, err
	}

	db.Exec(fmt.Sprintf("CREATE DATABASE %s;", config.Name))

	psqlInfo = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Host,
		config.Port,
		config.Username,
		config.Password,
		config.Name)

	db, err = gorm.Open(postgres.Open(psqlInfo), &gormConfig)
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

func getLogger(config DBConfig) logger.Interface {
	if config.Log {
		return gormutil.DefaultLogger
	}
	return logger.Default.LogMode(logger.Silent)
}
