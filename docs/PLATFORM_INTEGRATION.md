# IAG DMS — platform integration

Distribution Management Service runs behind the **API gateway** with **iag-authentication** IAM. Default compose port **4010**.

## Gateway routes

| Public path | Upstream |
|-------------|----------|
| `/api/v1/dms/v1/*` | DMS `:4010/v1/*` |
| `/api/v1/dms/health` | DMS `:4010/health` |
| `/api/v1/dms/` | DMS UI (`index.html`) |

Every API call must include `Authorization: Bearer <token>` with audience **`iag.dms`**. There is no `AUTH_MODE`, gateway trust header, or dev bypass in production.

## Next.js

Frontend developer guide: [`FRONTEND_INTEGRATION.md`](FRONTEND_INTEGRATION.md) (auth, response envelopes, RBAC gating, full endpoint reference, mutation→event map). Clients: [`dms-api.ts`](dms-api.ts) (typed fetch) and [`dms-react-query.ts`](dms-react-query.ts) (hooks). Browser calls go to `http://localhost:8080/api/v1/dms/v1`, not `:4010`.

## Environment

| Variable | Purpose |
|----------|---------|
| `GATEWAY_API_PREFIX` | `/api/v1/dms` |
| `PUBLIC_API_URL` | Gateway origin, e.g. `http://localhost:8080` |
| `DATABASE_URL` | `postgres://svc_iag_dms:…@pgbouncer:6432/iag_platform` |
| `AUDIENCE` | `iag.dms` |
| `SERVICE_CLIENT_ID` | `iag-dms` |
| `SERVICE_CLIENT_SECRET` | Registers `dms.*` permissions at startup (≥16 chars in prod) |
| `FINANCE_URL` | Optional finance upstream, e.g. `http://finance:3006` |
| `EVENT_BUS_ENABLED` | `true` + `KAFKA_BROKERS` for `iag.operations` events (outbox) |
| `CONSUMER_ENABLED` | `true` + `KAFKA_BROKERS` to consume `iag.commercial` CRM events |
| `CONSUMER_TOPIC` | Default `iag.commercial` |
| `CONSUMER_GROUP_ID` | Default `iag-dms` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP gRPC (default `otel-collector:4317`) |

### Production invariants

`ENVIRONMENT=production` requires non-wildcard `ALLOWED_ORIGINS`, `SERVICE_CLIENT_SECRET` (≥16 chars), `AUTO_MIGRATE=false`, `SEED_ON_EMPTY=false`, and Postgres (no memory store). RBAC is fail-closed when JWT `permissions` is empty. See [`config/.env.production.example`](../config/.env.production.example) and [`docs/PRODUCTION_CHECKLIST.md`](PRODUCTION_CHECKLIST.md).

## Platform API (RBAC, audit, admin)

| Route | Permission / gate |
|-------|-------------------|
| `GET /v1/bootstrap`, `/auth/session`, `/permissions/*`, `/lookups/:kind` | `dms.view_overview` |
| `GET /v1/insights/signals` | `dms.insights.read` |
| `GET/POST /v1/audit`, `GET /v1/audit/:id` | `dms.audit.read` / `dms.audit.create` |
| `GET /v1/admin/audit-logs`, `/admin/monitoring/*` | staff + `dms.admin.read` |
| `GET /v1/platform/status` | staff |

Paginated lists return `{ items, data, meta }` for CRM/Next.js compatibility.

## Events

| Topic | Direction | Types |
|-------|-----------|-------|
| `iag.operations` | DMS → platform | `dms.outlet.created`, `dms.outlet.updated`, `dms.order.created`, `dms.order.status_changed`, `dms.checkin.created`, `dms.checkin.completed`, `dms.visit.reported`, `dms.promotion.created`, `dms.claim.created`, `dms.invoice.created`, `dms.dispatch.created` |
| `iag.commercial` | CRM → DMS | `crm.deal.won`, `crm.lead.converted`, `crm.outlet.synced` (signals + `crm_ref`) |

Outbound events use a transactional **outbox** (`dms_event_outbox`) when Postgres is enabled.

## Local stack

```bash
pnpm infra:up
curl http://localhost:8080/api/v1/dms/v1/bootstrap
```
