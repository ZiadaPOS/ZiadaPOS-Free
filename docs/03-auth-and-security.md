# 03 — Auth & Security

## Roles

| Role | Meaning | Allowed on |
| --- | --- | --- |
| `admin` | Platform operator (Cameltech) | Subscriptions admin endpoints, everything owners can do |
| `owner` | Business owner / manager | Everything in their org (analytics, staff, stores, reports scheduling, subscriptions) |
| `staff` | Cashier / employee | POS, inventory, customers, credits, notebook, notifications, profile |

Middleware helpers:

- `middleware.Auth(tokens)` — requires a valid access JWT; injects `user_id`, `role`, `org_id`,
  `store_id` and claims into the Gin context.
- `middleware.RequireOwner()` — allows `admin` + `owner`.
- `middleware.RequireAdmin()` — allows only `admin`.

## Passwords

- bcrypt (`DefaultCost`) — hashed at registration and on every reset; never stored in plaintext.
- Registration normalises the phone to the 10-digit Tanzanian form (`0712345678`), validates the
  password, and stores a 4-digit staff `PIN` for POS clock-in.

## JWT (`internal/security`)

Two tokens, both HS256, signed with **different secrets**:

| Token | Secret | TTL | Purpose |
| --- | --- | --- | --- |
| access | `JWT_ACCESS_SECRET` | 15 min | Bearer auth on every request |
| refresh | `JWT_REFRESH_SECRET` | 7 days | `POST /auth/refresh` → new pair |

Claims (`Claims` struct):

```json
{
  "uid":      "<uuid>",      // user id
  "role":     "owner",       // admin | owner | staff
  "org_id":   "<uuid>",      // organisation (tenant) id
  "store_id": "<uuid>",      // active store id
  "typ":      "access"|"refresh",
  "sub": "<uuid>", "exp": …, "iat": …, "jti": …
}
```

Because `org_id` and `store_id` live **inside the token**, switching the active store requires a
fresh token pair: `POST /accounts/switch-store`.

## Login flow & lockout

`POST /auth/login` (identifier = phone or email, `password`):

1. Reads `login:lock:<identifier>` in Redis.
2. If attempts ≥ 5 → `423 Account temporarily locked …` (locked for **15 minutes**).
3. Finds the user, `bcrypt.CompareHashAndPassword`.
4. On failure: `recordFailedLogin` increments the counter (TTL 15 min) → wrong-password 401.
5. On success: clears the lock, updates `last_login`, issues tokens.

Lockout can be cleared from Redis:

```bash
redis-cli DEL "login:lock:<identifier>"
```

> This is exactly what bit the MCP agent when its healthcheck hammered login — see the agent's
> `docs/08-deployment.md`.

## OTP (phone verification)

- `POST /auth/otp/send` — generates a 6-digit code, stores `PhoneOTP` (purpose-scoped, expiry,
  attempt counter) and sends it via **SendAfrica SMS**.
- `POST /auth/otp/verify` — checks the code, marks verified (`is_phone_verified`), optionally
  returns a token pair.

## Email verification

- `user.registered` event → `sendConfirmationEmail` (via the Mailer) with a signed link to
  `GET /auth/confirm-email`.
- `POST /auth/resend-confirmation` re-sends.
- `is_email_verified` is set when confirmed.

## Password reset

- `POST /auth/password/forgot` — validates the identifier, sends an OTP by SMS.
- `POST /auth/password/reset` — OTP + new password → re-hash, clear the login lock.

## Rate limiting (`internal/middleware/ratelimit.go`)

Redis fixed-window limiter. Key prefers authenticated user id, else client IP, always keyed by
route (`rl:user:<id>:<path>` / `rl:ip:<ip>:<path>`). **Fails open** if Redis is down (the API
never hard-crashes on a limiter). Used for auth endpoints (login, register, OTP).

## Error model (`internal/apierr`)

Services raise structured `*apierr.Error{Status, Code, Message, Fields}`; handlers translate to
the response envelope. GORM not-found → 404; everything else → 500. Machine codes: `validation_error`,
`unauthorized`, `forbidden`, `not_found`, `conflict`, `account_locked`, `rate_limited`,
`internal_error`.

## Response envelope (`internal/response`)

Every endpoint returns:

```json
{ "success": true, "message": "OK", "data": {…}, "errors": null, "meta": {…} }
```

`meta` carries pagination on list endpoints: `total_count`, `current_page`, `page_size`,
`total_pages`, `has_next`, `has_prev`.
