package main

import (
	"net"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/iver-wharf/wharf-api/v5/internal/deprecated"
	"github.com/iver-wharf/wharf-core/pkg/ginutil"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
	"gorm.io/gorm"
)

func serveHTTP(listener net.Listener, config Config, db *gorm.DB) {
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

	r.RunListener(listener)
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
