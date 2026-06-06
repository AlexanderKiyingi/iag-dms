package store

import (
	"strings"

	"github.com/iag/dms/backend/internal/models"
)

func defaultLimit(opts ListOpts) ListOpts {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	return opts
}

func (m *memoryState) listDistributors(opts ListOpts) ([]models.Distributor, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.Distributor
	for _, d := range m.distributors {
		if opts.Status != "" && d.Status != opts.Status {
			continue
		}
		if !filterQ(opts.Q, d.ID, d.Name, d.Region) {
			continue
		}
		filtered = append(filtered, d)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) getDistributor(id string) (models.Distributor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.distributors {
		if d.ID == id {
			return d, nil
		}
	}
	return models.Distributor{}, ErrNotFound
}

func (m *memoryState) listOutlets(opts ListOpts) ([]models.Outlet, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.Outlet
	for _, o := range m.outlets {
		if opts.Channel != "" && !strings.EqualFold(o.Channel, opts.Channel) {
			continue
		}
		if opts.DistributorID != "" && o.DistributorID != opts.DistributorID {
			continue
		}
		if opts.BeatID != "" && o.BeatID != opts.BeatID {
			continue
		}
		if opts.Status != "" && o.Status != opts.Status {
			continue
		}
		if !filterQ(opts.Q, o.ID, o.Name, o.Address, o.Channel) {
			continue
		}
		filtered = append(filtered, o)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) getOutlet(id string) (models.Outlet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, o := range m.outlets {
		if o.ID == id {
			return o, nil
		}
	}
	return models.Outlet{}, ErrNotFound
}

func (m *memoryState) createOutlet(in models.OutletInput) (models.Outlet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	found := false
	for _, d := range m.distributors {
		if d.ID == in.DistributorID {
			found = true
			break
		}
	}
	if !found {
		return models.Outlet{}, ErrInvalidInput
	}
	id := m.nextID("OUT", &m.nextOutlet)
	o := models.Outlet{
		ID: id, Name: in.Name, Address: in.Address, Channel: in.Channel,
		DistributorID: in.DistributorID, BeatID: in.BeatID,
		Lat: in.Lat, Lng: in.Lng,
		Status: "active", Score: "B", Frequency: "1x/wk",
	}
	m.outlets = append(m.outlets, o)
	m.alerts = append([]models.Alert{{
		ID: newUUID(), Kind: "outlet", Title: "Outlet activated · " + id,
		Detail: in.Name,
	}}, m.alerts...)
	return o, nil
}

func (m *memoryState) listOrders(opts ListOpts) ([]models.Order, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.Order
	for _, o := range m.orders {
		if opts.Status != "" && o.Status != opts.Status {
			continue
		}
		if opts.DistributorID != "" && o.DistributorID != opts.DistributorID {
			continue
		}
		if !filterQ(opts.Q, o.ID, o.OutletName, o.OutletID) {
			continue
		}
		filtered = append(filtered, o)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) getOrder(id string) (models.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, o := range m.orders {
		if o.ID == id {
			return o, nil
		}
	}
	return models.Order{}, ErrNotFound
}

func (m *memoryState) createOrder(in models.OrderInput) models.Order {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextID("SO", &m.nextOrder)
	outletName := ""
	for _, o := range m.outlets {
		if o.ID == in.OutletID {
			outletName = o.Name
			break
		}
	}
	ts := now()
	o := models.Order{
		ID: id, OutletID: in.OutletID, OutletName: outletName,
		DistributorID: in.DistributorID, RepID: in.RepID,
		Status: "draft", AmountUGX: in.AmountUGX, Currency: "UGX",
		CreatedAt: ts, UpdatedAt: ts,
	}
	m.orders = append([]models.Order{o}, m.orders...)
	return o
}

func (m *memoryState) updateOrderStatus(id, status string) (models.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, o := range m.orders {
		if o.ID == id {
			m.orders[i].Status = status
			m.orders[i].UpdatedAt = now()
			return m.orders[i], nil
		}
	}
	return models.Order{}, ErrNotFound
}

func (m *memoryState) ordersBoard() map[string][]models.Order {
	m.mu.RLock()
	defer m.mu.RUnlock()
	board := map[string][]models.Order{
		"draft": {}, "submitted": {}, "picking": {}, "delivery": {}, "delivered": {},
	}
	for _, o := range m.orders {
		key := o.Status
		if _, ok := board[key]; !ok {
			key = "submitted"
		}
		board[key] = append(board[key], o)
	}
	return board
}

func (m *memoryState) listBeats(opts ListOpts) ([]models.Beat, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.Beat
	for _, b := range m.beats {
		if opts.RepID != "" && b.RepID != opts.RepID {
			continue
		}
		if !filterQ(opts.Q, b.ID, b.Name) {
			continue
		}
		filtered = append(filtered, b)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) getBeat(id string) (models.Beat, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, b := range m.beats {
		if b.ID == id {
			return b, nil
		}
	}
	return models.Beat{}, ErrNotFound
}

func (m *memoryState) listReps(opts ListOpts) ([]models.FieldRep, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.FieldRep
	for _, rep := range m.reps {
		if opts.Status != "" && rep.Status != opts.Status {
			continue
		}
		if !filterQ(opts.Q, rep.ID, rep.Name) {
			continue
		}
		filtered = append(filtered, rep)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) listCheckIns(opts ListOpts) ([]models.CheckIn, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.CheckIn
	for _, c := range m.checkIns {
		if opts.RepID != "" && c.RepID != opts.RepID {
			continue
		}
		filtered = append(filtered, c)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) createCheckIn(in models.CheckInInput) models.CheckIn {
	m.mu.Lock()
	defer m.mu.Unlock()
	ci := models.CheckIn{
		ID: newUUID(), RepID: in.RepID, OutletID: in.OutletID,
		Lat: in.Lat, Lng: in.Lng, ArrivedAt: now(), Status: "active",
	}
	m.checkIns = append([]models.CheckIn{ci}, m.checkIns...)
	for i, rep := range m.reps {
		if rep.ID == in.RepID {
			m.reps[i].Status = "clocked_in"
		}
	}
	return ci
}

func (m *memoryState) createVisitReport(in models.VisitReportInput) models.VisitReport {
	m.mu.Lock()
	defer m.mu.Unlock()
	v := models.VisitReport{
		ID: newUUID(), RepID: in.RepID, OutletID: in.OutletID,
		Outcome: in.Outcome, Notes: in.Notes, Lat: in.Lat, Lng: in.Lng,
		CreatedAt: now(),
	}
	m.visits = append([]models.VisitReport{v}, m.visits...)
	return v
}

func (m *memoryState) journey(repID string) models.JourneyDay {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stops := []models.JourneyStop{}
	seq := 1
	for _, o := range m.outlets {
		if seq > 6 {
			break
		}
		stops = append(stops, models.JourneyStop{
			Seq: seq, OutletID: o.ID, Outlet: o.Name, BeatID: o.BeatID,
			Planned: "09:00", Status: "planned",
		})
		seq++
	}
	if repID == "FF-04" && len(stops) > 0 {
		stops[0].Status = "completed"
		if len(stops) > 1 {
			stops[1].Status = "active"
		}
	}
	return models.JourneyDay{Date: "2026-05-26", RepID: repID, Stops: stops}
}

func (m *memoryState) listPromotions(opts ListOpts) ([]models.Promotion, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	return paginate(m.promos, opts)
}

func (m *memoryState) listClaims(opts ListOpts) ([]models.Claim, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.Claim
	for _, c := range m.claims {
		if opts.Status != "" && c.Status != opts.Status {
			continue
		}
		filtered = append(filtered, c)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) listDispatches(opts ListOpts) ([]models.Dispatch, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	return paginate(m.dispatches, opts)
}

func (m *memoryState) listStock(opts ListOpts) ([]models.StockPosition, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.StockPosition
	for _, s := range m.stock {
		if opts.DistributorID != "" && s.DistributorID != opts.DistributorID {
			continue
		}
		filtered = append(filtered, s)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) listSKUs(opts ListOpts) ([]models.SKU, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.SKU
	for _, s := range m.skus {
		if !filterQ(opts.Q, s.Code, s.Name) {
			continue
		}
		filtered = append(filtered, s)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) listInvoices(opts ListOpts) ([]models.Invoice, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	var filtered []models.Invoice
	for _, inv := range m.invoices {
		if opts.Status != "" && inv.Status != opts.Status {
			continue
		}
		if opts.DistributorID != "" && inv.DistributorID != opts.DistributorID {
			continue
		}
		filtered = append(filtered, inv)
	}
	return paginate(filtered, opts)
}

func (m *memoryState) createInvoice(in models.InvoiceInput) models.Invoice {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := ""
	for _, d := range m.distributors {
		if d.ID == in.DistributorID {
			name = d.Name
			break
		}
	}
	inv := models.Invoice{
		ID: m.nextID("INV", &m.nextOrder),
		DistributorID: in.DistributorID, Distributor: name,
		AmountUGX: in.AmountUGX, DueDate: in.DueDate,
		Status: "open", OrderID: in.OrderID,
	}
	m.invoices = append([]models.Invoice{inv}, m.invoices...)
	return inv
}

func (m *memoryState) listPricing() []models.PricingTemplate {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]models.PricingTemplate(nil), m.pricing...)
}

func (m *memoryState) listReports() []models.ReportTemplate {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]models.ReportTemplate(nil), m.reports...)
}

func (m *memoryState) listExecution(opts ListOpts) ([]models.ExecutionTask, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts = defaultLimit(opts)
	return paginate(m.execution, opts)
}

func (m *memoryState) financeSummary() models.FinanceSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var ar, overdue float64
	for _, inv := range m.invoices {
		ar += inv.AmountUGX
		if inv.Status == "overdue" {
			overdue += inv.AmountUGX
		}
	}
	var collected float64
	for _, inv := range m.invoices {
		if inv.Status == "paid" {
			collected += inv.AmountUGX
		}
	}
	return models.FinanceSummary{
		ARBalanceUGX: ar, DSODays: 0, OverdueUGX: overdue,
		CollectedUGX: collected,
	}
}

func (m *memoryState) kpiBoard() models.KPIBoard {
	return models.KPIBoard{
		Period: "May 2026",
		Leaderboard: []models.RepScore{
			{RepID: "FF-04", RepName: "A. Achieng", Points: 94.2, Rank: 1},
			{RepID: "FF-02", RepName: "J. Sebunya", Points: 88.6, Rank: 2},
			{RepID: "FF-09", RepName: "F. Wamala", Points: 82.1, Rank: 3},
		},
		Targets: []models.KPITarget{
			{Key: "strike_rate", Label: "Strike Rate", Actual: 62, Target: 65, Unit: "%"},
			{Key: "lines_per_order", Label: "Lines / Order", Actual: 4.8, Target: 5.0, Unit: ""},
		},
	}
}

func (m *memoryState) analytics() models.AnalyticsSummary {
	return models.AnalyticsSummary{
		Cohorts: []models.CohortRow{
			{Label: "HoReCa", Outlets: 684, Revenue: 111_000_000, Retention: 78},
			{Label: "Modern Trade", Outlets: 412, Revenue: 156_000_000, Retention: 84},
		},
		Funnel: []models.FunnelStep{
			{Stage: "Visited", Count: 2162, Rate: 100},
			{Stage: "Ordered", Count: 1344, Rate: 62},
			{Stage: "Invoiced", Count: 1180, Rate: 54},
		},
	}
}

func (m *memoryState) forecast(sku string) models.Forecast {
	if sku == "" {
		sku = "BG-AA-250"
	}
	return models.Forecast{
		SKU: sku, MAPE: 6.4, HorizonDays: 28,
		Points: []models.ForecastPoint{
			{Date: "2026-05-26", Forecast: 420, Actual: 398},
			{Date: "2026-06-02", Forecast: 445, LowerBound: 410, UpperBound: 480},
		},
	}
}

func (m *memoryState) search(q string, limit int) []models.SearchResult {
	q = strings.TrimSpace(strings.ToLower(q))
	if limit <= 0 {
		limit = 18
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []models.SearchResult
	add := func(kind, label, sub, page, id string) {
		if len(out) >= limit {
			return
		}
		if q == "" || filterQ(q, label, sub, id) {
			out = append(out, models.SearchResult{Kind: kind, Label: label, Sub: sub, Page: page, ID: id})
		}
	}

	if q == "" {
		for _, p := range pagesCatalog() {
			add("page", p.Title, "Overview module", p.ID, "")
		}
		return out
	}
	for _, o := range m.outlets {
		add("outlet", o.Name, o.ID+" · "+o.Channel, "outlets", o.ID)
	}
	for _, d := range m.distributors {
		add("distrib", d.Name, d.ID+" · "+d.Region, "network", d.ID)
	}
	for _, s := range m.skus {
		add("sku", s.Name, s.Code, "stockwh", s.Code)
	}
	for _, rep := range m.reps {
		add("rep", rep.Name, rep.ID+" · "+rep.Region, "field", rep.ID)
	}
	for _, inv := range m.invoices {
		add("invoice", inv.ID+" · "+inv.Distributor, inv.Status, "finance", inv.ID)
	}
	return out
}

func (m *memoryState) outletStats() map[string]any {
	return map[string]any{
		"universe": 2847, "productivePct": 76, "avgProductivityPct": 62,
		"avgDropUgx": 188000, "rangeSelling": 4.8, "mustStockPct": 71,
	}
}

func (m *memoryState) routesStats() map[string]any {
	return map[string]any{
		"activeBeats": 28, "avgStops": 14.2, "onTimePct": 88,
		"strikeRatePct": 62, "linesPerOrder": 4.8,
	}
}

func (m *memoryState) checkInStats() map[string]any {
	return map[string]any{
		"clockedIn": 14, "totalReps": 64, "visitsToday": 87,
		"avgMinutes": 11.4, "gpsVerifiedPct": 98, "failedReports": 6,
	}
}

func (m *memoryState) ordersStats() map[string]any {
	return map[string]any{
		"open": 187, "openValueUgx": 41_200_000, "picking": 42,
		"outForDelivery": 28, "deliveredToday": 64,
	}
}


