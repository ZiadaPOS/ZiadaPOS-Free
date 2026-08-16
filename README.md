# ZiadaPOS MCP Agent

A Python **MCP server + embedded AI agent** for [ZiadaPOS](https://github.com/Camel-Creatives/ZiadaPos) — the point-of-sale platform for Tanzanian shop owners. It exposes the entire ZiadaPOS Go API as MCP tools and wraps them in a natural-language agent (`ask`) so the shop owner can run their whole shop in plain Swahili or English:

- *"ongeza bidhaa hizi kwenye stock"* — add products
- *"ni mapato gani wiki iliyopita?"* — sales reports
- *"lipa credit ya Juma"* — record a credit payment
- *"tuma SMS kwa 0628587749 …"* — send a real SMS

## Highlights

- **111 tools** (110 domain tools + the `ask` agent tool) covering: inventory, sales/POS, customers, credit tabs, suppliers & deliveries, expenses, analytics, reports (incl. scheduled + CSV export), notifications, stores, staff, notebook, accounts, **product images** and **SMS/email**.
- **`ask` tool** — an embedded function-calling agent (Ngamia gateway, OpenAI-compatible) that plans multi-step shop tasks, calls the right tools, and answers in the owner's language.
- **Real SMS + email** — sends via the same providers the API uses: SendAfrica (SMS) and MailAfrica (email).
- **Auto product images** — when a product is added without a picture, the agent searches Wikimedia Commons, downloads the best match and uploads it to the shop's MinIO storage.
- **Single service account** — logs in once at startup, transparently refreshes the JWT, unwraps the API `{success, data}` envelope, and auto-switches stores by re-exchanging tokens.
- **HTTP (`streamable-http`) or stdio transport** — choose with `MCP_TRANSPORT`.
- **Swahili/English persona** — replies in the language the owner uses; confirms before destructive actions and before anything that spends SMS/email credits.
- **CI/CD** — lint + tests on push, then auto-deploy to the production VPS (`redeploy.md` is the manual runbook).

## Architecture

```
                ┌──────────────────────────────────────────────┐
                │            Any MCP client                    │
                │  (Claude Desktop, cursor, fastmcp, tests…)   │
                └──────────────────┬───────────────────────────┘
                                   │ MCP protocol
                                   │ HTTP (streamable-http) or stdio
                ┌──────────────────▼───────────────────────────┐
                │                agent/server.py               │
                │        FastMCP("ZiadaPOS") + lifespan        │
                │   ┌───────────────┬─────────────┬──────────┐ │
                │   │ tools/ (110)  │ agent.ask   │  prompts │ │
                │   └───────┬───────┴──────┬──────┴──────────┘ │
                └───────────┼──────────────┼───────────────────┘
                            │              │
                   client.py│              │ NgamiaClient (agent.py)
                   ZiadaClient│           │ /chat/completions
                            │              │
                ┌───────────▼──────────────▼───────────────────┐
                │          ZiadaPOS Go API (REST)              │
                │   auth/ inventory/ pos/ customers/ …         │
                └──────────────────────────────────────────────┘
```

**Source layout** (`src/ziada_mcp/`):

| File | Responsibility |
| --- | --- |
| `config.py` | All settings from `.env` (pydantic-settings), cached singleton |
| `client.py` | `ZiadaClient` — REST client: login, JWT refresh, store switching, envelope unwrapping, file upload |
| `errors.py` | Typed API error hierarchy (`ZiadaAuthError`, `ZiadaValidationError`, …) |
| `notify.py` | `NotifyClient` — SendAfrica SMS + MailAfrica email (used by the communication tools) |
| `prompts.py` | The "Ziada AI" persona / system prompt + list of destructive tools |
| `agent.py` | `NgamiaClient` + the tool-calling loop + the `ask` tool |
| `server.py` | Builds the FastMCP server, registers everything, entry point `ziada-mcp` |
| `tools/` | One module per API domain, each exposing `register(mcp, client)` |

## Quick start

Requires **Python 3.12+** and [uv](https://docs.astral.sh/uv/).

```bash
cd agent
uv sync
cp .env.example .env      # fill in ZIADA_* credentials + NGAMIA_API_KEY
uv run ziada-mcp          # starts the MCP server on :8081 (streamable-http)
```

Then point any MCP client at `http://localhost:8081/mcp` (note: **no trailing slash**) and call the `ask` tool, or use individual tools directly.

stdio mode (for Claude Desktop and other local clients):

```bash
MCP_TRANSPORT=stdio uv run ziada-mcp
```

## Example conversations

```
ask: "Ongeza bidhaa 'Pepsi 500ml', bei 2500, stock 50"
     → agent creates the product, fetches a Wikimedia image for it, and confirms.

ask: "Tuma SMS kwa 0628587749: 'Jumla ya deni lako ni 15,000, lipa kabla ya Ijumaa'"
     → agent sends a real SMS via SendAfrica (after confirming, it's a cost).

ask: "Ni bidhaa gani zinaisha?"            → list_low_stock
ask: "Mapato ya leo ni kiasi gani?"         → sales_summary
ask: "Ongeza credit ya Mwajuma TZS 5000"    → create_customer + complete_sale (Credit)
```

## Tool families (111 tools total)

| Family | Module | Tools |
| --- | --- | --- |
| Inventory & products | `tools/inventory.py` | 13 |
| Sales / POS transactions | `tools/transactions.py` | 7 |
| Customers | `tools/customers.py` | 5 |
| Credit tabs & reminders | `tools/credits.py` | 7 |
| Staff | `tools/staff.py` | 10 |
| Suppliers, deliveries & payments | `tools/suppliers.py` | 13 |
| Expenses | `tools/expenses.py` | 6 |
| Reports (incl. scheduled, CSV) | `tools/reports.py` | 9 |
| Analytics / BI | `tools/analytics.py` | 10 |
| Notifications | `tools/notifications.py` | 5 |
| Stores | `tools/stores.py` | 7 |
| Accounts / auth / org / users | `tools/accounts.py` | 8 |
| SMS / email | `tools/communicate.py` | 4 |
| Product images | `tools/images.py` | 1 |
| Notebook | `tools/notebook.py` | 5 |
| **AI agent** | `agent.py` | 1 (`ask`) |

Every tool is documented in **[docs/04-tools.md](docs/04-tools.md)**.

## Configuration

Everything is read from `.env` — see `.env.example` for the full list. The key groups:

| Group | Vars |
| --- | --- |
| API connection | `ZIADA_API_URL`, `ZIADA_IDENTIFIER`, `ZIADA_PASSWORD`, `ZIADA_STORE_ID` |
| AI gateway | `NGAMIA_BASE_URL`, `NGAMIA_API_KEY`, `NGAMIA_MODEL` |
| Agent limits | `AGENT_MAX_STEPS`, `AGENT_TIMEOUT_SECONDS` |
| SMS / email | `SENDAFRICA_*`, `MAILAFRICA_*` |
| Transport | `MCP_TRANSPORT`, `MCP_HOST`, `MCP_PORT`, `LOG_LEVEL` |

Full reference: **[docs/02-configuration.md](docs/02-configuration.md)**.

## Testing

```bash
uv sync --group dev
uvx ruff check src/ziada_mcp tests
uv run pytest -q        # 6 tests: registration, agent loop, tool schema, phone normalization
```

## Deployment

- **Local/dev**: `uv run ziada-mcp` (see `docker-compose.yml` for a one-container local stack).
- **Production**: a Docker container behind nginx at `https://agent.ziadapos.com/mcp`, built and deployed to the VPS automatically by CI/CD on every push to `main`. Manual restarts and `.env` changes are covered in **[redeploy.md](redeploy.md)** and **[docs/08-deployment.md](docs/08-deployment.md)**.

## More docs

- **[docs/00-index.md](docs/00-index.md)** — all documentation
- **[docs/01-architecture.md](docs/01-architecture.md)** — system context, auth flow, request flow
- **[docs/03-client-and-auth.md](docs/03-client-and-auth.md)** — how the API client and JWT refresh work
- **[docs/05-agent-and-persona.md](docs/05-agent-and-persona.md)** — the `ask` agent, Ngamia, safety rules
- **[docs/06-product-images.md](docs/06-product-images.md)** — the Wikimedia → MinIO image pipeline
- **[docs/07-communication.md](docs/07-communication.md)** — SendAfrica / MailAfrica integration
