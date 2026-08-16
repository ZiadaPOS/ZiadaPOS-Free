"""Expense tools: money going out of the shop."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def list_expenses(search: str | None = None,
                            category: str | None = None,
                            payment_method: str | None = None,
                            ordering: str | None = None,
                            page: int = 1,
                            page_size: int = 20) -> dict:
        """List expenses. categories: Rent/Utilities/Salaries/Supplies/Transport/Marketing/Maintenance/Licences & permits/Insurance/Other."""
        return await client.request("GET", "/expenses", params=param(
            search=search, category=category, payment_method=payment_method,
            ordering=ordering, page=page, page_size=page_size))

    @mcp.tool()
    async def create_expense(title: str, amount: int,
                             category: str = "Other",
                             payment_method: str = "Cash",
                             payment_reference: str | None = None,
                             notes: str | None = None,
                             receipt_url: str | None = None) -> dict:
        """Record an expense. amount in TZS; payment_method: Cash/M-Pesa/Tigo Pesa/Airtel Money/Bank."""
        return await client.request("POST", "/expenses", json={
            "title": title, "amount": amount, "category": category,
            "payment_method": payment_method,
            "payment_reference": payment_reference or None,
            "notes": notes or None, "receipt_url": receipt_url or None,
        })

    @mcp.tool()
    async def get_expense(expense_id: str) -> dict:
        """Get one expense."""
        return await client.request("GET", f"/expenses/{expense_id}")

    @mcp.tool()
    async def update_expense(expense_id: str, title: str | None = None,
                             amount: int | None = None,
                             category: str | None = None,
                             payment_method: str | None = None,
                             payment_reference: str | None = None,
                             notes: str | None = None,
                             receipt_url: str | None = None) -> dict:
        """Update an expense. Only provided fields change."""
        body: dict[str, Any] = {}
        for k, v in {
            "title": title, "amount": amount, "category": category,
            "payment_method": payment_method, "payment_reference": payment_reference,
            "notes": notes, "receipt_url": receipt_url,
        }.items():
            if v is not None:
                body[k] = v
        return await client.request("PATCH", f"/expenses/{expense_id}", json=body)

    @mcp.tool()
    async def delete_expense(expense_id: str) -> str:
        """Delete an expense. Use with care."""
        await client.request("DELETE", f"/expenses/{expense_id}")
        return "Expense deleted"

    @mcp.tool()
    async def expense_summary() -> dict:
        """Expense totals broken down by period and category."""
        return await client.request("GET", "/expenses/summary")
