package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

func (h *API) ListDistributors(c *gin.Context) {
	items, total := h.Repo.ListDistributors(listOpts(c))
	paginated(c, items, total)
}

func (h *API) GetDistributor(c *gin.Context) {
	item, err := h.Repo.GetDistributor(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) ListOutlets(c *gin.Context) {
	items, total := h.Repo.ListOutlets(listOpts(c))
	paginated(c, items, total)
}

func (h *API) OutletStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.OutletStats())
}

func (h *API) GetOutlet(c *gin.Context) {
	item, err := h.Repo.GetOutlet(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) CreateOutlet(c *gin.Context) {
	var in models.OutletInput
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == "" {
		badRequest(c, "name, channel, and distributorId are required")
		return
	}
	out := h.Repo.CreateOutlet(in)
	h.publish(c, "dms.outlet.created", gin.H{"id": out.ID, "name": out.Name})
	h.recordAudit(c, "CreateOutlet", store.AuditDetail("outlet", out.ID, "created"))
	c.JSON(http.StatusCreated, out)
}

func (h *API) ListOrders(c *gin.Context) {
	items, total := h.Repo.ListOrders(listOpts(c))
	paginated(c, items, total)
}

func (h *API) OrdersBoard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"columns": h.Repo.OrdersBoard(), "stats": h.Repo.OrdersStats()})
}

func (h *API) GetOrder(c *gin.Context) {
	item, err := h.Repo.GetOrder(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) CreateOrder(c *gin.Context) {
	var in models.OrderInput
	if err := c.ShouldBindJSON(&in); err != nil || in.OutletID == "" {
		badRequest(c, "outletId is required")
		return
	}
	o := h.Repo.CreateOrder(in)
	h.publish(c, "dms.order.created", gin.H{"id": o.ID, "status": o.Status})
	h.recordAudit(c, "CreateOrder", store.AuditDetail("order", o.ID, "created"))
	c.JSON(http.StatusCreated, o)
}

func (h *API) PatchOrderStatus(c *gin.Context) {
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Status == "" {
		badRequest(c, "status is required")
		return
	}
	item, err := h.Repo.UpdateOrderStatus(c.Param("id"), body.Status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			notFound(c)
			return
		}
		apierr.JSONStatus(c, http.StatusInternalServerError, "update failed")
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) ListBeats(c *gin.Context) {
	items, total := h.Repo.ListBeats(listOpts(c))
	paginated(c, items, total)
}

func (h *API) RoutesStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.RoutesStats())
}

func (h *API) GetBeat(c *gin.Context) {
	item, err := h.Repo.GetBeat(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) ListReps(c *gin.Context) {
	items, total := h.Repo.ListReps(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListCheckIns(c *gin.Context) {
	items, total := h.Repo.ListCheckIns(listOpts(c))
	paginated(c, items, total)
}

func (h *API) CheckInStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.CheckInStats())
}

func (h *API) CreateCheckIn(c *gin.Context) {
	var in models.CheckInInput
	if err := c.ShouldBindJSON(&in); err != nil || in.RepID == "" || in.OutletID == "" {
		badRequest(c, "repId and outletId are required")
		return
	}
	ci := h.Repo.CreateCheckIn(in)
	h.publish(c, "dms.checkin.created", gin.H{"id": ci.ID, "repId": ci.RepID})
	c.JSON(http.StatusCreated, ci)
}

func (h *API) CreateVisitReport(c *gin.Context) {
	var in models.VisitReportInput
	if err := c.ShouldBindJSON(&in); err != nil || in.RepID == "" || in.OutletID == "" {
		badRequest(c, "repId and outletId are required")
		return
	}
	v := h.Repo.CreateVisitReport(in)
	h.publish(c, "dms.visit.reported", gin.H{"id": v.ID, "outcome": v.Outcome})
	c.JSON(http.StatusCreated, v)
}

func (h *API) Journey(c *gin.Context) {
	repID := c.DefaultQuery("repId", "FF-04")
	c.JSON(http.StatusOK, h.Repo.Journey(repID))
}

func (h *API) ListPromotions(c *gin.Context) {
	items, total := h.Repo.ListPromotions(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListClaims(c *gin.Context) {
	items, total := h.Repo.ListClaims(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListDispatches(c *gin.Context) {
	items, total := h.Repo.ListDispatches(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListStock(c *gin.Context) {
	items, total := h.Repo.ListStock(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListSKUs(c *gin.Context) {
	items, total := h.Repo.ListSKUs(listOpts(c))
	paginated(c, items, total)
}

func (h *API) ListInvoices(c *gin.Context) {
	if h.Finance != nil && h.Finance.Enabled() {
		items, err := h.Finance.ListInvoices(c.Request.Context(), listOpts(c).Limit)
		if err == nil {
			out := make([]models.Invoice, 0, len(items))
			for _, it := range items {
				due, _ := time.Parse("2006-01-02", it.Due)
			out = append(out, models.Invoice{
					ID:            it.No,
					DistributorID: it.Customer,
					Distributor:   it.Customer,
					AmountUGX:     it.Balance,
					Status:        it.Status,
					DueDate:       due,
				})
			}
			paginated(c, out, len(out))
			return
		}
	}
	items, total := h.Repo.ListInvoices(listOpts(c))
	paginated(c, items, total)
}

func (h *API) CreateInvoice(c *gin.Context) {
	var in models.InvoiceInput
	if err := c.ShouldBindJSON(&in); err != nil || in.DistributorID == "" {
		badRequest(c, "distributorId is required")
		return
	}
	inv := h.Repo.CreateInvoice(in)
	h.publish(c, "dms.invoice.created", gin.H{"id": inv.ID})
	h.recordAudit(c, "CreateInvoice", store.AuditDetail("invoice", inv.ID, "created"))
	c.JSON(http.StatusCreated, inv)
}

func (h *API) FinanceSummary(c *gin.Context) {
	if h.Finance != nil && h.Finance.Enabled() {
		if s, err := h.Finance.Summary(c.Request.Context()); err == nil {
			c.JSON(http.StatusOK, models.FinanceSummary{
				ARBalanceUGX: parseAmountUGX(s.ARBalance),
				OverdueUGX:   parseAmountUGX(s.Overdue),
				CollectedUGX: parseAmountUGX(s.Collected),
				DSODays:      0,
			})
			return
		}
	}
	c.JSON(http.StatusOK, h.Repo.FinanceSummary())
}

func parseAmountUGX(raw string) float64 {
	var v float64
	_, _ = fmt.Sscanf(raw, "%f", &v)
	return v
}

func (h *API) ListPricing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.Repo.ListPricing()})
}

func (h *API) ListReports(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.Repo.ListReports()})
}

func (h *API) ListExecution(c *gin.Context) {
	items, total := h.Repo.ListExecution(listOpts(c))
	paginated(c, items, total)
}

func (h *API) KPIBoard(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.KPIBoard())
}

func (h *API) Analytics(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.Analytics())
}

func (h *API) Forecast(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.Forecast(c.Query("sku")))
}

func (h *API) publish(c *gin.Context, eventType string, data any) {
	if h.Events == nil || !h.Events.Enabled() {
		return
	}
	_ = h.Events.Publish(c.Request.Context(), eventType, data)
}

func (h *API) Notifications(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.Repo.Overview().Alerts})
}
