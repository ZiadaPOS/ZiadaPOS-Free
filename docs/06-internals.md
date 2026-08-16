# 06 — Internals

This page documents the non-HTTP machinery in `internal/` and how apps use it.

## `internal/config`

`Config` struct + `Load(path string)` which reads the given file with godotenv then overlays
process environment (process wins). One struct for the whole app — the same instance is threaded
through `Deps` so there is exactly one source of truth for settings. See
[02 — Configuration](02-configuration.md) for the full list.

## `internal/database`

`Connect(cfg)` opens the PostgreSQL pool with GORM, applies sane `SetMaxOpenConns` /
`SetMaxIdleConns` / `SetConnMaxLifetime`, and ping-checks it. Returns `*gorm.DB`. Apps register
their models via `AutoMigrate` on boot.

## `internal/cache`

Thin typed wrapper over the Redis client:

- `Set/Get` with JSON marshalling and TTL (`CACHE_TTL` default).
- `Del`, `DelByPattern` (SCAN-based), `Incr` / `Expire` for counters.
- Used for: reference lookups, AI usage, login lockout, rate limiting, `auth:otp` code checks.
- **Never a hard dependency**: all callers tolerate a `nil`/failed Redis gracefully.

## `internal/storage`

S3-compatible client wrapper (MinIO):

- `Put(ctx, key, reader, size, contentType)` — upload; returns the object.
- `Get`, `Delete` (delete the object + the image from S3).
- `URL(key)` / `PublicURL(key)` — public URL built from `PUBLIC_BASE_URL`.
- `PresignURL` — short-lived signed URL for private objects.
- Used by `apps/uploads` (product images, receipts) and `apps/reports` (CSV exports).

## `internal/mailer`

Transactional email with three pluggable backends chosen at boot:

1. **MailAfrica REST** (preferred) — when `MAILAFRICA_API_KEY` is set. Resolves the verified
   `MAILAFRICA_FROM_DOMAIN` to a numeric sender id, caches it, then `POST /email/send`.
2. **SMTP** — `net/smtp` against `SMTP_HOST:PORT` (Zoho default).
3. **Log** — development fallback: "emails" are logged, never sent.

`Send(to, subject, html)` returns a structured result; used by the welcome email and password
confirmations.

## `internal/integrations`

### `sms` (SendAfrica)

`Client` with `Send(ctx, to, message, senderID)`. Validates/normalises the phone
(`0…` → `255…`). When `SENDAFRICA_API_KEY` is empty it runs in **dry-run** mode (logs the
message). Used for OTPs, temp passwords and credit reminders.

### `ai` (Ngamia)

OpenAI-compatible HTTP client pointed at `NGAMIA_BASE_URL` with `NGAMIA_API_KEY` + `NGAMIA_MODEL`.
`Chat(ctx, system, messages)` returns the model reply. `apps/ai` wraps it with conversation
persistence and monthly credit counting.

### `mailafrica`

REST client for the email provider: `GetSenders()` → `SendEmail(to, subject, html, from…)`.
Used by `internal/mailer` mode 1.

## `internal/security`

`Manager` (a.k.a. `Tokens`):

- `GenerateTokens(user)` — mints access + refresh JWTs (HS256, separate secrets, separate TTLs),
  stores a hash of the refresh token (rotating) in `refresh_tokens`.
- `ParseAccess(token)` / `ParseRefresh(token)` — validate signature + expiry + `typ`.
- Claims carry `uid`, `role`, `org_id`, `store_id`, `typ`, `jti`.
- `HashPassword` / `CheckPassword` — bcrypt helpers used by `accounts`.

## `internal/middleware`

- `Auth(tokens)` — Bearer → `context.Set(auth_user_id/role/org_id/store_id/claims)`; helpers
  `UserID(ctx)`, `Role(ctx)`, `OrgID(ctx)`, `StoreID(ctx)`, `Claims(ctx)`.
- `RequireOwner()`, `RequireAdmin()`.
- `RateLimit(cache, max)` — Redis fixed window per user/IP+route.
- `RequestID()`, `Logger()`, `Recovery()`, `CORS()` (origins from `CORS_ALLOWED_ORIGINS`).

## `internal/events`

`Bus` — async in-process pub/sub:

- `Publish(name string, payload any)` — enqueue to a bounded channel (size 256).
- `On(name, handler)` — subscribe; 10 worker goroutines drain the queue.
- Handlers are functions `func(ctx, data any)`; payloads are re-marshalled to the handler's
  concrete type on delivery (safe to publish any struct).
- If the queue is full the event is dropped and logged — handlers are written to re-read state
  from the DB so nothing is lost irreversibly.

## `internal/scheduler`

`Scheduler` wraps `robfig/cron` with `cron.WithLocation(cfg.Timezone)` so jobs fire in
`Africa/Dar_es_Salaam`. `Every(spec, job)` + `Start()`. Current jobs:

- **06:30 EAT** — `apps/reports.DispatchDue` (scheduled reports).

## `internal/response` & `internal/apierr`

- `response.OK(c, data)` / `response.Created` / `response.Error(c, apierr)` / pagination helper.
- `apierr.Error{Status, Code, Message, Fields}` — the one type handlers use; the envelope
  serialises it. `Fields` becomes `errors` for validation messages.

## `internal/server`

`server.New(deps)` builds the Gin engine, global middleware, and the three meta routes
(`/healthz`, `/`, `/api/v1/health`). `Deps` is the single container handed to every app:

```go
type Deps struct {
  Cfg       *config.Config
  DB        *gorm.DB
  Cache     *cache.Cache
  Storage   *storage.Storage
  Mailer    *mailer.Mailer
  SMS       *sms.Client
  AI        *ai.Client
  Bus       *events.Bus
  Tokens    *security.Manager
  Scheduler *scheduler.Scheduler
}
```

## `internal/uuidx`

`New()` returns UUIDv4 strings; used for entity ids and refresh-token jti values.
