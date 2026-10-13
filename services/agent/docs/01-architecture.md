# 01 — Architecture

## System context

```
                    ┌──────────────────────────────┐
                    │         MCP client          │
                    │  (Claude Desktop, cursor,    │
                    │   fastmcp, home automation,  │
                    │   a mobile app, …)           │
                    └──────────────┬───────────────┘
                                   │  MCP protocol
                                   │  streamable-http (default) or stdio
                    ┌──────────────▼───────────────┐
                    │   agent/server.py            │
                    │   FastMCP("ZiadaPOS")        │
                    │                              │
                    │  tools/      agent/ask       │
                    │  110 tools   (Ngamia loop)   │
                    └──────┬──────────────┬────────┘
                           │              │
                 ZiadaClient│              │NgamiaClient
                 (REST, JWT)│              │(OpenAI-compatible)
                           │              │
        ┌──────────────────▼──────────────▼────────────────────┐
        │              ZiadaPOS Go API (REST)                 │
        │  /auth /accounts /inventory /pos /customers /credits│
        │  /staff /suppliers /expenses /reports /analytics    │
        │  /notifications /stores /notebook /uploads          │
        └──────────────────────────┬──────────────────────────┘
                                   │ external providers
              ┌────────────────────┼───────────────────────┐
              │                    │                       │
    ┌─────────▼───────┐  ┌─────────▼────────┐   ┌──────────▼──────────┐
    │   Ngamia        │  │  SendAfrica      │   │   Wikimedia Commons │
    │   (AI gateway)  │  │  (SMS)           │   │   (product images)  │
    └─────────────────┘  └──────────────────┘   └─────────────────────┘
                          ┌──────────────────┐   ┌─────────────────────┐
                          │  MailAfrica      │   │  MinIO (via API)    │
                          │  (email)         │   │  product images     │
                          └──────────────────┘   └─────────────────────┘
```

The agent is a **stateless MCP server**. All business data lives in the ZiadaPOS Go API + its database.
The agent holds **one service account** (a shop owner account) and speaks on its behalf.

## Source layout

```
agent/
├── pyproject.toml            package metadata, deps, console script "ziada-mcp"
├── uv.lock                   locked dependency versions
├── .env.example              template for all environment variables
├── Dockerfile                multi-stage build (uv → wheel → slim runtime)
├── docker-compose.yml        local one-container stack
├── redeploy.md               production runbook
├── .github/workflows/deploy.yml  CI/CD (ruff + pytest → rsync + docker deploy)
├── src/ziada_mcp/
│   ├── config.py             Settings (pydantic-settings) + get_settings() cache
│   ├── client.py             ZiadaClient — the REST + auth layer
│   ├── errors.py             ZiadaError hierarchy + envelope mapping
│   ├── notify.py             NotifyClient — SendAfrica / MailAfrica
│   ├── prompts.py            persona + destructive-tool list
│   ├── agent.py              NgamiaClient + run_agent loop + ask tool
│   ├── server.py             build_server(), lifespan, entry point
│   └── tools/
│       ├── __init__.py       register_all() — 15 domain modules
│       ├── _base.py          tzs(), today(), param() helpers
│       ├── accounts.py       8 tools
│       ├── analytics.py      10 tools
│       ├── communicate.py    4 tools (SMS/email)
│       ├── credits.py        7 tools
│       ├── customers.py      5 tools
│       ├── expenses.py       6 tools
│       ├── images.py         1 tool
│       ├── inventory.py      13 tools
│       ├── notebook.py       5 tools
│       ├── notifications.py  5 tools
│       ├── reports.py        9 tools
│       ├── staff.py          10 tools
│       ├── stores.py         7 tools
│       ├── suppliers.py      13 tools
│       └── transactions.py   7 tools
└── tests/
    └── test_server.py        6 tests
```

## How a tool call flows

1. An MCP client invokes a tool, e.g. `create_product`.
2. FastMCP resolves it to the `async def` handler in `tools/inventory.py`.
3. The handler builds query params with the `param()` helper (drops `None`/blank values)
   and calls `client.request(method, path, json=…)`.
4. `ZiadaClient` attaches `Authorization: Bearer <access_token>` and sends the request.
   On HTTP 401 it refreshes the token once (under a lock) and retries once.
5. The response body is unwrapped from the API envelope `{success, data, meta}` — the handler
   gets the `data` payload (or a typed `ZiadaError` is raised).
6. FastMCP serialises the result back to the client.

## How the `ask` agent flows

1. The client calls `ask(question)`.
2. `run_agent()` builds a message list: system prompt (`build_system_prompt`) + last 12
   history turns + the question.
3. It calls the Ngamia chat-completions endpoint with **every registered MCP tool** converted
   to OpenAI function schemas and `tool_choice: "auto"`.
4. Loop (max `AGENT_MAX_STEPS`): if the model returns text, that is the answer. If it returns
   `tool_calls`, the agent executes each tool locally and feeds the results back as
   `role: tool` messages, then asks the model again.
5. Tool errors are caught and returned to the model as text so it can recover gracefully.

## Auth model

- The Go API embeds `org_id` + `store_id` inside the JWT.
- The agent logs in once at startup (`start()` → `POST /auth/login`), loads `/accounts/me`,
  and optionally `switch_store()` if `ZIADA_STORE_ID` is set.
- Access tokens (~15 min TTL) are refreshed via `POST /auth/refresh` on the first 401.
- Switching stores re-exchanges tokens (`POST /accounts/switch-store`) because the new store
  must be inside the JWT.

## Timezone

All "today" logic uses **Africa/Dar_es_Salaam** (no DST). Money is handled in **TZS** integers.

## Related docs

- [02 — Configuration](02-configuration.md)
- [03 — Client & Auth](03-client-and-auth.md)
- [05 — Agent & Persona](05-agent-and-persona.md)
