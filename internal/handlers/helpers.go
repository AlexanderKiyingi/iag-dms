package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/store"
	"github.com/alvor-technologies/iag-platform-go/apierr"
)

func listOpts(c *gin.Context) store.ListOpts {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	return store.ListOpts{
		Limit: limit, Offset: offset, Q: c.Query("q"),
		Status: c.Query("status"), Channel: c.Query("channel"),
		DistributorID: c.Query("distributorId"), RepID: c.Query("repId"),
		BeatID: c.Query("beatId"),
	}
}

func paginated(c *gin.Context, items any, total int) {
	opts := listOpts(c)
	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"meta": gin.H{"total": total, "limit": opts.Limit, "offset": opts.Offset},
	})
}

func notFound(c *gin.Context) {
	apierr.JSONStatus(c, http.StatusNotFound, "not found")
}

func badRequest(c *gin.Context, msg string) {
	apierr.JSONStatus(c, http.StatusBadRequest, msg)
}
