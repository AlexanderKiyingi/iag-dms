package models

import "time"

// Secondary-sales desk — distributor → retailer. These are distinct from the
// primary objects already here (`Invoice` is IAG → distributor, `Claim` is an
// outlet's financial claim, `Dispatch` links a truck to order ids).
//
// Every record carries `Status`, `OutletID` and `RefID` as real columns for
// filtering; everything else lives in the JSON document.

// VanLoad is a primary movement: warehouse → van. It is never revenue.
type VanLoad struct {
	ID           string         `json:"id"`
	Date         string         `json:"date"`
	VehicleID    string         `json:"vehicleId"`
	Driver       string         `json:"driver"`
	BeatID       string         `json:"beatId"`
	ItemID       string         `json:"itemId"`
	Quantity     float64        `json:"quantity"`
	Batch        string         `json:"batch"`
	ExpiryDate   string         `json:"expiryDate"`
	Batches      []VanLoadBatch `json:"batches"`
	SaleKind     string         `json:"saleKind"`     // always "primary"
	PostingClass string         `json:"postingClass"` // always "internal-transfer"
	Status       string         `json:"status"`       // loaded | in_transit | returned
	Notes        string         `json:"notes"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type VanLoadBatch struct {
	Batch      string  `json:"batch"`
	ExpiryDate string  `json:"expiryDate"`
	Quantity   float64 `json:"quantity"`
}

func (v VanLoad) DocID() string                     { return v.ID }
func (v VanLoad) DocKeys() (string, string, string) { return v.Status, "", v.VehicleID }

// SecondaryInvoice is the revenue / AR event: van → retailer.
type SecondaryInvoice struct {
	ID             string    `json:"id"`
	Date           string    `json:"date"`
	OutletID       string    `json:"outletId"`
	BeatID         string    `json:"beatId"`
	VanID          string    `json:"vanId"`
	VanLoadID      string    `json:"vanLoadId"`
	ItemID         string    `json:"itemId"`
	Batch          string    `json:"batch"`
	Quantity       float64   `json:"quantity"`
	UnitPriceUGX   float64   `json:"unitPriceUgx"`
	AmountUGX      float64   `json:"amountUgx"`
	TaxUGX         float64   `json:"taxUgx"`
	PaymentTerms   string    `json:"paymentTerms"`
	DueDate        string    `json:"dueDate"`
	SchemeID       string    `json:"schemeId"`
	SchemeValueUGX float64   `json:"schemeValueUgx"`
	CreditOverride bool      `json:"creditOverride"`
	OverrideReason string    `json:"overrideReason"`
	OverrideBy     string    `json:"overrideBy"`
	InvoiceChannel string    `json:"invoiceChannel"`
	SaleKind       string    `json:"saleKind"`       // always "secondary"
	OutstandingUGX float64   `json:"outstandingUgx"` // derived: amount − collections − credited returns
	Status         string    `json:"status"`         // draft | invoiced | paid | void
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (s SecondaryInvoice) DocID() string { return s.ID }
func (s SecondaryInvoice) DocKeys() (string, string, string) {
	return s.Status, s.OutletID, s.VanLoadID
}

// Collection is a payment against one specific secondary invoice.
type Collection struct {
	ID                    string    `json:"id"`
	Date                  string    `json:"date"`
	OutletID              string    `json:"outletId"`
	InvoiceID             string    `json:"invoiceId"`
	AmountUGX             float64   `json:"amountUgx"`
	Method                string    `json:"method"` // cash | mobile_money | bank_transfer | cheque
	DepositRef            string    `json:"depositRef"`
	RemainingOnInvoiceUGX float64   `json:"remainingOnInvoiceUgx"` // derived
	Status                string    `json:"status"`                // collected | deposited | void
	Notes                 string    `json:"notes"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

func (c Collection) DocID() string                     { return c.ID }
func (c Collection) DocKeys() (string, string, string) { return c.Status, c.OutletID, c.InvoiceID }

// OutletReturn is reason-coded stock coming back from a retailer.
type OutletReturn struct {
	ID                   string    `json:"id"`
	Date                 string    `json:"date"`
	OutletID             string    `json:"outletId"`
	InvoiceID            string    `json:"invoiceId"`
	VanLoadID            string    `json:"vanLoadId"`
	ItemID               string    `json:"itemId"`
	Quantity             float64   `json:"quantity"`
	Reason               string    `json:"reason"`
	InventoryDisposition string    `json:"inventoryDisposition"` // derived: stock_receipt | write_off
	CreditAmountUGX      float64   `json:"creditAmountUgx"`
	Status               string    `json:"status"` // received | credited | cancelled
	Notes                string    `json:"notes"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

func (o OutletReturn) DocID() string                     { return o.ID }
func (o OutletReturn) DocKeys() (string, string, string) { return o.Status, o.OutletID, o.InvoiceID }

// VanRecon is the end-of-day close: stock and cash, each of which must
// balance or be explained.
type VanRecon struct {
	ID               string    `json:"id"`
	Date             string    `json:"date"`
	VanID            string    `json:"vanId"`
	VanLoadID        string    `json:"vanLoadId"`
	RepID            string    `json:"repId"`
	LoadedQty        float64   `json:"loadedQty"`
	SoldQty          float64   `json:"soldQty"`
	ReturnedQty      float64   `json:"returnedQty"`
	RemainingQty     float64   `json:"remainingQty"`
	StockVariance    float64   `json:"stockVariance"`  // derived
	StockException   string    `json:"stockException"` // derived: balanced | exception
	StockExplanation string    `json:"stockExplanation"`
	CollectedCashUGX float64   `json:"collectedCashUgx"`
	DepositedCashUGX float64   `json:"depositedCashUgx"`
	CashVariance     float64   `json:"cashVariance"`  // derived
	CashException    string    `json:"cashException"` // derived
	CashExplanation  string    `json:"cashExplanation"`
	Status           string    `json:"status"` // open | balanced | exception | confirmed
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (v VanRecon) DocID() string                     { return v.ID }
func (v VanRecon) DocKeys() (string, string, string) { return v.Status, "", v.VanLoadID }

// Scheme is a trade scheme with a budget that redemption draws down.
type Scheme struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Kind               string    `json:"kind"`
	AllocatedBudgetUGX float64   `json:"allocatedBudgetUgx"`
	AllocatedVolume    float64   `json:"allocatedVolume"`
	RedeemedValueUGX   float64   `json:"redeemedValueUgx"` // derived from invoices
	RedeemedVolume     float64   `json:"redeemedVolume"`   // derived from invoices
	Segment            string    `json:"segment"`
	VolumeTier         string    `json:"volumeTier"`
	StartDate          string    `json:"startDate"`
	EndDate            string    `json:"endDate"`
	Status             string    `json:"status"` // active | exhausted | inactive
	Notes              string    `json:"notes"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func (s Scheme) DocID() string                     { return s.ID }
func (s Scheme) DocKeys() (string, string, string) { return s.Status, "", "" }

// RepTarget: targets are typed, achievement and incentive are computed.
type RepTarget struct {
	ID               string    `json:"id"`
	RepID            string    `json:"repId"`
	RepName          string    `json:"repName"`
	Period           string    `json:"period"`
	VolumeTarget     float64   `json:"volumeTarget"`
	VolumeActual     float64   `json:"volumeActual"`
	CollectionTarget float64   `json:"collectionTarget"`
	CollectionActual float64   `json:"collectionActual"`
	OnboardingTarget float64   `json:"onboardingTarget"`
	OnboardedActual  float64   `json:"onboardedActual"`
	IncentiveRate    float64   `json:"incentiveRate"`
	VolumePct        float64   `json:"volumePct"`      // derived
	CollectionPct    float64   `json:"collectionPct"`  // derived
	OnboardingPct    float64   `json:"onboardingPct"`  // derived
	AchievementPct   float64   `json:"achievementPct"` // derived
	IncentiveUGX     float64   `json:"incentiveUgx"`   // derived
	Status           string    `json:"status"`         // active | closed
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (r RepTarget) DocID() string                     { return r.ID }
func (r RepTarget) DocKeys() (string, string, string) { return r.Status, "", r.RepID }

// DamageClaim: goods damaged on a van, raised against Fleet.
type DamageClaim struct {
	ID             string    `json:"id"`
	Date           string    `json:"date"`
	OutletReturnID string    `json:"outletReturnId"`
	VanID          string    `json:"vanId"`
	Driver         string    `json:"driver"`
	ItemID         string    `json:"itemId"`
	Quantity       float64   `json:"quantity"`
	Notes          string    `json:"notes"`
	Status         string    `json:"status"` // open | with_fleet | closed
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (d DamageClaim) DocID() string                     { return d.ID }
func (d DamageClaim) DocKeys() (string, string, string) { return d.Status, "", d.OutletReturnID }
