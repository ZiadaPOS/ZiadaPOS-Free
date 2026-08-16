"""Notification tools: in-app alerts for stock, credits, sales, system."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def list_notifications(category: str | None = None,
                                 unread: bool | None = None,
                                 page: int = 1,
                                 page_size: int = 30) -> dict:
        """List in-app notifications. category: stock/credit/sales/ai/system/supplier."""
        return await client.request("GET", "/notifications", params=param(
            category=category, unread=unread, page=page, page_size=page_size))

    @mcp.tool()
    async def unread_notification_count() -> dict:
        """Number of unread notifications."""
        return await client.request("GET", "/notifications/unread-count")

    @mcp.tool()
    async def mark_notification_read(notification_id: str) -> str:
        """Mark one notification as read."""
        await client.request("PATCH", f"/notifications/{notification_id}/read", json={})
        return "Notification marked read"

    @mcp.tool()
    async def mark_all_notifications_read() -> str:
        """Mark every notification as read."""
        await client.request("PATCH", "/notifications/read-all", json={})
        return "All notifications marked read"

    @mcp.tool()
    async def delete_notification(notification_id: str) -> str:
        """Delete one notification."""
        await client.request("DELETE", f"/notifications/{notification_id}")
        return "Notification deleted"
