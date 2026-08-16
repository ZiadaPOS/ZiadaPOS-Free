# 03 — Client & Auth

`src/ziada_mcp/client.py` provides `ZiadaClient` — the single gateway every tool uses to talk to
the ZiadaPOS Go API.

## Lifecycle

The server's lifespan (`server.py`) calls `client.start()` on startup and `client.aclose()` on
shutdown:

1. **`start()`** — validates that credentials exist, then:
   - `_login()` → `POST /auth/login` with `{identifier, password}` → stores the token pair.
   - `_load_me()` → `GET /accounts/me` → caches `_user` (profile, org, stores).
   - If `ZIADA_STORE_ID` is set and differs from the account's current store →
     `switch_store()` re-exchanges tokens so the JWT embeds the right store.
2. **`aclose()`** — closes the underlying `httpx.AsyncClient`.

## Auth: login, refresh, switch-store

- **Login** happens once at startup. The access token has a short TTL (~15 min).
- **Auto-refresh**: every authenticated request attaches `Authorization: Bearer <token>`.
  If the API answers **401**, `_request()` takes an `asyncio.Lock`, calls
  `POST /auth/refresh` with the stored refresh token, applies the new token pair, and retries
  the original request exactly once.
- **Store switching**: the API embeds `org_id` + `store_id` inside the JWT, so switching the
  active store must mint a new token. `switch_store(store_id)` → `POST /accounts/switch-store`
  then re-fetches `/accounts/me`.

## The response envelope

The API wraps responses as:

```json
{ "success": true, "data": { … }, "meta": { … } }
```

`request(method, path, json=…, params=…)` returns the unwrapped `data` payload.
On failure the error envelope `{status, code, message, fields}` is mapped to a typed exception
by `errors.from_payload()`.

## Request helpers

| Method | Use |
| --- | --- |
| `request(method, path, …)` | Standard JSON call; returns `data`. |
| `request_stream(method, path, …)` | Returns the raw `httpx.Response` (used for CSV report downloads). |
| `upload_file(content, filename, content_type, kind="image", folder="products")` | Multipart `POST /uploads` with `files={"file": …}` + `data={"kind", "folder"}`; 401 → refresh-and-retry; returns the stable public URL. |
| `_post_raw(path, …)` | Low-level POST returning the raw response. |

## Exposed context for tools

Tools read these properties to scope requests and to personalise answers:

- `user` / `user_id` — the logged-in account.
- `org_id` — the business (organisation) id.
- `store_id` / `store_name` — the active store.
- `stores` — all stores of the business (for switching / multi-branch context).
- `organisation` — business profile (name, TIN, plan…).

## Error model (`src/ziada_mcp/errors.py`)

| Exception | Triggered by |
| --- | --- |
| `ZiadaError` | Base; carries `message`, `status`, `code`, `fields`. |
| `ZiadaAuthError` | 401 / 403 — bad or expired token (after refresh), forbidden. |
| `ZiadaValidationError` | 400 / 422 — invalid input, `fields` details. |
| `ZiadaRateLimitedError` | 429 — too many requests. |
| `ZiadaNotFoundError` | 404 — resource not found (e.g. barcode not matched). |
| `ZiadaConflictError` | 409 — duplicate / conflicting state. |

Tools usually let these propagate to FastMCP. The `ask` agent catches them and hands the text
back to the model so it can correct itself.
