package store

import (
	"context"
	"encoding/json"
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
		SELECT id, distributor_id, distributor_name, amount_ugx, due_date, status, COALESCE(order_id,''),
		       COALESCE(efris_status,''), COALESCE(ura_receipt,''), COALESCE(document_url,'')
		FROM dms_invoices WHERE id = $1`, id).Scan(
		&inv.ID, &inv.DistributorID, &inv.Distributor, &inv.AmountUGX, &inv.DueDate, &inv.Status, &inv.OrderID,
		&inv.EFRISStatus, &inv.URAReceipt, &inv.DocumentURL)
	if err == pgx.ErrNoRows {
		return inv, ErrNotFound
	}
	return inv, err
}

// SetInvoiceEFRIS records fiscalisation status + URA receipt on an invoice.
func (r *Repository) SetInvoiceEFRIS(id, status, receipt string) (models.Invoice, error) {
	if r.pool != nil {
		tag, err := r.pool.Exec(r.bg(), `UPDATE dms_invoices SET efris_status=$2, ura_receipt=$3 WHERE id=$1`, id, status, receipt)
		if err != nil {
			return models.Invoice{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.Invoice{}, ErrNotFound
		}
		return r.pgGetInvoice(r.bg(), id)
	}
	return r.mem.setInvoiceEFRIS(id, status, receipt)
}

// SetInvoiceDocument records the generated document URL on an invoice.
func (r *Repository) SetInvoiceDocument(id, url string) error {
	if r.pool != nil {
		_, err := r.pool.Exec(r.bg(), `UPDATE dms_invoices SET document_url=$2 WHERE id=$1`, id, url)
		return err
	}
	return r.mem.setInvoiceDocument(id, url)
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

// ---- Generic deletes -------------------------------------------------------

func (r *Repository) pgDelete(ctx context.Context, table, id string) error {
	tag, err := r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id=$1`, table), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) pgDeleteOutlet(ctx context.Context, id string) error {
	return r.pgDelete(ctx, "dms_outlets", id)
}

func (r *Repository) pgDeleteOrder(ctx context.Context, id string) error {
	return r.pgDelete(ctx, "dms_orders", id)
}

func (r *Repository) pgDeleteCheckIn(ctx context.Context, id string) error {
	return r.pgDelete(ctx, "dms_check_ins", id)
}

func (r *Repository) pgDeletePromotion(ctx context.Context, id string) error {
	return r.pgDelete(ctx, "dms_promotions", id)
}

func (r *Repository) pgDeleteClaim(ctx context.Context, id string) error {
	return r.pgDelete(ctx, "dms_claims", id)
}

func (r *Repository) pgDeleteDispatch(ctx context.Context, id string) error {
	if err := r.pgDelete(ctx, "dms_dispatches", id); err != nil {
		return err
	}
	_, _ = r.pool.Exec(ctx, `DELETE FROM dms_dispatch_orders WHERE dispatch_id=$1`, id)
	return nil
}

func (r *Repository) pgDeleteInvoice(ctx context.Context, id string) error {
	return r.pgDelete(ctx, "dms_invoices", id)
}

// ---- Generic patches -------------------------------------------------------

func (r *Repository) pgPatchPromotion(ctx context.Context, id string, patch models.PromotionPatch) (models.Promotion, error) {
	var p models.Promotion
	err := r.pool.QueryRow(ctx, `SELECT id, name, sku, roi, status, outlets FROM dms_promotions WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.SKU, &p.ROI, &p.Status, &p.Outlets)
	if err == pgx.ErrNoRows {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if patch.Name != "" {
		p.Name = patch.Name
	}
	if patch.SKU != "" {
		p.SKU = patch.SKU
	}
	if patch.Status != "" {
		p.Status = patch.Status
	}
	if patch.ROI != nil {
		p.ROI = *patch.ROI
	}
	if patch.Outlets != nil {
		p.Outlets = *patch.Outlets
	}
	_, err = r.pool.Exec(ctx, `UPDATE dms_promotions SET name=$2, sku=$3, roi=$4, status=$5, outlets=$6 WHERE id=$1`,
		p.ID, p.Name, p.SKU, p.ROI, p.Status, p.Outlets)
	return p, err
}

func (r *Repository) pgPatchClaim(ctx context.Context, id string, patch models.ClaimPatch) (models.Claim, error) {
	var c models.Claim
	err := r.pool.QueryRow(ctx, `SELECT id, outlet_id, claim_type, status, amount_ugx, created_at FROM dms_claims WHERE id=$1`, id).
		Scan(&c.ID, &c.OutletID, &c.Type, &c.Status, &c.AmountUGX, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	if patch.Type != "" {
		c.Type = patch.Type
	}
	if patch.Status != "" {
		c.Status = patch.Status
	}
	if patch.AmountUGX != nil {
		c.AmountUGX = *patch.AmountUGX
	}
	_, err = r.pool.Exec(ctx, `UPDATE dms_claims SET claim_type=$2, status=$3, amount_ugx=$4 WHERE id=$1`,
		c.ID, c.Type, c.Status, c.AmountUGX)
	return c, err
}

func (r *Repository) pgPatchDispatch(ctx context.Context, id string, patch models.DispatchPatch) (models.Dispatch, error) {
	var d models.Dispatch
	err := r.pool.QueryRow(ctx, `SELECT id, truck_id, driver, status, eta, updated_at FROM dms_dispatches WHERE id=$1`, id).
		Scan(&d.ID, &d.TruckID, &d.Driver, &d.Status, &d.ETA, &d.UpdatedAt)
	if err == pgx.ErrNoRows {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	if patch.TruckID != "" {
		d.TruckID = patch.TruckID
	}
	if patch.Driver != "" {
		d.Driver = patch.Driver
	}
	if patch.Status != "" {
		d.Status = patch.Status
	}
	if patch.ETA != "" {
		d.ETA = patch.ETA
	}
	d.UpdatedAt = now()
	_, err = r.pool.Exec(ctx, `UPDATE dms_dispatches SET truck_id=$2, driver=$3, status=$4, eta=$5, updated_at=$6 WHERE id=$1`,
		d.ID, d.TruckID, d.Driver, d.Status, d.ETA, d.UpdatedAt)
	if err == nil {
		d.OrderIDs = r.pgDispatchOrderIDs(ctx, d.ID)
	}
	return d, err
}

func (r *Repository) pgDispatchOrderIDs(ctx context.Context, dispatchID string) []string {
	rows, err := r.pool.Query(ctx, `SELECT order_id FROM dms_dispatch_orders WHERE dispatch_id=$1`, dispatchID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var oid string
		if rows.Scan(&oid) == nil {
			out = append(out, oid)
		}
	}
	return out
}

func (r *Repository) pgPatchInvoice(ctx context.Context, id string, patch models.InvoicePatch) (models.Invoice, error) {
	inv, err := r.pgGetInvoice(ctx, id)
	if err != nil {
		return inv, err
	}
	if patch.AmountUGX != nil {
		inv.AmountUGX = *patch.AmountUGX
	}
	if patch.Status != "" {
		inv.Status = patch.Status
	}
	if patch.DueDate != nil {
		inv.DueDate = *patch.DueDate
	}
	_, err = r.pool.Exec(ctx, `UPDATE dms_invoices SET amount_ugx=$2, status=$3, due_date=$4 WHERE id=$1`,
		inv.ID, inv.AmountUGX, inv.Status, inv.DueDate)
	return inv, err
}

func (r *Repository) pgRunReport(ctx context.Context, in models.ReportRunInput) models.ReportRun {
	name := strings.TrimSpace(in.Name)
	page := "outlets"
	if name == "" && in.TemplateID != "" {
		_ = r.pool.QueryRow(ctx, `SELECT name, data_source FROM dms_report_templates WHERE id = $1`, in.TemplateID).Scan(&name, &page)
	}
	if name == "" {
		name = "Custom report"
	}
	if page == "" {
		page = "outlets"
	}
	jobID := uuid.NewString()
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO dms_report_jobs (id, name, template_id, status, email_to, created_at)
		VALUES ($1,$2,$3,'running',$4,NOW())
	`, jobID, name, in.TemplateID, in.EmailTo)
	payload := r.pgExportPage(ctx, page, "json")
	raw, _ := json.Marshal(payload)
	_, _ = r.pool.Exec(ctx, `
		UPDATE dms_report_jobs
		SET status = 'completed', row_count = $2, result = $3::jsonb, completed_at = NOW()
		WHERE id = $1
	`, jobID, payload.RowCount, raw)
	msg := fmt.Sprintf("Report generated (%d rows)", payload.RowCount)
	if strings.TrimSpace(in.EmailTo) != "" {
		msg += " — email delivery pending notifications integration"
	}
	return models.ReportRun{
		JobID: jobID, Name: name, Status: "completed",
		RowCount: payload.RowCount, Message: msg,
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
