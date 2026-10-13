"""Supplier tools: suppliers, deliveries and payments."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def list_suppliers(search: str | None = None,
                             status: str | None = None,
                             page: int = 1,
                             page_size: int = 20) -> dict:
        """List suppliers. status: active/inactive/on_hold."""
        return await client.request("GET", "/suppliers", params=param(
            search=search, status=status, page=page, page_size=page_size))

    @mcp.tool()
    async def create_supplier(name: str, category: str | None = None,
                              rep: str | None = None, phone: str | None = None,
                              email: str | None = None, address: str | None = None,
                              payment_terms: int = 30,
                              notes: str | None = None) -> dict:
        """Add a supplier. payment_terms in days (30/60/90)."""
        return await client.request("POST", "/suppliers", json={
            "name": name, "category": category or None, "rep": rep or None,
            "phone": phone or None, "email": email or None, "address": address or None,
            "payment_terms": payment_terms, "notes": notes or None,
        })

    @mcp.tool()
    async def get_supplier(supplier_id: str) -> dict:
        """Get one supplier including outstanding balance."""
        return await client.request("GET", f"/suppliers/{supplier_id}")

    @mcp.tool()
    async def update_supplier(supplier_id: str, name: str | None = None,
                              category: str | None = None, rep: str | None = None,
                              phone: str | None = None, email: str | None = None,
                              address: str | None = None,
                              payment_terms: int | None = None,
                              status: str | None = None,
                              notes: str | None = None) -> dict:
        """Update a supplier. Only provided fields change."""
        body: dict[str, Any] = {}
        for k, v in {
            "name": name, "category": category, "rep": rep, "phone": phone,
            "email": email, "address": address, "payment_terms": payment_terms,
            "status": status, "notes": notes,
        }.items():
            if v is not None:
                body[k] = v
        return await client.request("PATCH", f"/suppliers/{supplier_id}", json=body)

    @mcp.tool()
    async def deactivate_supplier(supplier_id: str) -> dict:
        """Deactivate a supplier (soft delete)."""
        return await client.request("DELETE", f"/suppliers/{supplier_id}")

    @mcp.tool()
    async def supplier_stats() -> dict:
        """Supplier counts and total outstanding balance."""
        return await client.request("GET", "/suppliers/stats")

    # ── Deliveries ──────────────────────────────────────────────────────────

    @mcp.tool()
    async def list_deliveries(supplier_id: str | None = None,
                              unpaid: bool | None = None,
                              page: int = 1,
                              page_size: int = 20) -> dict:
        """List supplier deliveries. unpaid: only deliveries not yet paid for; supplier: filter by supplier id."""
        return await client.request("GET", "/suppliers/deliveries", params=param(
            supplier=supplier_id, unpaid=unpaid, page=page, page_size=page_size))

    @mcp.tool()
    async def create_delivery(supplier_id: str, total_value: int,
                              invoice_number: str | None = None,
                              delivery_date: str | None = None,
                              items_count: int = 1,
                              status: str = "delivered",
                              is_paid: bool = False,
                              notes: str | None = None) -> dict:
        """Record a stock delivery from a supplier. total_value in TZS; delivery_date as YYYY-MM-DD."""
        return await client.request("POST", "/suppliers/deliveries", json={
            "supplier_id": supplier_id, "invoice_number": invoice_number or None,
            "delivery_date": delivery_date or None, "items_count": items_count,
            "total_value": total_value, "status": status, "is_paid": is_paid,
            "notes": notes or None,
        })

    @mcp.tool()
    async def get_delivery(delivery_id: str) -> dict:
        """Get one delivery."""
        return await client.request("GET", f"/suppliers/deliveries/{delivery_id}")

    @mcp.tool()
    async def update_delivery(delivery_id: str, invoice_number: str | None = None,
                              delivery_date: str | None = None,
                              items_count: int | None = None,
                              total_value: int | None = None,
                              status: str | None = None,
                              is_paid: bool | None = None,
                              notes: str | None = None) -> dict:
        """Update a delivery. Only provided fields change."""
        body: dict[str, Any] = {}
        for k, v in {
            "invoice_number": invoice_number, "delivery_date": delivery_date,
            "items_count": items_count, "total_value": total_value,
            "status": status, "is_paid": is_paid, "notes": notes,
        }.items():
            if v is not None:
                body[k] = v
        return await client.request("PATCH", f"/suppliers/deliveries/{delivery_id}", json=body)

    @mcp.tool()
    async def delete_delivery(delivery_id: str) -> str:
        """Delete a delivery record. Use with care."""
        await client.request("DELETE", f"/suppliers/deliveries/{delivery_id}")
        return "Delivery deleted"

    # ── Payments ────────────────────────────────────────────────────────────

    @mcp.tool()
    async def list_supplier_payments(supplier_id: str | None = None,
                                     page: int = 1,
                                     page_size: int = 20) -> dict:
        """List payments made to suppliers (optionally filtered by supplier)."""
        return await client.request("GET", "/suppliers/payments", params=param(
            supplier=supplier_id, page=page, page_size=page_size))

    @mcp.tool()
    async def record_supplier_payment(supplier_id: str, amount: int,
                                      payment_method: str = "cash",
                                      reference: str | None = None,
                                      delivery_id: str | None = None,
                                      notes: str | None = None) -> dict:
        """Record a payment to a supplier. payment_method: cash/bank/mobile. amount in TZS; attach delivery_id to clear a specific delivery."""
        return await client.request("POST", f"/suppliers/{supplier_id}/pay", json={
            "amount": amount, "payment_method": payment_method,
            "reference": reference or None, "delivery_id": delivery_id or None,
            "notes": notes or None,
        })
