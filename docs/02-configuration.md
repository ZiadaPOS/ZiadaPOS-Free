# 02 — Configuration

All configuration comes from the environment (loaded from `.env` by
[pydantic-settings](https://docs.pydantic.dev/latest/concepts/pydantic_settings/)). Copy
`.env.example` → `.env` and fill in the values.

| Env var | Field | Default | Purpose |
| --- | --- | --- | --- |
| `ZIADA_API_URL` | `ziada_api_url` | `http://localhost:8080/api/v1` | Base URL of the ZiadaPOS Go API. |
| `ZIADA_IDENTIFIER` | `ziada_identifier` | `""` | Login identifier of the service-account (email or phone). |
| `ZIADA_PASSWORD` | `ziada_password` | `""` | Password of the service account. |
| `ZIADA_STORE_ID` | `ziada_store_id` | `""` | Optional: force the agent onto one store. Empty = account's current store. |
| `NGAMIA_BASE_URL` | `ngamia_base_url` | `https://api.ngamia.cc/v1` | AI gateway (OpenAI-compatible chat completions). |
| `NGAMIA_API_KEY` | `ngamia_api_key` | `""` | API key for the AI gateway. |
| `NGAMIA_MODEL` | `ngamia_model` | `openai/gpt-4o-mini` | Model id used by the `ask` agent. |
| `AGENT_MAX_STEPS` | `agent_max_steps` | `8` | Max tool-calling iterations per `ask` request. |
| `AGENT_TIMEOUT_SECONDS` | `agent_timeout_seconds` | `120` | Per-request timeout to the AI gateway. |
| `SENDAFRICA_BASE_URL` | `sendafrica_base_url` | `https://api.sendafrica.online` | SendAfrica SMS API base. |
| `SENDAFRICA_API_KEY` | `sendafrica_api_key` | `""` | SendAfrica API key (`SA-…`). |
| `SENDAFRICA_SENDER_ID` | `sendafrica_sender_id` | `SendAfrika` | Sender id shown to the recipient. |
| `MAILAFRICA_BASE_URL` | `mailafrica_base_url` | `https://api.mailafrica.online` | MailAfrica email API base. |
| `MAILAFRICA_API_KEY` | `mailafrica_api_key` | `""` | MailAfrica API key (`MAIL_…`). |
| `MAILAFRICA_FROM_DOMAIN` | `mailafrica_from_domain` | `""` | Verified sending domain (e.g. `ziadapos.com`). |
| `MAILAFRICA_FROM_ADDRESS` | `mailafrica_from_address` | `""` | From address (e.g. `noreply@ziadapos.com`). |
| `MAILAFRICA_FROM_NAME` | `mailafrica_from_name` | `ZiadaPOS` | Display name for outgoing email. |
| `MCP_TRANSPORT` | `mcp_transport` | `http` | `http` (streamable-http) or `stdio`. |
| `MCP_HOST` | `mcp_host` | `0.0.0.0` | Bind address for the HTTP transport. |
| `MCP_PORT` | `mcp_port` | `8081` | Bind port for the HTTP transport. |
| `LOG_LEVEL` | `log_level` | `INFO` | Logging level. |

## Required vs optional

- **Required to start**: `ZIADA_API_URL`, `ZIADA_IDENTIFIER`, `ZIADA_PASSWORD`
  (the client refuses to start without credentials), plus `NGAMIA_API_KEY` if you want to use `ask`.
- **Required for SMS**: `SENDAFRICA_API_KEY`. Optional: `SENDAFRICA_SENDER_ID`.
- **Required for email**: `MAILAFRICA_API_KEY`, `MAILAFRICA_FROM_DOMAIN` (a **verified** domain
  on MailAfrica). Optional: `MAILAFRICA_FROM_ADDRESS`, `MAILAFRICA_FROM_NAME`.
- **Optional everywhere else.**

If an SMS/email key is missing, the corresponding tools raise a "not configured" error instead of
crashing the server.

## Security notes

- **Never commit `.env`** — it contains live credentials. It is gitignored.
- In production the container loads `.env` via `env_file:` at **creation time**. After editing
  `.env` you must recreate the container (`docker compose up -d --force-recreate <svc>`), a plain
  restart is not enough. See [redeploy.md](../redeploy.md).
- The agent's ZiadaPOS account is a normal shop-owner account. Abusive logins trigger the API's
  15-minute rate-lock (`login:lock:<identifier>` in Redis) — see [08 — Deployment](08-deployment.md).
