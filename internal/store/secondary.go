package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/iag/dms/backend/internal/models"
)

// ErrConflict is a write the rules refuse: editing a confirmed amount,
// over-collecting an invoice, confirming an unexplained close.
var ErrConflict = errors.New("conflict")

// Secondary holds the eight typed-document tables of the secondary-sales
// desk. It is attached to Repository so the rules below can also read
// outlets and beats.
type Secondary struct {
	vanLoads *docTable[models.VanLoad]
	invoices *docTable[models.SecondaryInvoice]
	collects *docTable[models.Collection]
	returns  *docTable[models.OutletReturn]
	recons   *docTable[models.VanRecon]
	schemes  *docTable[models.Scheme]
	targets  *docTable[models.RepTarget]
	claims   *docTable[models.DamageClaim]
}

func (r *Repository) secondary() *Secondary {
	if r.sec == nil {
		r.sec = &Secondary{
			vanLoads: newDocTable[models.VanLoad](r.pool, "dms_van_loads"),
			invoices: newDocTable[models.SecondaryInvoice](r.pool, "dms_secondary_invoices"),
			collects: newDocTable[models.Collection](r.pool, "dms_collections"),
			returns:  newDocTable[models.OutletReturn](r.pool, "dms_outlet_returns"),
			recons:   newDocTable[models.VanRecon](r.pool, "dms_van_recons"),
			schemes:  newDocTable[models.Scheme](r.pool, "dms_schemes"),
			targets:  newDocTable[models.RepTarget](r.pool, "dms_rep_targets"),
			claims:   newDocTable[models.DamageClaim](r.pool, "dms_damage_claims"),
		}
	}
	return r.sec
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

var cashTerms = regexp.MustCompile(`(?i)cash|cod|immediate`)

func isCashTerms(terms string) bool { return cashTerms.MatchString(terms) }

func isConfirmedInvoice(status string) bool {
	switch lower(status) {
	case "invoiced", "paid", "confirmed":
		return true
	}
	return false
}

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, a...))
}

func conflictf(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrConflict, fmt.Sprintf(format, a...))
}

/* ───────────────────────────── van loads ───────────────────────────── */

func (r *Repository) ListVanLoads(f DocFilter) ([]models.VanLoad, int) {
	return r.secondary().vanLoads.list(r.bg(), f)
}

func (r *Repository) GetVanLoad(id string) (models.VanLoad, error) {
	return r.secondary().vanLoads.get(r.bg(), id)
}

// normaliseVanLoad applies FEFO: batches are ordered nearest-expiry first and
// the headline batch/expiry is the first one when the caller left it blank.
func normaliseVanLoad(v *models.VanLoad) error {
	v.SaleKind = "primary"
	v.PostingClass = "internal-transfer"
	if v.Status == "" {
		v.Status = "loaded"
	}
	v.Status = lower(v.Status)
	if v.Quantity <= 0 && len(v.Batches) == 0 {
		return invalid("a van load needs a quantity")
	}
	sortByExpiry(v.Batches, func(b models.VanLoadBatch) string { return b.ExpiryDate })
	if len(v.Batches) > 0 {
		var total float64
		for _, b := range v.Batches {
			total += b.Quantity
		}
		if v.Quantity <= 0 {
			v.Quantity = total
		}
		if v.Batch == "" {
			v.Batch = v.Batches[0].Batch
			v.ExpiryDate = v.Batches[0].ExpiryDate
		}
	}
	v.Quantity = round2(v.Quantity)
	return nil
}

func (r *Repository) CreateVanLoad(in models.VanLoad) (models.VanLoad, error) {
	ctx := r.bg()
	s := r.secondary()
	if err := normaliseVanLoad(&in); err != nil {
		return models.VanLoad{}, err
	}
	if in.VehicleID == "" {
		return models.VanLoad{}, invalid("a van load needs a vehicle")
	}
	in.ID = s.vanLoads.nextID(ctx, r, "VAN")
	in.CreatedAt, in.UpdatedAt = now(), now()
	return in, s.vanLoads.insert(ctx, in)
}

func (r *Repository) UpdateVanLoad(id string, in models.VanLoad) (models.VanLoad, error) {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.vanLoads.get(ctx, id)
	if err != nil {
		return cur, err
	}
	in.ID, in.CreatedAt, in.UpdatedAt = cur.ID, cur.CreatedAt, now()
	if err := normaliseVanLoad(&in); err != nil {
		return cur, err
	}
	// Quantity on a load that already has invoices against it is history.
	if in.Quantity != cur.Quantity {
		if sold := r.vanSold(ctx, id); sold > 0 {
			return cur, conflictf("van load %s has %.2f sold against it; load a new van rather than editing this one", id, sold)
		}
	}
	return in, s.vanLoads.update(ctx, in)
}

func (r *Repository) DeleteVanLoad(id string) error {
	ctx := r.bg()
	if sold := r.vanSold(ctx, id); sold > 0 {
		return conflictf("van load %s has invoices against it", id)
	}
	return r.secondary().vanLoads.delete(ctx, id)
}

// vanSold is the confirmed quantity invoiced against a van load.
func (r *Repository) vanSold(ctx context.Context, vanLoadID string) float64 {
	rows, _ := r.secondary().invoices.list(ctx, DocFilter{RefID: vanLoadID, Limit: 5000})
	var sold float64
	for _, inv := range rows {
		if isConfirmedInvoice(inv.Status) {
			sold += inv.Quantity
		}
	}
	return round2(sold)
}

// vanReturned is the quantity outlets have sent back onto a van load —
// goods that are on the van again, not goods handed back to the warehouse.
func (r *Repository) vanReturned(ctx context.Context, vanLoadID string) float64 {
	rows, _ := r.secondary().returns.list(ctx, DocFilter{Limit: 5000})
	var back float64
	for _, ret := range rows {
		if ret.VanLoadID == vanLoadID && lower(ret.Status) != "cancelled" {
			back += ret.Quantity
		}
	}
	return round2(back)
}

// VanOnHand is loaded − sold + outlet returns for one van load.
func (r *Repository) VanOnHand(vanLoadID string) (float64, error) {
	ctx := r.bg()
	load, err := r.secondary().vanLoads.get(ctx, vanLoadID)
	if err != nil {
		return 0, err
	}
	return round2(math.Max(0, load.Quantity-r.vanSold(ctx, vanLoadID)+r.vanReturned(ctx, vanLoadID))), nil
}

/* ───────────────────────── secondary invoices ──────────────────────── */

func (r *Repository) ListSecondaryInvoices(f DocFilter) ([]models.SecondaryInvoice, int) {
	return r.secondary().invoices.list(r.bg(), f)
}

func (r *Repository) GetSecondaryInvoice(id string) (models.SecondaryInvoice, error) {
	return r.secondary().invoices.get(r.bg(), id)
}

// OutletOutstanding is the live receivable: the outstanding of every confirmed
// invoice for the outlet. Recomputed, never incremented, so it self-heals.
func (r *Repository) OutletOutstanding(outletID string) float64 {
	rows, _ := r.secondary().invoices.list(r.bg(), DocFilter{OutletID: outletID, Limit: 5000})
	var total float64
	for _, inv := range rows {
		if isConfirmedInvoice(inv.Status) {
			total += inv.OutstandingUGX
		}
	}
	return round2(total)
}

// invoiceOutstanding is amount − collections − credited returns, floored at 0.
func (r *Repository) invoiceOutstanding(ctx context.Context, inv models.SecondaryInvoice) float64 {
	if !isConfirmedInvoice(inv.Status) {
		return 0
	}
	s := r.secondary()
	paid := 0.0
	cols, _ := s.collects.list(ctx, DocFilter{RefID: inv.ID, Limit: 5000})
	for _, c := range cols {
		if lower(c.Status) != "void" {
			paid += c.AmountUGX
		}
	}
	rets, _ := s.returns.list(ctx, DocFilter{RefID: inv.ID, Limit: 5000})
	for _, ret := range rets {
		if lower(ret.Status) == "credited" {
			paid += ret.CreditAmountUGX
		}
	}
	return round2(math.Max(0, inv.AmountUGX-paid))
}

// refreshInvoice recomputes an invoice's outstanding, flips it to paid when
// settled, and pushes the outlet's live balance to the outlet row.
func (r *Repository) refreshInvoice(ctx context.Context, id string) (models.SecondaryInvoice, error) {
	s := r.secondary()
	inv, err := s.invoices.get(ctx, id)
	if err != nil {
		return inv, err
	}
	inv.OutstandingUGX = r.invoiceOutstanding(ctx, inv)
	if inv.OutstandingUGX <= 0.009 && lower(inv.Status) == "invoiced" {
		inv.Status = "paid"
	} else if inv.OutstandingUGX > 0.009 && lower(inv.Status) == "paid" {
		inv.Status = "invoiced"
	}
	inv.UpdatedAt = now()
	if err := s.invoices.update(ctx, inv); err != nil {
		return inv, err
	}
	r.refreshOutletBalance(ctx, inv.OutletID)
	if inv.SchemeID != "" {
		r.refreshScheme(ctx, inv.SchemeID)
	}
	return inv, nil
}

func (r *Repository) refreshOutletBalance(ctx context.Context, outletID string) {
	if outletID == "" {
		return
	}
	total := r.OutletOutstanding(outletID)
	if r.pool != nil {
		_, _ = r.pool.Exec(ctx, `UPDATE dms_outlets SET outstanding_ugx = $2 WHERE id = $1`, outletID, total)
		return
	}
	r.mem.mu.Lock()
	defer r.mem.mu.Unlock()
	for i := range r.mem.outlets {
		if r.mem.outlets[i].ID == outletID {
			r.mem.outlets[i].OutstandingUGX = total
		}
	}
}

func (r *Repository) findScheme(ctx context.Context, ref string) (models.Scheme, bool) {
	if ref == "" {
		return models.Scheme{}, false
	}
	s := r.secondary()
	if sch, err := s.schemes.get(ctx, ref); err == nil {
		return sch, true
	}
	rows, _ := s.schemes.list(ctx, DocFilter{Limit: 5000})
	for _, sch := range rows {
		if lower(sch.Name) == lower(ref) {
			return sch, true
		}
	}
	return models.Scheme{}, false
}

func schemeEligible(sch models.Scheme, o models.Outlet) bool {
	if sch.Segment != "" && lower(sch.Segment) != lower(o.Segment) {
		return false
	}
	if sch.VolumeTier != "" && lower(sch.VolumeTier) != lower(o.VolumeTier) {
		return false
	}
	return true
}

// SecondaryInvoiceInput is the create/update payload plus who is acting, so
// an override can be attributed even when the form left overrideBy blank.
type SecondaryInvoiceInput struct {
	models.SecondaryInvoice
	Actor string `json:"-"`
	// CanOverride is whether the caller holds the supervisor grant.
	CanOverride bool `json:"-"`
}

func (r *Repository) validateInvoice(ctx context.Context, in *SecondaryInvoiceInput, existing *models.SecondaryInvoice) error {
	in.SaleKind = "secondary"
	in.Status = lower(in.Status)
	if in.Status == "" {
		in.Status = "invoiced"
	}
	if in.OutletID == "" {
		return invalid("a secondary invoice needs an outlet")
	}
	outlet, err := r.GetOutlet(in.OutletID)
	if err != nil {
		return invalid("outlet %q not found", in.OutletID)
	}
	if in.Quantity > 0 && in.UnitPriceUGX > 0 {
		in.AmountUGX = round2(in.Quantity * in.UnitPriceUGX)
	}
	if in.AmountUGX <= 0 {
		return invalid("a secondary invoice needs a quantity and price")
	}
	// History is never edited: a confirmed amount is corrected by a linked
	// return or collection, not by typing a new figure.
	if existing != nil && isConfirmedInvoice(existing.Status) && round2(existing.AmountUGX) != in.AmountUGX {
		return conflictf("invoice %s is confirmed; its amount is not edited — raise a return or a reversing collection", existing.ID)
	}
	// Sell only what is on that van.
	if in.VanLoadID != "" && in.Quantity > 0 {
		onVan, err := r.VanOnHand(in.VanLoadID)
		if err != nil {
			return invalid("van load %q not found", in.VanLoadID)
		}
		if existing != nil && existing.VanLoadID == in.VanLoadID && isConfirmedInvoice(existing.Status) {
			onVan += existing.Quantity
		}
		if in.Quantity > onVan+0.009 {
			return conflictf("only %.2f on van %s; cannot invoice %.2f", onVan, in.VanLoadID, in.Quantity)
		}
	}
	// Credit rules.
	terms := in.PaymentTerms
	if terms == "" {
		terms = outlet.PaymentTerms
	}
	if terms == "" {
		terms = "Cash on delivery"
	}
	in.PaymentTerms = terms
	if !isCashTerms(terms) && isConfirmedInvoice(in.Status) {
		if lower(outlet.KYCStatus) != "approved" {
			return conflictf("outlet %s is cash-only until KYC is approved", outlet.ID)
		}
		current := r.OutletOutstanding(outlet.ID)
		if existing != nil && isConfirmedInvoice(existing.Status) {
			current -= existing.OutstandingUGX
		}
		over := round2(current + in.AmountUGX - outlet.CreditLimitUGX)
		if over > 0.009 {
			if !in.CreditOverride {
				return conflictf("credit limit %.2f would be exceeded by %.2f; a supervisor override with a reason is required", outlet.CreditLimitUGX, over)
			}
			if strings.TrimSpace(in.OverrideReason) == "" {
				return conflictf("a credit-limit override requires a reason")
			}
			if !in.CanOverride {
				return conflictf("a credit-limit override needs the supervisor grant")
			}
			if in.OverrideBy == "" {
				in.OverrideBy = in.Actor
			}
		}
	}
	// Scheme eligibility and budget.
	if in.SchemeID != "" {
		sch, ok := r.findScheme(ctx, in.SchemeID)
		if !ok {
			return invalid("scheme %q not found", in.SchemeID)
		}
		in.SchemeID = sch.ID
		if !schemeEligible(sch, outlet) {
			return conflictf("outlet %s is not eligible for scheme %s", outlet.ID, sch.Name)
		}
		if lower(sch.Status) != "active" {
			return conflictf("scheme %s is %s", sch.Name, sch.Status)
		}
	}
	return nil
}

func (r *Repository) CreateSecondaryInvoice(in SecondaryInvoiceInput) (models.SecondaryInvoice, error) {
	ctx := r.bg()
	s := r.secondary()
	if err := r.validateInvoice(ctx, &in, nil); err != nil {
		return models.SecondaryInvoice{}, err
	}
	inv := in.SecondaryInvoice
	inv.ID = s.invoices.nextID(ctx, r, "SINV")
	inv.CreatedAt, inv.UpdatedAt = now(), now()
	inv.OutstandingUGX = 0
	if isConfirmedInvoice(inv.Status) {
		inv.OutstandingUGX = inv.AmountUGX
	}
	if err := s.invoices.insert(ctx, inv); err != nil {
		return inv, err
	}
	return r.refreshInvoice(ctx, inv.ID)
}

func (r *Repository) UpdateSecondaryInvoice(id string, in SecondaryInvoiceInput) (models.SecondaryInvoice, error) {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.invoices.get(ctx, id)
	if err != nil {
		return cur, err
	}
	if err := r.validateInvoice(ctx, &in, &cur); err != nil {
		return cur, err
	}
	inv := in.SecondaryInvoice
	inv.ID, inv.CreatedAt, inv.UpdatedAt = cur.ID, cur.CreatedAt, now()
	if lower(inv.Status) == "void" && r.invoiceHasCollections(ctx, id) {
		return cur, conflictf("invoice %s has collections against it; void those first", id)
	}
	if err := s.invoices.update(ctx, inv); err != nil {
		return cur, err
	}
	out, err := r.refreshInvoice(ctx, id)
	if cur.OutletID != inv.OutletID {
		r.refreshOutletBalance(ctx, cur.OutletID)
	}
	return out, err
}

func (r *Repository) invoiceHasCollections(ctx context.Context, id string) bool {
	cols, _ := r.secondary().collects.list(ctx, DocFilter{RefID: id, Limit: 1})
	for _, c := range cols {
		if lower(c.Status) != "void" {
			return true
		}
	}
	return false
}

func (r *Repository) DeleteSecondaryInvoice(id string) error {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.invoices.get(ctx, id)
	if err != nil {
		return err
	}
	if isConfirmedInvoice(cur.Status) {
		return conflictf("invoice %s is confirmed; void it rather than deleting it", id)
	}
	if err := s.invoices.delete(ctx, id); err != nil {
		return err
	}
	r.refreshOutletBalance(ctx, cur.OutletID)
	return nil
}

/* ─────────────────────────── collections ───────────────────────────── */

func (r *Repository) ListCollections(f DocFilter) ([]models.Collection, int) {
	return r.secondary().collects.list(r.bg(), f)
}

func (r *Repository) validateCollection(ctx context.Context, in *models.Collection, existing *models.Collection) (models.SecondaryInvoice, error) {
	in.Status = lower(in.Status)
	if in.Status == "" {
		in.Status = "collected"
	}
	in.Method = lower(strings.ReplaceAll(in.Method, " ", "_"))
	if in.Method == "" {
		in.Method = "cash"
	}
	if in.InvoiceID == "" {
		return models.SecondaryInvoice{}, invalid("a collection is against a specific invoice")
	}
	inv, err := r.secondary().invoices.get(ctx, in.InvoiceID)
	if err != nil {
		return inv, invalid("invoice %q not found", in.InvoiceID)
	}
	if !isConfirmedInvoice(inv.Status) {
		return inv, conflictf("invoice %s is %s; only a confirmed invoice takes a payment", inv.ID, inv.Status)
	}
	in.OutletID = inv.OutletID
	if in.AmountUGX <= 0 {
		return inv, invalid("a collection needs an amount")
	}
	if in.Status != "void" {
		remaining := inv.OutstandingUGX
		if existing != nil && lower(existing.Status) != "void" && existing.InvoiceID == inv.ID {
			remaining += existing.AmountUGX
		}
		if in.AmountUGX > remaining+0.009 {
			return inv, conflictf("invoice %s has %.2f outstanding; cannot collect %.2f", inv.ID, remaining, in.AmountUGX)
		}
		in.RemainingOnInvoiceUGX = round2(remaining - in.AmountUGX)
	}
	return inv, nil
}

func (r *Repository) CreateCollection(in models.Collection) (models.Collection, error) {
	ctx := r.bg()
	s := r.secondary()
	if _, err := r.validateCollection(ctx, &in, nil); err != nil {
		return models.Collection{}, err
	}
	in.ID = s.collects.nextID(ctx, r, "COL")
	in.CreatedAt, in.UpdatedAt = now(), now()
	if err := s.collects.insert(ctx, in); err != nil {
		return in, err
	}
	_, err := r.refreshInvoice(ctx, in.InvoiceID)
	return in, err
}

func (r *Repository) UpdateCollection(id string, in models.Collection) (models.Collection, error) {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.collects.get(ctx, id)
	if err != nil {
		return cur, err
	}
	// A recorded payment's amount is history — void and re-collect.
	if lower(cur.Status) != "void" && in.AmountUGX > 0 && round2(in.AmountUGX) != round2(cur.AmountUGX) {
		return cur, conflictf("collection %s is recorded; void it and collect again rather than editing the amount", id)
	}
	if in.AmountUGX <= 0 {
		in.AmountUGX = cur.AmountUGX
	}
	if in.InvoiceID == "" {
		in.InvoiceID = cur.InvoiceID
	}
	if _, err := r.validateCollection(ctx, &in, &cur); err != nil {
		return cur, err
	}
	in.ID, in.CreatedAt, in.UpdatedAt = cur.ID, cur.CreatedAt, now()
	if err := s.collects.update(ctx, in); err != nil {
		return cur, err
	}
	if _, err := r.refreshInvoice(ctx, in.InvoiceID); err != nil {
		return in, err
	}
	if cur.InvoiceID != in.InvoiceID {
		_, _ = r.refreshInvoice(ctx, cur.InvoiceID)
	}
	return in, nil
}

func (r *Repository) DeleteCollection(id string) error {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.collects.get(ctx, id)
	if err != nil {
		return err
	}
	if lower(cur.Status) != "void" {
		return conflictf("collection %s is recorded; void it rather than deleting it", id)
	}
	if err := s.collects.delete(ctx, id); err != nil {
		return err
	}
	_, _ = r.refreshInvoice(ctx, cur.InvoiceID)
	return nil
}

/* ───────────────────────── outlet returns ──────────────────────────── */

var returnReasons = map[string]string{
	"unsold-slow-mover":  "stock_receipt",
	"expired":            "write_off",
	"damaged-in-transit": "write_off",
	"quality-complaint":  "write_off",
	"other":              "stock_receipt",
}

// ReturnReasons is the vocabulary, for the catalogue and the client.
func ReturnReasons() []string {
	return []string{"unsold-slow-mover", "expired", "damaged-in-transit", "quality-complaint", "other"}
}

func (r *Repository) ListOutletReturns(f DocFilter) ([]models.OutletReturn, int) {
	return r.secondary().returns.list(r.bg(), f)
}

func (r *Repository) validateReturn(ctx context.Context, in *models.OutletReturn) error {
	in.Status = lower(in.Status)
	if in.Status == "" {
		in.Status = "received"
	}
	in.Reason = lower(strings.ReplaceAll(in.Reason, " ", "-"))
	disposition, ok := returnReasons[in.Reason]
	if !ok {
		return invalid("reason must be one of %s", strings.Join(ReturnReasons(), ", "))
	}
	in.InventoryDisposition = disposition
	if in.Quantity <= 0 {
		return invalid("a return needs a quantity")
	}
	if in.InvoiceID != "" {
		inv, err := r.secondary().invoices.get(ctx, in.InvoiceID)
		if err != nil {
			return invalid("invoice %q not found", in.InvoiceID)
		}
		if in.OutletID == "" {
			in.OutletID = inv.OutletID
		}
		if in.VanLoadID == "" {
			in.VanLoadID = inv.VanLoadID
		}
		if in.CreditAmountUGX <= 0 && inv.Quantity > 0 {
			in.CreditAmountUGX = round2(inv.AmountUGX / inv.Quantity * in.Quantity)
		}
	}
	if in.OutletID == "" {
		return invalid("a return needs an outlet")
	}
	if _, err := r.GetOutlet(in.OutletID); err != nil {
		return invalid("outlet %q not found", in.OutletID)
	}
	return nil
}

func (r *Repository) CreateOutletReturn(in models.OutletReturn) (models.OutletReturn, error) {
	ctx := r.bg()
	s := r.secondary()
	if err := r.validateReturn(ctx, &in); err != nil {
		return models.OutletReturn{}, err
	}
	in.ID = s.returns.nextID(ctx, r, "RET")
	in.CreatedAt, in.UpdatedAt = now(), now()
	if err := s.returns.insert(ctx, in); err != nil {
		return in, err
	}
	if in.InvoiceID != "" {
		_, _ = r.refreshInvoice(ctx, in.InvoiceID)
	}
	return in, nil
}

func (r *Repository) UpdateOutletReturn(id string, in models.OutletReturn) (models.OutletReturn, error) {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.returns.get(ctx, id)
	if err != nil {
		return cur, err
	}
	if lower(cur.Status) == "credited" && lower(in.Status) != "cancelled" &&
		(round2(in.Quantity) != round2(cur.Quantity) || (in.CreditAmountUGX > 0 && round2(in.CreditAmountUGX) != round2(cur.CreditAmountUGX))) {
		return cur, conflictf("return %s is credited; cancel it and raise a new one rather than editing it", id)
	}
	if err := r.validateReturn(ctx, &in); err != nil {
		return cur, err
	}
	in.ID, in.CreatedAt, in.UpdatedAt = cur.ID, cur.CreatedAt, now()
	if err := s.returns.update(ctx, in); err != nil {
		return cur, err
	}
	for _, ref := range []string{in.InvoiceID, cur.InvoiceID} {
		if ref != "" {
			_, _ = r.refreshInvoice(ctx, ref)
		}
	}
	return in, nil
}

func (r *Repository) DeleteOutletReturn(id string) error {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.returns.get(ctx, id)
	if err != nil {
		return err
	}
	if lower(cur.Status) == "credited" {
		return conflictf("return %s is credited; cancel it rather than deleting it", id)
	}
	if err := s.returns.delete(ctx, id); err != nil {
		return err
	}
	if cur.InvoiceID != "" {
		_, _ = r.refreshInvoice(ctx, cur.InvoiceID)
	}
	return nil
}

/* ─────────────────────── van reconciliations ───────────────────────── */

func (r *Repository) ListVanRecons(f DocFilter) ([]models.VanRecon, int) {
	return r.secondary().recons.list(r.bg(), f)
}

// closeVanRecon computes the stock and cash close against the PRD identity
//
//	loaded = sold + returned + remaining
//
// where sold is net of outlet returns (goods back on the van) and returned is
// what the rep handed back to the warehouse. When the recon names a van load,
// loaded and sold come from the service's own records — the form's figures
// are not trusted over the ledger; returned and remaining are the count at
// the close. Confirming with an unexplained exception is refused.
func (r *Repository) closeVanRecon(ctx context.Context, in *models.VanRecon) error {
	in.Status = lower(in.Status)
	if in.VanLoadID != "" {
		load, err := r.secondary().vanLoads.get(ctx, in.VanLoadID)
		if err != nil {
			return invalid("van load %q not found", in.VanLoadID)
		}
		in.LoadedQty = load.Quantity
		in.SoldQty = round2(r.vanSold(ctx, in.VanLoadID) - r.vanReturned(ctx, in.VanLoadID))
		if in.VanID == "" {
			in.VanID = load.VehicleID
		}
	}
	in.StockVariance = round2(in.LoadedQty - in.SoldQty - in.ReturnedQty - in.RemainingQty)
	in.CashVariance = round2(in.CollectedCashUGX - in.DepositedCashUGX)
	stockOK := math.Abs(in.StockVariance) < 0.005
	cashOK := math.Abs(in.CashVariance) < 0.005
	in.StockException = map[bool]string{true: "balanced", false: "exception"}[stockOK]
	in.CashException = map[bool]string{true: "balanced", false: "exception"}[cashOK]

	wantConfirm := in.Status == "confirmed"
	if !stockOK && strings.TrimSpace(in.StockExplanation) == "" && wantConfirm {
		return conflictf("stock does not balance (variance %.2f) and no explanation was given", in.StockVariance)
	}
	if !cashOK && strings.TrimSpace(in.CashExplanation) == "" && wantConfirm {
		return conflictf("cash does not balance (variance %.2f) and no explanation was given", in.CashVariance)
	}
	switch {
	case wantConfirm:
		in.Status = "confirmed"
	case in.Status == "open" || in.Status == "":
		if stockOK && cashOK {
			in.Status = "balanced"
		} else {
			in.Status = "exception"
		}
	default:
		if stockOK && cashOK {
			in.Status = "balanced"
		} else {
			in.Status = "exception"
		}
	}
	return nil
}

func (r *Repository) CreateVanRecon(in models.VanRecon) (models.VanRecon, error) {
	ctx := r.bg()
	s := r.secondary()
	if in.VanLoadID == "" {
		return models.VanRecon{}, invalid("a van close needs a van load")
	}
	if err := r.closeVanRecon(ctx, &in); err != nil {
		return models.VanRecon{}, err
	}
	in.ID = s.recons.nextID(ctx, r, "VRC")
	in.CreatedAt, in.UpdatedAt = now(), now()
	return in, s.recons.insert(ctx, in)
}

func (r *Repository) UpdateVanRecon(id string, in models.VanRecon) (models.VanRecon, error) {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.recons.get(ctx, id)
	if err != nil {
		return cur, err
	}
	if lower(cur.Status) == "confirmed" {
		return cur, conflictf("van close %s is confirmed", id)
	}
	if in.VanLoadID == "" {
		in.VanLoadID = cur.VanLoadID
	}
	if err := r.closeVanRecon(ctx, &in); err != nil {
		return cur, err
	}
	in.ID, in.CreatedAt, in.UpdatedAt = cur.ID, cur.CreatedAt, now()
	return in, s.recons.update(ctx, in)
}

func (r *Repository) DeleteVanRecon(id string) error {
	ctx := r.bg()
	cur, err := r.secondary().recons.get(ctx, id)
	if err != nil {
		return err
	}
	if lower(cur.Status) == "confirmed" {
		return conflictf("van close %s is confirmed", id)
	}
	return r.secondary().recons.delete(ctx, id)
}

/* ──────────────────────────── schemes ──────────────────────────────── */

func (r *Repository) ListSchemes(f DocFilter) ([]models.Scheme, int) {
	return r.secondary().schemes.list(r.bg(), f)
}

// refreshScheme recomputes redemption from confirmed invoices and flips the
// scheme to exhausted the moment the budget or volume is used up.
func (r *Repository) refreshScheme(ctx context.Context, id string) {
	s := r.secondary()
	sch, err := s.schemes.get(ctx, id)
	if err != nil {
		return
	}
	rows, _ := s.invoices.list(ctx, DocFilter{Limit: 5000})
	var value, volume float64
	for _, inv := range rows {
		if inv.SchemeID == id && isConfirmedInvoice(inv.Status) {
			value += inv.SchemeValueUGX
			volume += inv.Quantity
		}
	}
	sch.RedeemedValueUGX, sch.RedeemedVolume = round2(value), round2(volume)
	exhausted := (sch.AllocatedBudgetUGX > 0 && sch.RedeemedValueUGX >= sch.AllocatedBudgetUGX) ||
		(sch.AllocatedVolume > 0 && sch.RedeemedVolume >= sch.AllocatedVolume)
	switch {
	case exhausted && lower(sch.Status) == "active":
		sch.Status = "exhausted"
	case !exhausted && lower(sch.Status) == "exhausted":
		sch.Status = "active"
	}
	sch.UpdatedAt = now()
	_ = s.schemes.update(ctx, sch)
}

func (r *Repository) CreateScheme(in models.Scheme) (models.Scheme, error) {
	ctx := r.bg()
	s := r.secondary()
	if strings.TrimSpace(in.Name) == "" {
		return models.Scheme{}, invalid("a scheme needs a name")
	}
	in.Status = lower(in.Status)
	if in.Status == "" {
		in.Status = "active"
	}
	in.ID = s.schemes.nextID(ctx, r, "SCH")
	in.RedeemedValueUGX, in.RedeemedVolume = 0, 0
	in.CreatedAt, in.UpdatedAt = now(), now()
	return in, s.schemes.insert(ctx, in)
}

func (r *Repository) UpdateScheme(id string, in models.Scheme) (models.Scheme, error) {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.schemes.get(ctx, id)
	if err != nil {
		return cur, err
	}
	in.ID, in.CreatedAt, in.UpdatedAt = cur.ID, cur.CreatedAt, now()
	in.Status = lower(in.Status)
	if in.Status == "" {
		in.Status = cur.Status
	}
	if err := s.schemes.update(ctx, in); err != nil {
		return cur, err
	}
	r.refreshScheme(ctx, id)
	return s.schemes.get(ctx, id)
}

func (r *Repository) DeleteScheme(id string) error {
	ctx := r.bg()
	s := r.secondary()
	rows, _ := s.invoices.list(ctx, DocFilter{Limit: 5000})
	for _, inv := range rows {
		if inv.SchemeID == id && isConfirmedInvoice(inv.Status) {
			return conflictf("scheme %s has been redeemed on invoice %s; set it inactive instead", id, inv.ID)
		}
	}
	return s.schemes.delete(ctx, id)
}

/* ────────────────────────── rep targets ────────────────────────────── */

func (r *Repository) ListRepTargets(f DocFilter) ([]models.RepTarget, int) {
	return r.secondary().targets.list(r.bg(), f)
}

func pct(actual, target float64) float64 {
	if target <= 0 {
		return 0
	}
	return round2(actual / target * 100)
}

// computeTarget: achievement is the mean of the three attainment percentages
// and incentive is rate × achievement — never typed as the source of truth.
func computeTarget(in *models.RepTarget) error {
	if strings.TrimSpace(in.RepID) == "" && strings.TrimSpace(in.RepName) == "" {
		return invalid("a target needs a rep")
	}
	if strings.TrimSpace(in.Period) == "" {
		return invalid("a target needs a period")
	}
	in.Status = lower(in.Status)
	if in.Status == "" {
		in.Status = "active"
	}
	in.VolumePct = pct(in.VolumeActual, in.VolumeTarget)
	in.CollectionPct = pct(in.CollectionActual, in.CollectionTarget)
	in.OnboardingPct = pct(in.OnboardedActual, in.OnboardingTarget)
	var sum, n float64
	for _, p := range []struct{ target, pct float64 }{
		{in.VolumeTarget, in.VolumePct}, {in.CollectionTarget, in.CollectionPct}, {in.OnboardingTarget, in.OnboardingPct},
	} {
		if p.target > 0 {
			sum += p.pct
			n++
		}
	}
	if n > 0 {
		in.AchievementPct = round2(sum / n)
	} else {
		in.AchievementPct = 0
	}
	in.IncentiveUGX = round2(in.IncentiveRate * in.AchievementPct / 100)
	return nil
}

func (r *Repository) CreateRepTarget(in models.RepTarget) (models.RepTarget, error) {
	ctx := r.bg()
	s := r.secondary()
	if err := computeTarget(&in); err != nil {
		return models.RepTarget{}, err
	}
	in.ID = s.targets.nextID(ctx, r, "TGT")
	in.CreatedAt, in.UpdatedAt = now(), now()
	return in, s.targets.insert(ctx, in)
}

func (r *Repository) UpdateRepTarget(id string, in models.RepTarget) (models.RepTarget, error) {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.targets.get(ctx, id)
	if err != nil {
		return cur, err
	}
	if lower(cur.Status) == "closed" && lower(in.Status) != "active" {
		return cur, conflictf("target %s is closed", id)
	}
	if err := computeTarget(&in); err != nil {
		return cur, err
	}
	in.ID, in.CreatedAt, in.UpdatedAt = cur.ID, cur.CreatedAt, now()
	return in, s.targets.update(ctx, in)
}

func (r *Repository) DeleteRepTarget(id string) error {
	return r.secondary().targets.delete(r.bg(), id)
}

/* ───────────────────────── damage claims ───────────────────────────── */

func (r *Repository) ListDamageClaims(f DocFilter) ([]models.DamageClaim, int) {
	return r.secondary().claims.list(r.bg(), f)
}

func (r *Repository) validateClaim(ctx context.Context, in *models.DamageClaim) error {
	in.Status = lower(strings.ReplaceAll(in.Status, " ", "_"))
	if in.Status == "" {
		in.Status = "open"
	}
	if in.OutletReturnID != "" {
		ret, err := r.secondary().returns.get(ctx, in.OutletReturnID)
		if err != nil {
			return invalid("outlet return %q not found", in.OutletReturnID)
		}
		if in.ItemID == "" {
			in.ItemID = ret.ItemID
		}
		if in.Quantity <= 0 {
			in.Quantity = ret.Quantity
		}
	}
	if in.Quantity <= 0 && in.OutletReturnID == "" {
		return invalid("a damage claim needs a quantity or an outlet return")
	}
	return nil
}

func (r *Repository) CreateDamageClaim(in models.DamageClaim) (models.DamageClaim, error) {
	ctx := r.bg()
	s := r.secondary()
	if err := r.validateClaim(ctx, &in); err != nil {
		return models.DamageClaim{}, err
	}
	in.ID = s.claims.nextID(ctx, r, "DCL")
	in.CreatedAt, in.UpdatedAt = now(), now()
	return in, s.claims.insert(ctx, in)
}

func (r *Repository) UpdateDamageClaim(id string, in models.DamageClaim) (models.DamageClaim, error) {
	ctx := r.bg()
	s := r.secondary()
	cur, err := s.claims.get(ctx, id)
	if err != nil {
		return cur, err
	}
	if err := r.validateClaim(ctx, &in); err != nil {
		return cur, err
	}
	in.ID, in.CreatedAt, in.UpdatedAt = cur.ID, cur.CreatedAt, now()
	return in, s.claims.update(ctx, in)
}

func (r *Repository) DeleteDamageClaim(id string) error {
	return r.secondary().claims.delete(r.bg(), id)
}
