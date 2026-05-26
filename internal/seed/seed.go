package seed

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iag/dms/backend/internal/store"
)

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	if err := store.SeedPostgres(ctx, pool); err != nil {
		return err
	}
	slog.Info("dms seed complete")
	return nil
}
