"""Report tools: generate reports, downloads and scheduled reports."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def report_types() -> dict:
        """List the available report types (sales/inventory/tax/credit)."""
        return await client.request("GET", "/reports/types")

    @mcp.tool()
    async def generate_report(report_type: str, range: str | None = None,
                              date_from: str | None = None,
                              date_to: str | None = None,
                              format: str = "json") -> dict:
        """Generate a business report.
        report_type: sales/inventory/tax/credit.
        range: 7d/30d/90d/month/ytd; or date_from/date_to (YYYY-MM-DD).
        format: 'json' returns data here; 'csv' returns CSV download bytes."""
        return await client.request("POST", "/reports/generate", json={
            "report_type": report_type, "format": format,
            "range": range or None, "date_from": date_from or None,
            "date_to": date_to or None,
        })

    @mcp.tool()
    async def list_report_exports(page: int = 1, page_size: int = 20) -> dict:
        """List past generated report exports."""
        return await client.request("GET", "/reports/exports", params=param(
            page=page, page_size=page_size))

    @mcp.tool()
    async def download_report_export(export_id: str) -> dict:
        """Re-generate and download a past report export."""
        return await client.request("GET", f"/reports/exports/{export_id}/download")

    @mcp.tool()
    async def list_scheduled_reports() -> dict:
        """List the shop's scheduled (auto-sent) reports."""
        return await client.request("GET", "/reports/scheduled")

    @mcp.tool()
    async def create_scheduled_report(report_type: str,
                                      name: str | None = None,
                                      frequency: str = "daily",
                                      date_range_preset: str = "30d",
                                      recipients: list[str] | None = None) -> dict:
        """Schedule a recurring report sent to recipients' email.
        frequency: daily/weekly/monthly. date_range_preset: 7d/30d/90d/month/ytd.
        recipients: list of email addresses."""
        return await client.request("POST", "/reports/scheduled", json={
            "report_type": report_type, "name": name or None,
            "frequency": frequency, "date_range_preset": date_range_preset,
            "recipients": recipients or [],
        })

    @mcp.tool()
    async def update_scheduled_report(scheduled_id: str,
                                      name: str | None = None,
                                      frequency: str | None = None,
                                      date_range_preset: str | None = None,
                                      recipients: list[str] | None = None,
                                      is_enabled: bool | None = None) -> dict:
        """Update a scheduled report. Only provided fields change."""
        body: dict[str, Any] = {}
        if name is not None:
            body["name"] = name
        if frequency is not None:
            body["frequency"] = frequency
        if date_range_preset is not None:
            body["date_range_preset"] = date_range_preset
        if recipients is not None:
            body["recipients"] = recipients
        if is_enabled is not None:
            body["is_enabled"] = is_enabled
        return await client.request("PATCH", f"/reports/scheduled/{scheduled_id}", json=body)

    @mcp.tool()
    async def delete_scheduled_report(scheduled_id: str) -> str:
        """Delete a scheduled report."""
        await client.request("DELETE", f"/reports/scheduled/{scheduled_id}")
        return "Scheduled report deleted"

    @mcp.tool()
    async def send_scheduled_report_now(scheduled_id: str) -> dict:
        """Send a scheduled report immediately to its recipients."""
        return await client.request("POST", f"/reports/scheduled/{scheduled_id}/send", json={})
