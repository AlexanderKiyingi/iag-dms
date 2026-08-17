package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alvor-technologies/iag-platform-go/corsenv"
)

type Config struct {
	ServiceName         string
	Addr                string
	Environment         string
	DatabaseURL         string
	UseMemoryStore      bool
	JWTIssuer           string
	JWKSURL             string
	Audience            string // aud claim the service requires on inbound tokens
	GatewayAPIPrefix    string
	ServiceClientID     string
	ServiceClientSecret string
	AuthTokenURL        string
	CORSOrigin          string
	PublicAPIURL        string
	AutoMigrate         bool
	SeedOnEmpty         bool
	EventBusEnabled     bool
	KafkaBrokers        []string
	ConsumerEnabled     bool
	ConsumerTopic       string
	ConsumerGroupID     string
	OperationsConsumerEnabled bool
	OperationsConsumerTopic   string
	OperationsConsumerGroupID string
	FinanceURL          string
	FileStorageDir      string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
}

// Load reads configuration from env. Hard cutover: every request must carry a
// verifiable Bearer token with aud=iag.dms. AUTH_MODE and
// GATEWAY_INTERNAL_SECRET no longer exist.
func Load() (Config, error) {
	env := strings.ToLower(strings.TrimSpace(envOr("ENVIRONMENT", envOr("APP_ENV", "development"))))
	issuer := envOr("JWT_ISSUER", "http://localhost:3001")
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	useMemory := strings.EqualFold(envOr("STORE_MODE", ""), "memory")

	cfg := Config{
		ServiceName:         envOr("SERVICE_NAME", "dms"),
		Addr:                ListenAddr(),
		Environment:         env,
		DatabaseURL:         dbURL,
		UseMemoryStore:      useMemory,
		JWTIssuer:           issuer,
		JWKSURL:             envOr("JWKS_URL", strings.TrimRight(issuer, "/")+"/.well-known/jwks.json"),
		Audience:            envOr("AUDIENCE", "iag.dms"),
		GatewayAPIPrefix:    strings.TrimSpace(envOr("GATEWAY_API_PREFIX", "/api/v1/dms")),
		ServiceClientID:     envOr("SERVICE_CLIENT_ID", "iag-dms"),
		ServiceClientSecret: strings.TrimSpace(os.Getenv("SERVICE_CLIENT_SECRET")),
		AuthTokenURL:        envOr("AUTH_TOKEN_URL", strings.TrimRight(issuer, "/")+"/oauth/token"),
		CORSOrigin:          corsenv.Allowlist(corsenv.DefaultDevOrigins),
		PublicAPIURL:        strings.TrimRight(strings.TrimSpace(envOr("PUBLIC_API_URL", "http://localhost:8080")), "/"),
		AutoMigrate:         envOr("AUTO_MIGRATE", "true") != "false",
		SeedOnEmpty:         envOr("SEED_ON_EMPTY", "true") != "false",
		EventBusEnabled:     strings.EqualFold(os.Getenv("EVENT_BUS_ENABLED"), "true"),
		KafkaBrokers:        parseBrokers(os.Getenv("KAFKA_BROKERS")),
		ConsumerEnabled:     strings.EqualFold(os.Getenv("CONSUMER_ENABLED"), "true"),
		ConsumerTopic:       envOr("CONSUMER_TOPIC", "iag.commercial"),
		ConsumerGroupID:     envOr("CONSUMER_GROUP_ID", "iag-dms"),
		OperationsConsumerEnabled: strings.EqualFold(os.Getenv("OPERATIONS_CONSUMER_ENABLED"), "true"),
		OperationsConsumerTopic:   envOr("OPERATIONS_CONSUMER_TOPIC", "iag.operations"),
		OperationsConsumerGroupID: envOr("OPERATIONS_CONSUMER_GROUP_ID", "iag-dms.operations"),
		FinanceURL:          strings.TrimRight(strings.TrimSpace(os.Getenv("FINANCE_URL")), "/"),
		FileStorageDir:      envOr("FILE_STORAGE_DIR", "./data/attachments"),
		ReadTimeout:         30 * time.Second,
		WriteTimeout:        30 * time.Second,
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if !c.UseMemoryStore && c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required unless STORE_MODE=memory")
	}
	if c.Audience == "" {
		return fmt.Errorf("AUDIENCE is required (e.g. iag.dms)")
	}
	if c.JWKSURL == "" {
		return fmt.Errorf("JWKS_URL is required")
	}
	if c.IsProduction() {
		if c.HasWildcardCORS() {
			return fmt.Errorf("set ALLOWED_ORIGINS in production (not *)")
		}
		if strings.TrimSpace(c.ServiceClientSecret) == "" {
			return fmt.Errorf("SERVICE_CLIENT_SECRET is required in production")
		}
		if len(strings.TrimSpace(c.ServiceClientSecret)) < 16 {
			return fmt.Errorf("SERVICE_CLIENT_SECRET must be at least 16 characters in production")
		}
		if c.AutoMigrate {
			return fmt.Errorf("AUTO_MIGRATE must be false in production (run migrations out of band)")
		}
		if c.SeedOnEmpty {
			return fmt.Errorf("SEED_ON_EMPTY must be false in production")
		}
		if c.UseMemoryStore {
			return fmt.Errorf("STORE_MODE=memory is not allowed in production")
		}
	}
	return nil
}

func (c Config) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "prod"
}

// StrictRBAC denies access when a verified token carries no permissions
// (fail-closed).
func (c Config) StrictRBAC() bool { return c.HardenedRuntime() }

// HardenedRuntime reports whether production safeguards apply.
//
// It deliberately does not just return IsProduction(). That required
// ENVIRONMENT=production, which the Railway runbooks never told anyone to set,
// so a hosted instance fell back to the "development" default and ran
// fail-OPEN: the permission middleware grants EVERY permission to a token
// carrying an empty permissions array. An unset ENVIRONMENT on a deployed
// instance now hardens instead; only an explicit dev-like value opts out.
//
// This cannot prevent boot — the worst case is a 403 for a caller that should
// never have had access. Boot-time validation stays keyed on ENVIRONMENT alone.
//
// Mirrors iag-fleet's config.HardenedRuntime; the intent is one shared
// implementation in shared/platform-go once every service is on it.
func (c Config) HardenedRuntime() bool {
	// An explicit production value always hardens, including on a Config built
	// by hand in a test rather than through Load.
	if c.IsProduction() {
		return true
	}
	if environmentExplicitlySet() {
		return !c.isDevLike()
	}
	return deployedRuntime()
}

// isDevLike reports an environment where fail-open behaviour is a deliberate
// local convenience rather than an accident.
func (c Config) isDevLike() bool {
	switch c.Environment {
	case "development", "dev", "local", "test":
		return true
	}
	return false
}

// environmentExplicitlySet distinguishes a deliberately configured environment
// from the "development" value Load falls back to when nothing is set. Read
// from the process rather than captured on Config: StrictRBAC is resolved once
// during startup wiring, and the environment does not change under us.
func environmentExplicitlySet() bool {
	return strings.TrimSpace(os.Getenv("ENVIRONMENT")) != "" ||
		strings.TrimSpace(os.Getenv("APP_ENV")) != ""
}

// deployedRuntime distinguishes a hosted instance from a laptop: Railway's
// injected variables, or gin in release mode, which the Dockerfiles set.
func deployedRuntime() bool {
	if strings.TrimSpace(os.Getenv("RAILWAY_ENVIRONMENT")) != "" ||
		strings.TrimSpace(os.Getenv("RAILWAY_PROJECT_ID")) != "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GIN_MODE")), "release")
}

func (c Config) HasWildcardCORS() bool {
	for _, o := range strings.Split(c.CORSOrigin, ",") {
		if strings.TrimSpace(o) == "*" {
			return true
		}
	}
	return c.CORSOrigin == "*"
}

func parseBrokers(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
