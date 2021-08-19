package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/logger"
	"github.com/iver-wharf/wharf-core/pkg/logger/consolepretty"

	"github.com/dustin/go-broadcast"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/iver-wharf/wharf-api/docs"
	"github.com/iver-wharf/wharf-api/pkg/httputils"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

var logBroadcaster broadcast.Broadcaster

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
		client, err := httputils.NewClientWithCerts(config.CA.CertsFile)
		if err != nil {
			log.Error().WithError(err).Message("Failed to get net/http.Client with certs")
			os.Exit(1)
		}
		http.DefaultClient = client
	}

	seed()

	db, err := openDatabase(config.DB)
	if err != nil {
		log.Error().WithError(err).Message("Database error")
		os.Exit(2)
	}

	err = runDatabaseMigrations(db)
	if err != nil {
		log.Error().WithError(err).Message("Migration error")
		os.Exit(3)
	}

	r := gin.New()
	r.Use(
		//disable GIN logs for path "/health". Probes won't clog up logs now.
		gin.LoggerWithWriter(gin.DefaultWriter, "/health"),
		gin.CustomRecovery(ginutil.RecoverProblemHandle),
	)

	if config.HTTP.CORS.AllowAllOrigins {
		log.Info().Message("Allowing all origins in CORS.")
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowAllOrigins = true
		r.Use(cors.New(corsConfig))
	}

	HealthModule{}.DeprecatedRegister(r)

	setupBasicAuth(r, config)

	mq, err := GetMQConnection(config.MQ)
	if err != nil {
		log.Error().WithError(err).Message("Message queue error.")
		os.Exit(4)
	} else if mq == nil {
		log.Info().Message("RabbitMQ integration has been disabled in the config. Skipping connection instantiation.")
	} else {
		err = mq.Connect()
		if err != nil {
			log.Error().WithError(err).Message("Unable to connect to the RabbitMQ queue.")
			os.Exit(5)
		}

		go func() {
			<-mq.UnexpectedClose
			log.Error().Message("Unexpected RabbitMQ close.")
			os.Exit(6)
		}()
	}

	modules := []HTTPModule{
		ProjectModule{Database: db, MessageQueue: mq, Config: &config},
		BuildModule{Database: db, MessageQueue: mq},
		TokenModule{Database: db},
		BranchModule{Database: db},
		ProviderModule{Database: db},
		HealthModule{}}

	api := r.Group("/api")
	for _, module := range modules {
		module.Register(api)
	}

	api.GET("/version", getVersionHandler)
	api.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	_ = r.Run(config.HTTP.BindAddress)
}

func setupBasicAuth(router *gin.Engine, config Config) {
	if config.HTTP.BasicAuth == "" {
		log.Info().Message("BasicAuth setting not set, skipping BasicAuth setup.")
		return
	}

	accounts := gin.Accounts{}
	var accountNames []string

	for _, account := range strings.Split(config.HTTP.BasicAuth, ",") {
		split := strings.Split(account, ":")
		user, pass := split[0], split[1]

		accounts[user] = pass
		accountNames = append(accountNames, user)
	}

	log.Debug().WithString("usernames", strings.Join(accountNames, ",")).
		Messagef("Set up basic authentication for %d users.", len(accountNames))

	router.Use(gin.BasicAuth(accounts))
}

func seed() {
	rand.Seed(time.Now().UTC().UnixNano())
}
