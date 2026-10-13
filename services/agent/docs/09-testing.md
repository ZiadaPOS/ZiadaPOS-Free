# 09 — Testing

## Running

```bash
uv sync --group dev
uv run pytest -q
```

Lint:

```bash
uvx ruff check src/ziada_mcp tests
```

## What is covered (`tests/test_server.py`, 6 tests)

| Test | Verifies |
| --- | --- |
| `test_all_domain_tools_registered` | Building the server registers ≥109 tools, the `ask` tool, and 17 representative tools across every domain. |
| `test_prompts_registered` | The `pos_help` MCP prompt is registered. |
| `test_openai_tool_schema_well_formed` | `_to_openai_tool` produces a valid OpenAI function schema from an MCP tool spec. |
| `test_agent_loop_tool_call_then_answer` | The agent loop executes a tool call and feeds the `tool` result back to the model before answering. Uses a fake Ngamia client (no network). |
| `test_agent_loop_handles_tool_error` | A raising tool is converted into an error string and the agent still finishes. |
| `test_normalize_phone` | Tanzanian phone normalisation (and rejection of invalid numbers). |

## Live smoke test

With a running API and valid `.env`:

```python
import asyncio
from fastmcp import Client

async def main():
    async with Client("http://localhost:8081/mcp") as c:   # no trailing slash
        tools = await c.list_tools()
        print("tools:", len(tools))
        r = await c.call_tool("ask", {"question": "Kuna bidhaa ngapi?"})
        print(r.content[0].text)

asyncio.run(main())
```

## CI

The same checks run automatically in GitHub Actions on every push to `main` before the deploy job
(see [08 — Deployment](08-deployment.md)).
