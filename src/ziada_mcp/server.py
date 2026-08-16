"""MCP server entry point — wires config, client, tools and the agent, then
runs over HTTP or stdio."""

from __future__ import annotations

import logging
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastmcp import FastMCP

from .agent import register_agent
from .client import ZiadaClient
from .config import get_settings
from .notify import NotifyClient
from .tools import register_all


def build_server() -> FastMCP:
    settings = get_settings()
    logging.basicConfig(
        level=settings.log_level.upper(),
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )
    log = logging.getLogger("ziada-mcp")

    client = ZiadaClient()

    @asynccontextmanager
    async def lifespan(server: FastMCP) -> AsyncIterator[None]:
        """Authenticate on startup, tear down on shutdown."""
        await client.start()
        try:
            yield
        finally:
            await client.aclose()

    mcp = FastMCP(
        "ZiadaPOS",
        instructions=(
            "Assistant that helps the ZiadaPOS shop owner run their shop: stock, "
            "sales, credit tabs, suppliers, expenses, reports and staff."
        ),
        lifespan=lifespan,
    )

    count = register_all(mcp, client, NotifyClient(settings))
    register_agent(mcp, client, settings)
    log.info("Registered %d tools + ask agent (store %s)", count, client.store_name)

    @mcp.prompt()
    def pos_help() -> str:
        """Quick-start prompt for the POS assistant."""
        return "Habari! Nisaidie kusimamia duka langu — unaweza kuona stock, mauzo na credit za wateja. Ni nini unahitaji leo?"

    return mcp


def main() -> None:
    settings = get_settings()
    mcp = build_server()
    kwargs: dict = {}
    if settings.mcp_transport == "http":
        kwargs = {"host": settings.mcp_host, "port": settings.mcp_port}
    mcp.run(transport=settings.mcp_transport, **kwargs)


if __name__ == "__main__":
    main()
