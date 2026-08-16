"""Staff tools: the shop's employees (owner/admin level)."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def list_staff(search: str | None = None,
                         role: str | None = None,
                         status: str | None = None,
                         shift: str | None = None,
                         include_stats: bool = False,
                         page: int = 1,
                         size: int = 20) -> dict:
        """List staff members. role: staff/owner; status: active/on_leave/inactive; shift: morning/evening/full_day/weekend."""
        return await client.request("GET", "/staff", params=param(
            search=search, role=role, status=status, shift=shift,
            include_stats=include_stats, page=page, size=size))

    @mcp.tool()
    async def create_staff(first_name: str, last_name: str | None = None,
                           phone: str | None = None, email: str | None = None,
                           role: str = "staff",
                           store_id: str | None = None,
                           shift: str = "morning",
                           employment_status: str = "active",
                           pin: str | None = None,
                           can_refund: bool = False,
                           can_discount: bool = True,
                           can_view_reports: bool = False) -> dict:
        """Hire a staff member. A temporary password is generated and sent by SMS/email."""
        return await client.request("POST", "/staff", json={
            "first_name": first_name, "last_name": last_name or None,
            "phone": phone or None, "email": email or None, "role": role,
            "store_id": store_id or None, "shift": shift,
            "employment_status": employment_status, "pin": pin or None,
            "can_refund": can_refund, "can_discount": can_discount,
            "can_view_reports": can_view_reports,
        })

    @mcp.tool()
    async def get_staff(staff_id: str) -> dict:
        """Get one staff member."""
        return await client.request("GET", f"/staff/{staff_id}")

    @mcp.tool()
    async def update_staff(staff_id: str, first_name: str | None = None,
                           last_name: str | None = None, email: str | None = None,
                           phone: str | None = None, role: str | None = None,
                           shift: str | None = None,
                           employment_status: str | None = None,
                           is_active: bool | None = None) -> dict:
        """Update a staff member's profile. Only provided fields change."""
        body: dict[str, Any] = {}
        for k, v in {
            "first_name": first_name, "last_name": last_name, "email": email,
            "phone": phone, "role": role, "shift": shift,
            "employment_status": employment_status, "is_active": is_active,
        }.items():
            if v is not None:
                body[k] = v
        return await client.request("PATCH", f"/staff/{staff_id}", json=body)

    @mcp.tool()
    async def deactivate_staff(staff_id: str) -> dict:
        """Deactivate a staff member (they can no longer log in)."""
        return await client.request("DELETE", f"/staff/{staff_id}")

    @mcp.tool()
    async def staff_stats(staff_id: str) -> dict:
        """Sales stats for one staff member (transactions, revenue, average ticket)."""
        return await client.request("GET", f"/staff/{staff_id}/stats")

    @mcp.tool()
    async def staff_kpis() -> dict:
        """Team performance KPIs per store."""
        return await client.request("GET", "/staff/kpis")

    @mcp.tool()
    async def staff_activity(staff_id: str, date: str | None = None) -> dict:
        """Activity/clocking log for a staff member (date as YYYY-MM-DD, defaults to today)."""
        return await client.request("GET", f"/staff/{staff_id}/activity", params=param(date=date))

    @mcp.tool()
    async def update_staff_shift(staff_id: str, shift: str) -> dict:
        """Change a staff member's shift. Values: morning/evening/full_day/weekend."""
        return await client.request("PATCH", f"/staff/{staff_id}/shift", json={"shift": shift})

    @mcp.tool()
    async def update_staff_permissions(staff_id: str, can_refund: bool | None = None,
                                       can_discount: bool | None = None,
                                       can_view_reports: bool | None = None) -> dict:
        """Change a staff member's POS permissions (refunds, discounts, reports)."""
        body: dict[str, Any] = {}
        for k, v in {
            "can_refund": can_refund, "can_discount": can_discount,
            "can_view_reports": can_view_reports,
        }.items():
            if v is not None:
                body[k] = v
        return await client.request("PATCH", f"/staff/{staff_id}/permissions", json=body)
