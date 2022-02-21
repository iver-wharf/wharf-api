package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/iver-wharf/wharf-core/pkg/cacertutil"
	"github.com/iver-wharf/wharf-core/pkg/logger"
	"github.com/iver-wharf/wharf-core/pkg/logger/consolepretty"
	"github.com/soheilhy/cmux"
	"gorm.io/gorm"

	"github.com/iver-wharf/wharf-api/v5/docs"
)

var log = logger.NewScoped("WHARF")

// @title Wharf main API
// @description Wharf backend API that manages data storage for projects,
// @description builds, providers, etc.
// @license.name MIT
// @license.url https://github.com/iver-wharf/wharf-api/blob/master/LICENSE
// @contact.name Iver Wharf API support
// @contact.url https://github.com/iver-wharf/wharf-api/issues
// @contact.email wharf@iver.se
// @basePath /api
// @query.collection.format multi
func main() {
	logger.AddOutput(logger.LevelDebug, consolepretty.Default)
	var (
		config Config
		err    error
	)
	if err = loadEmbeddedVersionFile(); err != nil {
		log.Error().WithError(err).Message("Failed to read embedded version.yaml file.")
		os.Exit(1)
	}

	if config, err = loadConfig(); err != nil {
		fmt.Println("Failed to read config:", err)
		os.Exit(1)
	}

	docs.SwaggerInfo.Version = AppVersion.Version

	if config.CA.CertsFile != "" {
		client, err := cacertutil.NewHTTPClientWithCerts(config.CA.CertsFile)
		if err != nil {
			log.Error().WithError(err).Message("Failed to get net/http.Client with certs")
			os.Exit(1)
		}
		http.DefaultClient = client
	}

	seed()

	db := setupDB(config.DB)
	serve(config, db)
}

func serve(config Config, db *gorm.DB) {
	listener, err := net.Listen("tcp", config.HTTP.BindAddress)
	if err != nil {
		log.Error().WithError(err).
			WithString("address", config.HTTP.BindAddress).
			Message("Failed to bind address.")
	}
	mux := cmux.New(listener)
	grpcListener := mux.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpListener := mux.Match(cmux.Any())

	go serveGRPC(grpcListener, db)
	go serveHTTP(httpListener, config, db)

	if err := mux.Serve(); err != nil {
		log.Error().WithError(err).
			Message("Failed to run multiplexed server.")
	}
}

func setupDB(dbConfig DBConfig) *gorm.DB {
	db, err := openDatabase(dbConfig)
	if err != nil {
		log.Error().
			WithString("driver", string(dbConfig.Driver)).
			WithError(err).
			Message("Database error")
		os.Exit(2)
	}

	err = runDatabaseMigrations(db, dbConfig.Driver)
	if err != nil {
		log.Error().WithError(err).Message("Migration error")
		os.Exit(3)
	}

	return db
}

func seed() {
	rand.Seed(time.Now().UTC().UnixNano())
}
