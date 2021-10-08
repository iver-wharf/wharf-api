package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/iver-wharf/wharf-core/pkg/gormutil"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var errUnsupportedDBDriver = errors.New("unsupported database driver")

func dbDriverSupportsForeignKeyConstraints(driver DBDriver) bool {
	switch driver {
	case DBDriverSqlite:
		return false
	default:
		return true
	}
}

func openDatabase(config DBConfig) (*gorm.DB, error) {
	log.Info().WithString("driver", string(config.Driver)).Message("Connecting to database.")

	switch config.Driver {
	case DBDriverPostgres:
		return openDatabasePostgres(config)
	case DBDriverSqlite:
		return openDatabaseSqlite(config)
	default:
		return nil, errUnsupportedDBDriver
	}
}

func openDatabaseSqlite(config DBConfig) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(config.Path), os.ModePerm); err != nil {
		return nil, fmt.Errorf("create directories for sqlite file: %w", err)
	}

	var gormConfig = getGormConfig(config)
	return gorm.Open(sqlite.Open(config.Path), &gormConfig)
}

func openDatabasePostgres(config DBConfig) (*gorm.DB, error) {
	const retryDelay = 2 * time.Second
	const maxAttempts = 3
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		config.Host,
		config.Port,
		config.Username,
		config.Password)

	var gormConfig = getGormConfig(config)
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

func getGormConfig(config DBConfig) gorm.Config {
	return gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: getLogger(config),
	}
}

func getLogger(config DBConfig) logger.Interface {
	if config.Log {
		return gormutil.DefaultLogger
	}
	return logger.Default.LogMode(logger.Silent)
}
