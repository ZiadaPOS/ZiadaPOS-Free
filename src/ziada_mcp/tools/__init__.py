"""MCP tool registry — one module per ZiadaPOS API domain.

Each module exposes ``register(mcp, client)`` which decorates async
functions as FastMCP tools. ``register_all`` wires every domain onto the
server.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from . import (
    accounts,
    analytics,
    communicate,
    credits,
    customers,
    expenses,
    images,
    inventory,
    notebook,
    notifications,
    reports,
    staff,
    stores,
    suppliers,
    transactions,
)

if TYPE_CHECKING:
    from fastmcp import FastMCP

    from ..notify import NotifyClient

_MODULES = [
    accounts,
    analytics,
    credits,
    customers,
    expenses,
    images,
    inventory,
    notebook,
    notifications,
    reports,
    staff,
    stores,
    suppliers,
    transactions,
]


def register_all(mcp: FastMCP, client: Any, notify: NotifyClient) -> int:
    """Register every domain tool on the server. Returns the tool count."""
    for mod in _MODULES:
        mod.register(mcp, client)
    communicate.register(mcp, client, notify)
    return len(mcp._tool_manager._tools)  # type: ignore[attr-defined]
