# 05 — Agent & Persona

The `ask` tool is the heart of the product: an **embedded function-calling agent** that turns a
plain-language request into the right sequence of MCP tool calls.

## Model provider — Ngamia

The agent calls an OpenAI-compatible `/chat/completions` endpoint on the **Ngamia gateway**
(see `agent.py`, `NgamiaClient`):

- URL: `{NGAMIA_BASE_URL}/chat/completions`
- Auth: `Authorization: Bearer {NGAMIA_API_KEY}`
- Body: `{model, messages, tools, tool_choice: "auto", temperature: 0.2}`
- Model default: `openai/gpt-4o-mini` (`NGAMIA_MODEL`).
- Timeout: `AGENT_TIMEOUT_SECONDS` (default 120).

## The `ask` flow

`register_agent()` converts **every registered MCP tool** into an OpenAI function schema
(`_to_openai_tool`) and `run_agent()` executes the loop:

1. **Build messages**
   - System prompt: `build_system_prompt(client)` — the persona filled in with the store name and
     today's date (Africa/Dar_es_Salaam).
   - Last 12 user/assistant history turns (kept within one session by FastMCP).
   - The user's question.
2. **Iterate** (up to `AGENT_MAX_STEPS` = 8):
   - Ask the model for the next step.
   - **No tool calls** → the reply text is the final answer; return it.
   - **Tool calls** → execute each one locally:
     - arguments are parsed as JSON (fallback `{}` if malformed),
     - unknown tool name → error string,
     - exceptions are caught and returned to the model as `"Error calling {name}: {err}"`,
     - tool output is JSON-serialised and truncated to 8000 chars,
   - append the assistant `tool_calls` message and one `role: tool` result per call, loop.
3. If the step limit is hit without a final answer, return a graceful
   "couldn't finish within the step limit" message.

## Persona (`prompts.py`)

**"Ziada AI"** — the shop owner's assistant:

- Bilingual **Swahili + English**; answers in the language the owner used.
- Always uses the tools for live data — never invents figures.
- Money in **TZS with thousands separators** (e.g. `TZS 12,500`).
- Sign-off: "Ziada AI".
- Read tasks → read tools; write tasks → write tools; confirm after writes.
- For unknown IDs → search first, then act.
- **Product images**: when adding a product without an image, call
  `suggest_product_image` automatically.
- **Safety**: before any refund/void, write-off, delete/deactivate, or any SMS/email send,
  confirm with the owner first.

### Destructive / guarded tools

`destructive_tool_names()` returns the set of tools the persona must not fire without explicit
confirmation (full list in [04 — Tools](04-tools.md) → "Destructive tools").

## Prompt & quick start

The server also registers an MCP **prompt** named `pos_help` — a Swahili quick-start prompt that
MCP clients can inject to help a new owner get going.
