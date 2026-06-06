# iag-dms

Distribution Management Service (**IAG SAFARI** UI in `index.html`) — domain microservice on the IAG platform.

## Stack

- **Gin** REST API under `/v1`
- **Postgres** persistence (migrations in `db/migrations/`) or **in-memory** store (`STORE_MODE=memory`)
- **Platform JWT** — every request requires Bearer token with `aud=iag.dms`
- **Kafka** events on `iag.operations` (outbox) when `EVENT_BUS_ENABLED=true`
- **Kafka consumer** on `iag.commercial` for CRM bridge events
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
set DATABASE_URL=postgres://user:pass@localhost:5432/iag_platform?sslmode=disable
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
| Writes | `PATCH /v1/outlets/:id`, `POST /v1/orders`, `POST /v1/dispatch`, `POST /v1/reports/run`, `POST /v1/exports/:page` |

Health: `GET /healthz`, `GET /ready` (Postgres ping when DB enabled).

Platform wiring: [`docs/PLATFORM_INTEGRATION.md`](docs/PLATFORM_INTEGRATION.md)  
Production: [`docs/PRODUCTION_CHECKLIST.md`](docs/PRODUCTION_CHECKLIST.md)

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

- Paginated lists use `{ items, data, meta }` — see [`docs/dms-api.ts`](docs/dms-api.ts).
- Set `FINANCE_URL` to proxy finance summary/invoices from iag-finance.
- Set `CONSUMER_ENABLED=true` to ingest CRM events into DMS signals.

## Production

Use [`config/.env.production.example`](config/.env.production.example): explicit `ALLOWED_ORIGINS` (no `*`), `AUTO_MIGRATE=false`, `SEED_ON_EMPTY=false`, and `SERVICE_CLIENT_SECRET` from your secret store. Run DB migrations out of band. Traces export via `OTEL_EXPORTER_OTLP_ENDPOINT`.

Registry: [`subrepos.json`](../../../subrepos.json) · Dev port: **4010** (gateway: `/api/v1/dms`)
