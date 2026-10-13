# 02 — Configuration

Loaded from `.env` (or process environment) by `internal/config`. Process env wins over `.env`.
Defaults are development-friendly; set real values in production.

## App

| Env var | Field | Default | Meaning |
| --- | --- | --- | --- |
| `APP_NAME` | `AppName` | `ZiadaPOS API` | Shown on `/` and `/api/v1/health`. |
| `APP_ENV` | `Environment` | `development` | `production`/`development`/… |
| `DEBUG` | `Debug` | `true` in development | Gin debug mode. |
| `PORT` | `Port` | `8080` | HTTP listen port. |
| `API_LISTEN_ADDR` | `ListenAddr` | `:<PORT>` | Optional full bind address, e.g. `127.0.0.1:8080` to accept local-only connections. |
| `API_BASE_PATH` | `APIBasePath` | `/api/v1` | Prefix for all app routes. |
| `SITE_URL` | `SiteURL` | `http://localhost:3000` | Frontend origin, used in email links. |
| `APP_TIMEZONE` | `Timezone` | `Africa/Dar_es_Salaam` | Scheduler + "today" logic. |

## PostgreSQL

| Env var | Default | Meaning |
| --- | --- | --- |
| `DB_HOST` | `localhost` | Host. |
| `DB_PORT` | `5432` | Port. |
| `DB_USER` | `ziada` | User. |
| `DB_PASSWORD` | `ziada` | Password. |
| `DB_NAME` | `ziada` | Database name. |
| `DB_SSLMODE` | `disable` | e.g. `require` in managed Postgres. |
| `DB_MAX_OPEN_CONNS` | `25` | Pool: max open. |
| `DB_MAX_IDLE_CONNS` | `10` | Pool: max idle. |
| `DB_CONN_MAX_LIFETIME` | `5m` | Max connection lifetime. |

## Redis

| Env var | Default | Meaning |
| --- | --- | --- |
| `REDIS_ADDR` | `localhost:6379` | Host:port. |
| `REDIS_PASSWORD` | *(empty)* | Password. |
| `REDIS_DB` | `0` | DB index. |
| `CACHE_TTL` | `5m` | Default TTL for cached values. |

## MinIO (object storage)

| Env var | Default | Meaning |
| --- | --- | --- |
| `MINIO_ENDPOINT` | `localhost:9000` | Endpoint. |
| `MINIO_ACCESS_KEY` | `minioadmin` | Access key. |
| `MINIO_SECRET_KEY` | `minioadmin` | Secret key. |
| `MINIO_BUCKET` | `ziada` | Bucket name. |
| `MINIO_USE_SSL` | `false` | TLS to MinIO. |
| `MINIO_REGION` | `us-east-1` | Region. |
| `PUBLIC_BASE_URL` | `http://localhost:9000/ziada` | Public URL prefix for stored objects (CDN/domain in front of MinIO). |

## JWT

| Env var | Default | Meaning |
| --- | --- | --- |
| `JWT_ACCESS_SECRET` | `change-me-access` | HS256 key for access tokens. **Set in production** (the server warns otherwise). |
| `JWT_REFRESH_SECRET` | `change-me-refresh` | HS256 key for refresh tokens. **Set in production**. |
| `MFA_ENCRYPTION_KEY` | falls back to `JWT_REFRESH_SECRET` | Stable secret used to encrypt admin authenticator seeds at rest. Set explicitly before enrolling admins; changing it later requires MFA re-enrollment. |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime. |
| `JWT_REFRESH_TTL` | `168h` (7 days) | Refresh token lifetime. |

## Email

| Env var | Default | Meaning |
| --- | --- | --- |
| `SMTP_HOST` | `smtp.zoho.com` | SMTP host. |
| `SMTP_PORT` | `587` | SMTP port. |
| `SMTP_USER` / `SMTP_PASSWORD` | *(empty)* | SMTP login. |
| `SMTP_FROM` | `no-reply@ziadapos.com` | From address. |
| `SMTP_FROM_NAME` | `ZiadaPOS` | From name. |

> Mailer selection: SMTP creds set → `smtp` mode (Zoho); else `log` mode
> (emails logged, never sent — safe for dev).

## SMS (SendAfrica)

| Env var | Default | Meaning |
| --- | --- | --- |
| `SENDAFRICA_BASE_URL` | `https://api.sendafrica.online` | API base. |
| `SENDAFRICA_API_KEY` | *(empty)* | API key (`SA-…`). When empty, SMS runs in dry-run (logged, not sent). |
| `SENDAFRICA_SENDER_ID` | `ZIADA POS` | Sender id shown to recipients. |

The registered API uses SendAfrica for OTPs, staff temporary-password messages, customer SMS,
and credit reminders. There is no registered SMS campaign/wallet app and no implemented
organisation-level 100-message wallet in the current schema. Do not infer campaign credit,
top-up, per-organisation provider-key or sender-ID approval behavior from these environment
variables.

## Snippe payments

Snippe configuration fields are loaded but **are not used by a registered payment integration**
in the current server. There are no active Snippe checkout or webhook routes. Do not enable or
advertise payment processing based on these values; the environment flags alone do not create a
working, verified payment workflow. When such a module is implemented, document its real route
registration and signature/replay protections before deployment.

`SENDAFRICA_SENDER_ID` is the configured sender value used by the current SendAfrica client. No
Snippe payment OTP or payment-confirmation flow is implemented by this API.

## AI (Ngamia)

| Env var | Default | Meaning |
| --- | --- | --- |
| `NGAMIA_BASE_URL` | `https://api.ngamia.cc/v1` | Gateway base (must include `/v1`). |
| `NGAMIA_API_KEY` | *(empty)* | API key (`ngm_…`). |
| `NGAMIA_MODEL` | `openai/gpt-4o-mini` | Model used by `/ai/chat`. |

## Scheduler

| Env var | Default | Meaning |
| --- | --- | --- |
| `DAILY_REPORT_HOUR` | `22` | Default daily report hour (scheduled reports dispatch at 06:30 EAT in `apps/reports`). |
| `DAILY_REPORT_MINUTE` | `0` | Default minute. |

## CORS

| Env var | Default | Meaning |
| --- | --- | --- |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,http://localhost:1420,http://localhost:5173,https://app.ziadapos.com,https://admin.ziadapos.com` | Comma-separated allowed origins. |

## Security notes

- **Never commit `.env`.** It holds DB/Redis/MinIO credentials and provider keys. It is gitignored.
- In production the container loads `.env` via `env_file:` at **creation time** — after editing it,
  recreate the container (`docker compose up -d --force-recreate api`), a plain restart is not
  enough. See [redeploy.md](../redeploy.md).
- Default JWT secrets print a warning at boot — always override in production.
