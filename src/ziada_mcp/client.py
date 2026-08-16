"""Async HTTP client for the ZiadaPOS API.

The client logs in once as the configured service account (a shop owner),
keeps the JWT pair in memory, transparently refreshes the access token on
401, and unwraps the API's ``{success, data, meta}`` envelope so tools get
clean payloads.

Important: the ZiadaPOS API embeds ``store_id``/``org_id`` inside the JWT
claims. Switching the active store means performing ``/auth/switch-store``
and exchanging the token pair — done via :meth:`ZiadaClient.switch_store`.
"""

from __future__ import annotations

import asyncio
import logging
from collections.abc import AsyncIterator
from typing import Any

import httpx

from .config import get_settings
from .errors import ZiadaAuthError, ZiadaError, from_payload

logger = logging.getLogger("ziada_mcp.client")


class ZiadaClient:
    """Thin authenticated client over the ZiadaPOS REST API."""

    def __init__(self) -> None:
        self._cfg = get_settings()
        self._http = httpx.AsyncClient(
            base_url=self._cfg.ziada_api_url,
            timeout=httpx.Timeout(30.0, connect=10.0),
        )
        self._access_token: str | None = None
        self._refresh_token: str | None = None
        self._lock = asyncio.Lock()
        self._user: dict[str, Any] | None = None

    # ── lifecycle ──────────────────────────────────────────────────────────

    async def start(self) -> None:
        """Authenticate as the configured service account."""
        if not self._cfg.ziada_identifier or not self._cfg.ziada_password:
            raise ZiadaAuthError(
                "ZIADA_IDENTIFIER / ZIADA_PASSWORD not set in .env — the agent "
                "needs the shop owner's login to act on their behalf."
            )
        await self._login()
        await self._load_me()
        if self._cfg.ziada_store_id and self._cfg.ziada_store_id != str(
            self._user.get("store", {}).get("id") or ""
        ):
            await self.switch_store(self._cfg.ziada_store_id)

    async def aclose(self) -> None:
        await self._http.aclose()

    # ── auth ───────────────────────────────────────────────────────────────

    async def _login(self) -> None:
        data = await self._post_raw(
            "/auth/login",
            json={
                "identifier": self._cfg.ziada_identifier,
                "password": self._cfg.ziada_password,
            },
            authed=False,
        )
        body = data["body"]
        self._apply_tokens(body.get("access_token"), body.get("refresh_token"))
        logger.info("Logged in to ZiadaPOS as %s", self._cfg.ziada_identifier)

    async def _refresh(self) -> None:
        if not self._refresh_token:
            raise ZiadaAuthError("No refresh token available — please log in again.")
        data = await self._post_raw(
            "/auth/refresh",
            json={"refresh_token": self._refresh_token},
            authed=False,
        )
        self._apply_tokens(data["body"].get("access_token"), data["body"].get("refresh_token"))

    async def switch_store(self, store_id: str) -> dict[str, Any]:
        """Switch the active store and exchange the token pair."""
        data = await self._post_raw(
            "/accounts/switch-store", json={"store_id": store_id}
        )
        body = data["body"]
        self._apply_tokens(body.get("access_token"), body.get("refresh_token"))
        await self._load_me()
        logger.info("Switched active store to %s", store_id)
        return body

    def _apply_tokens(self, access: str | None, refresh: str | None) -> None:
        if access:
            self._access_token = access
        if refresh:
            self._refresh_token = refresh

    async def _load_me(self) -> None:
        self._user = await self.request("GET", "/accounts/me")

    # ── request plumbing ───────────────────────────────────────────────────

    async def _request(
        self, method: str, path: str, *, json: Any = None, params: dict | None = None,
        authed: bool = True, _retry: bool = True,
    ) -> httpx.Response:
        if authed and not self._access_token:
            raise ZiadaAuthError("Not authenticated yet.")
        headers = {}
        if authed and self._access_token:
            headers["Authorization"] = f"Bearer {self._access_token}"

        resp = await self._http.request(
            method, path, json=json, params=params, headers=headers
        )

        if resp.status_code == 401 and authed and _retry:
            # Access token expired (15 min TTL) — refresh once and retry.
            async with self._lock:
                if _retry:
                    await self._refresh()
            return await self._request(method, path, json=json, params=params,
                                       authed=authed, _retry=False)

        return resp

    async def _post_raw(self, path: str, *, json: Any, authed: bool = True) -> dict:
        resp = await self._request("POST", path, json=json, authed=authed)
        body, ok = _parse(resp)
        if not ok:
            raise from_payload(body, resp.status_code)
        return {"body": body.get("data", {})}

    async def request(
        self, method: str, path: str, *, json: Any = None,
        params: dict | None = None, authed: bool = True,
    ) -> Any:
        """Perform an API request and return the unwrapped ``data`` payload."""
        resp = await self._request(method, path, json=json, params=params, authed=authed)
        body, ok = _parse(resp)
        if not ok:
            raise from_payload(body, resp.status_code)
        return body.get("data")

    async def request_stream(
        self, method: str, path: str, *, params: dict | None = None,
    ) -> AsyncIterator[httpx.Response]:
        """Stream a raw response (used for CSV report downloads)."""
        resp = await self._request(method, path, params=params)
        if resp.status_code >= 400:
            body, _ = _parse(resp)
            raise from_payload(body, resp.status_code)
        yield resp

    async def upload_file(
        self, content: bytes, filename: str, content_type: str,
        kind: str = "image", folder: str = "products",
    ) -> str:
        """Upload a file (e.g. a product image) to the shop's MinIO storage via
        POST /uploads and return the stable public URL."""
        if not self._access_token:
            raise ZiadaAuthError("Not authenticated yet.")

        async def _upload() -> httpx.Response:
            return await self._http.post(
                "/uploads",
                headers={"Authorization": f"Bearer {self._access_token}"},
                files={"file": (filename, content, content_type)},
                data={"kind": kind, "folder": folder},
            )

        resp = await _upload()
        if resp.status_code == 401:
            async with self._lock:
                await self._refresh()
            resp = await _upload()
        body, ok = _parse(resp)
        if not ok:
            raise from_payload(body, resp.status_code)
        url = body.get("data", {}).get("url")
        if not url:
            raise ZiadaError("Upload succeeded but no URL was returned.")
        return url

    # ── context helpers (for tools) ────────────────────────────────────────

    @property
    def user(self) -> dict[str, Any]:
        return self._user or {}

    @property
    def user_id(self) -> str | None:
        if not self._user:
            return None
        return self._user.get("user", {}).get("id")

    @property
    def org_id(self) -> str | None:
        if not self._user:
            return None
        return self._user.get("organisation", {}).get("id")

    @property
    def store_id(self) -> str | None:
        if not self._user:
            return None
        return self._user.get("store", {}).get("id")

    @property
    def store_name(self) -> str:
        if not self._user:
            return ""
        return self._user.get("store", {}).get("name") or ""

    @property
    def stores(self) -> list[dict[str, Any]]:
        """All stores available to the service account (for switching)."""
        if not self._user:
            return []
        org_stores = self._user.get("organisation", {}).get("stores")
        if org_stores:
            return org_stores
        return self._user.get("stores", []) or []

    @property
    def organisation(self) -> dict[str, Any]:
        return self._user.get("organisation", {}) if self._user else {}


def _parse(resp: httpx.Response) -> tuple[dict, bool]:
    """Return (body, success). Body is the JSON object or an empty dict."""
    try:
        body = resp.json()
    except Exception:  # noqa: BLE001 — any malformed body means empty dict
        body = {}
    if not isinstance(body, dict):
        body = {"data": body}
    if resp.status_code >= 400 or body.get("success") is False:
        return body, False
    return body, True
