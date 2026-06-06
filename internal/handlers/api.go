package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/config"
	"github.com/iag/dms/backend/internal/events"
	"github.com/iag/dms/backend/internal/financeclient"
	"github.com/iag/dms/backend/internal/store"
)

type API struct {
	Repo    *store.Repository
	Cfg     config.Config
	Events  *events.Bus
	Finance *financeclient.Client
}

func (h *API) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": h.Cfg.ServiceName})
}

func (h *API) Ready(c *gin.Context) {
	if err := h.Repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "service": h.Cfg.ServiceName})
}

func (h *API) Overview(c *gin.Context) {
	c.JSON(http.StatusOK, h.Repo.Overview())
}

func (h *API) Search(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "18"))
	c.JSON(http.StatusOK, gin.H{"results": h.Repo.Search(c.Query("q"), limit)})
}
