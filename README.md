# ZiadaPOS API

The backend for **ZiadaPOS** — the point-of-sale platform for Tanzanian shop owners. A
[REST](https://en.wikipedia.org/wiki/REST) API written in **Go** with the
[Gin](https://github.com/gin-gonic/gin) web framework and [GORM](https://gorm.io/) on PostgreSQL.

It powers every client: the desktop POS, the mobile app, the web dashboard, and the
**[ZiadaPOS MCP Agent](https://github.com/Camel-Creatives/ZiadaPOS-agent)** (the natural-language
assistant that exposes all of these endpoints as MCP tools).

## Highlights

- **Multi-tenant by design** — every row is scoped to an `organisation` (the business) and often a
  `store` (the branch). Tenancy comes from the JWT, so cross-tenant access is impossible by
  construction.
- **Roles**: `admin` (platform), `owner` (business), `staff` (cashier). Owner-only routes are
  protected by middleware.
- **Phone-first auth**: registration, OTP verification, password reset and login all centre on the
  Tanzanian phone number (10-digit `0…` form), with email as the secondary channel.
- **JWT access + refresh tokens**; access tokens embed `org_id` + `store_id`, so switching stores
  re-exchanges tokens.
- **Login rate limiting** in Redis (5 failed attempts → 15-minute lockout).
- **Event-driven side effects** — an in-process async event bus (email/SMS, credit tabs, customer
  stats, notifications) so a slow provider call never blocks the HTTP response.
- **Cron scheduler** in `Africa/Dar_es_Salaam` for scheduled reports and recurring jobs.
- **Object storage via MinIO** (S3-compatible) for product images, logos, report exports and
  receipts, served from a public base URL.
- **Two external providers** — [Ngamia](https://docs.ngamia.cc) (AI gateway, OpenAI-compatible)
  and [SendAfrica](https://docs.sendafrica.online) (SMS) — plus transactional email over
  SMTP (Zoho).
- **Auto schema migration** — every app calls `Migrate()` on boot (GORM `AutoMigrate`).
- **Graceful degradation** — Redis or MinIO down? The API boots anyway (production should run them,
  but dev never crashes on a missing dependency).
- **CI/CD** — vet/build/test on push, then auto-deploy to the production VPS.

## Quick start (local)

Requires **Go 1.25+** and Docker (for PostgreSQL, Redis and MinIO).

```bash
cd api
cp .env.example .env      # adjust as needed
docker compose up -d      # postgres + redis + minio

go build -o bin/api ./cmd/api
./bin/api                 # listens on :8080
```

Smoke test:

```bash
curl -s http://localhost:8080/api/v1/health
curl -s http://localhost:8080/healthz
```

Register an owner and log in:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "first_name":"Hamisi","phone":"0712345678","password":"secret-2026",
    "organisation_name":"Duka la Hamisi","business_type":"retail",
    "occupation":"shop_owner","region":"dar-es-salaam","district":"ilala"
  }'

curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"0712345678","password":"secret-2026"}'
# → { "data": { "access_token": "…", "refresh_token": "…", ... } }
```

Use the access token on every other call:

```bash
curl -s http://localhost:8080/api/v1/accounts/me \
  -H "Authorization: Bearer <access_token>"
```

## Apps & route groups

| Group | Base path | Description |
| --- | --- | --- |
| Auth & accounts | `/auth`, `/accounts`, `/meta` | Registration, OTP, login, JWT refresh, profile, org, users |
| Inventory | `/inventory` | Categories, products, barcodes, stock adjustments, low-stock |
| POS | `/pos` | Sales, transactions, refunds, voids, summaries |
| Customers | `/customers` | Customer profiles, segments, credit listings |
| Credits | `/credits` | Credit tabs, payments, reminders, write-offs |
| Suppliers | `/suppliers` | Suppliers, deliveries, supplier payments |
| Expenses | `/expenses` | Expense ledger + summaries |
| Staff | `/staff` | Staff management, shifts, permissions, KPIs |
| Stores | `/stores` | Multi-branch stores |
| Reports | `/reports` | Report types, generation, exports, scheduled reports |
| Analytics | `/analytics` | KPIs, trends, payment mix, top products, cashflow |
| Notifications | `/notifications` | In-app notifications |
| Messaging | `/messaging` | SendAfrica sender IDs, SMS wallet, templates, campaigns, and delivery logs |
| Public commerce | `/public`, `/commerce` | Shareable storefronts, carts, orders, and invoices |
| Notebook | `/notebook` | Store notes & reminders |
| AI | `/ai` | Chat with the Ngamia model, suggestions, usage |
| Subscriptions | `/subscriptions` | Plans, my-subscription, store limits, admin management |
| Reviews | `/reviews` | Public store reviews |
| Uploads | `/uploads` | File upload/delete to MinIO |

The complete endpoint reference is in **[docs/04-api-reference.md](docs/04-api-reference.md)**.

## Architecture

```
cmd/api/main.go          bootstrap: config, DB, Redis, MinIO, mailer, SMS, AI, event bus, scheduler
internal/config          env-driven configuration (Config struct + Load)
internal/database        PostgreSQL connection (GORM)
internal/cache           Redis wrapper (JSON-aware, TTL, counters, pattern deletes)
internal/security        JWT access/refresh issuing + parsing (HS256)
internal/middleware      Auth, RequireOwner/RequireAdmin, CORS, request id, logging, rate limit
internal/response        {success, data, message, errors, meta} envelope + pagination
internal/apierr          structured domain errors → HTTP mapping
internal/events          async in-process event bus (bounded queue + worker pool)
internal/scheduler       cron (Africa/Dar_es_Salaam)
internal/storage         MinIO client (put/get/delete/presign/url)
internal/mailer          transactional email over SMTP (Zoho) or log-only
internal/integrations    ai (Ngamia) and sms (SendAfrica) clients
internal/server          Gin engine + middleware + shared Deps container
apps/*                   each domain: models.go, repository.go, service.go, handler.go, routes.go
```

Each app follows the same pattern: `Migrate(db)` (schema), optional `RegisterHandlers(bus, deps)`
(event subscriptions), `RegisterRoutes(base, deps)` (routes). They are registered in
`cmd/api/main.go → registerApps`.

More: **[docs/01-architecture.md](docs/01-architecture.md)**.

## Configuration

Everything is driven by environment variables (loaded from `.env` via godotenv, process env wins).
Full reference: **[docs/02-configuration.md](docs/02-configuration.md)**.

## Testing

```bash
go vet ./...
go build ./...
go test ./...            # currently no Go tests are checked in — add them under each app
```

CI runs vet + build + test on every push to `main`, then auto-deploys.

## Deployment

- **Local**: `docker compose up -d` (postgres/redis/minio) + `go build ./cmd/api`.
- **Production**: Docker image `ziadapos-go-api:latest` behind nginx at
  `https://api.ziadapos.com`, in the `ziadapos-go` compose project on the VPS
  (Postgres/Redis/MinIO/mc-init + api). Host port `8086 → 8080`.
- **CI/CD**: `.github/workflows/deploy.yml` — vet/build/test, rsync to VPS, `docker compose build
  api && docker compose up -d api`, health check.
- Manual restarts / `.env` changes: **[redeploy.md](redeploy.md)** and
  **[docs/07-deployment.md](docs/07-deployment.md)**.

## Documentation

- **[docs/00-index.md](docs/00-index.md)** — index
- **[docs/01-architecture.md](docs/01-architecture.md)** — system context, request flow, event bus
- **[docs/02-configuration.md](docs/02-configuration.md)** — every env var
- **[docs/03-auth-and-security.md](docs/03-auth-and-security.md)** — JWT, roles, OTP, lockout
- **[docs/04-api-reference.md](docs/04-api-reference.md)** — every endpoint
- **[docs/05-data-model.md](docs/05-data-model.md)** — database tables
- **[docs/06-internals.md](docs/06-internals.md)** — Deps, cache, storage, mailer, integrations
- **[docs/07-deployment.md](docs/07-deployment.md)** — local + production + CI/CD
