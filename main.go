package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iver-wharf/wharf-core/pkg/cacertutil"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	"github.com/iver-wharf/wharf-core/pkg/logger"
	"github.com/iver-wharf/wharf-core/pkg/logger/consolepretty"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	v1 "github.com/iver-wharf/wharf-api/v5/api/wharfapi/v1"
	"github.com/iver-wharf/wharf-api/v5/docs"
	"github.com/iver-wharf/wharf-api/v5/internal/deprecated"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
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

	db, err := openDatabase(config.DB)
	if err != nil {
		log.Error().
			WithString("driver", string(config.DB.Driver)).
			WithError(err).
			Message("Database error")
		os.Exit(2)
	}

	err = runDatabaseMigrations(db, config.DB.Driver)
	if err != nil {
		log.Error().WithError(err).Message("Migration error")
		os.Exit(3)
	}

	gin.DefaultWriter = ginutil.DefaultLoggerWriter
	gin.DefaultErrorWriter = ginutil.DefaultLoggerWriter

	r := gin.New()
	r.Use(
		ginutil.LoggerWithConfig(ginutil.LoggerConfig{
			//disable GIN logs for path "/health". Probes won't clog up logs now.
			SkipPaths: []string{"/health"},
		}),
		ginutil.RecoverProblem,
	)

	if len(config.HTTP.CORS.AllowOrigins) > 0 {
		log.Info().
			WithStringf("origin", "%v", config.HTTP.CORS.AllowOrigins).
			Message("Allowing origins in CORS.")
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = config.HTTP.CORS.AllowOrigins
		corsConfig.AddAllowHeaders("Authorization")
		corsConfig.AllowCredentials = true
		r.Use(cors.New(corsConfig))
	} else if config.HTTP.CORS.AllowAllOrigins {
		log.Info().Message("Allowing all origins in CORS.")
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowAllOrigins = true
		r.Use(cors.New(corsConfig))
	}

	healthModule{}.DeprecatedRegister(r)
	healthModule{}.Register(r.Group("/api"))

	if config.HTTP.OIDC.Enable {
		rsaKeys, err := GetOIDCPublicKeys(config.HTTP.OIDC.KeysURL)
		if err != nil {
			log.Error().WithError(err).Message("Failed to obtain OIDC public keys.")
			os.Exit(1)
		}
		m := newOIDCMiddleware(rsaKeys, config.HTTP.OIDC)
		r.Use(m.VerifyTokenMiddleware)
		m.SubscribeToKeyURLUpdates()
	}

	setupBasicAuth(r, config)

	modules := []httpModule{
		engineModule{CIConfig: &config.CI},
		branchModule{Database: db},
		buildModule{Database: db, Config: &config},
		projectModule{Database: db},
		providerModule{Database: db},
		tokenModule{Database: db},
		deprecated.BranchModule{Database: db},
		deprecated.BuildModule{Database: db},
		deprecated.ProjectModule{Database: db},
		deprecated.ProviderModule{Database: db},
		deprecated.TokenModule{Database: db},
	}

	api := r.Group("/api")
	for _, module := range modules {
		module.Register(api)
	}

	api.GET("/version", getVersionHandler)
	api.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

	grpcServer := grpc.NewServer()
	grpcWharf := &grpcWharfServer{db: db}
	v1.RegisterBuildsServer(grpcServer, grpcWharf)

	go grpcServer.Serve(grpcListener)
	go r.RunListener(httpListener)

	if err := mux.Serve(); err != nil {
		log.Error().WithError(err).
			Message("Failed to run multiplexed server.")
	}
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
