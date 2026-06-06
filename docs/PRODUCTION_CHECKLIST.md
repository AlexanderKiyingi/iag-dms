# DMS — production checklist

Use this before enabling DMS in staging/production.

## Required

| Item | Env / setting | Verify |
|------|----------------|--------|
| Database | `DATABASE_URL` migrated through `0004_platform_hardening` | `GET /ready` returns Postgres healthy |
| Auth | `JWT_ISSUER`, `JWKS_URL`, `AUDIENCE=iag.dms` | Mutating API returns 401 without Bearer |
| Service account | `SERVICE_CLIENT_SECRET` (≥16 chars) | Startup log: permissions registered |
| Strict RBAC | `ENVIRONMENT=production` | Tokens without `permissions` are denied |
| Migrations | `AUTO_MIGRATE=false` | Run `db/migrations` out of band |
| Seed | `SEED_ON_EMPTY=false` | No demo seed in production |
| Kafka publish | `EVENT_BUS_ENABLED=true`, `KAFKA_BROKERS` | Outlet create emits `dms.outlet.created` via outbox |
| Kafka consumer | `CONSUMER_ENABLED=true` | CRM events update `dms_signals` / outlet `crm_ref` |
| Finance proxy | `FINANCE_URL` + service secret | `GET /v1/finance/summary` returns upstream data |

## Recommended

| Item | Notes |
|------|--------|
| `PUBLIC_API_URL` | Gateway origin for docs and callbacks |
| `ALLOWED_ORIGINS` | Explicit CORS list (not `*`) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Trace export to collector |

## Kubernetes

Manifests: [`deploy/kubernetes/dms/`](../../../../deploy/kubernetes/dms/)

1. Copy `secret.example.yaml` → sealed secret / external secrets operator.
2. Apply configmap + deployment + service.
3. Point gateway `UPSTREAM_DMS` at `iag-dms:4010`.

## Smoke test (post-deploy)

```bash
curl -s https://api.example.com/api/v1/dms/ready

curl -s -H "Authorization: Bearer $TOKEN" \
  https://api.example.com/api/v1/dms/v1/overview

curl -s -H "Authorization: Bearer $TOKEN" \
  "https://api.example.com/api/v1/dms/v1/outlets?limit=5"
```

## Commercial loop

- **Outlet created** → `dms.outlet.created` on `iag.operations` (outbox)
- **CRM deal won / lead converted / outlet synced** → DMS signals + optional `crm_ref` link
- **CRM bridge sync** → expects list payloads with `items` or `data` + `meta.total`
