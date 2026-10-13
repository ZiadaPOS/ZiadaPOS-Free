"""Embedded Ziada AI agent — a `ask` MCP tool that runs an OpenAI-compatible
function-calling loop against the Ngamia gateway, executing the other MCP
tools on the shop owner's behalf."""

from __future__ import annotations

import json
from typing import Any

import httpx

from .config import Settings
from .prompts import build_system_prompt


class NgamiaClient:
    """Minimal OpenAI-compatible chat-completions client for Ngamia."""

    def __init__(self, settings: Settings) -> None:
        self._base_url = settings.ngamia_base_url.rstrip("/")
        self._api_key = settings.ngamia_api_key
        self._model = settings.ngamia_model
        self._timeout = settings.agent_timeout_seconds

    async def chat(self, messages: list[dict], tools: list[dict]) -> dict:
        payload: dict[str, Any] = {
            "model": self._model,
            "messages": messages,
            "tools": tools,
            "tool_choice": "auto",
            "temperature": 0.2,
        }
        async with httpx.AsyncClient(timeout=self._timeout) as http:
            r = await http.post(
                f"{self._base_url}/chat/completions",
                headers={
                    "Authorization": f"Bearer {self._api_key}",
                    "Content-Type": "application/json",
                },
                json=payload,
            )
            r.raise_for_status()
            return r.json()


def _to_openai_tool(name: str, description: str, input_schema: dict) -> dict:
    return {
        "type": "function",
        "function": {
            "name": name,
            "description": description or name,
            "parameters": input_schema,
        },
    }


def _format_tool_result(payload: Any) -> str:
    text = json.dumps(payload, ensure_ascii=False, default=str)
    return text if len(text) <= 8000 else text[:8000] + "...[truncated]"


def _extract_choice(data: dict) -> dict:
    choices = data.get("choices") or []
    if not choices:
        raise RuntimeError(
            f"Agent gateway returned no choices: {json.dumps(data, default=str)[:500]}"
        )
    return choices[0].get("message", {})


async def run_agent(
    mcp: Any,
    client: Any,
    ngamia: NgamiaClient,
    question: str,
    history: list[dict] | None = None,
    max_steps: int | None = None,
) -> str:
    """Answer the owner's question by calling the Ngamia gateway in a
    function-calling loop over the registered MCP tools."""
    tools = await mcp._tool_manager.get_tools()  # type: ignore[attr-defined]
    openai_tools = [
        _to_openai_tool(name, tool.description, tool.to_mcp_tool().inputSchema)
        for name, tool in tools.items()
    ]
    if not openai_tools:
        raise RuntimeError("No MCP tools registered; cannot run the agent.")

    messages: list[dict] = [
        {"role": "system", "content": build_system_prompt(client)},
        *[m for m in (history or []) if m.get("role") in ("user", "assistant")][-12:],
        {"role": "user", "content": question},
    ]

    steps = 0
    limit = max_steps or 8
    while steps < limit:
        steps += 1
        data = await ngamia.chat(messages, openai_tools)
        message = _extract_choice(data)

        tool_calls = message.get("tool_calls")
        if not tool_calls:
            return (message.get("content") or "").strip()

        messages.append({"role": "assistant", "content": message.get("content") or "",
                         "tool_calls": tool_calls})

        for tc in tool_calls:
            fn = tc.get("function", {})
            name, raw_args = fn.get("name", ""), fn.get("arguments") or "{}"
            try:
                args = json.loads(raw_args) if isinstance(raw_args, str) else raw_args
            except json.JSONDecodeError:
                args = {}
            tool = tools.get(name)
            if tool is None:
                result = f"Error: unknown tool '{name}'"
            else:
                try:
                    result = _format_tool_result(await tool.fn(**args))
                except Exception as exc:  # noqa: BLE001 — surface to the model
                    result = f"Error calling {name}: {exc}"
            messages.append({"role": "tool", "tool_call_id": tc.get("id", ""),
                             "content": result})

    return ("I couldn't finish that within the step limit. "
            "Let me know if you'd like me to continue.")


def register_agent(mcp: Any, client: Any, settings: Settings) -> None:
    """Register the `ask` tool that wraps the agent loop."""
    ngamia = NgamiaClient(settings)

    @mcp.tool()
    async def ask(question: str) -> str:
        """Talk to Ziada AI: answer questions and run shop tasks in plain
        language (Swahili or English). Example: 'ongeza bidhaa hizi kwenye stock'
        or 'ni mapato gani wiki iliyopita?'"""
        return await run_agent(
            mcp, client, ngamia, question, max_steps=settings.agent_max_steps
        )
