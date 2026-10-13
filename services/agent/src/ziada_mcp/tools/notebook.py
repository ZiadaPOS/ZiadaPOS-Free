"""Notebook tools: quick notes and reminders for the shop."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def list_notes(tag: str | None = None,
                         search: str | None = None,
                         ordering: str | None = None) -> dict:
        """List the store's notes. ordering: created/updated/title (prepend - for desc)."""
        return await client.request("GET", "/notebook", params=param(
            tag=tag, search=search, ordering=ordering))

    @mcp.tool()
    async def create_note(title: str, content: str = "",
                          tags: list[str] | None = None) -> dict:
        """Create a note (e.g. a reminder like 'restock sugar on Friday')."""
        return await client.request("POST", "/notebook", json={
            "title": title, "content": content, "tags": tags or [],
        })

    @mcp.tool()
    async def get_note(note_id: str) -> dict:
        """Get one note."""
        return await client.request("GET", f"/notebook/{note_id}")

    @mcp.tool()
    async def update_note(note_id: str, title: str | None = None,
                          content: str | None = None,
                          tags: list[str] | None = None) -> dict:
        """Update a note. Only provided fields change."""
        body: dict[str, Any] = {}
        if title is not None:
            body["title"] = title
        if content is not None:
            body["content"] = content
        if tags is not None:
            body["tags"] = tags
        return await client.request("PATCH", f"/notebook/{note_id}", json=body)

    @mcp.tool()
    async def delete_note(note_id: str) -> str:
        """Delete a note."""
        await client.request("DELETE", f"/notebook/{note_id}")
        return "Note deleted"
