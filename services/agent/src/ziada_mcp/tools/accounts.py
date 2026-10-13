"""Accounts tools: the shop owner's profile, organisation and users."""
from __future__ import annotations

from typing import Any


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def get_me() -> dict:
        """Current account context: the owner's profile, active store and all stores."""
        return await client.request("GET", "/accounts/me")

    @mcp.tool()
    async def update_me(first_name: str | None = None,
                        last_name: str | None = None,
                        email: str | None = None,
                        occupation: str | None = None,
                        password: str | None = None) -> dict:
        """Update the owner's profile. Only provided fields change."""
        body: dict[str, Any] = {}
        for k, v in {
            "first_name": first_name, "last_name": last_name, "email": email,
            "occupation": occupation, "password": password,
        }.items():
            if v is not None:
                body[k] = v
        return await client.request("PATCH", "/accounts/me", json=body)

    @mcp.tool()
    async def get_organisation() -> dict:
        """The business / organisation profile (name, TIN, location, plan)."""
        return await client.request("GET", "/accounts/organisation")

    @mcp.tool()
    async def update_organisation(name: str | None = None,
                                  legal_name: str | None = None,
                                  tin: str | None = None,
                                  business_type: str | None = None,
                                  phone: str | None = None,
                                  email: str | None = None,
                                  address: str | None = None,
                                  region: str | None = None,
                                  district: str | None = None) -> dict:
        """Update the business profile. Only provided fields change."""
        body: dict[str, Any] = {}
        for k, v in {
            "name": name, "legal_name": legal_name, "tin": tin,
            "business_type": business_type, "phone": phone, "email": email,
            "address": address, "region": region, "district": district,
        }.items():
            if v is not None:
                body[k] = v
        return await client.request("PATCH", "/accounts/organisation", json=body)

    @mcp.tool()
    async def list_users() -> dict:
        """List all users in the organisation (owners and staff)."""
        return await client.request("GET", "/accounts/users")

    @mcp.tool()
    async def create_user(first_name: str, phone: str,
                          last_name: str | None = None,
                          email: str | None = None,
                          role: str = "staff",
                          store_id: str | None = None,
                          can_refund: bool = False,
                          can_discount: bool = True,
                          can_view_reports: bool = False) -> dict:
        """Create an organisation user (staff or co-owner). A temporary password is generated."""
        return await client.request("POST", "/accounts/users", json={
            "first_name": first_name, "phone": phone, "last_name": last_name or None,
            "email": email or None, "role": role, "store_id": store_id or None,
            "can_refund": can_refund, "can_discount": can_discount,
            "can_view_reports": can_view_reports,
        })

    @mcp.tool()
    async def deactivate_user(user_id: str) -> str:
        """Deactivate an organisation user (can no longer log in)."""
        await client.request("DELETE", f"/accounts/users/{user_id}")
        return "User deactivated"

    @mcp.tool()
    async def ai_credits() -> dict:
        """The organisation's AI credit balance for the month."""
        return await client.request("GET", "/accounts/ai-credits")
