package store

import (
	"errors"
	"testing"

	"github.com/iag/dms/backend/internal/models"
)

// deskRepo: one distributor, one beat, one KYC-approved outlet with a 100k
// credit limit and one cash-only (KYC pending) outlet.
func deskRepo(t *testing.T) (*Repository, models.Outlet, models.Outlet) {
	t.Helper()
	r := NewMemoryWith(Fixture{
		Distributors: []models.Distributor{{ID: "D-001", Name: "Kampala Premium"}},
		Beats:        []models.Beat{{ID: "BT-01", Name: "Nakawa", RepID: "FF-01"}},
	})
	credit, err := r.CreateOutlet(models.OutletInput{
		Name: "Credit Shop", Channel: "Supermarket", DistributorID: "D-001", KYCStatus: "approved",
		CreditLimitUGX: 100_000, PaymentTerms: "7 days", Segment: "Urban", VolumeTier: "Gold",
	})
	if err != nil {
		t.Fatal(err)
	}
	cash, err := r.CreateOutlet(models.OutletInput{Name: "Cash Kiosk", Channel: "Kiosk", DistributorID: "D-001"})
	if err != nil {
		t.Fatal(err)
	}
	return r, credit, cash
}

func mustLoad(t *testing.T, r *Repository, qty float64) models.VanLoad {
	t.Helper()
	load, err := r.CreateVanLoad(models.VanLoad{Date: "2026-09-15", VehicleID: "UBK 123A", ItemID: "SKU-1", Batches: []models.VanLoadBatch{
		{Batch: "B-LATE", ExpiryDate: "2027-03-01", Quantity: qty / 2},
		{Batch: "B-SOON", ExpiryDate: "2026-11-01", Quantity: qty / 2},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return load
}

func invoice(outlet models.Outlet, load models.VanLoad, qty, unit float64, terms string) SecondaryInvoiceInput {
	return SecondaryInvoiceInput{SecondaryInvoice: models.SecondaryInvoice{
		Date: "2026-09-15", OutletID: outlet.ID, VanLoadID: load.ID, ItemID: load.ItemID,
		Quantity: qty, UnitPriceUGX: unit, PaymentTerms: terms, Status: "invoiced",
	}}
}

// AC #8 — van loading selects nearest-expiry batches first; AC #4 — a primary
// load is never revenue.
func TestVanLoadIsFEFOAndPrimary(t *testing.T) {
	r, _, _ := deskRepo(t)
	load := mustLoad(t, r, 100)
	if load.Batch != "B-SOON" || load.Batches[0].Batch != "B-SOON" {
		t.Fatalf("FEFO not applied: %+v", load)
	}
	if load.SaleKind != "primary" || load.PostingClass != "internal-transfer" || load.Quantity != 100 {
		t.Fatalf("primary load mis-stamped: %+v", load)
	}
}

// AC #1 and #7 — outstanding updates immediately on invoice and on a payment
// against that specific invoice.
func TestInvoiceAndCollectionMoveOutletBalanceLive(t *testing.T) {
	r, credit, _ := deskRepo(t)
	load := mustLoad(t, r, 100)
	inv, err := r.CreateSecondaryInvoice(invoice(credit, load, 10, 5_000, "7 days"))
	if err != nil {
		t.Fatal(err)
	}
	if inv.AmountUGX != 50_000 || inv.OutstandingUGX != 50_000 || inv.SaleKind != "secondary" {
		t.Fatalf("invoice not computed: %+v", inv)
	}
	if o, _ := r.GetOutlet(credit.ID); o.OutstandingUGX != 50_000 {
		t.Fatalf("outlet balance not live after invoice: %v", o.OutstandingUGX)
	}
	if _, err := r.CreateCollection(models.Collection{InvoiceID: inv.ID, AmountUGX: 60_000, Method: "Cash"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("over-collection accepted: %v", err)
	}
	col, err := r.CreateCollection(models.Collection{InvoiceID: inv.ID, AmountUGX: 20_000, Method: "Mobile money"})
	if err != nil {
		t.Fatal(err)
	}
	if col.RemainingOnInvoiceUGX != 30_000 || col.OutletID != credit.ID || col.Method != "mobile_money" {
		t.Fatalf("collection not matched to invoice: %+v", col)
	}
	if o, _ := r.GetOutlet(credit.ID); o.OutstandingUGX != 30_000 {
		t.Fatalf("outlet balance not live after payment: %v", o.OutstandingUGX)
	}
	if _, err := r.CreateCollection(models.Collection{InvoiceID: inv.ID, AmountUGX: 30_000}); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.GetSecondaryInvoice(inv.ID); got.Status != "paid" || got.OutstandingUGX != 0 {
		t.Fatalf("settled invoice not flipped to paid: %+v", got)
	}
}

// AC #3 — over the credit limit needs a logged supervisor override with a reason.
func TestCreditLimitIsALimit(t *testing.T) {
	r, credit, _ := deskRepo(t)
	load := mustLoad(t, r, 1000)
	if _, err := r.CreateSecondaryInvoice(invoice(credit, load, 10, 8_000, "7 days")); err != nil {
		t.Fatal(err)
	}
	over := invoice(credit, load, 10, 5_000, "7 days") // 80k + 50k > 100k
	if _, err := r.CreateSecondaryInvoice(over); !errors.Is(err, ErrConflict) {
		t.Fatalf("over-limit accepted: %v", err)
	}
	over.CreditOverride = true
	if _, err := r.CreateSecondaryInvoice(over); !errors.Is(err, ErrConflict) {
		t.Fatalf("override without reason accepted: %v", err)
	}
	over.OverrideReason = "Long-standing customer, MD approved"
	if _, err := r.CreateSecondaryInvoice(over); !errors.Is(err, ErrConflict) {
		t.Fatalf("override without supervisor grant accepted: %v", err)
	}
	over.CanOverride, over.Actor = true, "supervisor@iag"
	inv, err := r.CreateSecondaryInvoice(over)
	if err != nil {
		t.Fatal(err)
	}
	if inv.OverrideBy != "supervisor@iag" {
		t.Fatalf("override not attributed: %+v", inv)
	}
}

// AC #10 — cash-only until KYC approved.
func TestUnapprovedOutletIsCashOnly(t *testing.T) {
	r, _, cash := deskRepo(t)
	load := mustLoad(t, r, 100)
	if _, err := r.CreateSecondaryInvoice(invoice(cash, load, 1, 1_000, "30 days")); !errors.Is(err, ErrConflict) {
		t.Fatalf("credit invoice to KYC-pending outlet accepted: %v", err)
	}
	if _, err := r.CreateSecondaryInvoice(invoice(cash, load, 1, 1_000, "Cash on delivery")); err != nil {
		t.Fatalf("cash invoice refused: %v", err)
	}
}

// Principle #2 — sell only what is on that van; #6 — history is never edited.
func TestVanStockIsAClosedLoopAndAmountsAreImmutable(t *testing.T) {
	r, credit, _ := deskRepo(t)
	load := mustLoad(t, r, 10)
	inv, err := r.CreateSecondaryInvoice(invoice(credit, load, 8, 1_000, "cash"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateSecondaryInvoice(invoice(credit, load, 3, 1_000, "cash")); !errors.Is(err, ErrConflict) {
		t.Fatalf("sold more than on van: %v", err)
	}
	if on, _ := r.VanOnHand(load.ID); on != 2 {
		t.Fatalf("on-hand %v, want 2", on)
	}
	edit := invoice(credit, load, 8, 2_000, "cash")
	if _, err := r.UpdateSecondaryInvoice(inv.ID, edit); !errors.Is(err, ErrConflict) {
		t.Fatalf("confirmed amount edited: %v", err)
	}
	if err := r.DeleteSecondaryInvoice(inv.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("confirmed invoice deleted: %v", err)
	}
}

// AC #5 — a return carries a reason and a disposition and credits the invoice.
func TestReturnIsReasonCodedAndCredits(t *testing.T) {
	r, credit, _ := deskRepo(t)
	load := mustLoad(t, r, 10)
	inv, _ := r.CreateSecondaryInvoice(invoice(credit, load, 10, 1_000, "7 days"))
	if _, err := r.CreateOutletReturn(models.OutletReturn{InvoiceID: inv.ID, Quantity: 2, Reason: "felt like it"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown reason accepted: %v", err)
	}
	ret, err := r.CreateOutletReturn(models.OutletReturn{InvoiceID: inv.ID, Quantity: 2, Reason: "Expired", Status: "credited"})
	if err != nil {
		t.Fatal(err)
	}
	if ret.InventoryDisposition != "write_off" || ret.CreditAmountUGX != 2_000 || ret.OutletID != credit.ID {
		t.Fatalf("return not derived: %+v", ret)
	}
	if got, _ := r.GetSecondaryInvoice(inv.ID); got.OutstandingUGX != 8_000 {
		t.Fatalf("credited return did not reduce outstanding: %+v", got)
	}
	if on, _ := r.VanOnHand(load.ID); on != 2 {
		t.Fatalf("returned stock not back on van: %v", on)
	}
	// 10 loaded, 10 sold, 2 back → 2 on the van.
}

// AC #2 and #12 — a close balances or is an exception needing an explanation;
// cash reconciles separately from stock.
func TestVanCloseMustBalanceOrExplain(t *testing.T) {
	r, credit, _ := deskRepo(t)
	load := mustLoad(t, r, 10)
	_, _ = r.CreateSecondaryInvoice(invoice(credit, load, 6, 1_000, "cash"))
	_, _ = r.CreateOutletReturn(models.OutletReturn{VanLoadID: load.ID, OutletID: credit.ID, Quantity: 1, Reason: "unsold-slow-mover"})

	// 10 loaded, 6 sold, 1 back from an outlet → net sold 5; rep hands 2 back
	// to the warehouse and counts 3 on the van.
	rec, err := r.CreateVanRecon(models.VanRecon{VanLoadID: load.ID, ReturnedQty: 2, RemainingQty: 3, CollectedCashUGX: 6_000, DepositedCashUGX: 6_000})
	if err != nil {
		t.Fatal(err)
	}
	if rec.LoadedQty != 10 || rec.SoldQty != 5 || rec.ReturnedQty != 2 || rec.StockVariance != 0 || rec.Status != "balanced" {
		t.Fatalf("balanced close mis-computed: %+v", rec)
	}

	short := models.VanRecon{VanLoadID: load.ID, ReturnedQty: 2, RemainingQty: 3, CollectedCashUGX: 6_000, DepositedCashUGX: 5_000, Status: "confirmed"}
	if _, err := r.CreateVanRecon(short); !errors.Is(err, ErrConflict) {
		t.Fatalf("cash shortfall confirmed without explanation: %v", err)
	}
	short.CashExplanation = "UGX 1,000 float retained for change"
	rec2, err := r.CreateVanRecon(short)
	if err != nil {
		t.Fatal(err)
	}
	if rec2.CashException != "exception" || rec2.StockException != "balanced" || rec2.Status != "confirmed" {
		t.Fatalf("stock-balanced/cash-short close mis-flagged: %+v", rec2)
	}
	lost := models.VanRecon{VanLoadID: load.ID, ReturnedQty: 2, RemainingQty: 2}
	rec3, _ := r.CreateVanRecon(lost)
	if rec3.StockVariance != 1 || rec3.Status != "exception" {
		t.Fatalf("missing unit not flagged: %+v", rec3)
	}
}

// AC #11 — scheme redemption is tracked against budget; over-budget is visible now.
func TestSchemeExhaustsAgainstBudgetAndChecksEligibility(t *testing.T) {
	r, credit, cash := deskRepo(t)
	load := mustLoad(t, r, 100)
	sch, err := r.CreateScheme(models.Scheme{Name: "Gold Q4", Kind: "Volume discount", AllocatedBudgetUGX: 10_000, VolumeTier: "Gold"})
	if err != nil {
		t.Fatal(err)
	}
	ineligible := invoice(cash, load, 1, 1_000, "cash")
	ineligible.SchemeID = "Gold Q4"
	if _, err := r.CreateSecondaryInvoice(ineligible); !errors.Is(err, ErrConflict) {
		t.Fatalf("ineligible outlet redeemed scheme: %v", err)
	}
	eligible := invoice(credit, load, 1, 1_000, "cash")
	eligible.SchemeID, eligible.SchemeValueUGX = "Gold Q4", 6_000
	if _, err := r.CreateSecondaryInvoice(eligible); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateSecondaryInvoice(eligible); err != nil {
		t.Fatal(err)
	}
	got, _ := r.secondary().schemes.get(r.bg(), sch.ID)
	if got.RedeemedValueUGX != 12_000 || got.Status != "exhausted" {
		t.Fatalf("scheme not exhausted at 12k of 10k: %+v", got)
	}
	if _, err := r.CreateSecondaryInvoice(eligible); !errors.Is(err, ErrConflict) {
		t.Fatalf("exhausted scheme still redeemable: %v", err)
	}
}

// AC #13 — incentive is computed, never typed.
func TestRepIncentiveIsComputed(t *testing.T) {
	r, _, _ := deskRepo(t)
	tg, err := r.CreateRepTarget(models.RepTarget{RepID: "FF-01", Period: "2026-09", VolumeTarget: 100, VolumeActual: 50,
		CollectionTarget: 1_000, CollectionActual: 1_000, IncentiveRate: 200_000, IncentiveUGX: 999_999_999})
	if err != nil {
		t.Fatal(err)
	}
	if tg.VolumePct != 50 || tg.CollectionPct != 100 || tg.AchievementPct != 75 || tg.IncentiveUGX != 150_000 {
		t.Fatalf("incentive not computed from achievement: %+v", tg)
	}
}

func TestDamageClaimInheritsFromReturn(t *testing.T) {
	r, credit, _ := deskRepo(t)
	ret, _ := r.CreateOutletReturn(models.OutletReturn{OutletID: credit.ID, ItemID: "SKU-1", Quantity: 4, Reason: "Damaged in transit"})
	cl, err := r.CreateDamageClaim(models.DamageClaim{OutletReturnID: ret.ID, VanID: "UBK 123A", Driver: "Okello", Status: "With fleet"})
	if err != nil {
		t.Fatal(err)
	}
	if cl.ItemID != "SKU-1" || cl.Quantity != 4 || cl.Status != "with_fleet" {
		t.Fatalf("claim not derived from return: %+v", cl)
	}
}
