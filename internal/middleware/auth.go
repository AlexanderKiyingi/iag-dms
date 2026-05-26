package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/alvor-technologies/iag-platform-go/authclient"
	"github.com/iag/dms/backend/internal/ctxkeys"
)

func Claims(c *gin.Context) (*authclient.Claims, bool) {
	v, ok := c.Get(ctxkeys.Claims)
	if !ok {
		return nil, false
	}
	cl, ok := v.(*authclient.Claims)
	return cl, ok
}

func ActorName(c *gin.Context) string {
	if claims, ok := Claims(c); ok && claims != nil {
		if n := strings.TrimSpace(claims.Name); n != "" {
			return n
		}
		if e := strings.TrimSpace(claims.Email); e != "" {
			return e
		}
		if claims.Subject != "" {
			return claims.Subject
		}
	}
	return "system"
}
