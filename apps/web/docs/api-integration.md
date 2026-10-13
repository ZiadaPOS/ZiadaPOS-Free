# Web app to API integration

This note documents the implemented browser integration. API endpoint details, payload schemas, and server-side constraints live in the API repository at `docs/04-api-reference.md` and `docs/business-workflows.md`.

## Runtime configuration

`POS_API_URL` is the server-side base URL for the Go API, including its version prefix. Default: `http://localhost:8080/api/v1`. Example production value: `https://api.ziadapos.com/api/v1`. It is deliberately not a `NEXT_PUBLIC_*` variable. Restart/redeploy the Next.js server after changing it.

## Request paths

### Business requests

Browser features call same-origin `/api/pos/...` using the shared POS API helper. The catch-all route handler under `src/app/api/pos/[...path]/route.ts`:

1. Appends the decoded path and incoming query string to `POS_API_URL`.
2. Reads the request body for methods other than GET/HEAD.
3. Reads `pos_access_token` from server cookies and sends it as `Authorization: Bearer ...`.
4. Sends content type, disables fetch caching, and forwards the upstream status/body/content type and content disposition.
5. On a 401 for a non-auth request, uses `pos_refresh_token` with `/auth/refresh`, updates HttpOnly cookies, and retries the original request once.
6. Clears both session cookies if the final response remains 401.

Supported methods: GET, POST, PATCH, PUT, DELETE. The route is a same-origin proxy and does not make an API call safe to retry if the upstream committed a mutation but the network failed before the browser received the response.

### Session requests

`src/app/api/pos-session/route.ts` owns browser session endpoints:

| Browser method | Purpose |
| --- | --- |
| `POST /api/pos-session` | For normal login, calls API `/auth/login`; for `{action: "verify-registration"}`, calls OTP verification. Fetches `/accounts/me` to expose business type/outlet type and sets access/refresh cookies on success. |
| `GET /api/pos-session` | Confirms session with `/accounts/me`; if needed, refreshes tokens and retries. |
| `PUT /api/pos-session` | Calls `/accounts/switch-store` and rotates tokens so the active store is in the new access token. |
| `DELETE /api/pos-session` | Clears the local auth cookies. |

Cookie attributes: HttpOnly, SameSite=Lax, path `/`, Secure in production. Access max age follows `expires_in` (fallback 15 minutes); refresh cookie max age is seven days. The browser JavaScript does not receive raw tokens.

## Typical client pattern

Use the POS API helper (`src/lib/pos-api.ts` if present; locate the canonical helper with `rg "function posApi|const posApi" src`) rather than ad hoc direct API fetches. Keep endpoint contracts close to the consuming feature, parse the response envelope, surface server errors, and invalidate or refresh affected data after successful writes. Never cache sensitive or branch-scoped records across an account/store change without including scope in the cache lifecycle.

For a sale, the UI may calculate a provisional display total, but should submit product IDs, quantities, chosen price mode and payment/customer details. The API re-reads the product in the active store, checks stock and price eligibility, calculates discount/tax/total, stores immutable transaction lines, and commits inventory changes. The UI must display the response as the committed result and should query the transaction record after ambiguous timeouts before retrying.

## Tenant and store scope

Organisation, user, role and active store scope are derived by the Go API from its verified access token. Do not accept a client-supplied org ID as authorization. `expected_store_id` on a sale is an optimistic context check, not a substitute for token scope. Branch switching is a server token exchange; merely changing a local selector is not enough.

UI navigation filters improve discoverability only. Server middleware and service validation are the access-control boundary. Pharmacy prescription register access has specific server role checks; owner/admin controls disposal/release. The API also validates business type for wholesale and pharmacy operations.

## Response/error handling

The API uses JSON envelopes for most application responses. Preserve useful server error messages and field details, but do not treat a successful HTTP response as proof an external M-Pesa/bank payment settled. A timeout around a write is ambiguous: inspect the ledger before resubmitting. A 401 after refresh clears session cookies; return the operator to sign-in. A 409 generally represents domain-state conflicts (for example stock, branch context, business type, or minimum quantity), and should be shown in a recoverable way.

## Endpoint families used by the workspace

The app calls the versioned API through the proxy. Principal families include `/auth`, `/accounts`, `/meta`, `/inventory`, `/pos`, `/customers`, `/credits`, `/suppliers`, `/expenses`, `/staff`, `/stores`, `/pharmacy`, `/analytics`, `/reports`, `/notifications`, `/subscriptions`, `/uploads`, `/ai`, and `/admin`. Exact methods, payloads, and auth requirements are in the API reference. A route being present in the Go API does not guarantee a page exists in this web app.

Known mismatch: the `/sms` UI calls `/messaging/*`, but those handlers are not registered in the Go API. Its wallet, sender-ID and key-management controls are currently nonfunctional against this backend. The API does implement customer SMS at `POST /customers/:id/sms`; this is a distinct endpoint and does not provide the `/sms` page's proposed wallet/campaign features.

## Local troubleshooting

| Symptom | Check |
| --- | --- |
| API requests return 502/connection errors | Go API reachable at `POS_API_URL`, correct scheme/port/path, API dependencies healthy. |
| Login succeeds upstream but session is absent | `/api/pos-session` response, `/accounts/me`, cookie Secure setting vs local HTTP, browser cookie policy. |
| Requests return 401 repeatedly | Refresh token validity, server signing configuration, clock, API refresh route, session cookies. |
| Wrong branch data | Active store in `/accounts/me`, successful `/api/pos-session` PUT and renewed access token. |
| UI says sale failed after a delay | Check `/transactions` before retry; transaction might have committed. |
| Browser CORS issue | Normal UI calls should be same-origin through Next.js. For direct API clients, configure API CORS origins separately. |

Never log passwords, raw access/refresh tokens, patient/prescriber details, or payment credentials. Keep secrets server-side and redact request/response bodies in diagnostics.
