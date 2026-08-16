"""Errors mapped from the ZiadaPOS API response envelope.

The API returns errors as ``{"status": <int>, "code": <str>,
"message": <str>, "fields": [...]}``. We translate those into a
typed exception so tools can surface the exact server message.
"""

from __future__ import annotations

from typing import Any


class ZiadaError(Exception):
    """Base error raised by the ZiadaPOS API client."""

    def __init__(self, message: str, status: int | None = None,
                 code: str | None = None, fields: list[Any] | None = None):
        super().__init__(message)
        self.message = message
        self.status = status
        self.code = code
        self.fields = fields or []

    def __str__(self) -> str:  # pragma: no cover - trivial
        bits = []
        if self.status:
            bits.append(f"HTTP {self.status}")
        if self.code:
            bits.append(self.code)
        text = f"[{' / '.join(bits)}] " if bits else ""
        return f"{text}{self.message}"


class ZiadaAuthError(ZiadaError):
    """Login, refresh, or 401/403 responses."""


class ZiadaValidationError(ZiadaError):
    """400-level request validation failures."""


class ZiadaRateLimitedError(ZiadaError):
    """429 responses."""


class ZiadaNotFoundError(ZiadaError):
    """404 responses."""


class ZiadaConflictError(ZiadaError):
    """409 responses."""


def from_payload(payload: dict[str, Any], status: int) -> ZiadaError:
    """Build the right ZiadaError subclass from an error envelope."""
    message = payload.get("message", "ZiadaPOS API error")
    code = payload.get("code")
    fields = payload.get("fields") or []

    if status == 401 or code == "unauthorized":
        return ZiadaAuthError(message, status, code, fields)
    if status == 403 or code == "forbidden":
        return ZiadaAuthError(message, status, code, fields)
    if status == 404 or code == "not_found":
        return ZiadaNotFoundError(message, status, code, fields)
    if status == 409 or code == "conflict":
        return ZiadaConflictError(message, status, code, fields)
    if status == 429 or code == "rate_limited":
        return ZiadaRateLimitedError(message, status, code, fields)
    if status == 422 or code == "validation_error" or status == 400:
        return ZiadaValidationError(message, status, code, fields)
    return ZiadaError(message, status, code, fields)
