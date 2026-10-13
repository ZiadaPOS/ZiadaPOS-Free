# 07 — Communication (SMS & Email)

The agent can send **real SMS and email** directly through the same providers the ZiadaPOS API
uses — no extra gateway. `src/ziada_mcp/notify.py` implements `NotifyClient`.

## SMS — SendAfrica

Base URL: `https://api.sendafrica.online` (`SENDAFRICA_BASE_URL`).

| Operation | Tool | Endpoint |
| --- | --- | --- |
| Send | `send_sms` | `POST /v1/sms/` (header `X-API-Key`, body `{to, message, from}`) |
| Balance | `sms_balance` | `GET /v1/credits/balance` |

- Phone numbers are normalised by `normalize_phone()` to the Tanzanian 10-digit `0…` form:
  `+255628587749` / `255628587749` → `0628587749`; `712345678` → `0712345678`; already-normalised
  numbers pass through; anything else raises `ValueError`.
- Returns the provider's `data` (message id / status / credits used).

## Email — MailAfrica

Base URL: `https://api.mailafrica.online` (`MAILAFRICA_BASE_URL`).

| Operation | Tool | Endpoint |
| --- | --- | --- |
| Send | `send_email` | `POST /api/outbound/emails` |
| Balance | `email_balance` | `GET /api/billing/balance` (returns `balance_tzs`) |

Sending flow:

1. `_resolve_from_domain_id()` — `GET /api/domains`, picks the **verified** domain
   (`MAILAFRICA_FROM_DOMAIN`), caches the result.
2. `POST /api/outbound/emails` with `{to: [address], subject, text_body, html_body?, from_name?,
   from_domain_id?, from_address?}`.
3. If the provider returns a 5xx / `PROVIDER_ERROR`, fall back to the platform's default sender so
   the email still goes out.

## Configuration

See [02 — Configuration](02-configuration.md): `SENDAFRICA_API_KEY` / `SENDAFRICA_SENDER_ID` for
SMS; `MAILAFRICA_API_KEY` + a **verified** `MAILAFRICA_FROM_DOMAIN` for email. Missing keys raise
a clear "not configured" error rather than failing silently.

## Safety

Sending costs provider credits, so the persona treats `send_sms` and `send_email` as
**destructive tools**: the agent must confirm with the owner before sending.
