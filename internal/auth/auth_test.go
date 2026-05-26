package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	platformauth "github.com/alvor-technologies/iag-platform-go/authclient"

	"github.com/iag/dms/backend/internal/ctxkeys"
)

func TestHasPermStrictRBACDeniesEmptyPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(strictRBACKey, true)
	c.Set(ctxkeys.Claims, &platformauth.Claims{
		PrincipalType: platformauth.PrincipalUser,
		Permissions:   nil,
	})

	if HasPerm(c, "dms.view_overview") {
		t.Fatal("strict RBAC should deny when permissions list is empty")
	}
}

func TestHasPermDevAllowsEmptyPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ctxkeys.Claims, &platformauth.Claims{
		PrincipalType: platformauth.PrincipalUser,
		Permissions:   nil,
	})

	if !HasPerm(c, "dms.view_overview") {
		t.Fatal("non-strict mode should allow empty permissions for local dev tokens")
	}
}

func TestHasPermExplicitPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(strictRBACKey, true)
	c.Set(ctxkeys.Claims, &platformauth.Claims{
		PrincipalType: platformauth.PrincipalUser,
		Permissions:   []string{"dms.view_overview"},
	})

	if !HasPerm(c, "dms.view_overview") {
		t.Fatal("expected explicit permission to pass under strict RBAC")
	}
	if HasPerm(c, "dms.manage_orders") {
		t.Fatal("expected missing permission to fail")
	}
}
