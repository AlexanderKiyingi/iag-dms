/**
 * DMS API client for Next.js (App Router / RSC / client components).
 * Always call through the gateway — never the service port in the browser.
 *
 *   NEXT_PUBLIC_DMS_API_URL=http://localhost:8080/api/v1/dms/v1
 *
 * List endpoints return `{ items, meta }` (not CRM's `{ data, meta }`).
 * Obtain a user token from `POST /api/v1/authentication/oauth/token`; the
 * token `aud` must include `iag.gateway` (gateway) and `iag.dms` (this service).
 */

export type Paginated<T> = {
  items: T[];
  meta: { total: number; limit: number; offset: number };
};

export type DmsFetchOptions = RequestInit & {
  token?: string;
};

function baseUrl(): string {
  const url =
    process.env.NEXT_PUBLIC_DMS_API_URL ?? "http://localhost:8080/api/v1/dms/v1";
  return url.replace(/\/$/, "");
}

export async function dmsFetch<T>(path: string, opts: DmsFetchOptions = {}): Promise<T> {
  const { token, headers, ...rest } = opts;
  const res = await fetch(`${baseUrl()}${path.startsWith("/") ? path : `/${path}`}`, {
    ...rest,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error(`DMS ${res.status}: ${await res.text()}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

function qs(params: Record<string, string | number | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") q.set(k, String(v));
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

function listResource<T>(path: string) {
  return (token?: string, params?: Record<string, string | number | undefined>) =>
    dmsFetch<Paginated<T>>(`${path}${qs(params ?? {})}`, { token });
}

function createResource<T>(path: string) {
  return (body: unknown, token?: string) =>
    dmsFetch<T>(path, { method: "POST", body: JSON.stringify(body), token });
}

export const dmsApi = {
  bootstrap: (token?: string) => dmsFetch("/bootstrap", { token }),
  session: (token?: string) => dmsFetch("/auth/session", { token }),
  permissionsCatalog: (token?: string) => dmsFetch("/permissions/catalog", { token }),
  permissionsBuiltin: (token?: string) => dmsFetch("/permissions/builtin", { token }),
  permissionsCheck: (keys: string[], token?: string) =>
    dmsFetch("/permissions/check", { method: "POST", body: JSON.stringify({ keys }), token }),
  permissionsMe: (token?: string) => dmsFetch("/permissions/me", { token }),
  lookups: (kind: string, token?: string) =>
    dmsFetch(`/lookups/${encodeURIComponent(kind)}`, { token }),
  search: (q: string, token?: string, limit = 18) =>
    dmsFetch(`/search${qs({ q, limit })}`, { token }),
  notifications: (token?: string) => dmsFetch("/notifications", { token }),
  platformStatus: (token?: string) => dmsFetch("/platform/status", { token }),

  overview: (token?: string) => dmsFetch("/overview", { token }),

  distributors: listResource<unknown>("/distributors"),
  getDistributor: (id: string, token?: string) => dmsFetch(`/distributors/${id}`, { token }),

  outletStats: (token?: string) => dmsFetch("/outlets/stats", { token }),
  outlets: listResource<unknown>("/outlets"),
  createOutlet: createResource<unknown>("/outlets"),
  getOutlet: (id: string, token?: string) => dmsFetch(`/outlets/${id}`, { token }),
  patchOutlet: (id: string, body: unknown, token?: string) =>
    dmsFetch(`/outlets/${id}`, { method: "PATCH", body: JSON.stringify(body), token }),

  ordersStats: (token?: string) => dmsFetch("/orders/stats", { token }),
  ordersBoard: (token?: string) => dmsFetch("/orders/board", { token }),
  orders: listResource<unknown>("/orders"),
  createOrder: createResource<unknown>("/orders"),
  getOrder: (id: string, token?: string) => dmsFetch(`/orders/${id}`, { token }),
  patchOrderStatus: (id: string, status: string, token?: string) =>
    dmsFetch(`/orders/${id}/status`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
      token,
    }),

  routesStats: (token?: string) => dmsFetch("/routes/stats", { token }),
  beats: listResource<unknown>("/beats"),
  getBeat: (id: string, token?: string) => dmsFetch(`/beats/${id}`, { token }),

  reps: (token?: string) => dmsFetch("/field/reps", { token }),
  checkInStats: (token?: string) => dmsFetch("/field/check-ins/stats", { token }),
  checkIns: listResource<unknown>("/field/check-ins"),
  createCheckIn: createResource<unknown>("/field/check-ins"),
  completeCheckIn: (id: string, token?: string) =>
    dmsFetch(`/field/check-ins/${id}`, { method: "PATCH", body: "{}", token }),
  visitReports: listResource<unknown>("/field/visit-reports"),
  createVisitReport: createResource<unknown>("/field/visit-reports"),
  journey: (token?: string) => dmsFetch("/field/journey", { token }),
  execution: (token?: string) => dmsFetch("/field/execution", { token }),

  promotions: listResource<unknown>("/promotions"),
  createPromotion: createResource<unknown>("/promotions"),
  claims: listResource<unknown>("/claims"),
  createClaim: createResource<unknown>("/claims"),
  dispatch: listResource<unknown>("/dispatch"),
  createDispatch: createResource<unknown>("/dispatch"),

  stock: (token?: string) => dmsFetch("/stock/distributor", { token }),
  skus: (token?: string) => dmsFetch("/stock/skus", { token }),

  financeSummary: (token?: string) => dmsFetch("/finance/summary", { token }),
  invoices: listResource<unknown>("/invoices"),
  getInvoice: (id: string, token?: string) => dmsFetch(`/invoices/${id}`, { token }),
  createInvoice: createResource<unknown>("/invoices"),

  pricingTemplates: (token?: string) => dmsFetch("/pricing/templates", { token }),
  reportTemplates: (token?: string) => dmsFetch("/reports/templates", { token }),
  runReport: (body: unknown, token?: string) =>
    dmsFetch("/reports/run", { method: "POST", body: JSON.stringify(body), token }),
  exportPage: (page: string, body: unknown = {}, token?: string) =>
    dmsFetch(`/exports/${encodeURIComponent(page)}`, {
      method: "POST",
      body: JSON.stringify(body),
      token,
    }),
  kpiBoard: (token?: string) => dmsFetch("/kpi/board", { token }),
  analytics: (token?: string) => dmsFetch("/analytics/summary", { token }),
  forecast: (token?: string) => dmsFetch("/forecast", { token }),

  insightsSignals: (token?: string) => dmsFetch("/insights/signals", { token }),

  audit: (token?: string) => dmsFetch("/audit", { token }),
  adminMonitoringSummary: (token?: string) => dmsFetch("/admin/monitoring/summary", { token }),
  adminAuditLogs: (token?: string) => dmsFetch("/admin/audit-logs", { token }),
  adminMonitoringActivity: (token?: string) => dmsFetch("/admin/monitoring/activity", { token }),
};

/** Map sidebar page ids (index.html) to primary data loaders for Next.js routes. */
export const DMS_PAGE_LOADERS = {
  overview: (token?: string) => dmsApi.overview(token),
  network: (token?: string) => dmsApi.distributors(token),
  outlets: (token?: string) => dmsApi.outlets(token),
  orders: (token?: string) => Promise.all([dmsApi.ordersBoard(token), dmsApi.orders(token)]),
  routes: (token?: string) => Promise.all([dmsApi.routesStats(token), dmsApi.beats(token)]),
  checkin: (token?: string) => dmsApi.checkIns(token),
  journey: (token?: string) => dmsApi.journey(token),
  field: (token?: string) => dmsApi.reps(token),
  execution: (token?: string) => dmsApi.execution(token),
  promo: (token?: string) => dmsApi.promotions(token),
  claims: (token?: string) => dmsApi.claims(token),
  dispatch: (token?: string) => dmsApi.dispatch(token),
  stock: (token?: string) => dmsApi.stock(token),
  stockwh: (token?: string) => dmsApi.skus(token),
  finance: (token?: string) => dmsApi.financeSummary(token),
  invoices: (token?: string) => dmsApi.invoices(token),
  pricing: (token?: string) => dmsApi.pricingTemplates(token),
  reports: (token?: string) => dmsApi.reportTemplates(token),
  kpi: (token?: string) => dmsApi.kpiBoard(token),
  analytics: (token?: string) => dmsApi.analytics(token),
  forecast: (token?: string) => dmsApi.forecast(token),
} as const;
