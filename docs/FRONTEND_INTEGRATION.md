# IAG DMS — frontend integration guide

How a web frontend (Next.js App Router, or any SPA) talks to the **Distribution
Management Service**. For platform/ops wiring (gateway routes, env vars, events
infra) see [`PLATFORM_INTEGRATION.md`](PLATFORM_INTEGRATION.md). For the ready-made
clients see [`dms-api.ts`](dms-api.ts) (typed fetch) and
[`dms-react-query.ts`](dms-react-query.ts) (hooks).

---

## 1. Base URL — always go through the gateway

The browser must call the **API gateway**, never the service port (`:4010`) directly.

```bash
# .env.local
NEXT_PUBLIC_DMS_API_URL=http://localhost:8080/api/v1/dms/v1
```

| Environment | `NEXT_PUBLIC_DMS_API_URL` |
|-------------|---------------------------|
| Local       | `http://localhost:8080/api/v1/dms/v1` |
| Production  | `https://iag-api-gateway-production.up.railway.app/api/v1/dms/v1` |

The gateway strips `/api/v1/dms` and forwards `/v1/*` to the service. Every path in
this guide is written relative to the base URL above (i.e. `/overview` →
`…/api/v1/dms/v1/overview`).

---

## 2. Authentication

Every request must carry a platform JWT:

```
Authorization: Bearer <access_token>
```

The token's `aud` claim **must include both** `iag.gateway` (so the gateway accepts
it) and `iag.dms` (so the service accepts it). There is no gateway-trust header, no
`AUTH_MODE`, and no dev bypass — a missing or invalid token is always `401`.

### Obtaining a token

```ts
const res = await fetch(`${PUBLIC_API}/api/v1/authentication/oauth/token`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    grant_type: "password",
    username,
    password,
    audience: ["iag.gateway", "iag.dms"],
  }),
});
const { access_token, refresh_token, expires_in } = await res.json();
```

Store the access token in memory (or an httpOnly cookie set by your Next.js route
handler) and refresh it before `expires_in` elapses. The DMS client reads the token
per-call, so wherever you keep it, pass it into `dmsFetch`/the hooks.

### Public (no-token) paths

`/`, `/health`, `/healthz`, `/ready` are open (service descriptor and probes).
Everything under `/v1/*` requires a token.

---

## 3. Bootstrapping the UI

Call **`GET /bootstrap`** once after login. It returns everything needed to render a
permission-aware shell in a single round-trip:

```jsonc
{
  "service": "dms",
  "version": "0.1.0",
  "api_prefix": "/api/v1/dms/v1",
  "public_api": "http://localhost:8080",
  "session": {
    "email": "rep@iag.local",
    "name": "Jane Rep",
    "role": "field_rep",
    "role_label": "Field Rep",
    "role_full": "Field Sales Representative",
    "pages": ["overview", "outlets", "checkin", "journey"],
    "modals": ["createCheckIn", "createVisitReport"],
    "permissions": ["dms.view_overview", "dms.field_checkin", "..."]
  },
  "permissions": {
    "role": "field_rep",
    "email": "rep@iag.local",
    "name": "Jane Rep",
    "permissions": ["dms.view_overview", "..."],
    "canMutate": true,
    "canManageAdmin": false,
    "isStaff": false
  },
  "pages": [{ "id": "overview", "title": "Distribution Tower" }, "..."],
  "page_titles": { "overview": "Distribution Tower", "...": "..." },
  "roles": { "field_rep": { "label": "Field Rep", "pages": ["..."] } },
  "sync_status": "connected",
  "modules": ["distribution", "field", "logistics", "finance", "intelligence"]
}
```

Use `session.pages` to drive the sidebar, and `permissions` (the
`PermissionContext`) to gate buttons/modals. Lighter alternatives:

| Endpoint | Returns |
|----------|---------|
| `GET /auth/session` | just the `session` block above |
| `GET /permissions/me` | just the `PermissionContext` (`canMutate`, `isStaff`, …) |
| `GET /permissions/catalog` | the full `dms.*` catalogue (for an admin role editor) |
| `POST /permissions/check` `{ "keys": ["dms.manage_orders"] }` | `{ "allowed": { "dms.manage_orders": true } }` |

---

## 4. Response conventions

### List envelope

Every paginated list returns the **same shape** (both `items` and `data` are
populated — `items` is canonical, `data` is for CRM/legacy compatibility):

```jsonc
{
  "items": [ /* rows */ ],
  "data":  [ /* same rows */ ],
  "meta": { "total": 184, "limit": 50, "offset": 0 }
}
```

Query params accepted by list endpoints:

| Param | Meaning | Default |
|-------|---------|---------|
| `limit` | page size | `50` |
| `offset` | rows to skip | `0` |
| `q` | free-text search | — |
| `status` | status filter (orders, outlets, claims, …) | — |
| `channel` | channel filter (outlets) | — |
| `distributorId` | scope to a distributor | — |
| `repId` | scope to a field rep | — |
| `beatId` | scope to a beat | — |

### Error envelope

All errors share one shape (top-level detail fields may be merged in):

```jsonc
{ "error": { "code": "FORBIDDEN", "message": "permission denied: dms.manage_orders" },
  "required_permission": "dms.manage_orders" }
```

| HTTP | `code` | When |
|------|--------|------|
| 400 | `BAD_REQUEST` / `VALIDATION_ERROR` | malformed body / missing required field |
| 401 | `UNAUTHORIZED` | missing/invalid/expired token |
| 403 | `FORBIDDEN` | authenticated but lacks the permission (`required_permission` tells you which) |
| 404 | `NOT_FOUND` | unknown id |
| 409 | `CONFLICT` | upstream write conflict |
| 500 | `INTERNAL` | server error |
| 503 | `SERVICE_UNAVAILABLE` | auth verifier not ready (JWKS still loading on boot) |

A `503` on boot is transient — retry with backoff. The `dmsFetch` helper throws
`Error("DMS <status>: <body>")`; parse `error.code`/`required_permission` to drive
toasts and disabled-control states.

### Response headers

`ETag` and `X-Request-ID` are exposed via CORS — log `X-Request-ID` with client
errors so backend traces can be correlated.

---

## 5. RBAC — gating the UI

Permission checks are enforced server-side per route; mirror them client-side only
to hide/disable controls (never as the security boundary). Rules:

- **Superuser / staff** bypass all permission checks.
- A user with permission `*` passes any check.
- In **production**, an empty `permissions` list **fails closed** (no access). In
  dev it's permissive, so don't rely on local behavior for gating logic — read
  `permissions.canMutate` / explicit `dms.*` keys instead.

### Permission catalogue

| Permission | Grants |
|------------|--------|
| `dms.view_overview` | overview, network, routes, search, lookups, notifications, promotions/claims (read) |
| `dms.view_outlets` / `dms.manage_outlets` | outlets read / create+patch |
| `dms.view_orders` / `dms.manage_orders` | orders + dispatch read / order create+status |
| `dms.field_checkin` | field reps, check-ins, visit reports, journey, execution |
| `dms.manage_promotions` | create promotions |
| `dms.manage_claims` | create claims |
| `dms.manage_dispatch` | create dispatch |
| `dms.view_finance` / `dms.manage_invoices` | finance + invoices read / create invoice |
| `dms.run_reports` | report templates, run report, exports |
| `dms.insights.read` | KPI board, analytics, forecast, insights signals |
| `dms.audit.read` / `dms.audit.create` | audit trail read / append |
| `dms.admin.read` / `dms.admin.update` | admin monitoring + audit-logs (also requires **staff**) |

---

## 6. Endpoint reference (by page)

`R` = read permission, `W` = write permission. Paths are relative to the base URL.

| Page | Read | Write |
|------|------|-------|
| **overview** | `GET /overview` — `dms.view_overview` | — |
| **network** | `GET /distributors`, `GET /distributors/:id` — `dms.view_overview` | — |
| **outlets** | `GET /outlets`, `/outlets/:id`, `/outlets/stats` — `dms.view_outlets` | `POST /outlets`, `PATCH /outlets/:id` — `dms.manage_outlets` |
| **orders** | `GET /orders`, `/orders/:id`, `/orders/board`, `/orders/stats` — `dms.view_orders` | `POST /orders`, `PATCH /orders/:id/status` — `dms.manage_orders` |
| **routes** | `GET /beats`, `/beats/:id`, `/routes/stats` — `dms.view_overview` | — |
| **checkin / field** | `GET /field/reps`, `/field/check-ins`, `/field/check-ins/stats`, `/field/visit-reports`, `/field/journey`, `/field/execution` — `dms.field_checkin` | `POST /field/check-ins`, `PATCH /field/check-ins/:id`, `POST /field/visit-reports` — `dms.field_checkin` |
| **promo** | `GET /promotions` — `dms.view_overview` | `POST /promotions` — `dms.manage_promotions` |
| **claims** | `GET /claims` — `dms.view_overview` | `POST /claims` — `dms.manage_claims` |
| **dispatch** | `GET /dispatch` — `dms.view_orders` | `POST /dispatch` — `dms.manage_dispatch` |
| **stock** | `GET /stock/distributor`, `/stock/skus` — `dms.view_overview` | — |
| **finance / invoices** | `GET /finance/summary`, `/invoices`, `/invoices/:id`, `/pricing/templates` — `dms.view_finance` | `POST /invoices` — `dms.manage_invoices` |
| **reports** | `GET /reports/templates` — `dms.run_reports` | `POST /reports/run`, `POST /exports/:page` — `dms.run_reports` |
| **kpi / analytics / forecast** | `GET /kpi/board`, `/analytics/summary`, `/forecast`, `/insights/signals` — `dms.insights.read` | — |
| **audit** | `GET /audit`, `/audit/:id` — `dms.audit.read` | `POST /audit` — `dms.audit.create` |
| **admin** | `GET /admin/audit-logs`, `/admin/monitoring/summary`, `/admin/monitoring/activity` — staff + `dms.admin.read` | — |
| **platform** | `GET /platform/status` — staff | — |
| **search** | `GET /search?q=&limit=` — `dms.view_overview` | — |

### Write payloads (minimum required fields)

```jsonc
// POST /outlets            { "name": "...", "channel": "CH-GT", "address": "..." }
// PATCH /outlets/:id       { "status": "active" }          // ≥1 field required
// POST /orders             { "outletId": "OUT-00318", "amountUgx": 1250000 }
// PATCH /orders/:id/status { "status": "delivery" }
// POST /field/check-ins    { "repId": "FF-04", "outletId": "OUT-00318" }
// POST /field/visit-reports{ "repId": "FF-04", "outletId": "OUT-00318", "outcome": "stocked" }
// POST /promotions         { "name": "Q3 GT push" }
// POST /claims             { "outletId": "OUT-00318", "type": "return" }
// POST /dispatch           { "truckId": "TRK-12", "orderIds": ["ORD-19854"] }
// POST /invoices           { "distributorId": "D-001", "amountUgx": 142000000, "dueDate": "2026-07-01T00:00:00Z" }
// POST /reports/run        { "templateId": "RT-01" } | { "name": "Custom", "emailTo": "..." }
// POST /exports/:page      { "format": "json" }
```

> **Invoices:** when the service is wired to `iag-finance` (`FINANCE_URL` set), reads
> (`GET /invoices`, `/invoices/:id`, `/finance/summary`) and `POST /invoices`
> round-trip to the finance ledger, and the returned `id` is the finance-assigned
> invoice number. Re-fetch the list after a create rather than assuming a local id.

> **Exports:** `POST /exports/:page` always returns JSON rows
> (`{ page, format, generatedAt, rowCount, rows }`) regardless of the requested
> `format`. Convert to CSV/XLSX client-side if needed.

---

## 7. Mutations → events (cache invalidation hints)

When the event bus is enabled, writes publish to `iag.operations`. The frontend
can't subscribe directly, but these are the signals to **refetch** after a mutation
(and what other services react to):

| Write | Event | Refetch |
|-------|-------|---------|
| `POST /outlets` | `dms.outlet.created` | outlets list, outlet stats |
| `PATCH /outlets/:id` | `dms.outlet.updated` | the outlet, outlets list |
| `POST /orders` | `dms.order.created` | orders board + list + stats |
| `PATCH /orders/:id/status` | `dms.order.status_changed` | orders board, the order |
| `POST /field/check-ins` | `dms.checkin.created` | check-ins list + stats |
| `PATCH /field/check-ins/:id` | `dms.checkin.completed` | check-ins, journey |
| `POST /field/visit-reports` | `dms.visit.reported` | visit reports, execution |
| `POST /promotions` | `dms.promotion.created` | promotions |
| `POST /claims` | `dms.claim.created` | claims |
| `POST /dispatch` | `dms.dispatch.created` | dispatch, orders board |
| `POST /invoices` | `dms.invoice.created` | invoices, finance summary |

Inbound: warehouse `warehouse.pick.confirmed` flips an order to `delivery`, and CRM
`crm.deal.won` / `crm.lead.converted` seed outlets/signals — so a stale orders board
or outlet list may change without a local mutation. Poll `/insights/signals` or
refetch on focus for near-real-time UIs.

---

## 8. Using the clients

### Typed fetch ([`dms-api.ts`](dms-api.ts))

```ts
import { dmsApi } from "@/lib/dms-api";

const overview = await dmsApi.overview(token);
const { items, meta } = await dmsApi.outlets(token, { q: "kampala", limit: 25 });
await dmsApi.createOrder({ outletId: "OUT-00318", amountUgx: 1_250_000 }, token);
await dmsApi.patchOrderStatus("ORD-19854", "delivery", token);
```

### React Query ([`dms-react-query.ts`](dms-react-query.ts))

```tsx
import { useOutlets, useCreateOrder } from "@/lib/dms-react-query";

function Outlets({ token }: { token: string }) {
  const { data, isLoading } = useOutlets(token, { q: "kampala" });
  const createOrder = useCreateOrder(token); // auto-invalidates orders on success
  // ...
}
```

Wrap your tree in a `QueryClientProvider` and pass the access token down (context or
prop). The hooks key queries by resource + params and invalidate the right lists on
mutation, matching the event table above.

---

## 9. Local quickstart

```bash
pnpm infra:up                                   # gateway + auth + postgres + kafka
curl -s localhost:8080/api/v1/dms/v1/bootstrap \
  -H "Authorization: Bearer $TOKEN" | jq .
```

No backend? Run DMS standalone with seeded in-memory data:

```bash
cd services/operations/dms
STORE_MODE=memory go run .          # http://localhost:4010
```

(Memory mode is dev-only — RBAC is permissive and data is not persisted.)
