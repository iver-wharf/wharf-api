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
	if err := loadEmbeddedVersionFile(); err != nil {
		fmt.Println("Failed to read embedded version.yaml file:", err)
		os.Exit(1)
	}

	docs.SwaggerInfo.Version = AppVersion.Version

	logger.AddOutput(logger.LevelDebug, consolepretty.Default)

	if localCertFile, _ := os.LookupEnv("CA_CERTS"); localCertFile != "" {
		client, err := httputils.NewClientWithCerts(localCertFile)
		if err != nil {
			log.Error().WithError(err).Message("Failed to get net/http.Client with certs")
			os.Exit(1)
		}
		http.DefaultClient = client
	}

	seed()

	config, err := getDatabaseConfigFromEnvironment()
	if err != nil {
		log.Error().WithError(err).Message("Config error")
		os.Exit(1)
	}

	db, err := openDatabase(config)
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

	allowCors, ok := os.LookupEnv("ALLOW_CORS")
	if ok && allowCors == "YES" {
		log.Info().Message("Allowing CORS")
		r.Use(cors.Default())
	}

	HealthModule{}.Register(r)

	setupBasicAuth(r)

	mq, err := GetMQConnection()
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
		ProjectModule{Database: db, MessageQueue: mq},
		BuildModule{Database: db, MessageQueue: mq},
		TokenModule{Database: db},
		BranchModule{Database: db},
		ProviderModule{Database: db}}

	api := r.Group("/api")
	for _, module := range modules {
		module.Register(api)
	}

	api.GET("/version", getVersionHandler)
	api.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	_ = r.Run(getBindAddress())
}

func getBindAddress() string {
	bindAddress, isBindAddressDefined := os.LookupEnv("BIND_ADDRESS")
	if !isBindAddressDefined || bindAddress == "" {
		return "0.0.0.0:8080"
	}
	return bindAddress
}

func setupBasicAuth(router *gin.Engine) {
	basicAuth := os.Getenv("BASIC_AUTH")
	if basicAuth == "" {
		log.Info().Message("BASIC_AUTH environment variable not set, skipping basic-auth setup.")
		return
	}

	accounts := gin.Accounts{}
	var accountNames []string

	for _, account := range strings.Split(basicAuth, ",") {
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
