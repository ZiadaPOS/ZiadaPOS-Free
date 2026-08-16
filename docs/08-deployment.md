# 08 — Deployment

## Local development

```bash
cd agent
uv sync
cp .env.example .env    # point ZIADA_API_URL at a running API; add credentials
uv run ziada-mcp        # HTTP transport on :8081
```

The repo ships a `docker-compose.yml` for a one-container local stack (builds the image, port
`8081:8081`, env from `.env`).

## MCP endpoint

- HTTP transport exposes the streamable-http endpoint at `/mcp`.
- Clients must connect **without a trailing slash**: `http://localhost:8081/mcp` or
  `https://agent.ziadapos.com/mcp`. A trailing slash makes the server 307-redirect, which some
  clients mishandle.

## Production (VPS)

- Runs as the Docker Compose service **`agent`** in project `ziadapos-go` on the production VPS.
- Host port **8087 → 8081**; nginx terminates TLS at `https://agent.ziadapos.com`.
- Source lives at `~/ziadapos-go/ziadapos-agent/`; compose project at `~/ziadapos-go/deploy/`.
- Image: `ziadapos-agent:latest`.

### Manual redeploy

Full runbook in [redeploy.md](../redeploy.md). Two flows:

**Code change**

```bash
rsync -avz --delete --exclude='.git' --exclude='.venv' --exclude='__pycache__' \
  -e "ssh -p 4444" ./ camel:~/ziadapos-go/ziadapos-agent/
ssh camel 'cd ~/ziadapos-go/deploy && docker compose build agent && docker compose up -d agent'
```

**`.env` change** (env is baked in at container creation)

```bash
ssh camel 'vim ~/ziadapos-go/deploy/.env'   # chmod 600 afterwards
ssh camel 'cd ~/ziadapos-go/deploy && docker compose config -q && docker compose up -d --force-recreate agent'
```

### CI/CD

`.github/workflows/deploy.yml` runs on every push to `main`:

1. **Test**: `uv sync --group dev` → `uvx ruff check src/ziada_mcp tests` → `uv run pytest -q`.
2. **Deploy**: rsync the source to the VPS → `docker compose build agent && docker compose up -d
   agent` → verify the container is healthy. Uses the `VPS_*` GitHub Actions secrets.

## Container healthcheck — important

The production healthcheck must be a **plain TCP connect to 8081**. Do **not** make it an
HTTP/`urllib` hit of `/mcp/`:

- Every probe opens an MCP session, which logs the agent into the ZiadaPOS API.
- Rapid probes → the API's **15-minute login rate-lock** (`423 Account temporarily locked`), which
  locks the agent's service account until cleared.

If the account does get locked, clear the lock in Redis:

```bash
ssh camel 'docker exec ziadapos-go-redis-1 redis-cli DEL "login:lock:owner@ziadapos.com"'
```

## Troubleshooting

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `400 Bad Request` when a client connects | Trailing slash in the MCP URL | Use `/mcp` (no slash) |
| `423 Account temporarily locked` | Login storm (healthcheck or bots) | Clear `login:lock:<identifier>` in Redis; keep healthcheck TCP-only |
| `No image found for '…'` | Wikimedia query too specific | Call `suggest_product_image` with shorter terms, or provide an image |
| `not configured` on SMS/email tools | Missing `SENDAFRICA_*` / `MAILAFRICA_*` vars | Add to `.env`, `--force-recreate` the container |
| Stale behaviour after `.env` edit | Container not recreated | `docker compose up -d --force-recreate agent` |
