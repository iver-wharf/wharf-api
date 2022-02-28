package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-core/pkg/config"
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

	// InstanceID may be an arbitrary string that is used to identify different
	// Wharf installations from each other. Needed when you use multiple Wharf
	// installations in the same environment, such as the same Kubernetes
	// namespace or the same Jenkins instance, to let Wharf know which builds
	// belong to which Wharf installation.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable WHARF_INSTANCE, which was added back in v0.7.9.
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
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable CI_URL, which was added back in v0.6.0.
	//
	// Deprecated: Use ci.engine.url (YAML) or WHARF_CI_ENGINE_URL (env var)
	// instead. Planned for removal in v6.0.0.
	//
	// Added in v4.2.0.
	TriggerURL string

	// TriggerToken is passed along as a credentials token via the "token" query
	// parameter. When using the Jenkins plugin "Generic Webhook Trigger"
	// (https://plugins.jenkins.io/generic-webhook-trigger) then this token is
	// configured in the webhook settings.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable CI_TOKEN, which was added back in v0.6.0.
	//
	// Deprecated: Use ci.engine.token (YAML) or WHARF_CI_ENGINE_TOKEN
	// (env var) instead. Planned for removal in v6.0.0.
	//
	// Added in v4.2.0.
	TriggerToken string

	// Engine defines the primary and default execution engine to be used when
	// starting new builds.
	//
	// Added in v5.1.0.
	Engine CIEngineConfig

	// Engine2 defines a secondary execution engine that can be used when
	// starting new builds.
	//
	// Added in v5.1.0.
	Engine2 CIEngineConfig

	// MockTriggerResponse will, when set to true, hinder wharf-api from sending
	// a HTTP POST trigger request when starting a new build and will instead
	// silently act like the build has been successfully scheduled.
	//
	// Useful when running Wharf locally and you want to test the behavior of
	// starting a new build, without actually needing a local Jenkins set up.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable MOCK_LOCAL_CI_RESPONSE, which was added back in v0.6.0.
	//
	// Added in v4.2.0.
	MockTriggerResponse bool
}

// CIEngineConfig holds settings for the execution engine used in CI
// (Continuous Integration).
type CIEngineConfig struct {
	// ID is the identifying name of the execution engine. Defaults to "primary"
	// or "secondary", depending on if this engine is defined by CIConfig.Engine
	// or CIConfig.Engine2, respectively.
	//
	// Added in v5.1.0.
	ID string

	// Name is the display name of the execution engine. Defaults to "Primary"
	// or "Secondary", depending on if this engine is defined by CIConfig.Engine
	// or CIConfig.Engine2, respectively.
	//
	// Added in v5.1.0.
	Name string

	// API is the type of API for this engine. If set to "wharf-cmd.v1" then the
	// wharf-api will have additional integration with this engine, such as
	// supporting cancelling builds. Possible values are:
	//
	// 	jenkins-generic-webhook-trigger
	// 	wharf-cmd.v1
	//
	// If no value is supplied, then "jenkins-generic-webhook-trigger" is
	// assumed.
	//
	// Added in v5.1.0.
	API CIEngineAPI

	// URL is the full URL that wharf-api will send a POST request to
	// with all of the build metadata. For example to trigger a Jenkins job via
	// the "Generic Webhook Trigger":
	// https://plugins.jenkins.io/generic-webhook-trigger
	//
	// Added in v5.1.0.
	URL string

	// Token is passed along as a credentials token via the "token" query
	// parameter. When using the Jenkins plugin "Generic Webhook Trigger"
	// (https://plugins.jenkins.io/generic-webhook-trigger) then this token is
	// configured in the webhook settings.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable CI_TOKEN, which was added back in v0.6.0.
	//
	// Added in v5.1.0.
	Token string
}

// CIEngineAPI is an enum of different engine API values.
type CIEngineAPI string

const (
	// CIEngineAPIJenkinsGenericWebhookTrigger means wharf-api will target the
	// Jenkins Generic Webhook Trigger plugin:
	// https://plugins.jenkins.io/generic-webhook-trigger/
	CIEngineAPIJenkinsGenericWebhookTrigger CIEngineAPI = "jenkins-generic-webhook-trigger"
	// CIEngineAPIWharfCMDv1 means that wharf-api will target the v1 of the
	// wharf-cmd-provisioner API.
	CIEngineAPIWharfCMDv1 CIEngineAPI = "wharf-cmd.v1"
)

// HTTPConfig holds settings for the HTTP server.
type HTTPConfig struct {
	CORS CORSConfig

	// BindAddress is the IP-address and port, separated by a colon, to bind
	// the HTTP server to. An IP-address of 0.0.0.0 will bind to all
	// IP-addresses.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable BIND_ADDRESS, which was added back in v4.1.0.
	//
	// Added in v4.2.0.
	BindAddress string

	// BasicAuth is a comma-separated list of username:password pairs.
	//
	// Example for user named "admin" with password "1234" and a user named
	// "john" with the password "secretpass":
	// 	BasicAuth="admin:1234,john:secretpass"
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable BASIC_AUTH, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	BasicAuth string

	// OIDC is only secure if using HTTPS/SSL.
	// Requires CORS set to specific origins.
	//
	// If enabled, HTTP requests without a valid OIDC access token in the Authorization
	// header will be rejected with Unauthorized 401.
	//
	// Added in v5.0.0.
	OIDC OIDCConfig
}

// CORSConfig holds settings for the HTTP server's CORS settings.
type CORSConfig struct {
	// AllowAllOrigins enables CORS and allows all hostnames and URLs in the
	// HTTP request origins when set to true. Practically speaking, this
	// results in the HTTP header "Access-Control-Allow-Origin" set to "*".
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable ALLOW_CORS, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	AllowAllOrigins bool

	// AllowOrigins enables CORS and allows the list of origins in the
	// HTTP request origins when set. Practically speaking, this
	// results in the HTTP header "Access-Control-Allow-Origin".
	//
	// Added in v5.0.0.
	AllowOrigins []string
}

// OIDCConfig holds settings for the HTTP server's OIDC access token validation settings.
type OIDCConfig struct {

	// Enable functions as a switch to enable or disable the validation of OIDC
	// access bearer tokens.
	//
	// Added in v5.0.0.
	Enable bool

	// IssuerURL is an integral part of the access token. It should be checked such that
	// only allowed OIDC targets can pass token validation.
	//
	// Added in v5.0.0.
	IssuerURL string

	// AudienceURL is an integral part of the access token. It should be checked such that
	// only the allowed application within a OIDC target can pass validation.
	//
	// Added in v5.0.0.
	AudienceURL string

	// KeysURL is an integral part of the access token. It should be checked such that
	// only OIDC targets with the expected keys pass validation.
	//
	// Added in v5.0.0.
	KeysURL string

	// UpdateInterval defines the key rotation of the public RSA keys obtained
	// by the OIDC keys URL. A value of 25 hours is both default and
	// recommended.
	//
	// Added in v5.0.0.
	UpdateInterval time.Duration
}

// CertConfig holds settings for certificates verification used when talking
// to remote services over HTTPS.
type CertConfig struct {
	// CertsFile points to a file of one or more PEM-formatted certificates to
	// use in addition to the certificates from the system
	// (such as from /etc/ssl/certs/).
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable CA_CERTS, which was added back in v0.7.5.
	//
	// Added in v4.2.0.
	CertsFile string
}

// DBDriver is an enum of different supported database drivers.
type DBDriver string

const (
	// DBDriverPostgres specifies usage of Postgres for persistence.
	//
	// Added in v5.0.0. Before v5.0.0, the database driver was assumed to be
	// Postgres, no matter what config you provided to wharf-api.
	DBDriverPostgres DBDriver = "postgres"

	// DBDriverSqlite specifies usage of Sqlite.
	//
	// Current limitation is that wharf-api must be compiled with CGO_ENABLED=1,
	// which by default the wharf-api Docker image is not.
	//
	// Added in v5.0.0.
	DBDriverSqlite DBDriver = "sqlite"
)

// DBConfig holds settings for connecting to a database, such as credentials and
// hostnames.
type DBConfig struct {
	// Driver sets what database engine to use for persistence. See the
	// DBDriver constants for the different supported values.
	//
	// The value is case sensitive.
	//
	// Added in v5.0.0.
	Driver DBDriver

	// Path defines where the database is located. Only applicable when the
	// driver is set to "sqlite", and is ignored otherwise.
	//
	// Non-existing directories in the path will be created, given the process
	// has write access in the regarded containing directories.
	//
	// The path is not dereferenced, so specifying "~/.local/share/wharf-api.db"
	// will result in a new directory named "~" to be created in the current
	// working directory, meaning it would be equivalent to
	// "./~/.local/share/wharf-api.db".
	//
	// Added in v5.0.0.
	Path string

	// Host is the network hostname wharf-api will connect to. Ignored when
	// the driver is set to "sqlite".
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable DBHOST, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	Host string

	// Port is the network port wharf-api will connect to. Ignored when
	// the driver is set to "sqlite".
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable DBPORT, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	Port int

	// Username is the username part of credentials used when connecting to the
	// database. Ignored when the driver is set to "sqlite".
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable DBUSER, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	Username string

	// Password is the username part of credentials used when connecting to the
	// database. Ignored when the driver is set to "sqlite".
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable DBPASS, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	Password string

	// Name is the database name that wharf-api will store its data in. Some
	// databases also call this the "schema" name.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable DBNAME, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	Name string

	// MaxIdleConns is the maximum number of idle connections that wharf-api
	// will keep alive.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable DBMAXIDLECONNS, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	MaxIdleConns int

	// MaxOpenConns is the maximum number of open connections that wharf-api
	// will use at a single point in time.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable DBMAXOPENCONNS, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	MaxOpenConns int

	// MaxConnLifetime is the maximum age for a given database connection. If
	// any connection exceeds this limit, while not in use, it will be
	// disconnected.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable DBMAXCONNLIFETIME, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	MaxConnLifetime time.Duration

	// Log enables/disables database SQL query logging.
	//
	// This corresponds to the deprecated (and unsupported since v5.0.0)
	// environment variable DBLOG, which was added back in v0.5.5.
	//
	// Added in v4.2.0.
	Log bool
}

// DefaultConfig is the hard-coded default values for wharf-api's configs.
var DefaultConfig = Config{
	CI: CIConfig{
		Engine: CIEngineConfig{
			ID:   "primary",
			Name: "Primary",
			API:  CIEngineAPIJenkinsGenericWebhookTrigger,
		},
		Engine2: CIEngineConfig{
			ID:   "secondary",
			Name: "Secondary",
			API:  CIEngineAPIJenkinsGenericWebhookTrigger,
		},
	},
	HTTP: HTTPConfig{
		BindAddress: "0.0.0.0:8080",
		CORS: CORSConfig{
			// :4200 is used when running wharf-web via `npm start` locally
			// :5000 is used when running wharf-web via docker-compose locally
			AllowOrigins: []string{"http://localhost:4200", "http://localhost:5000"},
		},
		OIDC: OIDCConfig{
			Enable:         false,
			IssuerURL:      "https://sts.windows.net/841df554-ef9d-48b1-bc6e-44cf8543a8fc/",
			AudienceURL:    "api://wharf-internal",
			KeysURL:        "https://login.microsoftonline.com/841df554-ef9d-48b1-bc6e-44cf8543a8fc/discovery/v2.0/keys",
			UpdateInterval: time.Hour * 25,
		},
	},
	DB: DBConfig{
		Driver: DBDriverPostgres,
		Path:   "wharf-api.db",
		// Current default in sql package according to docs
		// https://golang.org/pkg/database/sql/#DB.SetMaxIdleConns
		MaxIdleConns: 2,
		// Current default in sql package according to docs
		// https://golang.org/pkg/database/sql/#DB.SetMaxOpenConns
		MaxOpenConns:    0,
		MaxConnLifetime: 20 * time.Minute,
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
	cfg.addBackwardCompatibleConfigs()
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	if cfg.CI.Engine.URL != "" {
		cfg.CI.Engine.API, err = parseCIEngineAPI(cfg.CI.Engine.API)
		if err != nil {
			return Config{}, err
		}
	}
	if cfg.CI.Engine2.URL != "" {
		cfg.CI.Engine2.API, err = parseCIEngineAPI(cfg.CI.Engine2.API)
		if err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

func parseCIEngineAPI(api CIEngineAPI) (CIEngineAPI, error) {
	switch strings.TrimSpace(strings.ToLower(string(api))) {
	case "", string(CIEngineAPIJenkinsGenericWebhookTrigger):
		return CIEngineAPIJenkinsGenericWebhookTrigger, nil
	case string(CIEngineAPIWharfCMDv1):
		return CIEngineAPIWharfCMDv1, nil
	default:
		return "", fmt.Errorf("invalid CI engine API value: %q", api)
	}
}

func (cfg *Config) addBackwardCompatibleConfigs() {
	if cfg.CI.TriggerToken != "" {
		cfg.CI.Engine.Token = cfg.CI.TriggerToken
	}
	if cfg.CI.TriggerURL != "" {
		cfg.CI.Engine.URL = cfg.CI.TriggerURL
	}
}

func (cfg *Config) validate() error {
	if len(cfg.CI.Engine.ID) > database.BuildSizes.EngineID {
		return fmt.Errorf("primary engine ID is too large: max 32 chars, but was: %d", len(cfg.CI.Engine.ID))
	}
	if len(cfg.CI.Engine2.ID) > database.BuildSizes.EngineID {
		return fmt.Errorf("secondary engine ID is too large: max 32 chars, but was: %d", len(cfg.CI.Engine2.ID))
	}
	return nil
}
