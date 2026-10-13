"""Communication tools: send SMS (SendAfrica) and email (MailAfrica) to
any phone / address, so the owner can message customers and suppliers."""

from __future__ import annotations

from typing import Any

from ..notify import NotifyClient


def register(mcp: Any, client: Any, notify: NotifyClient) -> None:
    @mcp.tool()
    async def send_sms(to: str, message: str, from_name: str | None = None) -> dict:
        """Send an SMS to a phone number (Tanzanian numbers like 0628587749 or +255628587749).
        Returns the provider message_id, status and credits used."""
        return await notify.send_sms(to, message, from_name)

    @mcp.tool()
    async def sms_balance() -> dict:
        """Check the SendAfrica SMS credit balance."""
        return {"balance": await notify.sms_balance()}

    @mcp.tool()
    async def send_email(to: str, subject: str, body: str,
                         html: str | None = None) -> dict:
        """Send an email to any address. body is plain text; html is optional rich text.
        Returns the MailAfrica email id and delivery status."""
        return await notify.send_email(to, subject, body, html)

    @mcp.tool()
    async def email_balance() -> dict:
        """Check the MailAfrica email wallet balance in TZS."""
        return {"balance_tzs": await notify.email_balance()}
