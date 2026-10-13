"""Store tools: the shop's branches, status and weekly performance."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def list_stores(status: str | None = None) -> dict:
        """List the business's stores. status: open/closed/paused."""
        return await client.request("GET", "/stores", params=param(status=status))

    @mcp.tool()
    async def get_store(store_id: str) -> dict:
        """Get one store's details."""
        return await client.request("GET", f"/stores/{store_id}")

    @mcp.tool()
    async def store_stats() -> dict:
        """Org-wide store stats (sales by branch)."""
        return await client.request("GET", "/stores/stats")

    @mcp.tool()
    async def set_store_status(store_id: str, status: str) -> dict:
        """Open, close or pause a store. status: open/closed/paused."""
        return await client.request("PATCH", f"/stores/{store_id}/status", json={"status": status})

    @mcp.tool()
    async def store_weekly_performance(store_id: str) -> dict:
        """Weekly performance breakdown for one store."""
        return await client.request("GET", f"/stores/{store_id}/week")

    @mcp.tool()
    async def store_staff(store_id: str) -> dict:
        """Staff roster working at one store."""
        return await client.request("GET", f"/stores/{store_id}/staff")

    @mcp.tool()
    async def switch_active_store(store_id: str) -> dict:
        """Switch the agent's active store. All future stock/sales/analytics calls act on this store."""
        return await client.switch_store(store_id)
