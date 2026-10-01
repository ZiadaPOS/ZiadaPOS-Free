# ZiadaPOS Go API

The multi-tenant backend for ZiadaPOS point of sale and business management. It serves the web app, mobile clients and other authorized clients. The service is written in Go with Gin, GORM and PostgreSQL.

## What it supports

- Tanzanian shop registration, phone OTP verification, login, refresh and account management.
- Four supported business types: retail, wholesale, retail + wholesale, and pharmacy.
- Store-scoped product catalogue, categories, units, stock adjustments and inventory imports.
- Retail/wholesale POS checkout, immutable sale line snapshots, transaction ledger, refunds, voids, receipts and summaries.
- Customer profiles, credit tabs/payments, suppliers, procurement, expenses, staff, branches, reports and analytics.
- Pharmacy dispensing profiles, batch and expiry accounting, FEFO stock allocation, prescription register capture, quarantine and batch movement audit.

The API is authoritative for tenant scope, permissions, prices, stock, tax calculation and business workflow validation. UI state must not be treated as committed or trusted business data.

## Business workflow guide

Start with [docs/business-workflows.md](docs/business-workflows.md) for step-by-step setup, retail, wholesale, mixed pricing, pharmacy, returns, credit, procurement, daily close, and known boundaries. The endpoint catalog is [docs/04-api-reference.md](docs/04-api-reference.md). The documentation index is [docs/00-index.md](docs/00-index.md).

## Quick start

Requirements: Go version in `go.mod` and Docker Compose for PostgreSQL, Redis and MinIO in local development.

```bash
cp .env.example .env
# Review and set local values. Never commit production credentials.
docker compose up -d
go run ./cmd/api
```

The default listener is `:8080`; the versioned base path is `/api/v1` unless `API_BASE_PATH` is changed. Check `GET /healthz` for liveness and `GET /api/v1/health` for database health. Read [configuration](docs/02-configuration.md) before changing runtime dependencies.

Example login using a locally created test account:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"0712345678","password":"<local-test-password>"}'
```

Use the returned access token as `Authorization: Bearer <access_token>` for protected requests. Registration payload shape and OTP workflow are documented in the API reference; do not use real customer credentials in examples or logs.

## Route groups

| Group | Purpose |
| --- | --- |
| `/auth`, `/accounts`, `/meta` | Authentication, business/user profiles, store switching, public reference lists. |
| `/inventory` | Categories, units, products, import, barcodes, adjustments, batches and low stock. |
| `/pos` | Sales, transaction ledger, refund/void, receipts and sales summaries. |
| `/pharmacy` | Pharmacy-only batches, movements, expiry, dispensing profiles and prescription register. |
| `/customers`, `/credits` | Customer records and SMS, credit tabs, payments, reminders, notes and write-offs. |
| `/suppliers` | Suppliers, purchase orders, deliveries, receipt and payments. |
| `/expenses`, `/staff`, `/stores` | Operating expenses, owner-managed staff and branches. |
| `/analytics`, `/reports`, `/notifications` | Business analysis, generated/scheduled exports and in-app notifications. |
| `/subscriptions` | Plans, current organisation subscription and platform-admin management. |
| `/uploads` | Authenticated MinIO object upload/delete. |
| `/reviews` | Public review listing and authenticated review submission. |
| `/ai` | Ngamia chat, conversations, suggestions and usage. |
| `/admin` | Platform administration and monitoring; protected routes require admin MFA. |

The current server does **not** register `/messaging`, `/public` storefront/order or `/commerce` route groups. Existing `docs/08-messaging-and-commerce.md` describes future design only; do not integrate against those routes as if deployed. Payment callback paths mentioned there are not active until implemented and registered.

The web UI currently includes an `/sms` screen that calls the unregistered `/messaging/*` API. Its wallet/key/sender controls do not work end to end yet. The API's implemented customer SMS endpoint is `POST /customers/:id/sms` and uses the configured SendAfrica integration.

## Architecture

```text
cmd/api/main.go                 bootstrap, migrations, app registration, lifecycle
internal/config                 env configuration
internal/server                 Gin engine and shared dependency container
internal/middleware             JWT auth, roles, CORS, logging, request IDs, rate limits
internal/database               PostgreSQL/GORM connection
internal/cache                  Redis cache helpers
internal/events                 bounded in-process async event bus
internal/scheduler              Africa/Dar_es_Salaam scheduled jobs
internal/storage                MinIO object storage
internal/mailer                 SMTP mailer
internal/integrations/{ai,sms}  provider adapters
apps/*                          domain models, repository, service, handler, routes
```

Each app owns its route registration and persistence operations. Routes take organisation, user and store scope from the verified token, then domain services apply business-specific validation. POS writes run transactionally with inventory ledger changes. Events handle secondary work such as customer and credit statistics; they do not replace the main transaction.

## Core workflow references

- [Business workflows](docs/business-workflows.md)
- [API reference](docs/04-api-reference.md)
- [Data model](docs/05-data-model.md)
- [Authentication and security](docs/03-auth-and-security.md)
- [Configuration](docs/02-configuration.md)
- [Architecture](docs/01-architecture.md)
- [Internals](docs/06-internals.md)
- [Deployment](docs/07-deployment.md)
- [Redeploy runbook](redeploy.md)

## Development commands

```bash
go fmt ./...
go vet ./...
go build ./...
go test ./...
```

Tests exist in several domain packages (including transactions, inventory, pharmacy, accounts and suppliers). These commands are listed for maintainers; documentation-only changes do not require running them unless requested.

## Data and security invariants

- Tenant scope comes from signed access-token claims, not user-provided organisation IDs.
- Store-scoped data and transaction numbering follow the active store claim.
- Money is stored as integer TZS values (whole shillings in current API fields); dates and timezone behavior are described per endpoint/workflow.
- Retail/wholesale prices and stock are re-read and validated by the API during sale commit.
- Pharmacy sale stock is allocated from available unexpired batches by FEFO. Customer returns remain quarantined pending inspection.
- Prescription capture is a staff-entered register and authenticity confirmation, not electronic prescription validation.
- Do not log passwords, bearer/refresh tokens, provider credentials, or sensitive prescription register data.

See [03 — Auth & Security](docs/03-auth-and-security.md) for token, role and OTP details and [07 — Deployment](docs/07-deployment.md) for production operations.
