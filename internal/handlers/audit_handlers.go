package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/iag/dms/backend/internal/auth"
	"github.com/iag/dms/backend/internal/store"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

func (h *API) ListAudit(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, total, err := h.Repo.ListAudit(c.Request.Context(), limit)
	if err != nil {
		apierr.JSONStatus(c, http.StatusInternalServerError, "audit list failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "meta": gin.H{"total": total, "limit": limit}})
}

func (h *API) GetAuditEntry(c *gin.Context) {
	item, err := h.Repo.GetAudit(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
			notFound(c)
			return
		}
		apierr.JSONStatus(c, http.StatusInternalServerError, "audit get failed")
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *API) AppendAuditEntry(c *gin.Context) {
	var in struct {
		Action string `json:"action"`
		Detail string `json:"detail"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Action == "" {
		badRequest(c, "action is required")
		return
	}
	entry, err := h.Repo.AppendAudit(c.Request.Context(), in.Action, in.Detail, auth.ActorName(c))
	if err != nil {
		apierr.JSONStatus(c, http.StatusInternalServerError, "audit append failed")
		return
	}
	c.JSON(http.StatusCreated, entry)
}

func (h *API) AdminAuditLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, total, err := h.Repo.ListAPIAuditLogs(c.Request.Context(), limit)
	if err != nil {
		apierr.JSONStatus(c, http.StatusInternalServerError, "api audit failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "meta": gin.H{"total": total}})
}

func (h *API) AdminMonitoringSummary(c *gin.Context) {
	busEnabled := h.Events != nil && h.Events.Enabled()
	summary, err := h.Repo.MonitoringSummaryWithBus(c.Request.Context(), busEnabled)
	if err != nil {
		apierr.JSONStatus(c, http.StatusInternalServerError, "monitoring failed")
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *API) AdminMonitoringActivity(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	items, _, err := h.Repo.ListAPIAuditLogs(c.Request.Context(), limit)
	if err != nil {
		apierr.JSONStatus(c, http.StatusInternalServerError, "activity failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *API) InsightsSignals(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.Repo.ListSignals(c.Request.Context(), limit)
	if err != nil {
		apierr.JSONStatus(c, http.StatusInternalServerError, "signals failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
