# IAG DMS — platform integration

Distribution Management Service runs behind the **API gateway** with **iag-authentication** IAM. Default compose port **4010** (procurement uses 4009).

## Gateway routes

| Public path | Upstream |
|-------------|----------|
| `/api/v1/dms/v1/*` | DMS `:4010/v1/*` |
| `/api/v1/dms/health` | DMS `:4010/health` |
| `/api/v1/dms/` | DMS UI (`index.html`) |

Example:

```bash
curl http://localhost:8080/api/v1/dms/v1/overview
curl http://localhost:8080/api/v1/dms/ready
curl -X POST http://localhost:8080/api/v1/dms/v1/reports/run -H "Authorization: Bearer $TOKEN" -d '{"name":"Daily Sales Recap"}'
curl -X POST http://localhost:8080/api/v1/dms/v1/exports/outlets -H "Authorization: Bearer $TOKEN" -d '{"format":"json"}'
```

Write endpoints (gateway permissions): `dms.manage_outlets` (PATCH outlets), `dms.manage_orders`, `dms.manage_invoices`, `dms.field_checkin` (check-in complete), `dms.manage_claims`, `dms.manage_promotions`, `dms.manage_dispatch`, `dms.run_reports` (report run + export).

## Auth modes

| Mode | Use |
|------|-----|
| `jwt` | **Default in Compose** — verify forwarded `Authorization: Bearer` (audience `iag.dms`) after gateway checks `iag.gateway` |
| `gateway` | Legacy only (pre cutover); requires `X-IAG-*` headers the gateway no longer injects |
| `none` | UI-only dev with `STORE_MODE=memory` |

## Next.js

See [`README.md`](../README.md) and [`docs/dms-api.ts`](dms-api.ts). Browser calls go to `http://localhost:8080/api/v1/dms/v1`, not `:4010`.

## Environment

| Variable | Purpose |
|----------|---------|
| `AUTH_MODE` | `jwt` (Compose), `gateway` (legacy), or `none` |
| `GATEWAY_INTERNAL_SECRET` | Legacy `gateway` mode only |
| `GATEWAY_API_PREFIX` | `/api/v1/dms` |
| `PUBLIC_API_URL` | Gateway origin, e.g. `http://localhost:8080` |
| `DATABASE_URL` | `postgres://svc_iag_dms:…@pgbouncer:6432/iag_platform` |
| `AUDIENCE` | `iag.dms` (JWT mode) |
| `SERVICE_CLIENT_ID` | `iag-dms` |
| `SERVICE_CLIENT_SECRET` | Registers `dms.*` permissions at startup |
| `EVENT_BUS_ENABLED` | `true` + `KAFKA_BROKERS` for `iag.operations` events |
| `CONSUMER_ENABLED` | `true` + `KAFKA_BROKERS` to consume `iag.commercial` (CRM bridge events) |
| `CONSUMER_TOPIC` | Default `iag.commercial` |
| `CONSUMER_GROUP_ID` | Default `iag-dms` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP gRPC (default `otel-collector:4317`) |

### Production invariants

`ENVIRONMENT=production` requires `AUTH_MODE=jwt`, non-wildcard `ALLOWED_ORIGINS`, `SERVICE_CLIENT_SECRET` (≥16 chars), `AUTO_MIGRATE=false`, `SEED_ON_EMPTY=false`, and Postgres (no memory store). RBAC is fail-closed when JWT `permissions` is empty. See [`config/.env.production.example`](../config/.env.production.example).

## Platform API (RBAC, audit, admin)

| Route | Permission / gate |
|-------|-------------------|
| `GET /v1/bootstrap`, `/auth/session`, `/permissions/*`, `/lookups/:kind` | `dms.view_overview` |
| `GET /v1/insights/signals` | `dms.insights.read` |
| `GET/POST /v1/audit`, `GET /v1/audit/:id` | `dms.audit.read` / `dms.audit.create` |
| `GET /v1/admin/audit-logs`, `/admin/monitoring/*` | staff + `dms.admin.read` |
| `GET /v1/platform/status` | staff |

Handler-level `RequirePerm` mirrors gateway policies. Business mutations also append to `dms_audit_entries`; all `/v1` calls log to `dms_api_audit`.

## Events

| Topic | Types |
|-------|-------|
| `iag.operations` | `dms.outlet.created`, `dms.order.created`, `dms.checkin.created`, `dms.visit.reported`, `dms.invoice.created` |

## Local stack

```bash
pnpm infra:up
curl http://localhost:8080/api/v1/dms/v1/bootstrap
```
