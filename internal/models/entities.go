package models

import "time"

// Page domains mirror the DMS frontend (dmsiag) sidebar sections.

type Overview struct {
	KPIs           []KPI            `json:"kpis"`
	Regions        []RegionPin      `json:"regions"`
	ChannelMix     []ChannelMix     `json:"channelMix"`
	Alerts         []Alert          `json:"alerts"`
	TopDistributors []DistributorSummary `json:"topDistributors"`
	StockRisks     []StockRisk      `json:"stockRisks"`
}

type KPI struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	Value  string  `json:"value"`
	Unit   string  `json:"unit,omitempty"`
	Trend  string  `json:"trend,omitempty"`
	Sub    string  `json:"sub,omitempty"`
}

type RegionPin struct {
	Name        string  `json:"name"`
	RevenueUGX  float64 `json:"revenueUgx"`
	Distributors int    `json:"distributors"`
}

type ChannelMix struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Outlets   int     `json:"outlets"`
	ValueUGX  float64 `json:"valueUgx"`
	MixShare  float64 `json:"mixSharePct"`
}

type Alert struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Age     string `json:"age,omitempty"`
}

type DistributorSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Rep    string `json:"rep,omitempty"`
	Value  string `json:"value,omitempty"`
	Status string `json:"status,omitempty"`
}

type StockRisk struct {
	SKU            string `json:"sku"`
	DistributorID  string `json:"distributorId"`
	CoverDays      float64 `json:"coverDays"`
}

type Distributor struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Tier         int       `json:"tier"`
	Region       string    `json:"region"`
	Manager      string    `json:"manager,omitempty"`
	Outlets      int       `json:"outlets"`
	SellInRate   float64   `json:"sellInRatePct"`
	RevenueUGX   float64   `json:"revenueUgx"`
	Status       string    `json:"status"`
	OnboardedAt  time.Time `json:"onboardedAt,omitempty"`
}

type Outlet struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Address        string  `json:"address"`
	Channel        string  `json:"channel"`
	DistributorID  string  `json:"distributorId"`
	BeatID         string  `json:"beatId"`
	QTDValueUGX    float64 `json:"qtdValueUgx"`
	Frequency      string  `json:"frequency"`
	Score          string  `json:"score"`
	Status         string  `json:"status"`
	Lat            float64 `json:"lat,omitempty"`
	Lng            float64 `json:"lng,omitempty"`
}

type OutletInput struct {
	Name          string  `json:"name"`
	Address       string  `json:"address"`
	Channel       string  `json:"channel"`
	DistributorID string  `json:"distributorId"`
	BeatID        string  `json:"beatId,omitempty"`
	Lat           float64 `json:"lat,omitempty"`
	Lng           float64 `json:"lng,omitempty"`
}

type OutletPatch struct {
	Name     string `json:"name,omitempty"`
	Address  string `json:"address,omitempty"`
	Channel  string `json:"channel,omitempty"`
	BeatID   string `json:"beatId,omitempty"`
	Status   string `json:"status,omitempty"`
	Score    string `json:"score,omitempty"`
	Frequency string `json:"frequency,omitempty"`
}

type Order struct {
	ID            string    `json:"id"`
	OutletID      string    `json:"outletId"`
	OutletName    string    `json:"outletName,omitempty"`
	DistributorID string    `json:"distributorId"`
	RepID         string    `json:"repId,omitempty"`
	Status        string    `json:"status"`
	AmountUGX     float64   `json:"amountUgx"`
	Currency      string    `json:"currency"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type OrderInput struct {
	OutletID      string  `json:"outletId"`
	DistributorID string  `json:"distributorId"`
	RepID         string  `json:"repId,omitempty"`
	AmountUGX     float64 `json:"amountUgx"`
}

type Beat struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	RepID       string   `json:"repId"`
	RepName     string   `json:"repName,omitempty"`
	StopCount   int      `json:"stopCount"`
	DistanceKm  float64  `json:"distanceKm"`
	Status      string   `json:"status"`
	Progress    string   `json:"progress,omitempty"`
	OutletIDs   []string `json:"outletIds,omitempty"`
}

type FieldRep struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	BeatID   string `json:"beatId"`
	Region   string `json:"region"`
	Level    string `json:"level"`
	Status   string `json:"status"`
}

type CheckIn struct {
	ID        string    `json:"id"`
	RepID     string    `json:"repId"`
	OutletID  string    `json:"outletId"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	ArrivedAt time.Time `json:"arrivedAt"`
	Status    string    `json:"status"`
}

type CheckInInput struct {
	RepID    string  `json:"repId"`
	OutletID string  `json:"outletId"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type VisitReport struct {
	ID        string    `json:"id"`
	RepID     string    `json:"repId"`
	OutletID  string    `json:"outletId"`
	Outcome   string    `json:"outcome"`
	Notes     string    `json:"notes"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	CreatedAt time.Time `json:"createdAt"`
}

type VisitReportInput struct {
	RepID    string  `json:"repId"`
	OutletID string  `json:"outletId"`
	Outcome  string  `json:"outcome"`
	Notes    string  `json:"notes"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type JourneyDay struct {
	Date    string         `json:"date"`
	RepID   string         `json:"repId"`
	Stops   []JourneyStop  `json:"stops"`
}

type JourneyStop struct {
	Seq       int    `json:"seq"`
	OutletID  string `json:"outletId"`
	Outlet    string `json:"outletName"`
	BeatID    string `json:"beatId"`
	Planned   string `json:"plannedTime"`
	Status    string `json:"status"`
}

type Promotion struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	ROI      float64 `json:"roi"`
	Status   string  `json:"status"`
	Outlets  int     `json:"outlets,omitempty"`
}

type PromotionInput struct {
	Name    string  `json:"name"`
	SKU     string  `json:"sku"`
	Outlets int     `json:"outlets,omitempty"`
	ROI     float64 `json:"roi,omitempty"`
}

// PromotionPatch carries partial updates; nil pointers / empty strings are left
// unchanged so callers can PATCH a single field.
type PromotionPatch struct {
	Name    string   `json:"name,omitempty"`
	SKU     string   `json:"sku,omitempty"`
	Status  string   `json:"status,omitempty"`
	ROI     *float64 `json:"roi,omitempty"`
	Outlets *int     `json:"outlets,omitempty"`
}

type Claim struct {
	ID        string    `json:"id"`
	OutletID  string    `json:"outletId"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	AmountUGX float64   `json:"amountUgx"`
	CreatedAt time.Time `json:"createdAt"`
}

type ClaimInput struct {
	OutletID  string  `json:"outletId"`
	Type      string  `json:"type"`
	AmountUGX float64 `json:"amountUgx"`
}

type ClaimPatch struct {
	Type      string   `json:"type,omitempty"`
	Status    string   `json:"status,omitempty"`
	AmountUGX *float64 `json:"amountUgx,omitempty"`
}

type Dispatch struct {
	ID         string    `json:"id"`
	TruckID    string    `json:"truckId"`
	Driver     string    `json:"driver,omitempty"`
	OrderIDs   []string  `json:"orderIds"`
	Status     string    `json:"status"`
	ETA        string    `json:"eta,omitempty"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type DispatchInput struct {
	TruckID  string   `json:"truckId"`
	Driver   string   `json:"driver,omitempty"`
	OrderIDs []string `json:"orderIds"`
	ETA      string   `json:"eta,omitempty"`
}

type DispatchPatch struct {
	TruckID string `json:"truckId,omitempty"`
	Driver  string `json:"driver,omitempty"`
	Status  string `json:"status,omitempty"`
	ETA     string `json:"eta,omitempty"`
}

type SKU struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	CoverDays  float64 `json:"coverDays,omitempty"`
	QtyOnHand  int     `json:"qtyOnHand,omitempty"`
	Warehouse  string  `json:"warehouse,omitempty"`
	SCAScore   float64 `json:"scaScore,omitempty"`
}

type StockPosition struct {
	DistributorID string  `json:"distributorId"`
	SKU           string  `json:"sku"`
	CoverDays     float64 `json:"coverDays"`
	Qty           int     `json:"qty"`
	Status        string  `json:"status"`
}

type Invoice struct {
	ID            string    `json:"id"`
	DistributorID string    `json:"distributorId"`
	Distributor   string    `json:"distributorName,omitempty"`
	AmountUGX     float64   `json:"amountUgx"`
	DueDate       time.Time `json:"dueDate"`
	Status        string    `json:"status"`
	OrderID       string    `json:"orderId,omitempty"`
	EFRISStatus   string    `json:"efrisStatus,omitempty"`
	URAReceipt    string    `json:"uraReceipt,omitempty"`
	DocumentURL   string    `json:"documentUrl,omitempty"`
}

type InvoiceInput struct {
	OrderID       string    `json:"orderId,omitempty"`
	DistributorID string    `json:"distributorId"`
	AmountUGX     float64   `json:"amountUgx"`
	DueDate       time.Time `json:"dueDate"`
}

type InvoicePatch struct {
	AmountUGX *float64   `json:"amountUgx,omitempty"`
	Status    string     `json:"status,omitempty"`
	DueDate   *time.Time `json:"dueDate,omitempty"`
}

type PricingTemplate struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Channel  string `json:"channel"`
	Version  string `json:"version"`
	Currency string `json:"currency"`
	Status   string `json:"status,omitempty"`
}

type PricingInput struct {
	Name     string `json:"name"`
	Channel  string `json:"channel"`
	Currency string `json:"currency"`
}

type PricingPatch struct {
	Name     string `json:"name,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Currency string `json:"currency,omitempty"`
	Status   string `json:"status,omitempty"`
}

type PricingVersion struct {
	ID         string    `json:"id"`
	TemplateID string    `json:"templateId"`
	Version    string    `json:"version"`
	CreatedBy  string    `json:"createdBy,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

// JourneyAssignment binds a beat to a rep on a date with an ordered stop seq.
type JourneyAssignment struct {
	ID     string `json:"id"`
	RepID  string `json:"repId"`
	Date   string `json:"date"`
	BeatID string `json:"beatId"`
	Seq    int    `json:"seq"`
	Status string `json:"status"`
}

type JourneyAssignmentInput struct {
	RepID  string `json:"repId"`
	Date   string `json:"date"`
	BeatID string `json:"beatId"`
	Seq    int    `json:"seq,omitempty"`
}

type JourneyAssignmentPatch struct {
	BeatID string `json:"beatId,omitempty"`
	Date   string `json:"date,omitempty"`
	Status string `json:"status,omitempty"`
	Seq    *int   `json:"seq,omitempty"`
}

type ReportSchedule struct {
	ID         string     `json:"id"`
	TemplateID string     `json:"templateId,omitempty"`
	Name       string     `json:"name"`
	Cron       string     `json:"cron,omitempty"`
	Channel    string     `json:"channel"`
	Recipient  string     `json:"recipient"`
	Active     bool       `json:"active"`
	LastRunAt  *time.Time `json:"lastRunAt,omitempty"`
	NextRunAt  *time.Time `json:"nextRunAt,omitempty"`
}

type ReportScheduleInput struct {
	TemplateID string `json:"templateId,omitempty"`
	Name       string `json:"name"`
	Cron       string `json:"cron,omitempty"`
	Channel    string `json:"channel,omitempty"`
	Recipient  string `json:"recipient"`
}

type ReportTemplate struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	DataSource string `json:"dataSource"`
	Schedule   string `json:"schedule,omitempty"`
}

type ReportRunInput struct {
	Name       string `json:"name"`
	TemplateID string `json:"templateId,omitempty"`
	EmailTo    string `json:"emailTo,omitempty"`
}

type ReportRun struct {
	JobID    string `json:"jobId"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	RowCount int    `json:"rowCount"`
	Message  string `json:"message,omitempty"`
}

type ExportInput struct {
	Page   string `json:"page"`
	Format string `json:"format,omitempty"`
}

type ExportPayload struct {
	Page        string           `json:"page"`
	Format      string           `json:"format"`
	GeneratedAt time.Time        `json:"generatedAt"`
	RowCount    int              `json:"rowCount"`
	Rows        []map[string]any `json:"rows"`
}

type KPIBoard struct {
	Period      string       `json:"period"`
	Leaderboard []RepScore   `json:"leaderboard"`
	Targets     []KPITarget  `json:"targets"`
}

type RepScore struct {
	RepID   string  `json:"repId"`
	RepName string  `json:"repName"`
	Points  float64 `json:"points"`
	Rank    int     `json:"rank"`
}

type KPITarget struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	Actual float64 `json:"actual"`
	Target float64 `json:"target"`
	Unit   string  `json:"unit"`
}

type AnalyticsSummary struct {
	Cohorts []CohortRow `json:"cohorts"`
	Funnel  []FunnelStep `json:"funnel"`
}

type CohortRow struct {
	Label   string  `json:"label"`
	Outlets int     `json:"outlets"`
	Revenue float64 `json:"revenueUgx"`
	Retention float64 `json:"retentionPct"`
}

type FunnelStep struct {
	Stage string  `json:"stage"`
	Count int     `json:"count"`
	Rate  float64 `json:"ratePct"`
}

type Forecast struct {
	SKU         string           `json:"sku"`
	MAPE        float64          `json:"mapePct"`
	HorizonDays int              `json:"horizonDays"`
	Points      []ForecastPoint  `json:"points"`
}

type ForecastPoint struct {
	Date       string  `json:"date"`
	Forecast   float64 `json:"forecast"`
	Actual     float64 `json:"actual,omitempty"`
	LowerBound float64 `json:"lowerBound,omitempty"`
	UpperBound float64 `json:"upperBound,omitempty"`
}

type FinanceSummary struct {
	ARBalanceUGX float64 `json:"arBalanceUgx"`
	DSODays      float64 `json:"dsoDays"`
	OverdueUGX   float64 `json:"overdueUgx"`
	CollectedUGX float64 `json:"collectedUgx"`
}

type ExecutionTask struct {
	ID       string   `json:"id"`
	OutletID string   `json:"outletId"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Detail   string   `json:"detail"`
	Photos   []string `json:"photos,omitempty"`
}

// Attachment is a stored binary object's metadata; the bytes live in the
// configured storage.Store under StorageKey.
type Attachment struct {
	ID          string    `json:"id"`
	OwnerType   string    `json:"ownerType,omitempty"`
	OwnerID     string    `json:"ownerId,omitempty"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	StorageKey  string    `json:"-"`
	URL         string    `json:"url"`
	UploadedBy  string    `json:"uploadedBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SearchResult struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Sub   string `json:"sub"`
	Page  string `json:"page"`
	ID    string `json:"id,omitempty"`
}

type ListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type PageInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
