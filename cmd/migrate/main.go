// Command migrate applies the DMS schema migrations out of band.
//
// The service itself refuses to auto-migrate in production: config.Validate
// rejects AUTO_MIGRATE=true when ENVIRONMENT=production, on the principle that
// a schema change should be a deliberate act rather than a side effect of a
// deploy. That principle was sound but incomplete - there was no out-of-band
// path, so production had no way to migrate at all, and the dms schema was
// simply never created.
//
// This is that path. It shares the runner, the embedded migration files and the
// ledger with the in-process path, so running it is equivalent to what a dev
// boot does with AUTO_MIGRATE=true - the same versions, the same checksums, the
// same dms.schema_migrations rows.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/migrate
//	migrate -timeout 20m
//
// It is safe to re-run: already-applied versions are skipped, and a version
// whose file has changed is reported as a checksum mismatch rather than being
// silently re-applied.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	dmsdb "github.com/iag/dms/backend/db"
	"github.com/iag/dms/backend/internal/db"
	"github.com/iag/dms/backend/internal/migrate"
)

func main() {
	var (
		databaseURL = flag.String("database-url", "", "postgres connection string (default: $DATABASE_URL)")
		timeout     = flag.Duration("timeout", 10*time.Minute, "overall deadline for the migration run")
	)
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if err := run(*databaseURL, *timeout); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1)
	}
}

func run(databaseURL string, timeout time.Duration) error {
	url := strings.TrimSpace(databaseURL)
	if url == "" {
		url = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if url == "" {
		return errors.New("no database URL: pass -database-url or set DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	connectCtx, cancelConnect := context.WithTimeout(ctx, 30*time.Second)
	pool, err := db.Connect(connectCtx, url)
	cancelConnect()
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	applied, err := migrate.Up(ctx, pool, dmsdb.Migrations())
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if len(applied) == 0 {
		slog.Info("schema already up to date, nothing applied")
		return nil
	}
	slog.Info("migrations applied", "count", len(applied), "versions", applied)
	return nil
}
