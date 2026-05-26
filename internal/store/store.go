package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/iag/dms/backend/internal/models"
)

var ErrNotFound = errors.New("not found")

type ListOpts struct {
	Limit  int
	Offset int
	Q      string
	Status string
	Channel string
	DistributorID string
	RepID string
	BeatID string
}

type memoryState struct {
	mu sync.RWMutex

	distributors []models.Distributor
	outlets      []models.Outlet
	orders       []models.Order
	beats        []models.Beat
	reps         []models.FieldRep
	checkIns     []models.CheckIn
	visits       []models.VisitReport
	promos       []models.Promotion
	claims       []models.Claim
	dispatches   []models.Dispatch
	skus         []models.SKU
	stock        []models.StockPosition
	invoices     []models.Invoice
	pricing      []models.PricingTemplate
	reports      []models.ReportTemplate
	execution    []models.ExecutionTask
	alerts       []models.Alert
	auditEntries []models.AuditEntry
	apiAudit     []apiAuditRow
	signals      []map[string]any

	nextOutlet int
	nextOrder  int
}

func newMemoryState() *memoryState {
	m := &memoryState{nextOutlet: 2849, nextOrder: 19853}
	seedMemory(m)
	return m
}

func (m *memoryState) isEmpty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.distributors) == 0
}

func (r *Repository) Overview() models.Overview {
	if r.pool != nil {
		return r.pgOverview(context.Background())
	}
	return r.mem.overview()
}

func (m *memoryState) overview() models.Overview {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return models.Overview{
		KPIs: []models.KPI{
			{Key: "sell_out", Label: "Sell-Out (Wk)", Value: "412", Unit: "M UGX", Trend: "▲ 14.3% WoW", Sub: "vs UGX 360M LW"},
			{Key: "sell_in", Label: "Sell-In Rate", Value: "87.4", Unit: "%", Trend: "▲ 2.1 pts", Sub: "Target 85%"},
			{Key: "outlets", Label: "Active Outlets", Value: "2,162", Unit: "/2.8k", Trend: "▲ 76% coverage", Sub: "+184 net new"},
			{Key: "cycle", Label: "Order Cycle", Value: "26", Unit: "hrs", Trend: "▼ 4 hrs", Sub: "Order → delivery"},
			{Key: "fill", Label: "Fill Rate", Value: "94.2", Unit: "%", Trend: "▼ 1.4 pts", Sub: "OTIF 91%"},
		},
		Regions: []models.RegionPin{
			{Name: "Kampala", RevenueUGX: 142_000_000, Distributors: 8},
			{Name: "Mbarara", RevenueUGX: 68_000_000, Distributors: 5},
			{Name: "Mbale", RevenueUGX: 84_000_000, Distributors: 7},
			{Name: "Gulu", RevenueUGX: 41_000_000, Distributors: 4},
			{Name: "Fort Portal", RevenueUGX: 33_000_000, Distributors: 3},
			{Name: "Arua", RevenueUGX: 24_000_000, Distributors: 2},
			{Name: "Soroti", RevenueUGX: 19_000_000, Distributors: 2},
			{Name: "Jinja", RevenueUGX: 38_000_000, Distributors: 4},
		},
		ChannelMix: []models.ChannelMix{
			{Code: "CH-MT", Name: "Modern Trade", Outlets: 412, ValueUGX: 156_000_000, MixShare: 38},
			{Code: "CH-HC", Name: "HoReCa", Outlets: 684, ValueUGX: 111_000_000, MixShare: 27},
			{Code: "CH-GT", Name: "General Trade", Outlets: 1512, ValueUGX: 90_000_000, MixShare: 22},
		},
		Alerts: append([]models.Alert(nil), m.alerts...),
		TopDistributors: []models.DistributorSummary{
			{ID: "D-001", Name: "Kampala Premium Beverages", Rep: "J. Sebunya", Value: "UGX 142M", Status: "on_track"},
			{ID: "D-007", Name: "Mbale Coffee Hub Ltd", Rep: "F. Wamala", Value: "UGX 84M", Status: "on_track"},
		},
		StockRisks: []models.StockRisk{
			{SKU: "BG-AA-250", DistributorID: "D-001", CoverDays: 2.1},
			{SKU: "HK-IN-100", DistributorID: "D-007", CoverDays: 3.4},
		},
	}
}

func (r *Repository) Pages() []models.PageInfo {
	return pagesCatalog()
}

func pagesCatalog() []models.PageInfo {
	return []models.PageInfo{
		{ID: "overview", Title: "Distribution Tower"},
		{ID: "network", Title: "Distributor Network"},
		{ID: "outlets", Title: "Outlets & Universe"},
		{ID: "orders", Title: "Secondary Orders"},
		{ID: "routes", Title: "Routes & Beats"},
		{ID: "checkin", Title: "Field Check-In"},
		{ID: "journey", Title: "Journey Planner"},
		{ID: "field", Title: "Field Force"},
		{ID: "execution", Title: "Retail Execution"},
		{ID: "promo", Title: "Trade Promotions"},
		{ID: "claims", Title: "Claims & Returns"},
		{ID: "dispatch", Title: "Dispatch & Fleet"},
		{ID: "stock", Title: "Distributor Stock"},
		{ID: "stockwh", Title: "Stock & Warehouse"},
		{ID: "finance", Title: "Finance Hub"},
		{ID: "invoices", Title: "Invoice Studio"},
		{ID: "pricing", Title: "Pricing Templates"},
		{ID: "reports", Title: "Reports Studio"},
		{ID: "kpi", Title: "KPI & Incentives"},
		{ID: "analytics", Title: "Analytics Studio"},
		{ID: "forecast", Title: "AI Demand Forecast"},
	}
}

func paginate[T any](items []T, opts ListOpts) ([]T, int) {
	total := len(items)
	start := opts.Offset
	if start > total {
		start = total
	}
	end := start + opts.Limit
	if opts.Limit <= 0 || end > total {
		end = total
	}
	return items[start:end], total
}

func filterQ(q string, parts ...string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	for _, p := range parts {
		if strings.Contains(strings.ToLower(p), q) {
			return true
		}
	}
	return false
}

func (m *memoryState) nextID(prefix string, seq *int) string {
	*seq++
	return fmt.Sprintf("%s-%05d", prefix, *seq)
}

func now() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

func newUUID() string {
	return uuid.NewString()
}
