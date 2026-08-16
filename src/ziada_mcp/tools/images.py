"""Image tools: fetch a product photo for the shop when the owner doesn't
supply one (e.g. 'Pepsi 500ml' → a Pepsi image). Images are uploaded to the
shop's MinIO storage and returned as stable URLs."""

from __future__ import annotations

from typing import Any

import httpx

ALLOWED_TYPES = {"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}
COMMONS_API = "https://commons.wikimedia.org/w/api.php"
USER_AGENT = "ZiadaPOS-Agent/0.1 (ziadapos-mcp; contact: noreply@ziadapos.com)"


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def suggest_product_image(query: str, max_bytes: int = 2 * 1024 * 1024) -> dict:
        """Find and store a product image for a name like 'Pepsi 500ml bottle'.
        Searches Wikimedia Commons, downloads the best match, uploads it to the
        shop's image storage (MinIO) and returns the public URL + source so you
        can attach it to a product's `image` field."""
        hits = await _search_commons(query)
        if not hits:
            shorter = " ".join(query.split()[:3])
            if shorter != query:
                hits = await _search_commons(shorter)
        if not hits:
            return {"error": f"No image found for {query!r}. Ask the owner to provide one."}

        async with httpx.AsyncClient(timeout=30.0, follow_redirects=True,
                                     headers={"User-Agent": USER_AGENT}) as http:
            for hit in hits:
                thumb = hit.get("thumburl") or hit.get("url")
                if not thumb:
                    continue
                try:
                    r = await http.get(thumb)
                    r.raise_for_status()
                except httpx.HTTPError:
                    continue
                if len(r.content) == 0 or len(r.content) > max_bytes:
                    continue
                content_type = r.headers.get("content-type", "").split(";")[0].strip()
                ext = ALLOWED_TYPES.get(content_type)
                if not ext:
                    continue
                url = await client.upload_file(
                    r.content, filename=f"{hit.get('title', 'product')[:50]}{ext}",
                    content_type=content_type,
                )
                return {
                    "url": url,
                    "source": hit.get("descriptionurl") or hit.get("url"),
                    "title": hit.get("title"),
                    "content_type": content_type,
                    "size": len(r.content),
                }
        return {"error": f"Could not download a usable image for {query!r}."}


async def _search_commons(query: str) -> list[dict]:
    """Search Wikimedia Commons for images matching the query."""
    params = {
        "action": "query",
        "generator": "search",
        "gsrsearch": query,
        "gsrnamespace": 6,
        "gsrlimit": 5,
        "prop": "imageinfo",
        "iiprop": "url|mime",
        "iiurlwidth": 600,
        "format": "json",
        "origin": "*",
    }
    async with httpx.AsyncClient(timeout=30.0,
                                 headers={"User-Agent": USER_AGENT}) as http:
        r = await http.get(COMMONS_API, params=params)
        r.raise_for_status()
        data = r.json()
    pages = (data.get("query") or {}).get("pages") or {}
    hits = []
    for page in pages.values():
        info = (page.get("imageinfo") or [{}])[0]
        mime = info.get("mime", "")
        if mime not in ALLOWED_TYPES:
            continue
        hits.append({
            "title": page.get("title"),
            "url": info.get("url"),
            "thumburl": info.get("thumburl"),
            "descriptionurl": info.get("descriptionurl"),
        })
    return hits
