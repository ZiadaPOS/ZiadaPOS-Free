"""Direct SMS + email clients mirroring the Go API's SendAfrica / MailAfrica
integrations, so the agent can send to any phone / address the owner names.
"""

from __future__ import annotations

from typing import Any

import httpx

from .config import Settings

SMS_PROVIDER = "SendAfrica"
EMAIL_PROVIDER = "MailAfrica"


def normalize_phone(phone: str) -> str:
    """Tanzanian number → 10-digit 0-prefixed form SendAfrica expects."""
    digits = "".join(ch for ch in phone if ch.isdigit())
    if len(digits) in (12, 13) and digits.startswith("255"):
        return "0" + digits[3:]
    if len(digits) == 10 and digits.startswith("0"):
        return digits
    if len(digits) == 9 and digits.startswith("7"):
        return "0" + digits
    raise ValueError("invalid phone number (expected a Tanzanian number like 0628587749 or +255628587749)")


class SendAfricaError(RuntimeError):
    def __init__(self, code: str, message: str, status: int = 0) -> None:
        super().__init__(f"{SMS_PROVIDER}: {message}")
        self.code = code
        self.status = status


class MailAfricaError(RuntimeError):
    def __init__(self, code: str, message: str, status: int = 0) -> None:
        super().__init__(f"{EMAIL_PROVIDER}: {message}")
        self.code = code
        self.status = status


class NotifyClient:
    """Sends SMS via SendAfrica and email via MailAfrica."""

    def __init__(self, settings: Settings) -> None:
        self._sms_base = settings.sendafrica_base_url.rstrip("/")
        self._sms_key = settings.sendafrica_api_key
        self._sender_id = settings.sendafrica_sender_id
        self._mail_base = settings.mailafrica_base_url.rstrip("/")
        self._mail_key = settings.mailafrica_api_key
        self._from_domain = settings.mailafrica_from_domain.lower().strip()
        self._from_address = settings.mailafrica_from_address
        self._from_name = settings.mailafrica_from_name
        self._resolved_domain_id: int | None = None

    async def send_sms(self, to: str, message: str, from_name: str | None = None) -> dict[str, Any]:
        """Send one SMS. Returns the SendAfrica message_id / status."""
        if not self._sms_key:
            raise SendAfricaError("not_configured", f"{SMS_PROVIDER} API key not configured")
        to = normalize_phone(to)
        payload: dict[str, str] = {"to": to, "message": message}
        if from_name:
            payload["from"] = from_name
        elif self._sender_id:
            payload["from"] = self._sender_id

        async with httpx.AsyncClient(timeout=30.0) as http:
            resp = await http.post(
                f"{self._sms_base}/v1/sms/",
                headers={"X-API-Key": self._sms_key, "Content-Type": "application/json"},
                json=payload,
            )
            data = _safe_json(resp)
        if resp.status_code >= 400 or not data.get("success"):
            msg = data.get("error") or data.get("message") or str(data)[:300]
            raise SendAfricaError(f"http_{resp.status_code}", msg, resp.status_code)
        return data.get("data", {})

    async def sms_balance(self) -> float:
        if not self._sms_key:
            raise SendAfricaError("not_configured", f"{SMS_PROVIDER} API key not configured")
        async with httpx.AsyncClient(timeout=30.0) as http:
            resp = await http.get(
                f"{self._sms_base}/v1/credits/balance",
                headers={"X-API-Key": self._sms_key},
            )
            data = _safe_json(resp)
        if resp.status_code >= 400:
            raise SendAfricaError(f"http_{resp.status_code}", "balance request failed", resp.status_code)
        return float((data.get("data") or {}).get("balance", 0))

    async def send_email(self, to: str, subject: str, body: str,
                         html: str | None = None,
                         from_name: str | None = None) -> dict[str, Any]:
        """Send one transactional email. Returns the MailAfrica email id/status."""
        if not self._mail_key:
            raise MailAfricaError("not_configured", f"{EMAIL_PROVIDER} API key not configured")

        req: dict[str, Any] = {"to": [to], "subject": subject, "text_body": body}
        if html:
            req["html_body"] = html
        if from_name or self._from_name:
            req["from_name"] = from_name or self._from_name

        domain_id = await self._resolve_from_domain_id() if self._from_domain else None
        if domain_id is not None:
            req["from_domain_id"] = domain_id
            if self._from_address:
                req["from_address"] = self._from_address
            try:
                return await self._post_email(req)
            except MailAfricaError as exc:
                if not (exc.status >= 500 or exc.code == "PROVIDER_ERROR"):
                    raise
                # Fall back to the MailAfrica platform sender (matches the Go client).
                req.pop("from_domain_id", None)
                req.pop("from_address", None)
                try:
                    return await self._post_email(req)
                except MailAfricaError as fb:
                    raise MailAfricaError(
                        "send_failed", f"own-domain send failed ({exc}) and "
                        f"platform-sender fallback also failed ({fb})"
                    ) from fb
        return await self._post_email(req)

    async def email_balance(self) -> int:
        if not self._mail_key:
            raise MailAfricaError("not_configured", f"{EMAIL_PROVIDER} API key not configured")
        async with httpx.AsyncClient(timeout=30.0) as http:
            resp = await http.get(
                f"{self._mail_base}/api/billing/balance",
                headers={"X-API-Key": self._mail_key},
            )
            data = _safe_json(resp)
        if resp.status_code >= 400:
            raise MailAfricaError(f"http_{resp.status_code}", "balance request failed", resp.status_code)
        return int((data.get("data") or {}).get("balance_tzs", 0))

    async def _resolve_from_domain_id(self) -> int:
        if self._resolved_domain_id is not None:
            return self._resolved_domain_id
        async with httpx.AsyncClient(timeout=30.0) as http:
            resp = await http.get(
                f"{self._mail_base}/api/domains",
                headers={"X-API-Key": self._mail_key},
            )
            data = _safe_json(resp)
        if resp.status_code >= 400:
            raise MailAfricaError(f"http_{resp.status_code}", "domain lookup failed", resp.status_code)
        for d in (data.get("data") or []):
            if (d.get("domain") or "").lower() == self._from_domain:
                if d.get("status") != "verified":
                    raise MailAfricaError(
                        "domain_not_verified",
                        f"sending domain {self._from_domain!r} is not verified (status {d.get('status')!r})",
                    )
                self._resolved_domain_id = int(d["id"])
                return self._resolved_domain_id
        raise MailAfricaError(
            "domain_not_found",
            f"sending domain {self._from_domain!r} not found on this MailAfrica account",
        )

    async def _post_email(self, req: dict[str, Any]) -> dict[str, Any]:
        async with httpx.AsyncClient(timeout=30.0) as http:
            resp = await http.post(
                f"{self._mail_base}/api/outbound/emails",
                headers={"X-API-Key": self._mail_key, "Content-Type": "application/json"},
                json=req,
            )
            data = _safe_json(resp)
        if resp.status_code >= 400 or not data.get("success"):
            errors = data.get("errors") or []
            code, msg = "", ""
            for e in errors:
                if e.get("code"):
                    code, msg = e["code"], e.get("message", "")
                    break
            raise MailAfricaError(code or f"http_{resp.status_code}",
                                  msg or data.get("message") or str(data)[:300],
                                  resp.status_code)
        return data.get("data", {})


def _safe_json(resp: httpx.Response) -> dict:
    try:
        data = resp.json()
    except Exception:  # noqa: BLE001
        data = {}
    return data if isinstance(data, dict) else {}
