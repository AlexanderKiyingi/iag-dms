package store

import (
	"errors"
	"testing"

	"github.com/iag/dms/backend/internal/models"
)

func outletRepo() *Repository {
	return NewMemoryWith(Fixture{
		Distributors: []models.Distributor{{ID: "D-001", Name: "Kampala Premium"}},
		Beats:        []models.Beat{{ID: "BT-01", Name: "Nakawa", RepID: "FF-01"}},
	})
}

func TestCreateOutletKeepsCommercialProfileAndDefaultsKYC(t *testing.T) {
	r := outletRepo()
	o, err := r.CreateOutlet(models.OutletInput{
		Name: "Shop A", Channel: "Kiosk", DistributorID: "D-001", BeatID: "BT-01",
		Lat: 0.31, Lng: 32.58, Contact: "Amina", Phone: "0700", RadiusM: 120,
		CreditLimitUGX: 500000, PaymentTerms: "7 days", Segment: "Urban", VolumeTier: "Gold",
		LicenseExpiry: "2027-01-31", Notes: "corner shop",
		Attrs: map[string]any{"kycAttachments": []any{"att-1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if o.KYCStatus != "pending" {
		t.Fatalf("kyc defaulted to %q, want pending", o.KYCStatus)
	}
	if o.CreditLimitUGX != 500000 || o.RadiusM != 120 || o.Segment != "Urban" || o.LicenseExpiry != "2027-01-31" {
		t.Fatalf("commercial profile not stored: %+v", o)
	}
	if o.Attrs["kycAttachments"] == nil {
		t.Fatal("attrs dropped")
	}
}

func TestCreateOutletRejectsUnknownBeat(t *testing.T) {
	r := outletRepo()
	_, err := r.CreateOutlet(models.OutletInput{Name: "Shop", Channel: "Bar", DistributorID: "D-001", BeatID: "Nakawa"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for beat name as id, got %v", err)
	}
}

func TestPatchOutletMovesGPSAndClearsFields(t *testing.T) {
	r := outletRepo()
	o, _ := r.CreateOutlet(models.OutletInput{Name: "Shop", Channel: "Bar", DistributorID: "D-001", Lat: 1, Lng: 2, Notes: "x"})
	lat, lng, zero, empty, kyc := 0.35, 32.6, 0.0, "", "approved"
	got, err := r.PatchOutlet(o.ID, models.OutletPatch{Lat: &lat, Lng: &lng, CreditLimitUGX: &zero, Notes: &empty, KYCStatus: &kyc})
	if err != nil {
		t.Fatal(err)
	}
	if got.Lat != 0.35 || got.Lng != 32.6 {
		t.Fatalf("gps not patched: %+v", got)
	}
	if got.Notes != "" || got.CreditLimitUGX != 0 {
		t.Fatalf("explicit zero/empty not applied: %+v", got)
	}
	if got.KYCStatus != "approved" {
		t.Fatalf("kyc not patched: %+v", got)
	}
	if got.Name != "Shop" || got.Channel != "Bar" {
		t.Fatalf("untouched fields changed: %+v", got)
	}
}

func TestPatchOutletRejectsUnknownBeat(t *testing.T) {
	r := outletRepo()
	o, _ := r.CreateOutlet(models.OutletInput{Name: "Shop", Channel: "Bar", DistributorID: "D-001"})
	bogus := "Nakawa"
	if _, err := r.PatchOutlet(o.ID, models.OutletPatch{BeatID: &bogus}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	real := "BT-01"
	if got, err := r.PatchOutlet(o.ID, models.OutletPatch{BeatID: &real}); err != nil || got.BeatID != "BT-01" {
		t.Fatalf("real beat refused: %v %+v", err, got)
	}
}

func TestCreateDistributorThenFileAnOutletUnderIt(t *testing.T) {
	r := New(nil)
	d, err := r.CreateDistributor(models.DistributorInput{Name: "Kampala Premium", Region: "Kampala"})
	if err != nil || d.ID == "" || d.Status != "active" || d.Tier != 1 {
		t.Fatalf("distributor not created: %v %+v", err, d)
	}
	if _, err := r.CreateDistributor(models.DistributorInput{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("nameless distributor accepted: %v", err)
	}
	if _, err := r.CreateOutlet(models.OutletInput{Name: "Shop", Channel: "Kiosk", DistributorID: d.ID}); err != nil {
		t.Fatalf("outlet under new distributor refused: %v", err)
	}
}
