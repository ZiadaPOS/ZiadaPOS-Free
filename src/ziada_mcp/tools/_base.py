"""Shared helpers for MCP tools."""
from __future__ import annotations

from typing import Any


def tzs(amount: int | None) -> str:
    """Format a TZS int64 amount with thousands separators."""
    if amount is None:
        return "0"
    return f"{amount:,}"


def today() -> str:
    from datetime import datetime
    from zoneinfo import ZoneInfo

    return datetime.now(ZoneInfo("Africa/Dar_es_Salaam")).date().isoformat()


def param(**kwargs: Any) -> dict | None:
    """Build a query-param dict, dropping None/empty values."""
    out = {}
    for k, v in kwargs.items():
        if v is None:
            continue
        if isinstance(v, str) and not v.strip():
            continue
        out[k] = v
    return out or None
