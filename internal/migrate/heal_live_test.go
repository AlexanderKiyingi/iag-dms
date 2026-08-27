package migrate

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iag/dms/backend/db"
)

const oldForecastChecksum = "c23c47ebca091f5a31a8cb3c88a31625e5d1a5a4c789095f1d2935ea7fb79aee"

func TestChecksumHealLive(t *testing.T) {
	dsn := os.Getenv("DMS_TEST_DSN")
	if dsn == "" {
		t.Skip("DMS_TEST_DSN not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if _, err := Up(ctx, pool, db.Migrations()); err != nil {
		t.Fatalf("initial Up: %v", err)
	}

	// Simulate a database that applied the original 0002.
	if _, err := pool.Exec(ctx,
		`UPDATE dms.schema_migrations SET checksum=$1 WHERE version='0002_forecast_points'`,
		oldForecastChecksum); err != nil {
		t.Fatalf("stamp old checksum: %v", err)
	}

	if _, err := Up(ctx, pool, db.Migrations()); err != nil {
		t.Fatalf("Up should have healed the superseded checksum, got: %v", err)
	}

	var now string
	if err := pool.QueryRow(ctx,
		`SELECT checksum FROM dms.schema_migrations WHERE version='0002_forecast_points'`).Scan(&now); err != nil {
		t.Fatalf("read checksum: %v", err)
	}
	if now == oldForecastChecksum {
		t.Fatal("checksum was not re-stamped")
	}
	t.Logf("healed: %s -> %s", oldForecastChecksum[:12], now[:12])

	// Any OTHER drift must still be fatal.
	if _, err := pool.Exec(ctx,
		`UPDATE dms.schema_migrations SET checksum='deadbeef' WHERE version='0002_forecast_points'`); err != nil {
		t.Fatalf("stamp bogus checksum: %v", err)
	}
	_, err = Up(ctx, pool, db.Migrations())
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("unrecognised drift must still fail, got: %v", err)
	}
	t.Logf("unrecognised drift still rejected: %v", err)
}
