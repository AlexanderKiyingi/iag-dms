package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/store"
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
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}
