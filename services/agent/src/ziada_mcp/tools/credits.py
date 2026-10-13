"""Credit tools: customer tabs (madeni) — payments, reminders and notes."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def list_credit_tabs(status: str | None = None,
                               customer_id: str | None = None,
                               overdue: bool | None = None,
                               page: int = 1,
                               page_size: int = 20) -> dict:
        """List credit tabs. status: open/partially_paid/settled/written_off; overdue: only past-due."""
        return await client.request("GET", "/credits", params=param(
            status=status, customer_id=customer_id, overdue=overdue,
            page=page, page_size=page_size))

    @mcp.tool()
    async def customer_credit_profile(customer_id: str) -> dict:
        """Full credit profile for one customer: open tabs, balance and payment history."""
        return await client.request("GET", f"/credits/customers/{customer_id}")

    @mcp.tool()
    async def record_credit_payment(customer_id: str, amount: int,
                                    method: str = "Cash",
                                    reference: str | None = None,
                                    note: str | None = None) -> dict:
        """Record a payment against a customer's credit balance. method: Cash/M-Pesa/Tigo Pesa/Bank. amount in TZS."""
        return await client.request("POST", f"/credits/customers/{customer_id}/payments", json={
            "amount": amount, "method": method, "reference": reference or None,
            "note": note or None,
        })

    @mcp.tool()
    async def send_credit_reminder(customer_id: str) -> dict:
        """Send the customer an SMS reminder about their outstanding credit. Requires a phone + configured SMS."""
        return await client.request("POST", f"/credits/customers/{customer_id}/remind", json={})

    @mcp.tool()
    async def add_credit_message(customer_id: str, body: str,
                                 kind: str = "whatsapp") -> dict:
        """Log a message sent to a customer about their credit. kind: whatsapp/call/sms."""
        return await client.request("POST", f"/credits/customers/{customer_id}/messages", json={
            "kind": kind, "body": body,
        })

    @mcp.tool()
    async def add_credit_note(customer_id: str, body: str) -> dict:
        """Add an internal note to a customer's credit record (only the shop sees it)."""
        return await client.request("POST", f"/credits/customers/{customer_id}/notes", json={
            "body": body,
        })

    @mcp.tool()
    async def write_off_credit(tab_id: str, reason: str | None = None) -> dict:
        """Forgive (write off) a credit tab. Irreversible — confirm the amount with the shop owner first."""
        return await client.request("POST", f"/credits/{tab_id}/write-off",
                                    json={"reason": reason or ""})
