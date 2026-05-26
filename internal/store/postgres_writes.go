package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/iag/dms/backend/internal/models"
)

func (r *Repository) pgPatchOutlet(ctx context.Context, id string, patch models.OutletPatch) (models.Outlet, error) {
	o, err := r.pgGetOutlet(ctx, id)
	if err != nil {
		return o, err
	}
	if patch.Name != "" {
		o.Name = patch.Name
	}
	if patch.Address != "" {
		o.Address = patch.Address
	}
	if patch.Channel != "" {
		o.Channel = patch.Channel
	}
	if patch.BeatID != "" {
		o.BeatID = patch.BeatID
	}
	if patch.Status != "" {
		o.Status = patch.Status
	}
	if patch.Score != "" {
		o.Score = patch.Score
	}
	if patch.Frequency != "" {
		o.Frequency = patch.Frequency
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE dms_outlets SET name=$2, address=$3, channel=$4, beat_id=$5, status=$6, score=$7, frequency=$8
		WHERE id=$1`,
		o.ID, o.Name, o.Address, o.Channel, o.BeatID, o.Status, o.Score, o.Frequency)
	if err != nil {
		return models.Outlet{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Outlet{}, ErrNotFound
	}
	return o, nil
}

func (r *Repository) pgGetInvoice(ctx context.Context, id string) (models.Invoice, error) {
	var inv models.Invoice
	err := r.pool.QueryRow(ctx, `
		SELECT id, distributor_id, distributor_name, amount_ugx, due_date, status, COALESCE(order_id,'')
		FROM dms_invoices WHERE id = $1`, id).Scan(
		&inv.ID, &inv.DistributorID, &inv.Distributor, &inv.AmountUGX, &inv.DueDate, &inv.Status, &inv.OrderID)
	if err == pgx.ErrNoRows {
		return inv, ErrNotFound
	}
	return inv, err
}

func (r *Repository) pgListVisitReports(ctx context.Context, opts ListOpts) ([]models.VisitReport, int) {
	opts = defaultLimit(opts)
	q := `SELECT id, rep_id, outlet_id, outcome, notes, lat, lng, created_at FROM dms_visit_reports WHERE ($1 = '' OR rep_id = $1) ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, opts.RepID)
	if err != nil {
		return nil, 0
	}
	defer rows.Close()
	var all []models.VisitReport
	for rows.Next() {
		var v models.VisitReport
		if rows.Scan(&v.ID, &v.RepID, &v.OutletID, &v.Outcome, &v.Notes, &v.Lat, &v.Lng, &v.CreatedAt) == nil {
			all = append(all, v)
		}
	}
	return paginate(all, opts)
}

func (r *Repository) pgCompleteCheckIn(ctx context.Context, id string) (models.CheckIn, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE dms_check_ins SET status = 'completed' WHERE id = $1`, id)
	if err != nil {
		return models.CheckIn{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.CheckIn{}, ErrNotFound
	}
	var ci models.CheckIn
	err = r.pool.QueryRow(ctx, `
		SELECT id, rep_id, outlet_id, lat, lng, arrived_at, status FROM dms_check_ins WHERE id = $1`, id).
		Scan(&ci.ID, &ci.RepID, &ci.OutletID, &ci.Lat, &ci.Lng, &ci.ArrivedAt, &ci.Status)
	if err != nil {
		return ci, err
	}
	_, _ = r.pool.Exec(ctx, `UPDATE dms_field_reps SET status = 'active' WHERE id = $1 AND status = 'clocked_in'`, ci.RepID)
	return ci, nil
}

func (r *Repository) pgCreateClaim(ctx context.Context, in models.ClaimInput) models.Claim {
	id, _ := r.pgNextID(ctx, "CLM")
	c := models.Claim{
		ID: id, OutletID: in.OutletID, Type: in.Type,
		Status: "open", AmountUGX: in.AmountUGX, CreatedAt: now(),
	}
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO dms_claims (id, outlet_id, claim_type, status, amount_ugx, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		c.ID, c.OutletID, c.Type, c.Status, c.AmountUGX, c.CreatedAt)
	return c
}

func (r *Repository) pgCreatePromotion(ctx context.Context, in models.PromotionInput) models.Promotion {
	id, _ := r.pgNextID(ctx, "TPM")
	roi := in.ROI
	if roi == 0 {
		roi = 2.0
	}
	p := models.Promotion{
		ID: id, Name: in.Name, SKU: in.SKU, ROI: roi, Status: "active", Outlets: in.Outlets,
	}
	_, _ = r.pool.Exec(ctx, `INSERT INTO dms_promotions (id, name, sku, roi, status, outlets) VALUES ($1,$2,$3,$4,$5,$6)`,
		p.ID, p.Name, p.SKU, p.ROI, p.Status, p.Outlets)
	return p
}

func (r *Repository) pgCreateDispatch(ctx context.Context, in models.DispatchInput) models.Dispatch {
	id, _ := r.pgNextID(ctx, "DXP")
	eta := in.ETA
	if eta == "" {
		eta = "< 6h"
	}
	d := models.Dispatch{
		ID: id, TruckID: in.TruckID, Driver: in.Driver,
		OrderIDs: in.OrderIDs, Status: "planned", ETA: eta, UpdatedAt: now(),
	}
	_, _ = r.pool.Exec(ctx, `INSERT INTO dms_dispatches (id, truck_id, driver, status, eta, updated_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		d.ID, d.TruckID, d.Driver, d.Status, d.ETA, d.UpdatedAt)
	for _, oid := range in.OrderIDs {
		_, _ = r.pool.Exec(ctx, `INSERT INTO dms_dispatch_orders (dispatch_id, order_id) VALUES ($1,$2)`, d.ID, oid)
	}
	return d
}

func (r *Repository) pgRunReport(ctx context.Context, in models.ReportRunInput) models.ReportRun {
	name := strings.TrimSpace(in.Name)
	if name == "" && in.TemplateID != "" {
		_ = r.pool.QueryRow(ctx, `SELECT name FROM dms_report_templates WHERE id = $1`, in.TemplateID).Scan(&name)
	}
	if name == "" {
		name = "Custom report"
	}
	var rowCount int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM dms_outlets`).Scan(&rowCount)
	return models.ReportRun{
		JobID: uuid.NewString(), Name: name, Status: "queued",
		RowCount: rowCount, Message: "Report queued for generation and email delivery",
	}
}

func (r *Repository) pgExportPage(ctx context.Context, page, format string) models.ExportPayload {
	if format == "" {
		format = "json"
	}
	rows := r.pgExportRows(ctx, page)
	return models.ExportPayload{
		Page: page, Format: format, GeneratedAt: now(),
		RowCount: len(rows), Rows: rows,
	}
}

func (r *Repository) pgExportRows(ctx context.Context, page string) []map[string]any {
	switch page {
	case "outlets":
		items, _ := r.pgListOutlets(ctx, ListOpts{Limit: 5000})
		out := make([]map[string]any, 0, len(items))
		for _, o := range items {
			out = append(out, map[string]any{"id": o.ID, "name": o.Name, "channel": o.Channel, "status": o.Status})
		}
		return out
	case "orders":
		items, _ := r.pgListOrders(ctx, ListOpts{Limit: 5000})
		out := make([]map[string]any, 0, len(items))
		for _, o := range items {
			out = append(out, map[string]any{"id": o.ID, "outlet": o.OutletName, "status": o.Status, "amountUgx": o.AmountUGX})
		}
		return out
	case "invoices":
		items, _ := r.pgListInvoices(ctx, ListOpts{Limit: 5000})
		out := make([]map[string]any, 0, len(items))
		for _, inv := range items {
			out = append(out, map[string]any{"id": inv.ID, "distributor": inv.Distributor, "amountUgx": inv.AmountUGX, "status": inv.Status})
		}
		return out
	case "network":
		items, _ := r.pgListDistributors(ctx, ListOpts{Limit: 5000})
		out := make([]map[string]any, 0, len(items))
		for _, d := range items {
			out = append(out, map[string]any{"id": d.ID, "name": d.Name, "region": d.Region, "revenueUgx": d.RevenueUGX})
		}
		return out
	default:
		return []map[string]any{{"page": page, "note": fmt.Sprintf("export snapshot for %s", page)}}
	}
}
