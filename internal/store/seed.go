package store

import (
	"time"

	"github.com/iag/dms/backend/internal/models"
)

func seedMemory(m *memoryState) {
	onboard := time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC)
	dueSoon := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	dueToday := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)

	m.distributors = []models.Distributor{
		{ID: "D-001", Name: "Kampala Premium Beverages", Tier: 1, Region: "Central", Manager: "J. Sebunya", Outlets: 412, SellInRate: 91.2, RevenueUGX: 142_000_000, Status: "active", OnboardedAt: onboard},
		{ID: "D-007", Name: "Mbale Coffee Hub Ltd", Tier: 1, Region: "Eastern", Manager: "F. Wamala", Outlets: 198, SellInRate: 88.4, RevenueUGX: 84_000_000, Status: "active", OnboardedAt: onboard},
		{ID: "D-014", Name: "Rwenzori Trade Co", Tier: 2, Region: "Western", Outlets: 156, SellInRate: 82.1, RevenueUGX: 48_000_000, Status: "active"},
		{ID: "D-022", Name: "Jinja Distributors Ltd", Tier: 2, Region: "Eastern", Outlets: 124, SellInRate: 79.8, RevenueUGX: 38_000_000, Status: "active"},
		{ID: "D-029", Name: "Gulu Northern Ltd", Tier: 2, Region: "Northern", Outlets: 88, SellInRate: 74.2, RevenueUGX: 24_000_000, Status: "watch"},
		{ID: "D-031", Name: "Fort Portal Beverages", Tier: 3, Region: "Western", Outlets: 62, SellInRate: 68.5, RevenueUGX: 18_000_000, Status: "active"},
	}

	m.outlets = []models.Outlet{
		{ID: "OUT-00214", Name: "Cafe Javas Kololo", Address: "Plot 14, Acacia Ave · Kampala", Channel: "HoReCa", DistributorID: "D-001", BeatID: "B-08", QTDValueUGX: 8_400_000, Frequency: "2x/wk", Score: "A+", Status: "active", Lat: 0.34288, Lng: 32.60241},
		{ID: "OUT-00318", Name: "Shoprite Lugogo", Address: "Lugogo Mall · Kampala", Channel: "MT", DistributorID: "D-001", BeatID: "B-02", QTDValueUGX: 12_800_000, Frequency: "3x/wk", Score: "A+", Status: "active"},
		{ID: "OUT-00482", Name: "Quality Supermarket Lubowa", Address: "Entebbe Rd · Wakiso", Channel: "MT", DistributorID: "D-001", BeatID: "B-04", QTDValueUGX: 6_200_000, Frequency: "2x/wk", Score: "A", Status: "active"},
		{ID: "OUT-00694", Name: "Endiro Coffee Kisementi", Address: "Kisementi · Kampala", Channel: "HoReCa", DistributorID: "D-001", BeatID: "B-12", QTDValueUGX: 4_800_000, Frequency: "2x/wk", Score: "A", Status: "active"},
		{ID: "OUT-00821", Name: "Kamir Mini-Mart", Address: "Bwaise · Kampala", Channel: "GT", DistributorID: "D-001", BeatID: "B-15", QTDValueUGX: 1_400_000, Frequency: "1x/wk", Score: "B", Status: "active"},
		{ID: "OUT-01124", Name: "Sironko Mountain Café", Address: "Mbale Rd · Sironko", Channel: "HoReCa", DistributorID: "D-007", BeatID: "B-21", QTDValueUGX: 2_200_000, Frequency: "1x/wk", Score: "B+", Status: "active"},
		{ID: "OUT-01408", Name: "Jinja Source Café", Address: "Source of the Nile · Jinja", Channel: "HoReCa", DistributorID: "D-022", BeatID: "B-24", QTDValueUGX: 3_100_000, Frequency: "2x/wk", Score: "A", Status: "active"},
		{ID: "OUT-01872", Name: "Gulu Roasters Lounge", Address: "Awere Rd · Gulu", Channel: "HoReCa", DistributorID: "D-029", BeatID: "B-26", QTDValueUGX: 1_800_000, Frequency: "1x/wk", Score: "B", Status: "watch"},
		{ID: "OUT-02214", Name: "Fort Western Trading", Address: "Kasese Rd · Fort Portal", Channel: "GT", DistributorID: "D-031", BeatID: "B-19", QTDValueUGX: 800_000, Frequency: "1x/wk", Score: "C", Status: "dormant"},
	}

	ts := time.Date(2026, 5, 2, 8, 0, 0, 0, time.UTC)
	m.orders = []models.Order{
		{ID: "SO-19852", OutletID: "OUT-00214", OutletName: "Cafe Javas Kololo", DistributorID: "D-001", RepID: "FF-04", Status: "draft", AmountUGX: 2_100_000, Currency: "UGX", CreatedAt: ts, UpdatedAt: ts},
		{ID: "SO-19848", OutletID: "OUT-00318", OutletName: "Shoprite Lugogo", DistributorID: "D-001", RepID: "FF-04", Status: "picking", AmountUGX: 4_500_000, Currency: "UGX", CreatedAt: ts, UpdatedAt: ts},
		{ID: "SO-19844", OutletID: "OUT-01408", OutletName: "Jinja Source Café", DistributorID: "D-022", RepID: "FF-09", Status: "delivery", AmountUGX: 3_100_000, Currency: "UGX", CreatedAt: ts, UpdatedAt: ts},
		{ID: "SO-19841", OutletID: "OUT-00482", OutletName: "Quality Lubowa", DistributorID: "D-001", Status: "submitted", AmountUGX: 2_800_000, Currency: "UGX", CreatedAt: ts, UpdatedAt: ts},
		{ID: "SO-19836", OutletID: "OUT-00821", Status: "delivered", AmountUGX: 980_000, Currency: "UGX", DistributorID: "D-001", CreatedAt: ts, UpdatedAt: ts},
	}

	m.beats = []models.Beat{
		{ID: "B-08", Name: "Kololo–Bukoto", RepID: "FF-04", RepName: "A. Achieng", StopCount: 14, DistanceKm: 22.4, Status: "in_progress", Progress: "stop 4 of 14", OutletIDs: []string{"OUT-00214", "OUT-00318"}},
		{ID: "B-02", Name: "Lugogo–Ntinda", RepID: "FF-02", RepName: "J. Sebunya", StopCount: 12, DistanceKm: 18.1, Status: "planned"},
		{ID: "B-21", Name: "Mbale HoReCa", RepID: "FF-09", RepName: "F. Wamala", StopCount: 10, DistanceKm: 45.2, Status: "active"},
	}

	m.reps = []models.FieldRep{
		{ID: "FF-04", Name: "A. Achieng", BeatID: "B-08", Region: "Central", Level: "Senior", Status: "clocked_in"},
		{ID: "FF-02", Name: "J. Sebunya", BeatID: "B-02", Region: "Central", Level: "Senior", Status: "active"},
		{ID: "FF-05", Name: "D. Kasozi", BeatID: "B-12", Region: "Central", Level: "HoReCa", Status: "active"},
		{ID: "FF-09", Name: "F. Wamala", BeatID: "B-21", Region: "Eastern", Level: "Senior", Status: "active"},
		{ID: "FF-13", Name: "P. Okello", BeatID: "B-26", Region: "Northern", Level: "Rep", Status: "idle"},
	}

	m.promos = []models.Promotion{
		{ID: "TPM-024", Name: "Bugisu AA BOGO", SKU: "BG-AA-250", ROI: 2.8, Status: "active", Outlets: 142},
		{ID: "TPM-019", Name: "HARAKA Instant bundle", SKU: "HK-IN-100", ROI: 1.9, Status: "active", Outlets: 88},
	}

	m.claims = []models.Claim{
		{ID: "CLM-1042", OutletID: "OUT-00318", Type: "damaged_cartons", Status: "open", AmountUGX: 420_000, CreatedAt: ts},
		{ID: "CLM-1038", OutletID: "OUT-01124", Type: "pricing_dispute", Status: "review", AmountUGX: 180_000, CreatedAt: ts},
	}

	m.dispatches = []models.Dispatch{
		{ID: "DXP-2814", TruckID: "TRK-12", Driver: "M. Okello", OrderIDs: []string{"SO-19848"}, Status: "out_for_delivery", ETA: "< 6h", UpdatedAt: ts},
		{ID: "DXP-2811", TruckID: "TRK-08", OrderIDs: []string{"SO-19844"}, Status: "loading", ETA: "14:30", UpdatedAt: ts},
	}

	m.skus = []models.SKU{
		{Code: "BG-AA-250", Name: "Bugisu AA Reserve · 250g", CoverDays: 8.2, QtyOnHand: 12400, Warehouse: "FG-KLA", SCAScore: 88.5},
		{Code: "BG-AB-500", Name: "Bugisu AB Washed · 500g", CoverDays: 6.1, QtyOnHand: 8200, Warehouse: "FG-KLA", SCAScore: 84},
		{Code: "SF-NT-250", Name: "Sipi Falls Natural · 250g", CoverDays: 5.4, QtyOnHand: 4100, Warehouse: "FG-MBL", SCAScore: 86},
		{Code: "WN-RB-500", Name: "West Nile Robusta · 500g", CoverDays: 4.2, QtyOnHand: 2900, Warehouse: "FG-ARU", SCAScore: 82.5},
		{Code: "HK-IN-100", Name: "HARAKA Instant · 100g", CoverDays: 3.1, QtyOnHand: 15600, Warehouse: "FG-KLA", SCAScore: 0},
	}

	m.stock = []models.StockPosition{
		{DistributorID: "D-001", SKU: "BG-AA-250", CoverDays: 2.1, Qty: 840, Status: "critical"},
		{DistributorID: "D-001", SKU: "BG-AB-500", CoverDays: 6.8, Qty: 2200, Status: "ok"},
		{DistributorID: "D-007", SKU: "HK-IN-100", CoverDays: 3.4, Qty: 1100, Status: "watch"},
	}

	m.invoices = []models.Invoice{
		{ID: "INV-2418", DistributorID: "D-001", Distributor: "Kampala Premium", AmountUGX: 18_400_000, DueDate: dueToday, Status: "due_today"},
		{ID: "INV-2421", DistributorID: "D-007", Distributor: "Mbale Coffee Hub", AmountUGX: 12_600_000, DueDate: dueSoon, Status: "open"},
		{ID: "INV-2417", DistributorID: "D-031", Distributor: "Fort Portal Bev", AmountUGX: 4_200_000, DueDate: time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC), Status: "overdue"},
	}

	m.pricing = []models.PricingTemplate{
		{ID: "PT-001", Name: "Distrib T1", Channel: "distributor", Version: "v3.4", Currency: "UGX"},
		{ID: "PT-002", Name: "Distrib T2", Channel: "distributor", Version: "v3.4", Currency: "UGX"},
		{ID: "PT-003", Name: "HoReCa", Channel: "horeca", Version: "v3.4", Currency: "UGX"},
		{ID: "PT-004", Name: "Export USD", Channel: "export", Version: "v3.2", Currency: "USD"},
	}

	m.reports = []models.ReportTemplate{
		{ID: "RPT-001", Name: "Weekly Sell-Out by Region", DataSource: "sell_through", Schedule: "Weekly Mon 07:30"},
		{ID: "RPT-002", Name: "Outlet Productivity", DataSource: "outlet_master", Schedule: "Monthly 1st"},
		{ID: "RPT-003", Name: "Field Activity Summary", DataSource: "field_activity", Schedule: "Daily 06:00"},
	}

	m.execution = []models.ExecutionTask{
		{ID: "EXE-1001", OutletID: "OUT-00214", Type: "planogram", Status: "active", Detail: "9 photos · 2 done"},
		{ID: "EXE-1002", OutletID: "OUT-00318", Type: "osa", Status: "pending", Detail: "Must-stock 8 SKUs"},
	}

	m.alerts = []models.Alert{
		{ID: "ALT-1", Kind: "stock", Title: "Stock-out risk · Bugisu AA", Detail: "D-001 Kampala · 2.1d cover"},
		{ID: "ALT-2", Kind: "invoice", Title: "INV-2418 due today", Detail: "Kampala Premium · UGX 18.4M"},
		{ID: "ALT-3", Kind: "field", Title: "FF-15 idle 34 min", Detail: "D. Anyama · Arua"},
		{ID: "ALT-4", Kind: "promo", Title: "TPM-024 ROI tracking 2.8x", Detail: "Bugisu AA BOGO"},
	}

	m.checkIns = []models.CheckIn{
		{ID: "CI-9001", RepID: "FF-04", OutletID: "OUT-00214", Lat: 0.34288, Lng: 32.60241, ArrivedAt: time.Date(2026, 5, 26, 7, 38, 0, 0, time.UTC), Status: "active"},
	}

	m.signals = defaultDMSSignals()
}
