# 01 — Architecture

## System context

```
                     ┌────────────────────────────────────────────────┐
                     │              ZiadaPOS API (this repo)          │
                     │                                                │
    Web dashboard ───┤                                                │
    Desktop POS   ───┤   cmd/api/main.go ── registerApps()            │
    Mobile app    ───┤   ├─ apps/*  (routes, handlers, services)     │
    MCP agent     ───┤   ├─ internal/server (Gin engine + Deps)      │
                     │   ├─ internal/security (JWT)                  │
                     │   ├─ internal/events (async bus)              │
                     │   └─ internal/scheduler (cron)                │
                     └───────┬──────────────┬───────────────┬────────┘
                             │              │               │
                    ┌────────▼─────┐ ┌──────▼──────┐ ┌──────▼──────┐
                    │  PostgreSQL  │ │   Redis     │ │    MinIO    │
                    │  (GORM)      │ │ (cache, rl) │ │ (objects)   │
                    └──────────────┘ └─────────────┘ └─────────────┘
                             │
                 ┌───────────┼───────────────┬──────────────────────┐
                 │           │               │                      │
           ┌──────▼─────┐ ┌───▼────────┐ ┌────────▼──────┐
           │   Ngamia   │ │ SendAfrica │ │  SMTP (Zoho)  │
           │  (AI chat) │ │ (SMS)      │ │    (email)    │
           └────────────┘ └────────────┘ └───────────────┘
```

The API is **stateless** (except the DB/Redis), horizontally scalable, and multi-tenant via the JWT.

## Bootstrap (`cmd/api/main.go`)

1. `config.Load(".env")` — loads environment.
2. `database.Connect(cfg)` — PostgreSQL via GORM.
3. `cache.New(ctx, cfg)` — Redis. **Optional at boot**: if Redis is unreachable, the API logs
   "running without Redis" and continues (rate limiting, caching and login lockout degrade
   gracefully).
4. `storage.New(ctx, cfg)` — MinIO. **Optional at boot** — uploads fail until it is up.
5. `events.New(10, 256)` — 10 worker goroutines, 256-event queue.
6. `scheduler.New(cfg.Timezone)` — cron in Africa/Dar_es_Salaam.
7. Build `server.Deps` (the shared dependency container) and `server.New(deps)` (Gin engine +
   global middleware + `/healthz`, `/`, `/api/v1/health`).
8. `registerApps(db, deps, srv)` — for each app: `Migrate` (schema) → `RegisterHandlers` (event
   subscriptions) → `RegisterRoutes` (HTTP routes).
9. `sched.Start()` then listen. Graceful shutdown on SIGINT/SIGTERM (15 s drain).

## Request flow

```
HTTP → Gin engine
  → middleware.RequestID()   add X-Request-Id
  → middleware.Recovery()    panic → 500
  → middleware.Logger()      access log
  → middleware.CORS()        allowed origins
  → app router group (/api/v1/...)
      → middleware.Auth()    parse Bearer JWT → set user_id/role/org_id/store_id in context
      → (some groups) RequireOwner() / RequireAdmin()
      → handler (binds JSON, validates)
      → service (business logic, GORM queries)
      → response envelope {success, data, meta} or apierr → {success:false, message, errors}
```

## The `Deps` container (`internal/server/server.go`)

Every app receives `*server.Deps` so routes/services never construct their own clients:

| Field | Type | Use |
| --- | --- | --- |
| `Cfg` | `*config.Config` | runtime settings |
| `DB` | `*gorm.DB` | database |
| `Cache` | `*cache.Cache` | Redis |
| `Storage` | `*storage.Storage` | MinIO |
| `Mailer` | `*mailer.Mailer` | transactional email |
| `SMS` | `*sms.Client` | SendAfrica |
| `AI` | `*ai.Client` | Ngamia |
| `Bus` | `*events.Bus` | async events |
| `Tokens` | `*security.Manager` | JWT |
| `Scheduler` | `*scheduler.Scheduler` | cron |

## Event bus (`internal/events`)

Services `Publish` domain events; apps `On` them. Handlers run in a bounded worker pool so a slow
SMS/email handler never blocks the HTTP response. Dropped events are logged (handlers re-read DB
state anyway).

Registered events include:

| Event | Subscribers |
| --- | --- |
| `user.registered` | accounts → welcome email; subscriptions → create trial |
| `staff.created` | accounts → SMS temp password |
| `transaction.created` | credits → open tab for Credit sales; customers → spend/segment; notifications; analytics |
| `transaction.refunded` / `transaction.voided` | credits → settle tab; customers → reduce spend; notifications |

## Scheduler (`internal/scheduler`)

Wraps `robfig/cron` pinned to `APP_TIMEZONE` (Africa/Dar_es_Salaam). Current job:
**reports daily dispatch** at 06:30 EAT — sends every enabled `ScheduledReport` whose
`next_send_at` has arrived.

## Tenancy

- `Organisation` = the registered business (top level).
- `Store` = a physical branch, belongs to an organisation.
- `User` = owner or staff, scoped to an organisation + store.
- The JWT carries `org_id` + `store_id`; repositories scope queries with those ids. There is no
  tenant id in URLs.

## Related docs

- [02 — Configuration](02-configuration.md)
- [03 — Auth & Security](03-auth-and-security.md)
- [06 — Internals](06-internals.md)
