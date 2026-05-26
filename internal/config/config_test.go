package config

import "testing"

func TestValidateProductionRequiresJWTAndSecret(t *testing.T) {
	cfg := Config{
		Environment:   "production",
		DatabaseURL:   "postgres://u:p@localhost/db",
		Audience:      "iag.dms",
		AuthMode:      "gateway",
		CORSOrigin:    "https://app.example.com",
		GatewaySecret: "gateway-secret-min-16",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected AUTH_MODE=jwt requirement in production")
	}

	cfg.AuthMode = "jwt"
	cfg.ServiceClientSecret = "short"
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
		AuthMode:            "jwt",
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
	if !(Config{Environment: "production", AuthMode: "jwt"}).StrictRBAC() {
		t.Fatal("production jwt should use strict RBAC")
	}
	if (Config{Environment: "development", AuthMode: "jwt"}).StrictRBAC() {
		t.Fatal("development should not use strict RBAC")
	}
}
