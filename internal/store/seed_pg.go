package store

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedPostgres copies in-memory demo data into Postgres when distributors table is empty.
func SeedPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	repo := New(pool)
	empty, err := repo.IsEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		slog.Info("dms seed skipped — data present")
		return nil
	}
	m := newMemoryState()
	seedMemory(m)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, d := range m.distributors {
		var ob any
		if !d.OnboardedAt.IsZero() {
			ob = d.OnboardedAt
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO dms_distributors (id, name, tier, region, manager, outlets, sell_in_rate, revenue_ugx, status, onboarded_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			d.ID, d.Name, d.Tier, d.Region, d.Manager, d.Outlets, d.SellInRate, d.RevenueUGX, d.Status, ob); err != nil {
			return err
		}
	}
	for _, o := range m.outlets {
		if _, err := tx.Exec(ctx, `
			INSERT INTO dms_outlets (id, name, address, channel, distributor_id, beat_id, qtd_value_ugx, frequency, score, status, lat, lng)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			o.ID, o.Name, o.Address, o.Channel, o.DistributorID, o.BeatID, o.QTDValueUGX, o.Frequency, o.Score, o.Status, o.Lat, o.Lng); err != nil {
			return err
		}
	}
	for _, o := range m.orders {
		if _, err := tx.Exec(ctx, `
			INSERT INTO dms_orders (id, outlet_id, outlet_name, distributor_id, rep_id, status, amount_ugx, currency, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			o.ID, o.OutletID, o.OutletName, o.DistributorID, o.RepID, o.Status, o.AmountUGX, o.Currency, o.CreatedAt, o.UpdatedAt); err != nil {
			return err
		}
	}
	for _, b := range m.beats {
		if _, err := tx.Exec(ctx, `
			INSERT INTO dms_beats (id, name, rep_id, rep_name, stop_count, distance_km, status, progress)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			b.ID, b.Name, b.RepID, b.RepName, b.StopCount, b.DistanceKm, b.Status, b.Progress); err != nil {
			return err
		}
		for seq, oid := range b.OutletIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO dms_beat_outlets (beat_id, outlet_id, seq) VALUES ($1,$2,$3)`, b.ID, oid, seq+1); err != nil {
				return err
			}
		}
	}
	for _, rep := range m.reps {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_field_reps (id, name, beat_id, region, level, status) VALUES ($1,$2,$3,$4,$5,$6)`,
			rep.ID, rep.Name, rep.BeatID, rep.Region, rep.Level, rep.Status); err != nil {
			return err
		}
	}
	for _, p := range m.promos {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_promotions (id, name, sku, roi, status, outlets) VALUES ($1,$2,$3,$4,$5,$6)`,
			p.ID, p.Name, p.SKU, p.ROI, p.Status, p.Outlets); err != nil {
			return err
		}
	}
	for _, c := range m.claims {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_claims (id, outlet_id, claim_type, status, amount_ugx, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
			c.ID, c.OutletID, c.Type, c.Status, c.AmountUGX, c.CreatedAt); err != nil {
			return err
		}
	}
	for _, d := range m.dispatches {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_dispatches (id, truck_id, driver, status, eta, updated_at) VALUES ($1,$2,$3,$4,$5,$6)`,
			d.ID, d.TruckID, d.Driver, d.Status, d.ETA, d.UpdatedAt); err != nil {
			return err
		}
		for _, oid := range d.OrderIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO dms_dispatch_orders (dispatch_id, order_id) VALUES ($1,$2)`, d.ID, oid); err != nil {
				return err
			}
		}
	}
	for _, s := range m.skus {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_skus (code, name, cover_days, qty_on_hand, warehouse, sca_score) VALUES ($1,$2,$3,$4,$5,$6)`,
			s.Code, s.Name, s.CoverDays, s.QtyOnHand, s.Warehouse, s.SCAScore); err != nil {
			return err
		}
	}
	for _, s := range m.stock {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_stock_positions (distributor_id, sku, cover_days, qty, status) VALUES ($1,$2,$3,$4,$5)`,
			s.DistributorID, s.SKU, s.CoverDays, s.Qty, s.Status); err != nil {
			return err
		}
	}
	for _, inv := range m.invoices {
		if _, err := tx.Exec(ctx, `
			INSERT INTO dms_invoices (id, distributor_id, distributor_name, amount_ugx, due_date, status, order_id)
			VALUES ($1,$2,$3,$4,$5,$6,NULL)`,
			inv.ID, inv.DistributorID, inv.Distributor, inv.AmountUGX, inv.DueDate, inv.Status); err != nil {
			return err
		}
	}
	for _, p := range m.pricing {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_pricing_templates (id, name, channel, version, currency) VALUES ($1,$2,$3,$4,$5)`,
			p.ID, p.Name, p.Channel, p.Version, p.Currency); err != nil {
			return err
		}
	}
	for _, rt := range m.reports {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_report_templates (id, name, data_source, schedule) VALUES ($1,$2,$3,$4)`,
			rt.ID, rt.Name, rt.DataSource, rt.Schedule); err != nil {
			return err
		}
	}
	for _, e := range m.execution {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_execution_tasks (id, outlet_id, task_type, status, detail) VALUES ($1,$2,$3,$4,$5)`,
			e.ID, e.OutletID, e.Type, e.Status, e.Detail); err != nil {
			return err
		}
	}
	for _, a := range m.alerts {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_alerts (id, kind, title, detail) VALUES ($1,$2,$3,$4)`,
			a.ID, a.Kind, a.Title, a.Detail); err != nil {
			return err
		}
	}
	for _, c := range m.checkIns {
		if _, err := tx.Exec(ctx, `INSERT INTO dms_check_ins (id, rep_id, outlet_id, lat, lng, arrived_at, status) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			c.ID, c.RepID, c.OutletID, c.Lat, c.Lng, c.ArrivedAt, c.Status); err != nil {
			return err
		}
	}
	_, _ = tx.Exec(ctx, `INSERT INTO dms_id_counters (prefix, next_value) VALUES ('OUT', 2849), ('SO', 19853), ('INV', 2426), ('CLM', 1043), ('TPM', 25), ('DXP', 2815) ON CONFLICT DO NOTHING`)
	return tx.Commit(ctx)
}
