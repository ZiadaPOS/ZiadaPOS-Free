# ZiadaPOS MCP Agent

A Python **MCP server + embedded AI agent** for [ZiadaPOS](https://github.com/Camel-Creatives/ZiadaPos) — the POS for Tanzanian shop owners. It exposes the whole ZiadaPOS Go API as MCP tools and wraps them in an agent (`ask`) so the shop owner can run their shop in plain Swahili or English: *"ongeza bidhaa hizi kwenye stock"*, *"ni mapato gani wiki iliyopita?"*, *"lipa credit ya Juma"*.

## Features

- **110 tools** across: inventory, sales/transactions, customers, credit tabs, suppliers/deliveries, expenses, analytics, reports (incl. scheduled + CSV), notifications, stores, staff, notebook, accounts, and **SMS/email**.
- **`ask` tool** — an agent loop that calls the tools for you. Powered by the [Ngamia](https://ngamia.cc) gateway (OpenAI-compatible function calling).
- **Sends real SMS + email** to any phone/address via the same providers the API uses — [SendAfrica](https://docs.sendafrica.online) (SMS) and [MailAfrica](https://docs.mailafrica.online) (email) — so "tuma SMS kwa 0628587749…" works exactly like sending email.
- **Single service account** auth: logs in once at startup, auto-refreshes the JWT on expiry, transparently unwraps the API envelope.
- **HTTP (`streamable-http`) or stdio transport** — pick with `MCP_TRANSPORT`.
- Swahili/English bilingual persona; confirms before destructive actions and before spending SMS/email credits.

## Architecture

```
src/ziada_mcp/
  config.py      settings from .env (pydantic-settings)
  client.py      ZiadaPOS REST client (login, refresh, switch-store, envelope unwrap)
  errors.py      API error mapping
  notify.py      SendAfrica SMS + MailAfrica email clients (used by tools)
  prompts.py     agent persona / system prompt
  agent.py       Ngamia function-calling loop + ask tool
  server.py      FastMCP server entry (HTTP or stdio)
  tools/         one module per API domain, each exposing register(mcp, client)
```

The Go API embeds `store_id`/`org_id` in the JWT, so switching stores re-exchanges tokens (`POST /accounts/switch-store`) — wired through `client.switch_store`.

## Setup

Requires Python 3.12+ and [uv](https://docs.astral.sh/uv/).

```bash
cd agent
uv sync
cp .env.example .env   # fill in credentials + Ngamia key
```

## Run

```bash
# HTTP server (default): exposes the MCP endpoint for MCP clients
uv run ziada-mcp

# stdio (for Claude Desktop / other MCP clients)
MCP_TRANSPORT=stdio uv run ziada-mcp
```

Point an MCP client at the HTTP endpoint (default `http://localhost:8081/mcp/`) and ask the `ask` tool anything — or use any individual tool directly.

## Test

```bash
uv run pytest
```

## Env vars

| Variable | Meaning |
| --- | --- |
| `ZIADA_API_URL` | ZiadaPOS API base, e.g. `http://localhost:8080/api/v1` |
| `ZIADA_IDENTIFIER` / `ZIADA_PASSWORD` | shop owner (service account) login |
| `ZIADA_STORE_ID` | optional; force a specific store |
| `NGAMIA_BASE_URL` / `NGAMIA_API_KEY` / `NGAMIA_MODEL` | AI gateway |
| `SENDAFRICA_*` | SendAfrica SMS key/sender (used by `send_sms`) |
| `MAILAFRICA_*` | MailAfrica email key/domain (used by `send_email`) |
| `MCP_TRANSPORT` / `MCP_HOST` / `MCP_PORT` | `http` or `stdio` |
| `AGENT_MAX_STEPS` / `AGENT_TIMEOUT_SECONDS` | agent loop limits |

See `.env.example` for everything.
