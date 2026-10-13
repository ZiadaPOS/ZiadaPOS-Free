# 06 — Product Images

When a shop owner adds a product without a picture, the agent can find one automatically via the
`suggest_product_image` tool — **Wikimedia Commons → MinIO**, all behind the scenes.

## Tool

`tools/images.py` → `suggest_product_image(query, max_bytes=2 MiB)`
returns:

```json
{
  "url": "https://files.ziadapos.com/ziada/<org>/products/<uuid>.jpg",
  "source": "https://commons.wikimedia.org/wiki/File:…",
  "title": "File:…",
  "content_type": "image/jpeg",
  "size": 357040
}
```

or `{"error": "No image found for '…'. Ask the owner to provide one."}`.

## Pipeline

1. **Search Wikimedia Commons**
   `GET https://commons.wikimedia.org/w/api.php?action=query&generator=search&gsrsearch=<query>&gsrnamespace=6&gsrlimit=5&prop=imageinfo&iiprop=url|mime&iiurlwidth=600&format=json&origin=*`
   (a proper `User-Agent: ZiadaPOS-Agent/0.1 …` header is required by Commons).
   Only MIME types in `{image/jpeg, image/png, image/webp, image/gif}` are kept.
2. **Query relaxation** — long queries return nothing (e.g. `Pepsi Cola 500ml` → 0 results), so the
   tool progressively drops the last word (`Pepsi Cola 500ml` → `Pepsi Cola` → `Pepsi`) until hits
   are found.
3. **Download** — fetch the 600px thumbnail (or full image); skip on HTTP error, empty body,
   body larger than `max_bytes`, or an unexpected content-type.
4. **Upload** — `client.upload_file(content, filename, content_type)` → multipart
   `POST /uploads` to the ZiadaPOS API → stored in the shop's **MinIO** bucket at
   `<org>/products/<uuid>.<ext>`.
5. **Return** the stable public URL + attribution details.

## Why this matters

- The URL returned is the **permanent public** product image (served from
  `files.ziadapos.com` in production), which the agent can store on the product via
  `update_product` / `create_product(image=…)`.
- Attribution is returned so the owner knows exactly where the picture came from
  (Wikimedia Commons requires attribution on reuse).
