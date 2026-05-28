package config

import "testing"

func TestValidateProductionRequiresJWKSAndSecret(t *testing.T) {
	cfg := Config{
		Environment:         "production",
		DatabaseURL:         "postgres://u:p@localhost/db",
		Audience:            "iag.dms",
		JWKSURL:             "https://auth.example.com/.well-known/jwks.json",
		CORSOrigin:          "https://app.example.com",
		ServiceClientSecret: "short",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected SERVICE_CLIENT_SECRET min length in production")
	}

	cfg.ServiceClientSecret = "production-secret-min-16"
	cfg.AutoMigrate = true
	cfg.SeedOnEmpty = true
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected AUTO_MIGRATE/SEED rejection in production")
	}

	cfg.AutoMigrate = false
	cfg.SeedOnEmpty = false
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid production config: %v", err)
	}
}

func TestValidateRejectsWildcardCORSInProduction(t *testing.T) {
	cfg := Config{
		Environment:         "production",
		DatabaseURL:         "postgres://u:p@localhost/db",
		Audience:            "iag.dms",
		JWKSURL:             "https://auth.example.com/.well-known/jwks.json",
		CORSOrigin:          "https://app.example.com,*",
		ServiceClientSecret: "production-secret-min-16",
		AutoMigrate:         false,
		SeedOnEmpty:         false,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected wildcard CORS rejection in production")
	}
}

func TestStrictRBAC(t *testing.T) {
	if !(Config{Environment: "production"}).StrictRBAC() {
		t.Fatal("production should use strict RBAC")
	}
	if (Config{Environment: "development"}).StrictRBAC() {
		t.Fatal("development should not use strict RBAC")
	}
}
