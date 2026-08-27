package store

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedPostgres is intentionally a no-op.
//
// It used to copy the in-memory demo dataset into Postgres when dms_distributors was
// empty. Those rows were removed from every environment by migration
// 0008_purge_demo_seed.sql and are not reloaded; a DMS database starts empty and fills
// up with what operators actually enter.
func SeedPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	slog.Info("dms seed skipped — the demo dataset was purged and is not reloaded")
	return nil
}
