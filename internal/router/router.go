package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	platformmw "github.com/alvor-technologies/iag-platform-go/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/iag/dms/backend/internal/auth"
	"github.com/iag/dms/backend/internal/config"
	"github.com/iag/dms/backend/internal/events"
	"github.com/iag/dms/backend/internal/financeclient"
	"github.com/iag/dms/backend/internal/handlers"
	"github.com/iag/dms/backend/internal/middleware"
	"github.com/iag/dms/backend/internal/storage"
	"github.com/iag/dms/backend/internal/store"
)

type Options struct {
	Cfg          config.Config
	Repo         *store.Repository
	PlatformAuth *middleware.PlatformAuth
	Events       *events.Bus
	Finance      *financeclient.Client
	Storage      storage.Store
}

func New(opts Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware(opts.Cfg.ServiceName))
	r.Use(platformmw.RequestID())
	r.Use(securityHeaders())
	r.Use(corsMiddleware(opts.Cfg.CORSOrigin))

	api := &handlers.API{Repo: opts.Repo, Cfg: opts.Cfg, Events: opts.Events, Finance: opts.Finance, Storage: opts.Storage}

	// Root service descriptor. The embedded static UI was removed — DMS is a
	// headless API; the browser UI lives in the separate dmsiag repo and talks
	// to this service through the gateway.
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": opts.Cfg.ServiceName,
			"status":  "ok",
			"api":     "/v1",
		})
	})

	r.GET("/healthz", api.Health)
	r.GET("/health", api.Health)
	r.GET("/ready", api.Ready)

	v1 := r.Group("/v1")
	if opts.PlatformAuth != nil {
		v1.Use(opts.PlatformAuth.AttachPrincipal())
	}
	if opts.Cfg.StrictRBAC() {
		v1.Use(auth.StrictRBAC())
	}
	v1.Use(middleware.RequestAudit(opts.Repo))

	registerPlatformRoutes(v1, api)
	registerDomainRoutes(v1, api)
	registerAuditRoutes(v1, api)
	registerAdminRoutes(v1, api)
	registerIntelligenceRoutes(v1, api)

	return r
}

func registerPlatformRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	v1.GET("/bootstrap", auth.RequirePerm("dms.view_overview"), api.Bootstrap)
	v1.GET("/auth/session", auth.RequirePerm("dms.view_overview"), api.Session)
	v1.GET("/permissions/catalog", auth.RequirePerm("dms.view_overview"), api.PermissionsCatalog)
	v1.GET("/permissions/builtin", auth.RequirePerm("dms.view_overview"), api.PermissionsBuiltin)
	v1.POST("/permissions/check", auth.RequirePerm("dms.view_overview"), api.PermissionsCheck)
	v1.GET("/permissions/me", auth.RequirePerm("dms.view_overview"), api.PermissionsMe)
	v1.GET("/lookups/:kind", auth.RequirePerm("dms.view_overview"), api.Lookups)
	v1.GET("/search", auth.RequirePerm("dms.view_overview"), api.Search)
	v1.GET("/notifications", auth.RequirePerm("dms.view_overview"), api.Notifications)
	v1.POST("/messages", auth.RequirePerm("dms.view_overview"), api.SendMessage)
	v1.GET("/platform/status", auth.RequireStaff(), api.PlatformStatus)
}

func registerDomainRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	v1.GET("/overview", auth.RequirePerm("dms.view_overview"), api.Overview)

	v1.GET("/distributors", auth.RequirePerm("dms.view_overview"), api.ListDistributors)
	v1.GET("/distributors/:id", auth.RequirePerm("dms.view_overview"), api.GetDistributor)

	v1.GET("/outlets/stats", auth.RequirePerm("dms.view_outlets"), api.OutletStats)
	v1.GET("/outlets", auth.RequirePerm("dms.view_outlets"), api.ListOutlets)
	v1.POST("/outlets", auth.RequirePerm("dms.manage_outlets"), api.CreateOutlet)
	v1.GET("/outlets/:id", auth.RequirePerm("dms.view_outlets"), api.GetOutlet)
	v1.PATCH("/outlets/:id", auth.RequirePerm("dms.manage_outlets"), api.PatchOutlet)
	v1.DELETE("/outlets/:id", auth.RequirePerm("dms.manage_outlets"), api.DeleteOutlet)

	v1.GET("/orders/stats", auth.RequirePerm("dms.view_orders"), func(c *gin.Context) {
		c.JSON(http.StatusOK, api.Repo.OrdersStats())
	})
	v1.GET("/orders/board", auth.RequirePerm("dms.view_orders"), api.OrdersBoard)
	v1.GET("/orders", auth.RequirePerm("dms.view_orders"), api.ListOrders)
	v1.POST("/orders", auth.RequirePerm("dms.manage_orders"), api.CreateOrder)
	v1.GET("/orders/:id", auth.RequirePerm("dms.view_orders"), api.GetOrder)
	v1.PATCH("/orders/:id/status", auth.RequirePerm("dms.manage_orders"), api.PatchOrderStatus)
	v1.DELETE("/orders/:id", auth.RequirePerm("dms.manage_orders"), api.DeleteOrder)

	v1.GET("/routes/stats", auth.RequirePerm("dms.view_overview"), api.RoutesStats)
	v1.GET("/beats", auth.RequirePerm("dms.view_overview"), api.ListBeats)
	v1.GET("/beats/:id", auth.RequirePerm("dms.view_overview"), api.GetBeat)

	v1.GET("/field/reps", auth.RequirePerm("dms.field_checkin"), api.ListReps)
	v1.GET("/field/check-ins/stats", auth.RequirePerm("dms.field_checkin"), api.CheckInStats)
	v1.GET("/field/check-ins", auth.RequirePerm("dms.field_checkin"), api.ListCheckIns)
	v1.POST("/field/check-ins", auth.RequirePerm("dms.field_checkin"), api.CreateCheckIn)
	v1.PATCH("/field/check-ins/:id", auth.RequirePerm("dms.field_checkin"), api.CompleteCheckIn)
	v1.DELETE("/field/check-ins/:id", auth.RequirePerm("dms.field_checkin"), api.DeleteCheckIn)
	v1.GET("/field/visit-reports", auth.RequirePerm("dms.field_checkin"), api.ListVisitReports)
	v1.POST("/field/visit-reports", auth.RequirePerm("dms.field_checkin"), api.CreateVisitReport)
	v1.GET("/field/journey", auth.RequirePerm("dms.field_checkin"), api.Journey)
	v1.GET("/field/journey/assignments", auth.RequirePerm("dms.field_checkin"), api.ListJourneyAssignments)
	v1.POST("/field/journey/assign", auth.RequirePerm("dms.field_checkin"), api.CreateJourneyAssignment)
	v1.PATCH("/field/journey/assign/:id", auth.RequirePerm("dms.field_checkin"), api.PatchJourneyAssignment)
	v1.DELETE("/field/journey/assign/:id", auth.RequirePerm("dms.field_checkin"), api.DeleteJourneyAssignment)
	v1.GET("/field/execution", auth.RequirePerm("dms.field_checkin"), api.ListExecution)
	v1.POST("/field/execution/:id/photos", auth.RequirePerm("dms.field_checkin"), api.UploadExecutionPhoto)
	v1.GET("/field/execution/:id/photos", auth.RequirePerm("dms.field_checkin"), api.ListExecutionPhotos)

	v1.GET("/promotions", auth.RequirePerm("dms.view_overview"), api.ListPromotions)
	v1.POST("/promotions", auth.RequirePerm("dms.manage_promotions"), api.CreatePromotion)
	v1.PATCH("/promotions/:id", auth.RequirePerm("dms.manage_promotions"), api.PatchPromotion)
	v1.DELETE("/promotions/:id", auth.RequirePerm("dms.manage_promotions"), api.DeletePromotion)
	v1.GET("/claims", auth.RequirePerm("dms.view_overview"), api.ListClaims)
	v1.POST("/claims", auth.RequirePerm("dms.manage_claims"), api.CreateClaim)
	v1.PATCH("/claims/:id", auth.RequirePerm("dms.manage_claims"), api.PatchClaim)
	v1.DELETE("/claims/:id", auth.RequirePerm("dms.manage_claims"), api.DeleteClaim)
	v1.GET("/dispatch", auth.RequirePerm("dms.view_orders"), api.ListDispatches)
	v1.POST("/dispatch", auth.RequirePerm("dms.manage_dispatch"), api.CreateDispatch)
	v1.PATCH("/dispatch/:id", auth.RequirePerm("dms.manage_dispatch"), api.PatchDispatch)
	v1.DELETE("/dispatch/:id", auth.RequirePerm("dms.manage_dispatch"), api.DeleteDispatch)

	v1.GET("/stock/distributor", auth.RequirePerm("dms.view_overview"), api.ListStock)
	v1.GET("/stock/skus", auth.RequirePerm("dms.view_overview"), api.ListSKUs)

	v1.GET("/finance/summary", auth.RequirePerm("dms.view_finance"), api.FinanceSummary)
	v1.GET("/invoices", auth.RequirePerm("dms.view_finance"), api.ListInvoices)
	v1.GET("/invoices/:id", auth.RequirePerm("dms.view_finance"), api.GetInvoice)
	v1.POST("/invoices", auth.RequirePerm("dms.manage_invoices"), api.CreateInvoice)
	v1.PATCH("/invoices/:id", auth.RequirePerm("dms.manage_invoices"), api.PatchInvoice)
	v1.DELETE("/invoices/:id", auth.RequirePerm("dms.manage_invoices"), api.DeleteInvoice)
	v1.POST("/invoices/:id/efris", auth.RequirePerm("dms.manage_invoices"), api.SubmitInvoiceEFRIS)
	v1.GET("/invoices/:id/document", auth.RequirePerm("dms.view_finance"), api.GetInvoiceDocument)
	v1.POST("/invoices/:id/send", auth.RequirePerm("dms.manage_invoices"), api.SendInvoice)

	v1.GET("/pricing/templates", auth.RequirePerm("dms.view_finance"), api.ListPricing)
	v1.POST("/pricing/templates", auth.RequirePerm("dms.manage_pricing"), api.CreatePricing)
	v1.PATCH("/pricing/templates/:id", auth.RequirePerm("dms.manage_pricing"), api.PatchPricing)
	v1.POST("/pricing/templates/:id/approve", auth.RequirePerm("dms.manage_pricing"), api.ApprovePricing)
	v1.GET("/pricing/templates/:id/versions", auth.RequirePerm("dms.view_finance"), api.ListPricingVersions)
	v1.GET("/reports/templates", auth.RequirePerm("dms.run_reports"), api.ListReports)
	v1.POST("/reports/run", auth.RequirePerm("dms.run_reports"), api.RunReport)
	v1.GET("/reports/scheduled", auth.RequirePerm("dms.run_reports"), api.ListReportSchedules)
	v1.POST("/reports/schedule", auth.RequirePerm("dms.run_reports"), api.CreateReportSchedule)
	v1.DELETE("/reports/schedule/:id", auth.RequirePerm("dms.run_reports"), api.DeleteReportSchedule)
	v1.POST("/exports/:page", auth.RequirePerm("dms.run_reports"), api.ExportPage)

	v1.POST("/attachments", auth.RequirePerm("dms.view_overview"), api.UploadAttachment)
	v1.GET("/attachments", auth.RequirePerm("dms.view_overview"), api.ListAttachments)
	v1.GET("/attachments/:id/download", auth.RequirePerm("dms.view_overview"), api.DownloadAttachment)

	v1.GET("/kpi/board", auth.RequirePerm("dms.insights.read"), api.KPIBoard)
	v1.GET("/analytics/summary", auth.RequirePerm("dms.insights.read"), api.Analytics)
	v1.GET("/forecast", auth.RequirePerm("dms.insights.read"), api.Forecast)
}

func registerAuditRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	v1.GET("/audit", auth.RequirePerm("dms.audit.read"), api.ListAudit)
	v1.POST("/audit", auth.RequirePerm("dms.audit.create"), api.AppendAuditEntry)
	v1.GET("/audit/:id", auth.RequirePerm("dms.audit.read"), api.GetAuditEntry)
}

func registerAdminRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	admin := v1.Group("/admin")
	admin.Use(auth.RequireStaff())
	{
		admin.GET("/audit", auth.RequirePerm("dms.audit.read"), api.ListAudit)
		admin.GET("/audit-logs", auth.RequirePerm("dms.admin.read"), api.AdminAuditLogs)
		admin.GET("/monitoring/summary", auth.RequirePerm("dms.admin.read"), api.AdminMonitoringSummary)
		admin.GET("/monitoring/activity", auth.RequirePerm("dms.admin.read"), api.AdminMonitoringActivity)
	}
}

func registerIntelligenceRoutes(v1 *gin.RouterGroup, api *handlers.API) {
	v1.GET("/insights/signals", auth.RequirePerm("dms.insights.read"), api.InsightsSignals)
}

// securityHeaders emits the platform-wide baseline browser security header
// set. DMS is a headless JSON API (no bundled UI), so the Content-Security-Policy
// locks everything down to 'none' — nothing is meant to be rendered in a browser.
//
// HSTS is gated on TLS termination (direct TLS or trusted X-Forwarded-Proto)
// so plain-HTTP dev environments (http://localhost) do not lock browsers
// into HTTPS for the developer's whole domain.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), interest-cohort=()")
		c.Header("X-XSS-Protection", "1; mode=block")
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		c.Next()
	}
}

// cors helpers below

func corsMiddleware(allowed string) gin.HandlerFunc {
	allowAny := allowed == "" || allowed == "*"
	allowedOrigins := splitAllowedOrigins(allowed)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAny || (origin != "" && originAllowed(origin, allowedOrigins)) {
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else if allowAny {
				c.Header("Access-Control-Allow-Origin", "*")
			}
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "ETag, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func splitAllowedOrigins(allowed string) []string {
	if allowed == "" || allowed == "*" {
		return nil
	}
	parts := strings.Split(allowed, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func originAllowed(origin string, allowed []string) bool {
	for _, candidate := range allowed {
		if origin == candidate {
			return true
		}
	}
	return false
}
