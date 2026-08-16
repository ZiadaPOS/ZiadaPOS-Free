"""Analytics tools: KPIs, trends and business intelligence."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def analytics_overview(
        range: str | None = None,
        date_from: str | None = None,
        date_to: str | None = None,
        sections: str | None = None,
    ) -> dict:
        """Combined dashboard: KPIs, revenue trend, payment mix and top products.
        range: 7d/30d/90d/ytd (default 30d); or give date_from/date_to (YYYY-MM-DD).
        sections: comma list e.g. 'kpis,trend,mix,products' (all by default)."""
        return await client.request("GET", "/analytics/overview", params=param(
            range=range, date_from=date_from, date_to=date_to, sections=sections))

    @mcp.tool()
    async def analytics_summary(range: str | None = None,
                                date_from: str | None = None,
                                date_to: str | None = None) -> dict:
        """KPI summary (revenue, profit, transactions, margin, averages) for a range."""
        return await client.request("GET", "/analytics/summary", params=param(
            range=range, date_from=date_from, date_to=date_to))

    @mcp.tool()
    async def analytics_trend(range: str | None = None,
                              date_from: str | None = None,
                              date_to: str | None = None) -> dict:
        """Revenue trend over a range (daily buckets)."""
        return await client.request("GET", "/analytics/trend", params=param(
            range=range, date_from=date_from, date_to=date_to))

    @mcp.tool()
    async def analytics_payment_mix(range: str | None = None,
                                    date_from: str | None = None,
                                    date_to: str | None = None) -> dict:
        """Sales split by payment method (Cash/M-Pesa/Tigo Pesa/Bank/Credit)."""
        return await client.request("GET", "/analytics/payment-mix", params=param(
            range=range, date_from=date_from, date_to=date_to))

    @mcp.tool()
    async def analytics_top_products(range: str | None = None,
                                     date_from: str | None = None,
                                     date_to: str | None = None,
                                     limit: int = 10) -> dict:
        """Best-selling products for a range (default 10, max 50)."""
        return await client.request("GET", "/analytics/top-products", params=param(
            range=range, date_from=date_from, date_to=date_to, limit=limit))

    @mcp.tool()
    async def analytics_sales(range: str | None = None,
                              date_from: str | None = None,
                              date_to: str | None = None) -> dict:
        """Full sales report: KPIs + trend + breakdown by method/hour."""
        return await client.request("GET", "/analytics/sales", params=param(
            range=range, date_from=date_from, date_to=date_to))

    @mcp.tool()
    async def analytics_products(range: str | None = None,
                                 date_from: str | None = None,
                                 date_to: str | None = None,
                                 category: str | None = None,
                                 sort: str | None = None) -> dict:
        """Per-product performance (revenue, qty, margin) for a range."""
        return await client.request("GET", "/analytics/products", params=param(
            range=range, date_from=date_from, date_to=date_to,
            category=category, sort=sort))

    @mcp.tool()
    async def analytics_customers(range: str | None = None,
                                  date_from: str | None = None,
                                  date_to: str | None = None) -> dict:
        """Customer analytics for a range (new vs repeat, top spenders, segments)."""
        return await client.request("GET", "/analytics/customers", params=param(
            range=range, date_from=date_from, date_to=date_to))

    @mcp.tool()
    async def analytics_cashflow(range: str | None = None,
                                 date_from: str | None = None,
                                 date_to: str | None = None) -> dict:
        """Cashflow report: money in (sales, payments) vs out (expenses, supplier payments)."""
        return await client.request("GET", "/analytics/cashflow", params=param(
            range=range, date_from=date_from, date_to=date_to))

    @mcp.tool()
    async def analytics_dashboard() -> dict:
        """Pre-aggregated dashboard data (quick daily picture)."""
        return await client.request("GET", "/analytics/dashboard")
