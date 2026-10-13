"""Customer tools: the people who buy from the shop."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def list_customers(search: str | None = None,
                             segment: str | None = None,
                             has_credit: bool | None = None,
                             page: int = 1,
                             page_size: int = 20) -> dict:
        """List customers. segment: New/Regular/VIP/Occasional; has_credit: only customers with an open balance."""
        return await client.request("GET", "/customers", params=param(
            search=search, segment=segment, has_credit=has_credit,
            page=page, page_size=page_size))

    @mcp.tool()
    async def create_customer(name: str, phone: str | None = None,
                              email: str | None = None,
                              segment: str | None = None,
                              credit_limit: int | None = None,
                              notes: str | None = None) -> dict:
        """Register a new customer. Phone is required for SMS credit reminders. credit_limit in TZS."""
        return await client.request("POST", "/customers", json={
            "name": name, "phone": phone or None, "email": email or None,
            "segment": segment or None, "credit_limit": credit_limit,
            "notes": notes or None,
        })

    @mcp.tool()
    async def get_customer(customer_id: str) -> dict:
        """Get one customer's profile."""
        return await client.request("GET", f"/customers/{customer_id}")

    @mcp.tool()
    async def update_customer(customer_id: str, name: str | None = None,
                              phone: str | None = None, email: str | None = None,
                              segment: str | None = None,
                              credit_limit: int | None = None,
                              notes: str | None = None,
                              is_active: bool | None = None) -> dict:
        """Update a customer. Only provided fields change. Pass credit_limit=-1 to clear it."""
        body: dict[str, Any] = {}
        for k, v in {
            "name": name, "phone": phone, "email": email, "segment": segment,
            "credit_limit": credit_limit, "notes": notes, "is_active": is_active,
        }.items():
            if v is not None:
                body[k] = v
        return await client.request("PATCH", f"/customers/{customer_id}", json=body)

    @mcp.tool()
    async def credit_customers(overdue: bool | None = None) -> list[dict]:
        """List customers with credit tabs (money owed to the shop). overdue: only past-due balances."""
        return await client.request("GET", "/customers/credits", params=param(overdue=overdue))
