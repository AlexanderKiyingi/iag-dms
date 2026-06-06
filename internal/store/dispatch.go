package store

import (
	"context"

	"github.com/iag/dms/backend/internal/models"
)

func (r *Repository) bg() context.Context { return context.Background() }

func (r *Repository) ListDistributors(opts ListOpts) ([]models.Distributor, int) {
	if r.pool != nil {
		return r.pgListDistributors(r.bg(), opts)
	}
	return r.mem.listDistributors(opts)
}

func (r *Repository) GetDistributor(id string) (models.Distributor, error) {
	if r.pool != nil {
		return r.pgGetDistributor(r.bg(), id)
	}
	return r.mem.getDistributor(id)
}

func (r *Repository) ListOutlets(opts ListOpts) ([]models.Outlet, int) {
	if r.pool != nil {
		return r.pgListOutlets(r.bg(), opts)
	}
	return r.mem.listOutlets(opts)
}

func (r *Repository) GetOutlet(id string) (models.Outlet, error) {
	if r.pool != nil {
		return r.pgGetOutlet(r.bg(), id)
	}
	return r.mem.getOutlet(id)
}

func (r *Repository) CreateOutlet(in models.OutletInput) (models.Outlet, error) {
	if r.pool != nil {
		return r.pgCreateOutlet(r.bg(), in)
	}
	return r.mem.createOutlet(in)
}

func (r *Repository) ListOrders(opts ListOpts) ([]models.Order, int) {
	if r.pool != nil {
		return r.pgListOrders(r.bg(), opts)
	}
	return r.mem.listOrders(opts)
}

func (r *Repository) GetOrder(id string) (models.Order, error) {
	if r.pool != nil {
		return r.pgGetOrder(r.bg(), id)
	}
	return r.mem.getOrder(id)
}

func (r *Repository) CreateOrder(in models.OrderInput) models.Order {
	if r.pool != nil {
		return r.pgCreateOrder(r.bg(), in)
	}
	return r.mem.createOrder(in)
}

func (r *Repository) UpdateOrderStatus(id, status string) (models.Order, error) {
	if r.pool != nil {
		return r.pgUpdateOrderStatus(r.bg(), id, status)
	}
	return r.mem.updateOrderStatus(id, status)
}

func (r *Repository) OrdersBoard() map[string][]models.Order {
	if r.pool != nil {
		return r.pgOrdersBoard(r.bg())
	}
	return r.mem.ordersBoard()
}

func (r *Repository) ListBeats(opts ListOpts) ([]models.Beat, int) {
	if r.pool != nil {
		return r.pgListBeats(r.bg(), opts)
	}
	return r.mem.listBeats(opts)
}

func (r *Repository) GetBeat(id string) (models.Beat, error) {
	if r.pool != nil {
		return r.pgGetBeat(r.bg(), id)
	}
	return r.mem.getBeat(id)
}

func (r *Repository) ListReps(opts ListOpts) ([]models.FieldRep, int) {
	if r.pool != nil {
		return r.pgListReps(r.bg(), opts)
	}
	return r.mem.listReps(opts)
}

func (r *Repository) ListCheckIns(opts ListOpts) ([]models.CheckIn, int) {
	if r.pool != nil {
		return r.pgListCheckIns(r.bg(), opts)
	}
	return r.mem.listCheckIns(opts)
}

func (r *Repository) CreateCheckIn(in models.CheckInInput) models.CheckIn {
	if r.pool != nil {
		return r.pgCreateCheckIn(r.bg(), in)
	}
	return r.mem.createCheckIn(in)
}

func (r *Repository) CreateVisitReport(in models.VisitReportInput) models.VisitReport {
	if r.pool != nil {
		return r.pgCreateVisitReport(r.bg(), in)
	}
	return r.mem.createVisitReport(in)
}

func (r *Repository) Journey(repID string) models.JourneyDay {
	if r.pool != nil {
		return r.pgJourney(r.bg(), repID)
	}
	return r.mem.journey(repID)
}

func (r *Repository) ListPromotions(opts ListOpts) ([]models.Promotion, int) {
	if r.pool != nil {
		return r.pgListPromotions(r.bg(), opts)
	}
	return r.mem.listPromotions(opts)
}

func (r *Repository) ListClaims(opts ListOpts) ([]models.Claim, int) {
	if r.pool != nil {
		return r.pgListClaims(r.bg(), opts)
	}
	return r.mem.listClaims(opts)
}

func (r *Repository) ListDispatches(opts ListOpts) ([]models.Dispatch, int) {
	if r.pool != nil {
		return r.pgListDispatches(r.bg(), opts)
	}
	return r.mem.listDispatches(opts)
}

func (r *Repository) ListStock(opts ListOpts) ([]models.StockPosition, int) {
	if r.pool != nil {
		return r.pgListStock(r.bg(), opts)
	}
	return r.mem.listStock(opts)
}

func (r *Repository) ListSKUs(opts ListOpts) ([]models.SKU, int) {
	if r.pool != nil {
		return r.pgListSKUs(r.bg(), opts)
	}
	return r.mem.listSKUs(opts)
}

func (r *Repository) ListInvoices(opts ListOpts) ([]models.Invoice, int) {
	if r.pool != nil {
		return r.pgListInvoices(r.bg(), opts)
	}
	return r.mem.listInvoices(opts)
}

func (r *Repository) CreateInvoice(in models.InvoiceInput) models.Invoice {
	if r.pool != nil {
		return r.pgCreateInvoice(r.bg(), in)
	}
	return r.mem.createInvoice(in)
}

func (r *Repository) ListPricing() []models.PricingTemplate {
	if r.pool != nil {
		return r.pgListPricing(r.bg())
	}
	return r.mem.listPricing()
}

func (r *Repository) ListReports() []models.ReportTemplate {
	if r.pool != nil {
		return r.pgListReports(r.bg())
	}
	return r.mem.listReports()
}

func (r *Repository) ListExecution(opts ListOpts) ([]models.ExecutionTask, int) {
	if r.pool != nil {
		return r.pgListExecution(r.bg(), opts)
	}
	return r.mem.listExecution(opts)
}

func (r *Repository) FinanceSummary() models.FinanceSummary {
	if r.pool != nil {
		return r.pgFinanceSummary(r.bg())
	}
	return r.mem.financeSummary()
}

func (r *Repository) PatchOutlet(id string, patch models.OutletPatch) (models.Outlet, error) {
	if r.pool != nil {
		return r.pgPatchOutlet(r.bg(), id, patch)
	}
	return r.mem.patchOutlet(id, patch)
}

func (r *Repository) GetInvoice(id string) (models.Invoice, error) {
	if r.pool != nil {
		return r.pgGetInvoice(r.bg(), id)
	}
	return r.mem.getInvoice(id)
}

func (r *Repository) ListVisitReports(opts ListOpts) ([]models.VisitReport, int) {
	if r.pool != nil {
		return r.pgListVisitReports(r.bg(), opts)
	}
	return r.mem.listVisitReports(opts)
}

func (r *Repository) CompleteCheckIn(id string) (models.CheckIn, error) {
	if r.pool != nil {
		return r.pgCompleteCheckIn(r.bg(), id)
	}
	return r.mem.completeCheckIn(id)
}

func (r *Repository) CreateClaim(in models.ClaimInput) models.Claim {
	if r.pool != nil {
		return r.pgCreateClaim(r.bg(), in)
	}
	return r.mem.createClaim(in)
}

func (r *Repository) CreatePromotion(in models.PromotionInput) models.Promotion {
	if r.pool != nil {
		return r.pgCreatePromotion(r.bg(), in)
	}
	return r.mem.createPromotion(in)
}

func (r *Repository) CreateDispatch(in models.DispatchInput) models.Dispatch {
	if r.pool != nil {
		return r.pgCreateDispatch(r.bg(), in)
	}
	return r.mem.createDispatch(in)
}

func (r *Repository) RunReport(in models.ReportRunInput) models.ReportRun {
	if r.pool != nil {
		return r.pgRunReport(r.bg(), in)
	}
	return r.mem.runReport(in)
}

func (r *Repository) ExportPage(page, format string) models.ExportPayload {
	if r.pool != nil {
		return r.pgExportPage(r.bg(), page, format)
	}
	return r.mem.exportPage(page, format)
}

func (r *Repository) KPIBoard() models.KPIBoard {
	if r.pool != nil {
		return r.pgKPIBoard(r.bg())
	}
	return r.mem.kpiBoard()
}

func (r *Repository) Analytics() models.AnalyticsSummary {
	if r.pool != nil {
		return r.pgAnalytics(r.bg())
	}
	return r.mem.analytics()
}

func (r *Repository) Forecast(sku string) models.Forecast {
	if r.pool != nil {
		return r.pgForecast(r.bg(), sku)
	}
	return r.mem.forecast(sku)
}

func (r *Repository) Search(q string, limit int) []models.SearchResult {
	if r.pool != nil {
		return r.pgSearch(r.bg(), q, limit)
	}
	return r.mem.search(q, limit)
}

func (r *Repository) OutletStats() map[string]any {
	if r.pool != nil {
		return r.pgOutletStats(r.bg())
	}
	return r.mem.outletStats()
}

func (r *Repository) RoutesStats() map[string]any {
	if r.pool != nil {
		return r.pgRoutesStats(r.bg())
	}
	return r.mem.routesStats()
}

func (r *Repository) CheckInStats() map[string]any {
	if r.pool != nil {
		return r.pgCheckInStats(r.bg())
	}
	return r.mem.checkInStats()
}

func (r *Repository) OrdersStats() map[string]any {
	if r.pool != nil {
		return r.pgOrdersStats(r.bg())
	}
	return r.mem.ordersStats()
}
