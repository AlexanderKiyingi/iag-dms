# iag-dms

Distribution Management Service (**IAG SAFARI** UI in `index.html`) — domain microservice on the IAG platform.

## Stack

- **Gin** REST API under `/v1`
- **Postgres** persistence (migrations in `db/migrations/`) or **in-memory** store (`STORE_MODE=memory`)
- **Platform JWT** via `shared/platform-go` when `AUTH_MODE=jwt`
- **Kafka** events on `iag.operations` when `EVENT_BUS_ENABLED=true`
- Static UI + `assets/dms-api.js` client wired to create outlet, check-in, visit report, invoice

## Run locally

```bash
cd services/operations/dms
go mod tidy
go run .
```

Open [http://localhost:4010/](http://localhost:4010/) (direct) or [http://localhost:8080/api/v1/dms/](http://localhost:8080/api/v1/dms/) via gateway.

### With Postgres

```bash
# example
set DATABASE_URL=postgres://user:pass@localhost:5432/iag_dms?sslmode=disable
set STORE_MODE=
go run .
```

## API map

| Page | Endpoints |
|------|-----------|
| `overview` | `GET /v1/overview` |
| `network` | `GET /v1/distributors` |
| `outlets` | `GET/POST /v1/outlets` |
| `orders` | `GET /v1/orders/board`, `PATCH /v1/orders/:id/status` |
| `routes` | `GET /v1/beats` |
| `checkin` | `POST /v1/field/check-ins`, `POST /v1/field/visit-reports` |
| `finance` | `GET /v1/finance/summary`, `GET/POST /v1/invoices` |
| Search | `GET /v1/search?q=` |
| Writes | `PATCH /v1/outlets/:id`, `POST /v1/orders`, `PATCH /v1/orders/:id/status`, `PATCH /v1/field/check-ins/:id` (complete), `POST /v1/claims`, `POST /v1/promotions`, `POST /v1/dispatch`, `POST /v1/reports/run`, `POST /v1/exports/:page` |
| Detail | `GET /v1/invoices/:id`, `GET /v1/field/visit-reports` |

Health: `GET /healthz`, `GET /ready` (Postgres ping when DB enabled).

Platform wiring: [`docs/PLATFORM_INTEGRATION.md`](docs/PLATFORM_INTEGRATION.md)

## Next.js integration

```bash
# .env.local
NEXT_PUBLIC_DMS_API_URL=http://localhost:8080/api/v1/dms/v1
```

Obtain a user token from `POST /api/v1/authentication/oauth/token` (audiences must include `iag.gateway` and `iag.dms`), then:

```ts
import { dmsApi } from "./docs/dms-api";

const boot = await dmsApi.bootstrap(accessToken);
const outlets = await dmsApi.outlets(accessToken, { limit: 20 });
```

- `GET /v1/bootstrap` returns `session`, RBAC `roles`, `permissions`, `pages`, and `page_titles`.
- Platform: `/auth/session`, `/permissions/*`, `/lookups/:kind`, `/insights/signals`, `/audit`, `/admin/monitoring/*`.
- Set `CONSUMER_ENABLED=true` to ingest `iag.commercial` events (audit trail).
- Paginated lists use `{ items, meta }` — see [`docs/dms-api.ts`](docs/dms-api.ts) (copy into your Next app or import via path alias).
- **Server Components / Route Handlers** can call the same URL without CORS; **client components** rely on gateway proxy + DMS `ALLOWED_ORIGINS` (default includes `http://localhost:3000`).
- Gateway enforces auth and write permissions (`dms.manage_outlets`, etc.); compose uses `AUTH_MODE=jwt` so the forwarded Bearer is verified with audience `iag.dms`.

## Production

Use [`config/.env.production.example`](config/.env.production.example): `AUTH_MODE=jwt`, `AUTO_MIGRATE=false`, `SEED_ON_EMPTY=false`, explicit `ALLOWED_ORIGINS` (no `*`), and `SERVICE_CLIENT_SECRET` from your secret store. Run DB migrations out of band. Traces export via `OTEL_EXPORTER_OTLP_ENDPOINT` (see compose `otel-collector:4317`). In production, RBAC is fail-closed when JWT `permissions` is empty.

Registry: [`subrepos.json`](../../../subrepos.json) · Dev port: **4010** (gateway: `/api/v1/dms`)
