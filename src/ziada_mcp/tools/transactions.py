"""POS / transactions tools: sales, refunds, voids and summaries."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def complete_sale(
        lines: list[dict[str, Any]],
        payment_method: str,
        payment_reference: str | None = None,
        discount_pct: float = 0,
        customer_name: str | None = None,
        customer_phone: str | None = None,
        customer_id: str | None = None,
        till_number: str | None = None,
        notes: str | None = None,
    ) -> dict:
        """Complete a POS sale and decrement stock.
        lines: list of {"product_id": uuid, "qty": int} — one entry per item.
        payment_method: Cash, M-Pesa, Tigo Pesa, Bank or Credit (credit opens/extends a customer credit tab).
        All money is in TZS. Use 'Credit' only when the customer has a credit relationship."""
        return await client.request("POST", "/pos/sale", json={
            "lines": lines, "payment_method": payment_method,
            "payment_reference": payment_reference or None, "discount_pct": discount_pct,
            "customer_name": customer_name or None, "customer_phone": customer_phone or None,
            "customer_id": customer_id or None, "till_number": till_number or None,
            "notes": notes or None,
        })

    @mcp.tool()
    async def list_transactions(
        status: str | None = None,
        method: str | None = None,
        search: str | None = None,
        customer_id: str | None = None,
        date_from: str | None = None,
        date_to: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """List sales. status: paid/credit/refunded/void; method: payment method; search: txn number/customer; dates as YYYY-MM-DD."""
        return await client.request("GET", "/pos/transactions", params=param(
            status=status, method=method, search=search, customer_id=customer_id,
            date_from=date_from, date_to=date_to, page=page, page_size=page_size))

    @mcp.tool()
    async def get_transaction(transaction_id: str) -> dict:
        """Get one sale with its itemised lines."""
        return await client.request("GET", f"/pos/transactions/{transaction_id}")

    @mcp.tool()
    async def refund_transaction(transaction_id: str, reason: str | None = None) -> dict:
        """Refund a paid sale — stock is restored and the txn becomes 'refunded'. Destructive: confirm the amount with the shop owner first."""
        return await client.request("POST", f"/pos/transactions/{transaction_id}/refund",
                                    json={"reason": reason or ""})

    @mcp.tool()
    async def void_transaction(transaction_id: str, reason: str | None = None) -> dict:
        """Void a transaction (mark as void). Destructive: confirm with the shop owner first."""
        return await client.request("POST", f"/pos/transactions/{transaction_id}/void",
                                    json={"reason": reason or ""})

    @mcp.tool()
    async def sales_summary(date_from: str | None = None, date_to: str | None = None) -> dict:
        """Sales summary (revenue, transactions, profit) for a date range. Defaults to today. Dates as YYYY-MM-DD."""
        return await client.request("GET", "/pos/summary", params=param(
            date_from=date_from, date_to=date_to))

    @mcp.tool()
    async def hourly_sales(date: str | None = None) -> list[dict]:
        """Hourly sales breakdown for a day (YYYY-MM-DD, defaults to today)."""
        return await client.request("GET", "/pos/summary/hourly", params=param(date=date))
