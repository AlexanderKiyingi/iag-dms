package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *API) PlatformStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":        h.Cfg.ServiceName,
		"authMode":       h.Cfg.AuthMode,
		"gatewayPrefix":  h.Cfg.GatewayAPIPrefix,
		"publicApiUrl":   h.Cfg.PublicAPIURL,
		"store":          map[bool]string{true: "memory", false: "postgres"}[h.Cfg.UseMemoryStore],
		"events":         h.Events != nil && h.Events.Enabled(),
		"consumer":       h.Cfg.ConsumerEnabled,
		"consumerTopic":  h.Cfg.ConsumerTopic,
	})
}
