package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/iag/dms/backend/internal/models"
)

func (r *Repository) pgListDistributors(ctx context.Context, opts ListOpts) ([]models.Distributor, int) {
	opts = defaultLimit(opts)
	q := `SELECT id, name, tier, region, manager, outlets, sell_in_rate, revenue_ugx, status, onboarded_at
		FROM dms_distributors WHERE ($1 = '' OR status = $1)
		AND ($2 = '' OR id ILIKE '%'||$2||'%' OR name ILIKE '%'||$2||'%' OR region ILIKE '%'||$2||'%')
		ORDER BY revenue_ugx DESC`
	rows, err := r.pool.Query(ctx, q, opts.Status, opts.Q)
	if err != nil {
		return nil, 0
	}
	defer rows.Close()
	var all []models.Distributor
	for rows.Next() {
		var d models.Distributor
		var onboarded *time.Time
		if err := rows.Scan(&d.ID, &d.Name, &d.Tier, &d.Region, &d.Manager, &d.Outlets, &d.SellInRate, &d.RevenueUGX, &d.Status, &onboarded); err != nil {
			continue
		}
		if onboarded != nil {
			d.OnboardedAt = *onboarded
		}
		all = append(all, d)
	}
	return paginate(all, opts)
}

func (r *Repository) pgGetDistributor(ctx context.Context, id string) (models.Distributor, error) {
	var d models.Distributor
	var onboarded *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, tier, region, manager, outlets, sell_in_rate, revenue_ugx, status, onboarded_at
		FROM dms_distributors WHERE id = $1`, id).Scan(
		&d.ID, &d.Name, &d.Tier, &d.Region, &d.Manager, &d.Outlets, &d.SellInRate, &d.RevenueUGX, &d.Status, &onboarded)
	if err != nil {
		if err == pgx.ErrNoRows {
			return d, ErrNotFound
		}
		return d, err
	}
	if onboarded != nil {
		d.OnboardedAt = *onboarded
	}
	return d, nil
}

func (r *Repository) pgListOutlets(ctx context.Context, opts ListOpts) ([]models.Outlet, int) {
	opts = defaultLimit(opts)
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, address, channel, distributor_id, beat_id, qtd_value_ugx, frequency, score, status, lat, lng
		FROM dms_outlets
		WHERE ($1 = '' OR channel ILIKE $1)
		  AND ($2 = '' OR distributor_id = $2)
		  AND ($3 = '' OR beat_id = $3)
		  AND ($4 = '' OR status = $4)
		  AND ($5 = '' OR id ILIKE '%'||$5||'%' OR name ILIKE '%'||$5||'%')
		ORDER BY qtd_value_ugx DESC`, opts.Channel, opts.DistributorID, opts.BeatID, opts.Status, opts.Q)
	if err != nil {
		return nil, 0
	}
	defer rows.Close()
	var all []models.Outlet
	for rows.Next() {
		var o models.Outlet
		var lat, lng *float64
		if rows.Scan(&o.ID, &o.Name, &o.Address, &o.Channel, &o.DistributorID, &o.BeatID,
			&o.QTDValueUGX, &o.Frequency, &o.Score, &o.Status, &lat, &lng) == nil {
			if lat != nil {
				o.Lat = *lat
			}
			if lng != nil {
				o.Lng = *lng
			}
			all = append(all, o)
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgGetOutlet(ctx context.Context, id string) (models.Outlet, error) {
	items, _ := r.pgListOutlets(ctx, ListOpts{Q: id, Limit: 1})
	for _, o := range items {
		if o.ID == id {
			return o, nil
		}
	}
	var o models.Outlet
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, address, channel, distributor_id, beat_id, qtd_value_ugx, frequency, score, status, lat, lng
		FROM dms_outlets WHERE id = $1`, id).Scan(
		&o.ID, &o.Name, &o.Address, &o.Channel, &o.DistributorID, &o.BeatID,
		&o.QTDValueUGX, &o.Frequency, &o.Score, &o.Status, &o.Lat, &o.Lng)
	if err == pgx.ErrNoRows {
		return o, ErrNotFound
	}
	return o, err
}

func (r *Repository) pgCreateOutlet(ctx context.Context, in models.OutletInput) models.Outlet {
	id, _ := r.pgNextID(ctx, "OUT")
	o := models.Outlet{
		ID: id, Name: in.Name, Address: in.Address, Channel: in.Channel,
		DistributorID: in.DistributorID, BeatID: in.BeatID, Lat: in.Lat, Lng: in.Lng,
		Status: "active", Score: "B", Frequency: "1x/wk",
	}
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO dms_outlets (id, name, address, channel, distributor_id, beat_id, lat, lng, status, score, frequency)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		o.ID, o.Name, o.Address, o.Channel, o.DistributorID, o.BeatID, o.Lat, o.Lng, o.Status, o.Score, o.Frequency)
	_, _ = r.pool.Exec(ctx, `INSERT INTO dms_alerts (id, kind, title, detail) VALUES ($1,'outlet',$2,$3)`,
		uuid.NewString(), "Outlet activated · "+id, in.Name)
	return o
}

func (r *Repository) pgListOrders(ctx context.Context, opts ListOpts) ([]models.Order, int) {
	opts = defaultLimit(opts)
	rows, err := r.pool.Query(ctx, `
		SELECT id, outlet_id, outlet_name, distributor_id, rep_id, status, amount_ugx, currency, created_at, updated_at
		FROM dms_orders
		WHERE ($1 = '' OR status = $1) AND ($2 = '' OR distributor_id = $2)
		ORDER BY updated_at DESC`, opts.Status, opts.DistributorID)
	if err != nil {
		return nil, 0
	}
	defer rows.Close()
	var all []models.Order
	for rows.Next() {
		var o models.Order
		if rows.Scan(&o.ID, &o.OutletID, &o.OutletName, &o.DistributorID, &o.RepID, &o.Status, &o.AmountUGX, &o.Currency, &o.CreatedAt, &o.UpdatedAt) == nil {
			all = append(all, o)
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgGetOrder(ctx context.Context, id string) (models.Order, error) {
	var o models.Order
	err := r.pool.QueryRow(ctx, `
		SELECT id, outlet_id, outlet_name, distributor_id, rep_id, status, amount_ugx, currency, created_at, updated_at
		FROM dms_orders WHERE id = $1`, id).Scan(
		&o.ID, &o.OutletID, &o.OutletName, &o.DistributorID, &o.RepID, &o.Status, &o.AmountUGX, &o.Currency, &o.CreatedAt, &o.UpdatedAt)
	if err == pgx.ErrNoRows {
		return o, ErrNotFound
	}
	return o, err
}

func (r *Repository) pgCreateOrder(ctx context.Context, in models.OrderInput) models.Order {
	id, _ := r.pgNextID(ctx, "SO")
	outletName := ""
	_ = r.pool.QueryRow(ctx, `SELECT name FROM dms_outlets WHERE id = $1`, in.OutletID).Scan(&outletName)
	ts := now()
	o := models.Order{
		ID: id, OutletID: in.OutletID, OutletName: outletName, DistributorID: in.DistributorID,
		RepID: in.RepID, Status: "draft", AmountUGX: in.AmountUGX, Currency: "UGX", CreatedAt: ts, UpdatedAt: ts,
	}
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO dms_orders (id, outlet_id, outlet_name, distributor_id, rep_id, status, amount_ugx, currency, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`,
		o.ID, o.OutletID, o.OutletName, o.DistributorID, o.RepID, o.Status, o.AmountUGX, o.Currency, ts)
	return o
}

func (r *Repository) pgUpdateOrderStatus(ctx context.Context, id, status string) (models.Order, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE dms_orders SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return models.Order{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Order{}, ErrNotFound
	}
	return r.pgGetOrder(ctx, id)
}

func (r *Repository) pgOrdersBoard(ctx context.Context) map[string][]models.Order {
	items, _ := r.pgListOrders(ctx, ListOpts{Limit: 500})
	board := map[string][]models.Order{
		"draft": {}, "submitted": {}, "picking": {}, "delivery": {}, "delivered": {},
	}
	for _, o := range items {
		key := o.Status
		if _, ok := board[key]; !ok {
			key = "submitted"
		}
		board[key] = append(board[key], o)
	}
	return board
}

func (r *Repository) pgListBeats(ctx context.Context, opts ListOpts) ([]models.Beat, int) {
	opts = defaultLimit(opts)
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, rep_id, rep_name, stop_count, distance_km, status, progress FROM dms_beats
		WHERE ($1 = '' OR rep_id = $1) ORDER BY name`, opts.RepID)
	if err != nil {
		return nil, 0
	}
	defer rows.Close()
	var all []models.Beat
	for rows.Next() {
		var b models.Beat
		if rows.Scan(&b.ID, &b.Name, &b.RepID, &b.RepName, &b.StopCount, &b.DistanceKm, &b.Status, &b.Progress) == nil {
			all = append(all, b)
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgGetBeat(ctx context.Context, id string) (models.Beat, error) {
	var b models.Beat
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, rep_id, rep_name, stop_count, distance_km, status, progress FROM dms_beats WHERE id = $1`, id).
		Scan(&b.ID, &b.Name, &b.RepID, &b.RepName, &b.StopCount, &b.DistanceKm, &b.Status, &b.Progress)
	if err == pgx.ErrNoRows {
		return b, ErrNotFound
	}
	return b, err
}

func (r *Repository) pgListReps(ctx context.Context, opts ListOpts) ([]models.FieldRep, int) {
	opts = defaultLimit(opts)
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, beat_id, region, level, status FROM dms_field_reps
		WHERE ($1 = '' OR status = $1) ORDER BY name`, opts.Status)
	if err != nil {
		return nil, 0
	}
	defer rows.Close()
	var all []models.FieldRep
	for rows.Next() {
		var rep models.FieldRep
		if rows.Scan(&rep.ID, &rep.Name, &rep.BeatID, &rep.Region, &rep.Level, &rep.Status) == nil {
			all = append(all, rep)
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgListCheckIns(ctx context.Context, opts ListOpts) ([]models.CheckIn, int) {
	opts = defaultLimit(opts)
	rows, err := r.pool.Query(ctx, `
		SELECT id, rep_id, outlet_id, lat, lng, arrived_at, status FROM dms_check_ins
		WHERE ($1 = '' OR rep_id = $1) ORDER BY arrived_at DESC`, opts.RepID)
	if err != nil {
		return nil, 0
	}
	defer rows.Close()
	var all []models.CheckIn
	for rows.Next() {
		var c models.CheckIn
		if rows.Scan(&c.ID, &c.RepID, &c.OutletID, &c.Lat, &c.Lng, &c.ArrivedAt, &c.Status) == nil {
			all = append(all, c)
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgCreateCheckIn(ctx context.Context, in models.CheckInInput) models.CheckIn {
	ci := models.CheckIn{
		ID: uuid.NewString(), RepID: in.RepID, OutletID: in.OutletID,
		Lat: in.Lat, Lng: in.Lng, ArrivedAt: now(), Status: "active",
	}
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO dms_check_ins (id, rep_id, outlet_id, lat, lng, arrived_at, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, ci.ID, ci.RepID, ci.OutletID, ci.Lat, ci.Lng, ci.ArrivedAt, ci.Status)
	_, _ = r.pool.Exec(ctx, `UPDATE dms_field_reps SET status = 'clocked_in' WHERE id = $1`, in.RepID)
	return ci
}

func (r *Repository) pgCreateVisitReport(ctx context.Context, in models.VisitReportInput) models.VisitReport {
	v := models.VisitReport{
		ID: uuid.NewString(), RepID: in.RepID, OutletID: in.OutletID,
		Outcome: in.Outcome, Notes: in.Notes, Lat: in.Lat, Lng: in.Lng, CreatedAt: now(),
	}
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO dms_visit_reports (id, rep_id, outlet_id, outcome, notes, lat, lng, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		v.ID, v.RepID, v.OutletID, v.Outcome, v.Notes, v.Lat, v.Lng, v.CreatedAt)
	return v
}

func (r *Repository) pgListPromotions(ctx context.Context, opts ListOpts) ([]models.Promotion, int) {
	opts = defaultLimit(opts)
	rows, _ := r.pool.Query(ctx, `SELECT id, name, sku, roi, status, outlets FROM dms_promotions ORDER BY roi DESC`)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	var all []models.Promotion
	if rows != nil {
		for rows.Next() {
			var p models.Promotion
			if rows.Scan(&p.ID, &p.Name, &p.SKU, &p.ROI, &p.Status, &p.Outlets) == nil {
				all = append(all, p)
			}
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgListClaims(ctx context.Context, opts ListOpts) ([]models.Claim, int) {
	opts = defaultLimit(opts)
	rows, _ := r.pool.Query(ctx, `
		SELECT id, outlet_id, claim_type, status, amount_ugx, created_at FROM dms_claims
		WHERE ($1 = '' OR status = $1) ORDER BY created_at DESC`, opts.Status)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	var all []models.Claim
	if rows != nil {
		for rows.Next() {
			var c models.Claim
			if rows.Scan(&c.ID, &c.OutletID, &c.Type, &c.Status, &c.AmountUGX, &c.CreatedAt) == nil {
				all = append(all, c)
			}
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgListDispatches(ctx context.Context, opts ListOpts) ([]models.Dispatch, int) {
	opts = defaultLimit(opts)
	rows, _ := r.pool.Query(ctx, `SELECT id, truck_id, driver, status, eta, updated_at FROM dms_dispatches ORDER BY updated_at DESC`)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	var all []models.Dispatch
	if rows != nil {
		for rows.Next() {
			var d models.Dispatch
			if rows.Scan(&d.ID, &d.TruckID, &d.Driver, &d.Status, &d.ETA, &d.UpdatedAt) == nil {
				rows2, _ := r.pool.Query(ctx, `SELECT order_id FROM dms_dispatch_orders WHERE dispatch_id = $1`, d.ID)
				if rows2 != nil {
					for rows2.Next() {
						var oid string
						if rows2.Scan(&oid) == nil {
							d.OrderIDs = append(d.OrderIDs, oid)
						}
					}
					rows2.Close()
				}
				all = append(all, d)
			}
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgListStock(ctx context.Context, opts ListOpts) ([]models.StockPosition, int) {
	opts = defaultLimit(opts)
	rows, _ := r.pool.Query(ctx, `
		SELECT distributor_id, sku, cover_days, qty, status FROM dms_stock_positions
		WHERE ($1 = '' OR distributor_id = $1)`, opts.DistributorID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	var all []models.StockPosition
	if rows != nil {
		for rows.Next() {
			var s models.StockPosition
			if rows.Scan(&s.DistributorID, &s.SKU, &s.CoverDays, &s.Qty, &s.Status) == nil {
				all = append(all, s)
			}
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgListSKUs(ctx context.Context, opts ListOpts) ([]models.SKU, int) {
	opts = defaultLimit(opts)
	rows, _ := r.pool.Query(ctx, `
		SELECT code, name, cover_days, qty_on_hand, warehouse, sca_score FROM dms_skus
		WHERE ($1 = '' OR code ILIKE '%'||$1||'%' OR name ILIKE '%'||$1||'%')
		ORDER BY code`, opts.Q)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	var all []models.SKU
	if rows != nil {
		for rows.Next() {
			var s models.SKU
			if rows.Scan(&s.Code, &s.Name, &s.CoverDays, &s.QtyOnHand, &s.Warehouse, &s.SCAScore) == nil {
				all = append(all, s)
			}
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgListInvoices(ctx context.Context, opts ListOpts) ([]models.Invoice, int) {
	opts = defaultLimit(opts)
	rows, _ := r.pool.Query(ctx, `
		SELECT id, distributor_id, distributor_name, amount_ugx, due_date, status, COALESCE(order_id,'')
		FROM dms_invoices
		WHERE ($1 = '' OR status = $1) AND ($2 = '' OR distributor_id = $2)
		ORDER BY due_date`, opts.Status, opts.DistributorID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	var all []models.Invoice
	if rows != nil {
		for rows.Next() {
			var inv models.Invoice
			var due time.Time
			if rows.Scan(&inv.ID, &inv.DistributorID, &inv.Distributor, &inv.AmountUGX, &due, &inv.Status, &inv.OrderID) == nil {
				inv.DueDate = due
				all = append(all, inv)
			}
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgCreateInvoice(ctx context.Context, in models.InvoiceInput) models.Invoice {
	id, _ := r.pgNextID(ctx, "INV")
	name := ""
	_ = r.pool.QueryRow(ctx, `SELECT name FROM dms_distributors WHERE id = $1`, in.DistributorID).Scan(&name)
	due := in.DueDate
	if due.IsZero() {
		due = time.Now().UTC().AddDate(0, 1, 0)
	}
	inv := models.Invoice{
		ID: id, DistributorID: in.DistributorID, Distributor: name,
		AmountUGX: in.AmountUGX, DueDate: due, Status: "open", OrderID: in.OrderID,
	}
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO dms_invoices (id, distributor_id, distributor_name, amount_ugx, due_date, status, order_id)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''))`,
		inv.ID, inv.DistributorID, inv.Distributor, inv.AmountUGX, inv.DueDate, inv.Status, inv.OrderID)
	return inv
}

func (r *Repository) pgListPricing(ctx context.Context) []models.PricingTemplate {
	rows, _ := r.pool.Query(ctx, `SELECT id, name, channel, version, currency FROM dms_pricing_templates ORDER BY id`)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	var out []models.PricingTemplate
	if rows != nil {
		for rows.Next() {
			var p models.PricingTemplate
			if rows.Scan(&p.ID, &p.Name, &p.Channel, &p.Version, &p.Currency) == nil {
				out = append(out, p)
			}
		}
	}
	return out
}

func (r *Repository) pgListReports(ctx context.Context) []models.ReportTemplate {
	rows, _ := r.pool.Query(ctx, `SELECT id, name, data_source, schedule FROM dms_report_templates ORDER BY id`)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	var out []models.ReportTemplate
	if rows != nil {
		for rows.Next() {
			var rt models.ReportTemplate
			if rows.Scan(&rt.ID, &rt.Name, &rt.DataSource, &rt.Schedule) == nil {
				out = append(out, rt)
			}
		}
	}
	return out
}

func (r *Repository) pgListExecution(ctx context.Context, opts ListOpts) ([]models.ExecutionTask, int) {
	opts = defaultLimit(opts)
	rows, _ := r.pool.Query(ctx, `SELECT id, outlet_id, task_type, status, detail FROM dms_execution_tasks ORDER BY id`)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	var all []models.ExecutionTask
	if rows != nil {
		for rows.Next() {
			var t models.ExecutionTask
			if rows.Scan(&t.ID, &t.OutletID, &t.Type, &t.Status, &t.Detail) == nil {
				all = append(all, t)
			}
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgFinanceSummary(ctx context.Context) models.FinanceSummary {
	var ar, overdue float64
	_ = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_ugx),0) FROM dms_invoices`).Scan(&ar)
	_ = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_ugx),0) FROM dms_invoices WHERE status = 'overdue'`).Scan(&overdue)
	return models.FinanceSummary{ARBalanceUGX: ar, DSODays: 32.4, OverdueUGX: overdue, CollectedUGX: 18_400_000}
}

func (r *Repository) pgSearch(ctx context.Context, q string, limit int) []models.SearchResult {
	if limit <= 0 {
		limit = 18
	}
	var out []models.SearchResult
	if q == "" {
		for _, p := range pagesCatalog() {
			if len(out) >= limit {
				break
			}
			out = append(out, models.SearchResult{Kind: "page", Label: p.Title, Sub: "Module", Page: p.ID})
		}
		return out
	}
	outlets, _ := r.pgListOutlets(ctx, ListOpts{Q: q, Limit: limit})
	for _, o := range outlets {
		out = append(out, models.SearchResult{Kind: "outlet", Label: o.Name, Sub: o.ID + " · " + o.Channel, Page: "outlets", ID: o.ID})
	}
	if len(out) >= limit {
		return out[:limit]
	}
	dists, _ := r.pgListDistributors(ctx, ListOpts{Q: q, Limit: limit})
	for _, d := range dists {
		out = append(out, models.SearchResult{Kind: "distrib", Label: d.Name, Sub: d.ID + " · " + d.Region, Page: "network", ID: d.ID})
	}
	return out
}

func (r *Repository) pgNextID(ctx context.Context, prefix string) (string, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO dms_id_counters (prefix, next_value) VALUES ($1, 2)
		ON CONFLICT (prefix) DO UPDATE SET next_value = dms_id_counters.next_value + 1
		RETURNING next_value - 1`, prefix).Scan(&n)
	if err != nil {
		return prefix + "-" + uuid.NewString()[:8], err
	}
	return fmt.Sprintf("%s-%05d", prefix, n), nil
}
