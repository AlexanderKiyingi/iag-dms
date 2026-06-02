package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

func (h *API) PatchOutlet(c *gin.Context) {
	var patch models.OutletPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		badRequest(c, "invalid body")
		return
	}
	if patch.Name == "" && patch.Address == "" && patch.Channel == "" && patch.BeatID == "" &&
		patch.Status == "" && patch.Score == "" && patch.Frequency == "" {
		badRequest(c, "at least one field required")
		return
	}
	item, err := h.Repo.PatchOutlet(c.Param("id"), patch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			notFound(c)
			return
		}
		apierr.JSONStatus(c, http.StatusInternalServerError, "update failed")
		return
	}
	h.recordAudit(c, "PatchOutlet", store.AuditDetail("outlet", item.ID, "updated"))
	c.JSON(http.StatusOK, item)
}

func (h *API) GetInvoice(c *gin.Context) {
	item, err := h.Repo.GetInvoice(c.Param("id"))
	if err != nil {
		notFound(c)
		return
	}
	h.recordAudit(c, "CompleteCheckIn", store.AuditDetail("check-in", item.ID, "completed"))
	c.JSON(http.StatusOK, item)
}

func (h *API) ListVisitReports(c *gin.Context) {
	items, total := h.Repo.ListVisitReports(listOpts(c))
	paginated(c, items, total)
}

func (h *API) CompleteCheckIn(c *gin.Context) {
	item, err := h.Repo.CompleteCheckIn(c.Param("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			notFound(c)
			return
		}
		apierr.JSONStatus(c, http.StatusInternalServerError, "complete failed")
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) CreateClaim(c *gin.Context) {
	var in models.ClaimInput
	if err := c.ShouldBindJSON(&in); err != nil || in.OutletID == "" || in.Type == "" {
		badRequest(c, "outletId and type are required")
		return
	}
	cl := h.Repo.CreateClaim(in)
	h.recordAudit(c, "CreateClaim", store.AuditDetail("claim", cl.ID, "created"))
	c.JSON(http.StatusCreated, cl)
}

func (h *API) CreatePromotion(c *gin.Context) {
	var in models.PromotionInput
	if err := c.ShouldBindJSON(&in); err != nil || in.Name == "" {
		badRequest(c, "name is required")
		return
	}
	p := h.Repo.CreatePromotion(in)
	c.JSON(http.StatusCreated, p)
}

func (h *API) CreateDispatch(c *gin.Context) {
	var in models.DispatchInput
	if err := c.ShouldBindJSON(&in); err != nil || in.TruckID == "" || len(in.OrderIDs) == 0 {
		badRequest(c, "truckId and orderIds are required")
		return
	}
	d := h.Repo.CreateDispatch(in)
	h.publish(c, "dms.dispatch.created", gin.H{"id": d.ID, "truckId": d.TruckID})
	h.recordAudit(c, "CreateDispatch", store.AuditDetail("dispatch", d.ID, "created"))
	c.JSON(http.StatusCreated, d)
}

func (h *API) RunReport(c *gin.Context) {
	var in models.ReportRunInput
	if err := c.ShouldBindJSON(&in); err != nil {
		badRequest(c, "invalid body")
		return
	}
	if strings.TrimSpace(in.Name) == "" && strings.TrimSpace(in.TemplateID) == "" {
		badRequest(c, "name or templateId is required")
		return
	}
	run := h.Repo.RunReport(in)
	c.JSON(http.StatusAccepted, run)
}

func (h *API) ExportPage(c *gin.Context) {
	page := c.Param("page")
	if page == "" {
		badRequest(c, "page is required")
		return
	}
	var body models.ExportInput
	_ = c.ShouldBindJSON(&body)
	payload := h.Repo.ExportPage(page, body.Format)
	c.JSON(http.StatusOK, payload)
}
