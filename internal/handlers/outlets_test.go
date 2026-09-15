package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/store"
)

func outletAPI(t *testing.T) (*API, string) {
	t.Helper()
	repo := store.NewMemoryWith(store.Fixture{
		Distributors: []models.Distributor{{ID: "D-001", Name: "Kampala Premium"}},
		Beats:        []models.Beat{{ID: "BT-01", Name: "Nakawa", RepID: "FF-01"}},
	})
	o, err := repo.CreateOutlet(models.OutletInput{Name: "Shop", Channel: "Bar", DistributorID: "D-001"})
	if err != nil {
		t.Fatal(err)
	}
	return &API{Repo: repo}, o.ID
}

func patchOutlet(api *API, id, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/v1/outlets/"+id, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: id}}
	api.PatchOutlet(c)
	return w
}

func TestPatchOutletCoercesFormStringsAndPersistsGPS(t *testing.T) {
	api, id := outletAPI(t)
	w := patchOutlet(api, id, `{"lat":"0.3476","lng":"32.5825","creditLimitUgx":"750000","radiusM":"90","kycStatus":"approved"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var o models.Outlet
	if err := json.Unmarshal(w.Body.Bytes(), &o); err != nil {
		t.Fatal(err)
	}
	if o.Lat != 0.3476 || o.Lng != 32.5825 || o.CreditLimitUGX != 750000 || o.RadiusM != 90 || o.KYCStatus != "approved" {
		t.Fatalf("patch not applied: %+v", o)
	}
}

func TestPatchOutletRejectsEmptyAndUnknownBeat(t *testing.T) {
	api, id := outletAPI(t)
	if w := patchOutlet(api, id, `{}`); w.Code != http.StatusBadRequest {
		t.Fatalf("empty patch: status %d", w.Code)
	}
	if w := patchOutlet(api, id, `{"beatId":"Nakawa"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("bogus beat: status %d body %s", w.Code, w.Body.String())
	}
	if w := patchOutlet(api, id, `{"beatId":"BT-01"}`); w.Code != http.StatusOK {
		t.Fatalf("real beat: status %d body %s", w.Code, w.Body.String())
	}
}
