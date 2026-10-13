# Redeploy — ZiadaPOS MCP Agent

Production stack lives on the VPS (`13.140.167.42`, SSH `camel` alias or port 4444) as a Docker
Compose project named **`ziadapos-go`**.

| Service | Container image | Host port | Source dir on VPS              |
|---------|-----------------|-----------|--------------------------------|
| `agent` | `ziadapos-agent:latest` | 8087 (→ 8081) | `~/ziadapos-go/ziadapos-agent` |

Compose project dir: `~/ziadapos-go/deploy` (`docker-compose.yml` + `.env`). Public endpoint:
`https://agent.ziadapos.com/mcp` (nginx → `127.0.0.1:8087`).

> Both redeploy flows below are exactly what CI runs; doing them manually is just running the same
> commands by hand.

## Prerequisites

- SSH access to the VPS: `ssh camel` (or `ssh -p 4444 camel@13.140.167.42`).
- Local Python 3.12+ for the manual test step.

---

## 1. Redeploy after code changes

No rebuild of the environment/`.env` is needed — only the application code changed.

```bash
# from this repo's root
rsync -avz --delete \
  --exclude='.git' --exclude='.venv' --exclude='__pycache__' --exclude='*.pyc' \
  -e "ssh -p 4444" ./ camel:~/ziadapos-go/ziadapos-agent/

# build the new image and recreate the container
ssh camel 'cd ~/ziadapos-go/deploy && docker compose build agent && docker compose up -d agent'
```

Optional: keep the previous image for rollback before rebuilding.

```bash
ssh camel 'docker tag ziadapos-agent:latest ziadapos-agent:prev'
```

## 2. Redeploy after `.env` changes

Environment variables are baked into the container **at creation time**, so after editing `.env`
the container must be recreated (no rebuild required).

```bash
# edit the env file on the VPS
ssh camel 'vim ~/ziadapos-go/deploy/.env'
chmod 600 ~/ziadapos-go/deploy/.env   # if you touched perms

# validate then recreate the container to pick up the new variables
ssh camel 'cd ~/ziadapos-go/deploy && docker compose config -q && docker compose up -d --force-recreate agent'
```

> Never commit `.env` to git. It holds `ZIADA_IDENTIFIER` / `ZIADA_PASSWORD` (agent API login) and
> provider keys. There is no `.env` in this repo.

## 3. Verify

```bash
# container status + log in (watch for "Registered N tools")
ssh camel 'cd ~/ziadapos-go/deploy && docker compose ps agent'
ssh camel 'docker logs --tail 50 ziadapos-go-agent-1'

# connect a test MCP client (note: use the URL WITHOUT a trailing slash)
# e.g. with the fastmcp python client:
#   async with Client("https://agent.ziadapos.com/mcp") as c:
#       tools = await c.list_tools()
```

## 4. Logs

```bash
ssh camel 'docker logs -f --tail 100 ziadapos-go-agent-1'
ssh camel 'cd ~/ziadapos-go/deploy && docker compose logs -f agent'
```

## 5. Rollback

If the new build is bad, restore the previous source, rebuild, and restart.

```bash
# (optional) if you tagged the previous image:
ssh camel 'docker tag ziadapos-agent:prev ziadapos-agent:latest'
ssh camel 'cd ~/ziadapos-go/deploy && docker compose up -d --force-recreate --no-build agent'

# otherwise: checkout the previous commit locally, rsync again, rebuild
git log --oneline -5
git checkout <previous-good-commit>
# ... run the Section 1 commands ...
git checkout main
```

## Notes

- **Do not** change the agent's `healthcheck` to an HTTP/`urllib` hit of `/mcp/`: every probe opens
  an MCP session and logs the agent into the API, which trips the API's 15-minute login-rate lock.
  The healthcheck must stay a plain TCP connect on `8081` (as in `docker-compose.yml`).
- The agent account (`.env` → `ZIADA_IDENTIFIER`/`ZIADA_PASSWORD`) is a normal shop-owner account;
  if it gets locked, clear the lock in Redis:
  `docker exec ziadapos-go-redis-1 redis-cli DEL "login:lock:<identifier>"`.
