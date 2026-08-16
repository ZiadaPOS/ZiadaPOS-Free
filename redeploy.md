# Redeploy — ZiadaPOS Go API

Production stack lives on the VPS (`13.140.167.42`, SSH `camel` alias or port 4444) as a Docker
Compose project named **`ziadapos-go`**.

| Service | Container image | Host port | Source dir on VPS            |
|---------|-----------------|-----------|------------------------------|
| `api`   | `ziadapos-go-api:latest` | 8086 (→ 8080) | `~/ziadapos-go/ziadapos-api` |

Compose project dir: `~/ziadapos-go/deploy` (`docker-compose.yml` + `.env`). Public endpoint:
`https://api.ziadapos.com` (nginx → `127.0.0.1:8086`).

> Both redeploy flows below are exactly what CI runs; doing them manually is just running the same
> commands by hand.

## Prerequisites

- SSH access to the VPS: `ssh camel` (or `ssh -p 4444 camel@13.140.167.42`).
- Your local user must be able to reach the VPS over SSH with the configured key.

---

## 1. Redeploy after code changes

No rebuild of the environment/`.env` is needed — only the application code changed.

```bash
# from this repo's root
rsync -avz --delete \
  --exclude='.git' --exclude='bin' \
  -e "ssh -p 4444" ./ camel:~/ziadapos-go/ziadapos-api/

# build the new image and recreate the container
ssh camel 'cd ~/ziadapos-go/deploy && docker compose build api && docker compose up -d api'
```

Optional: keep the previous image for rollback before rebuilding.

```bash
ssh camel 'docker tag ziadapos-go-api:latest ziadapos-go-api:prev'
```

## 2. Redeploy after `.env` changes

Environment variables are baked into the container **at creation time**, so after editing `.env`
the container must be recreated (no rebuild required).

```bash
# edit the env file on the VPS
ssh camel 'vim ~/ziadapos-go/deploy/.env'
chmod 600 ~/ziadapos-go/deploy/.env   # if you touched perms

# validate then recreate the container to pick up the new variables
ssh camel 'cd ~/ziadapos-go/deploy && docker compose config -q && docker compose up -d --force-recreate api'
```

> Never commit `.env` to git. It holds DB/Redis/MinIO credentials, JWT secrets and provider keys.
> There is no `.env` in this repo.

## 3. Verify

```bash
# service health + container status
curl -fsS https://api.ziadapos.com/api/v1/health
ssh camel 'cd ~/ziadapos-go/deploy && docker compose ps api'

# quick auth smoke test (use a real account)
curl -s -X POST https://api.ziadapos.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"owner@ziadapos.com","password":"..."}'
```

## 4. Logs

```bash
ssh camel 'docker logs -f --tail 100 ziadapos-go-api-1'
ssh camel 'cd ~/ziadapos-go/deploy && docker compose logs -f api'
```

## 5. Rollback

If the new build is bad, restore the previous source, rebuild, and restart.

```bash
# (optional) if you tagged the previous image:
ssh camel 'docker tag ziadapos-go-api:prev ziadapos-go-api:latest'
ssh camel 'cd ~/ziadapos-go/deploy && docker compose up -d --force-recreate --no-build api'

# otherwise: checkout the previous commit locally, rsync again, rebuild
git log --oneline -5
git checkout <previous-good-commit>
# ... run the Section 1 commands ...
git checkout main
```
