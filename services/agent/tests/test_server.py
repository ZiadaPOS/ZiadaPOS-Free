import pytest


@pytest.fixture(scope="module")
def mcp():
    from ziada_mcp.server import build_server

    return build_server()


def test_all_domain_tools_registered(mcp):
    import asyncio

    tools = asyncio.run(mcp._tool_manager.get_tools())  # type: ignore[attr-defined]
    assert len(tools) >= 109
    assert "ask" in tools
    for name in (
        "list_products",
        "adjust_stock",
        "complete_sale",
        "sales_summary",
        "credit_customers",
        "record_credit_payment",
        "create_expense",
        "analytics_overview",
        "generate_report",
        "list_notifications",
        "list_stores",
        "create_staff",
        "get_me",
        "send_sms",
        "send_email",
        "sms_balance",
        "email_balance",
    ):
        assert name in tools, f"missing tool {name}"


def test_prompts_registered(mcp):
    import asyncio

    prompts = asyncio.run(mcp.get_prompts())
    assert "pos_help" in prompts


def test_openai_tool_schema_well_formed():
    from ziada_mcp.agent import _to_openai_tool

    spec = _to_openai_tool("add", "Add two numbers", {
        "type": "object",
        "properties": {"a": {"type": "integer"}},
        "required": ["a"],
    })
    assert spec["type"] == "function"
    assert spec["function"]["name"] == "add"
    assert spec["function"]["parameters"]["required"] == ["a"]


@pytest.mark.asyncio
async def test_agent_loop_tool_call_then_answer():
    """The agent should execute a tool call, feed the result back, and
    return the final assistant message."""
    from fastmcp import FastMCP

    from ziada_mcp.agent import run_agent

    mcp = FastMCP("t")

    @mcp.tool()
    async def add(a: int, b: int) -> int:
        """Add two numbers."""
        return a + b

    calls = []

    class FakeNgamia:
        async def chat(self, messages, tools):
            calls.append(messages)
            if len(messages) == 2:  # system + user
                return {
                    "choices": [{
                        "message": {
                            "role": "assistant",
                            "content": None,
                            "tool_calls": [{
                                "id": "call_1",
                                "type": "function",
                                "function": {"name": "add", "arguments": '{"a": 1, "b": 2}'},
                            }],
                        }
                    }]
                }
            return {"choices": [{"message": {
                "role": "assistant", "content": "Jibu ni 3."
            }}]}

    class FakeClient:
        store_name = "Duka La Maziwa"

    answer = await run_agent(mcp, FakeClient(), FakeNgamia(), "What is 1+2?")
    assert answer == "Jibu ni 3."
    # tool result must have been fed back
    assert any(m.get("role") == "tool" for m in calls[-1])


@pytest.mark.asyncio
async def test_agent_loop_handles_tool_error():
    from fastmcp import FastMCP

    from ziada_mcp.agent import run_agent

    mcp = FastMCP("t")

    @mcp.tool()
    async def boom() -> str:
        """Explodes."""
        raise ValueError("kaboom")

    class FakeNgamia:
        async def chat(self, messages, tools):
            if len(messages) == 2:
                return {"choices": [{"message": {
                    "role": "assistant", "content": None,
                    "tool_calls": [{
                        "id": "c1", "type": "function",
                        "function": {"name": "boom", "arguments": "{}"},
                    }],
                }}]}
            return {"choices": [{"message": {
                "role": "assistant", "content": "Kuna tatizo."}}]}

    class FakeClient:
        store_name = "Duka"

    answer = await run_agent(mcp, FakeClient(), FakeNgamia(), "do it")
    assert answer == "Kuna tatizo."


@pytest.mark.asyncio
async def test_ask_bridge_registered_on_real_server(mcp):
    """The real server must expose the /ask HTTP bridge route."""
    routes = mcp._additional_http_routes  # type: ignore[attr-defined]
    paths = [getattr(r, "path", None) for r in routes]
    assert "/ask" in paths


@pytest.mark.asyncio
async def test_ask_bridge_http_endpoint():
    """POST /ask must answer a plain-HTTP question for mobile clients."""
    import httpx
    from fastmcp import FastMCP
    from starlette.requests import Request
    from starlette.responses import JSONResponse

    mcp = FastMCP("t")

    @mcp.tool()
    async def ask(question: str) -> str:
        """Talk to Ziada AI."""
        return f"Answer to: {question}"

    @mcp.custom_route("/ask", methods=["POST"])
    async def ask_bridge(request: Request) -> JSONResponse:
        body = await request.json()
        question = (body or {}).get("question", "")
        if not question or not str(question).strip():
            return JSONResponse({"error": "question is required"}, status_code=400)
        tool = await mcp.get_tool("ask")
        answer = await tool.fn(question=str(question).strip())
        return JSONResponse({"answer": answer})

    app = mcp.http_app()
    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(
        transport=transport, base_url="http://testserver"
    ) as client:
        r = await client.post("/ask", json={"question": "ni mapato gani?"})
        assert r.status_code == 200, r.text
        assert r.json() == {"answer": "Answer to: ni mapato gani?"}

        r = await client.post("/ask", json={})
        assert r.status_code == 400, r.text


def test_normalize_phone():
    from ziada_mcp.notify import normalize_phone

    assert normalize_phone("0628587749") == "0628587749"
    assert normalize_phone("+255628587749") == "0628587749"
    assert normalize_phone("255628587749") == "0628587749"
    assert normalize_phone("712345678") == "0712345678"
    with pytest.raises(ValueError):
        normalize_phone("1122")
