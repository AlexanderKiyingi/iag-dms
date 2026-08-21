package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/auth"
	"github.com/iag/dms/backend/internal/events"
	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
)

// ---- Journey assignments ---------------------------------------------------

func (h *API) ListJourneyAssignments(c *gin.Context) {
	items := h.Repo.ListJourneyAssignments(c.Query("repId"), c.Query("date"))
	c.JSON(http.StatusOK, gin.H{"items": items, "data": items})
}

func (h *API) CreateJourneyAssignment(c *gin.Context) {
	var in models.JourneyAssignmentInput
	if err := bindJSONCoerced(c, &in); err != nil || in.RepID == "" || in.Date == "" || in.BeatID == "" {
		badRequest(c, "repId, date and beatId are required")
		return
	}
	a, err := h.Repo.CreateJourneyAssignment(in)
	if err != nil {
		writeStoreErr(c, err, "create failed")
		return
	}
	h.publish(c, "dms.journey.assigned", gin.H{"id": a.ID, "repId": a.RepID, "beatId": a.BeatID})
	h.recordAudit(c, "CreateJourneyAssignment", store.AuditDetail("journey", a.ID, "assigned"))
	c.JSON(http.StatusCreated, a)
}

func (h *API) PatchJourneyAssignment(c *gin.Context) {
	var patch models.JourneyAssignmentPatch
	if err := bindJSONCoerced(c, &patch); err != nil {
		badRequest(c, "invalid body")
		return
	}
	a, err := h.Repo.PatchJourneyAssignment(c.Param("id"), patch)
	if err != nil {
		writeStoreErr(c, err, "update failed")
		return
	}
	h.recordAudit(c, "PatchJourneyAssignment", store.AuditDetail("journey", a.ID, "updated"))
	c.JSON(http.StatusOK, a)
}

func (h *API) DeleteJourneyAssignment(c *gin.Context) {
	h.deleteEntity(c, "journey", h.Repo.DeleteJourneyAssignment)
}

// ---- Pricing lifecycle -----------------------------------------------------

func (h *API) CreatePricing(c *gin.Context) {
	var in models.PricingInput
	if err := bindJSONCoerced(c, &in); err != nil || in.Name == "" {
		badRequest(c, "name is required")
		return
	}
	p, err := h.Repo.CreatePricing(in, auth.ActorName(c))
	if err != nil {
		writeStoreErr(c, err, "create failed")
		return
	}
	h.publish(c, "dms.pricing.created", gin.H{"id": p.ID, "name": p.Name})
	h.recordAudit(c, "CreatePricing", store.AuditDetail("pricing", p.ID, "created"))
	c.JSON(http.StatusCreated, p)
}

func (h *API) PatchPricing(c *gin.Context) {
	var patch models.PricingPatch
	if err := bindJSONCoerced(c, &patch); err != nil {
		badRequest(c, "invalid body")
		return
	}
	p, err := h.Repo.PatchPricing(c.Param("id"), patch, auth.ActorName(c))
	if err != nil {
		writeStoreErr(c, err, "update failed")
		return
	}
	h.publish(c, "dms.pricing.updated", gin.H{"id": p.ID, "version": p.Version})
	h.recordAudit(c, "PatchPricing", store.AuditDetail("pricing", p.ID, "updated to "+p.Version))
	c.JSON(http.StatusOK, p)
}

func (h *API) ApprovePricing(c *gin.Context) {
	p, err := h.Repo.ApprovePricing(c.Param("id"), auth.ActorName(c))
	if err != nil {
		writeStoreErr(c, err, "approve failed")
		return
	}
	h.publish(c, "dms.pricing.approved", gin.H{"id": p.ID, "version": p.Version})
	// A pricing template carries no requester address, so the decision goes to
	// the ops desk. Approving one changes what every distributor is charged,
	// which is worth a record outside the audit log.
	if h.Events != nil && h.Events.Enabled() {
		if desk := events.DefaultNotifyRecipient(); desk != "" {
			h.Events.PublishAlert(c.Request.Context(), "", desk, "approval.decision", map[string]string{
				"Title": "Pricing approved: " + p.Name,
				"Body": "Pricing template " + p.Name + " (version " +
					p.Version + ") was approved by " + auth.ActorName(c) + ".",
			}, p.ID)
		}
	}
	h.recordAudit(c, "ApprovePricing", store.AuditDetail("pricing", p.ID, "approved"))
	c.JSON(http.StatusOK, p)
}

func (h *API) ListPricingVersions(c *gin.Context) {
	items := h.Repo.ListPricingVersions(c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"items": items, "data": items})
}

// ---- Report schedules ------------------------------------------------------

func (h *API) ListReportSchedules(c *gin.Context) {
	items := h.Repo.ListReportSchedules()
	c.JSON(http.StatusOK, gin.H{"items": items, "data": items})
}

func (h *API) CreateReportSchedule(c *gin.Context) {
	var in models.ReportScheduleInput
	if err := bindJSONCoerced(c, &in); err != nil || in.Recipient == "" {
		badRequest(c, "recipient is required")
		return
	}
	s, err := h.Repo.CreateReportSchedule(in)
	if err != nil {
		writeStoreErr(c, err, "create failed")
		return
	}
	h.recordAudit(c, "CreateReportSchedule", store.AuditDetail("report-schedule", s.ID, "created"))
	c.JSON(http.StatusCreated, s)
}

func (h *API) DeleteReportSchedule(c *gin.Context) {
	h.deleteEntity(c, "report-schedule", h.Repo.DeleteReportSchedule)
}
