# 07 — Deployment

## Local development

Requirements: **Go 1.25+**, **Docker** (for Postgres/Redis/MinIO).

```bash
cd api
cp .env.example .env        # defaults work for local dev
docker compose up -d        # postgres, redis, minio (MinIO console on :9001)
go build -o bin/api ./cmd/api
./bin/api                   # → http://localhost:8080
```

Useful checks:

```bash
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/api/v1/health
curl -s http://localhost:8080/
```

> **Redis/MinIO are optional at boot** — dev survives with neither, but rate limiting, lockout,
> caching and uploads won't work until they're up. Production runs all three.

## Docker image

`Dockerfile` — two-stage build:

1. `golang:1.25-alpine` builder: `go mod download`, then
   `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /ziada-api ./cmd/api`.
2. `alpine:3.20` runtime with `ca-certificates` + `tzdata`, `TZ=Africa/Dar_es_Salaam`,
   exposes `8080`, entrypoint `/app/ziada-api`.

`docker-compose.yml` — the api service plus its infra:

| Service | Image | Host port | Notes |
| --- | --- | --- | --- |
| `api` | build from Dockerfile | `8080:8080` | env_file `.env`; service-name overrides for DB/Redis/MinIO; healthcheck GET `/api/v1/health` |
| `postgres` | postgres:16-alpine | `5432` | user/pass/db `ziada`; named volume |
| `redis` | redis:7-alpine | `6379` | appendonly yes; named volume |
| `minio` | minio/minio | `9000` + `9001` (console) | `minioadmin`; named volume |

## Production (VPS)

Production lives on the VPS (`13.140.167.42`) in the `ziadapos-go` compose project, source tree
`~/ziadapos-go/ziadapos-api`. The image is `ziadapos-go-api:latest`, published on
**`https://api.ziadapos.com`** via nginx, plus a public file domain
**`https://files.ziadapos.com`** in front of MinIO (`PUBLIC_BASE_URL=https://files.ziadapos.com/ziada`).

Infra in the same project: Postgres 16, Redis 7, MinIO, and `mc-init` (creates the `ziada`
bucket with public-read policy so stored images are web-accessible).

Host port mapping: `8086 → 8080` (reverse-proxied to the domain).

`.env` on the VPS holds real secrets: DB/Redis/MinIO credentials, fresh JWT secrets,
`SMTP_*` (Zoho), `SENDAFRICA_API_KEY`, `NGAMIA_API_KEY`.

> Editing `.env` on the VPS requires recreating the api container — env_file is read at
> container **creation** time, not restart. See [redeploy.md](../redeploy.md).

## CI/CD (GitHub Actions)

`.github/workflows/deploy.yml` — runs on push to `main`:

1. **Test** — `setup-go 1.25`, `go vet ./...`, `go build ./...`, `go test ./...`.
2. **Deploy** — SSH (via `VPS_*` secrets + deploy key) → rsync the source to the VPS → in
   `~/ziadapos-go/deploy` run `docker compose build api && docker compose up -d api` →
   health check `curl /api/v1/health`.

Secrets required in the repo (all set for `Camel-Creatives/ZiadaPOS-GO-API`):

| Secret | Value |
| --- | --- |
| `VPS_HOST` | `13.140.167.42` |
| `VPS_USER` | `camel` |
| `VPS_PORT` | `4444` |
| `VPS_SSH_KEY` | CI deploy key (`ziada_ci_ed25519`, in the VPS `authorized_keys`) |
| `VPS_KNOWN_HOSTS` | the VPS host key line |

## Operational notes

- **Health endpoint**: `GET /api/v1/health` returns `db`/`redis` status; a Redis outage reports
  `degraded` rather than failing the container.
- **MCP agent note**: the agent container's healthcheck must be TCP-only. HTTP probes that hit
  `/mcp/` redirect to a 307 and a misconfigured agent can trigger the **login rate lock**
  (`login:lock:<identifier>`, 5 attempts / 15 min). If that happens:
  `redis-cli DEL "login:lock:<identifier>"`.
- **Logs**: `docker compose -p ziadapos-go logs -f api` (deploy project `~/ziadapos-go/deploy`).
- **Backups**: daily Postgres dumps (`~/backups/`); keep the last Django-era dump
  (`ziada_django_*.sql.gz`) until the platform has no rollback path.
