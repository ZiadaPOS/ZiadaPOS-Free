# 04 — API Reference

Base path: **`/api/v1`** (configurable via `API_BASE_PATH`). Everything except the health/meta/
public endpoints requires `Authorization: Bearer <access_token>`.

Conventions:

- **Envelope**: `{success, message, data, errors, meta}` (see [03 — Auth & Security](03-auth-and-security.md)).
- **IDs** are UUIDv4. **Money** is integer **TZS**. **Dates** are `YYYY-MM-DD`.
- **List endpoints** accept `page` + `page_size` and return a `meta` pagination block.
- Scope is derived from the token (`org_id` + `store_id`) — never from the URL.
- Owner-only = requires `admin` or `owner` role. Admin-only = requires `admin`.

---

## System

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/healthz` | — | Liveness: always `{status: ok}`. |
| GET | `/` | — | App info: name, version, env. |
| GET | `/api/v1/health` | — | Deep health: DB ping → `status` `ok`/`degraded`, `db`, `redis`, `uptime`, `runtime`. |

---

## Auth — `/auth`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/auth/register` | — | Register an owner: creates organisation + main store + owner user + **7-day trial** subscription. Payload: `first_name`, `last_name?`, `phone`, `password`, `organisation_name`, `business_type` (slug), `occupation` (slug), `region` (slug), `district` (slug), `latitude?`, `longitude?`. Returns user + token pair. |
| POST | `/auth/otp/send` | — | Send a 6-digit OTP by SMS (`phone`, `purpose`). |
| POST | `/auth/otp/verify` | — | Verify an OTP (`phone`, `purpose`, `code`); marks phone verified. |
| POST | `/auth/login` | — | Login with `identifier` (phone or email) + `password`; returns `{access_token, refresh_token, expires_in, token_type, user}`. 5 failures → 15-min lockout. |
| POST | `/auth/refresh` | — | Exchange `refresh_token` for a new pair. |
| GET | `/auth/confirm-email` | — | `token` → confirm email address (link from the welcome email). |
| POST | `/auth/password/forgot` | — | `identifier` → sends a reset OTP by SMS. |
| POST | `/auth/password/reset` | — | `identifier`, `otp`, `new_password` → re-hash + clear lock. |
| POST | `/auth/logout` | JWT | Log out (client discards tokens). |
| POST | `/auth/resend-confirmation` | JWT | Re-send the email confirmation. |

---

## Meta — `/meta` (public reference data)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/meta/business-types` | — | Onboarding list of business types (slugs). |
| GET | `/meta/regions` | — | Tanzanian regions (slugs). |
| GET | `/meta/regions/:slug/districts` | — | Districts for a region. |
| GET | `/meta/occupations` | — | Owner job options (slugs). |

---

## Accounts — `/accounts`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/accounts/me` | JWT | Current user: profile, active store, all stores. |
| PATCH | `/accounts/me` | JWT | Update profile (`first_name`, `last_name`, `email`, `occupation`, `password`). |
| GET | `/accounts/organisation` | Owner | Business profile: name, TIN, plan, trial, location, integrations. |
| PATCH | `/accounts/organisation` | Owner | Update business profile + integration keys (SendAfrica, Ngamia). |
| POST | `/accounts/switch-store` | JWT | `store_id` → mint a new token pair with the new store embedded. |
| GET | `/accounts/users` | Owner | All org users (owners + staff). |
| POST | `/accounts/users` | Owner | Create a user (staff by default); temp password is sent by SMS/email (`staff.created` event). |
| DELETE | `/accounts/users/:id` | Owner | Deactivate a user. |
| GET | `/accounts/ai-credits` | JWT | Org's monthly AI credit usage + remaining. |

---

## Inventory — `/inventory`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/inventory/categories` | JWT | Org categories. |
| POST | `/inventory/categories` | JWT | Create category (`name`, `description?`, `sort_order?`). |
| PATCH | `/inventory/categories/:id` | JWT | Partial update. |
| DELETE | `/inventory/categories/:id` | JWT | Delete (products keep data, lose the link). |
| GET | `/inventory/products` | JWT | List: `search?`, `category_id?`, `status?`, `low_stock?`, `page`, `page_size`. |
| POST | `/inventory/products` | JWT | Create product (`name`, `selling_price`, `cost_price?`, `category_id?`, `sku?`, `barcode?`, `stock?`, `reorder_point?`, `image?`, `tax_rate?`, `description?`, `is_service?`, `is_active?`). Stock is opening quantity. |
| GET | `/inventory/products/:id` | JWT | One product. |
| PATCH | `/inventory/products/:id` | JWT | **Partial update** — only provided fields change (`UpdateProductInput`, pointer fields). Stock must use `/adjust`. |
| DELETE | `/inventory/products/:id` | JWT | Soft-delete (deactivate). |
| GET | `/inventory/products/barcode/:code` | JWT | Find by barcode (scanner / phone lookup). |
| POST | `/inventory/products/:id/adjust` | JWT | `qty_change`, `reason` (restock/damage/correction/return), `note?` → record `StockAdjustment` + update stock. |
| GET | `/inventory/products/:id/adjustments` | JWT | Adjustment history (`limit`). |
| GET | `/inventory/low-stock` | JWT | Products at/below reorder point. |

---

## POS — `/pos`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/pos/sale` | JWT | Complete a sale. Body: `lines` (`[{product_id, qty}]`), `payment_method` (`Cash|M-Pesa|Tigo Pesa|Bank|Credit`), `payment_reference?`, `discount_pct?`, `customer_name?`, `customer_phone?`, `customer_id?`, `till_number?`, `notes?`. Decrements stock; generates a per-store `txn_number`; `Credit` fires events that open a credit tab. |
| GET | `/pos/transactions` | JWT | List: `status?` (`paid|credit|refunded|void`), `method?`, `search?`, `customer_id?`, `date_from?`, `date_to?`, `page`, `page_size`. |
| GET | `/pos/transactions/:id` | JWT | One transaction with itemised lines. |
| POST | `/pos/transactions/:id/refund` | JWT | Refund a paid sale → restore stock, mark `refunded`. |
| POST | `/pos/transactions/:id/void` | JWT | Void a transaction → restore stock, mark `void`. |
| GET | `/pos/summary` | JWT | Revenue / transactions / profit for `date_from?`–`date_to?` (default today). |
| GET | `/pos/summary/hourly` | JWT | Hourly revenue breakdown for `date?`. |

> `txn_number` is **sequential per store**: a composite unique index
> `(store_id, txn_number)` guarantees no duplicates across branches.

---

## Customers — `/customers`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/customers` | JWT | List: `search?`, `segment?` (`New|Regular|VIP|Occasional`), `has_credit?`, paged. |
| POST | `/customers` | JWT | Create (`name`, `phone?`, `email?`, `segment?`, `credit_limit?`, `notes?`). |
| GET | `/customers/credits` | JWT | Customers with outstanding credit (`overdue?`). |
| GET | `/customers/:id` | JWT | Full profile (spend stats maintained by POS events). |
| PATCH | `/customers/:id` | JWT | Partial update (`credit_limit: -1` clears). |

---

## Credits — `/credits`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/credits` | JWT | Tabs: `status?` (`open|partially_paid|settled|written_off`), `customer_id?`, `overdue?`, paged. |
| GET | `/credits/customers/:customer_id` | JWT | Tab + balance + payment history for one customer. |
| POST | `/credits/customers/:customer_id/payments` | JWT | `amount`, `method` (`Cash|M-Pesa|Tigo Pesa|Bank`), `reference?`, `note?` → record payment, update tab. |
| POST | `/credits/customers/:customer_id/remind` | JWT | SMS reminder about outstanding credit (needs phone + SMS configured). |
| POST | `/credits/customers/:customer_id/messages` | JWT | Log a message (`body`, `kind` = `whatsapp|call|sms`). |
| POST | `/credits/customers/:customer_id/notes` | JWT | Internal note (`body`). |
| POST | `/credits/:tab_id/write-off` | JWT | Forgive a tab (irreversible). |

> A `transaction.created` with method `Credit` **auto-creates** the tab; `transaction.refunded` /
> `transaction.voided` auto-settle it.

---

## Suppliers & deliveries — `/suppliers`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/suppliers` | JWT | List: `search?`, `status?` (`active|inactive|on_hold`), paged. |
| POST | `/suppliers` | JWT | Create (`name`, `category?`, `rep?`, `phone?`, `email?`, `address?`, `payment_terms?`, `notes?`). |
| GET | `/suppliers/stats` | JWT | Counts + total outstanding. |
| GET | `/suppliers/deliveries` | JWT | All deliveries: `supplier_id?`, `unpaid?`, paged. |
| POST | `/suppliers/deliveries` | JWT | Record a delivery (`supplier_id`, `total_value`, `invoice_number?`, `delivery_date?`, `items_count?`, `status?`, `is_paid?`, `notes?`). |
| GET | `/suppliers/deliveries/:id` | JWT | One delivery. |
| PATCH | `/suppliers/deliveries/:id` | JWT | Partial update. |
| DELETE | `/suppliers/deliveries/:id` | JWT | Delete a delivery. |
| POST | `/suppliers/deliveries/:id/pay` | Owner | Pay a specific delivery. |
| GET | `/suppliers/payments` | JWT | Payments ledger. |
| GET | `/suppliers/:id` | JWT | One supplier incl. outstanding balance. |
| PATCH | `/suppliers/:id` | JWT | Partial update. |
| DELETE | `/suppliers/:id` | JWT | Soft-delete. |
| GET | `/suppliers/:id/deliveries` | JWT | Deliveries for one supplier. |
| POST | `/suppliers/:id/deliveries` | JWT | Add a delivery for one supplier. |
| GET | `/suppliers/:id/payments` | JWT | Payments for one supplier. |
| POST | `/suppliers/:id/pay` | Owner | `amount`, `payment_method` (`cash|bank|mobile`), `reference?`, `delivery_id?`, `notes?`. |

---

## Expenses — `/expenses`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/expenses` | JWT | List: `search?`, `category?`, `payment_method?`, `ordering?`, paged. |
| POST | `/expenses` | JWT | Create (`title`, `amount`, `category?`, `payment_method?`, `payment_reference?`, `notes?`, `receipt_url?`). |
| GET | `/expenses/summary` | JWT | Totals by period + category. |
| GET | `/expenses/:id` | JWT | One expense. |
| PATCH | `/expenses/:id` | JWT | Partial update. |
| DELETE | `/expenses/:id` | JWT | Delete. |

---

## Staff — `/staff` (owner)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/staff` | Owner | List: `search?`, `role?`, `status?`, `shift?`, `include_stats?`, paged. |
| POST | `/staff` | Owner | Create staff (`first_name`, `last_name?`, `phone?`, `email?`, `role?`, `store_id?`, `shift?`, `employment_status?`, `pin?`, `can_refund?`, `can_discount?`, `can_view_reports?`). Temp password SMS'd via `staff.created` event. |
| GET | `/staff/kpis` | Owner | Team KPIs per store. |
| GET | `/staff/:id` | Owner | One staff member. |
| PATCH | `/staff/:id` | Owner | Update profile. |
| DELETE | `/staff/:id` | Owner | Deactivate. |
| GET | `/staff/:id/stats` | Owner | Sales stats (transactions, revenue, avg ticket). |
| GET | `/staff/:id/activity` | Owner | Clocking/activity log for `date?` (default today). |
| PATCH | `/staff/:id/shift` | Owner | Change shift. |
| PATCH | `/staff/:id/permissions` | Owner | Change POS permissions. |

> Staff identity reuses `accounts.users`; stats are computed from the transactions ledger (no
> extra tables).

---

## Stores — `/stores` (owner)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/stores` | Owner | List branches (`status?`). |
| POST | `/stores` | Owner | Create branch (name, code, area, address, GPS, hours…). |
| GET | `/stores/stats` | Owner | Org-wide per-branch stats. |
| GET | `/stores/:id` | Owner | Branch details. |
| PATCH | `/stores/:id` | Owner | Update branch. |
| DELETE | `/stores/:id` | Owner | Delete branch. |
| GET | `/stores/:id/staff` | Owner | Staff roster. |
| GET | `/stores/:id/week` | Owner | Weekly performance. |
| PATCH | `/stores/:id/status` | Owner | `open` / `closed` / `paused`. |

---

## Notebook — `/notebook`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/notebook` | JWT | Notes: `tag?`, `search?`, `ordering?`, paged. |
| POST | `/notebook` | JWT | Create (`title`, `content?`, `tags?`). |
| GET | `/notebook/:id` | JWT | One note. |
| PATCH | `/notebook/:id` | JWT | Update. |
| DELETE | `/notebook/:id` | JWT | Delete. |

---

## Reviews — `/reviews`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/reviews/public` | — | Public store reviews. |
| POST | `/reviews` | JWT | Submit a review. |

---

## AI — `/ai` (Ngamia)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/ai/chat` | JWT | Start a chat with the model; streams a response; consumes AI credits. |
| POST | `/ai/conversations/:id/chat` | JWT | Continue a conversation. |
| GET | `/ai/conversations` | JWT | List conversations. |
| GET | `/ai/conversations/:id` | JWT | One conversation + messages. |
| PATCH | `/ai/conversations/:id` | JWT | Rename/update a conversation. |
| GET | `/ai/suggestions` | JWT | Starter suggestions. |
| GET | `/ai/usage` | JWT | Monthly credit usage. |

---

## Reports — `/reports`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/reports/types` | JWT | Available types (`sales|inventory|tax|credit`). |
| POST | `/reports/generate` | JWT | `report_type`, `range?` (`7d|30d|90d|month|ytd`), `date_from?`, `date_to?`, `format?` (`json|csv`) → report data (CSV may be stored as an export). |
| GET | `/reports/exports` | JWT | Past generated exports. |
| GET | `/reports/exports/:id/download` | JWT | Re-generate + download an export. |
| GET | `/reports/scheduled` | Owner | Scheduled reports. |
| POST | `/reports/scheduled` | Owner | Create (`report_type`, `name?`, `frequency` `daily|weekly|monthly`, `date_range_preset?`, `recipients?`). |
| PATCH | `/reports/scheduled/:id` | Owner | Update. |
| DELETE | `/reports/scheduled/:id` | Owner | Delete. |
| POST | `/reports/scheduled/:id/send` | Owner | Send immediately. |

> A daily cron job (06:30 EAT) dispatches every enabled scheduled report whose `next_send_at`
> has arrived (`apps/reports → DispatchDue`).

---

## Notifications — `/notifications`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/notifications` | JWT | List: `category?` (`stock|credit|sales|ai|system|supplier`), `unread?`, paged. |
| GET | `/notifications/unread-count` | JWT | Unread count. |
| POST | `/notifications` | JWT | Create a notification. |
| PATCH | `/notifications/read-all` | JWT | Mark all read. |
| PATCH | `/notifications/:id/read` | JWT | Mark one read. |
| DELETE | `/notifications/:id` | JWT | Delete. |

---

## Subscriptions — `/subscriptions`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/subscriptions/plans` | — | Public plan catalogue. |
| GET | `/subscriptions/plans/:id` | — | One plan. |
| GET | `/subscriptions/my-subscription` | Owner | Current org subscription. |
| GET | `/subscriptions/store-limit` | Owner | Store limit vs used. |
| POST | `/subscriptions/plans` | Admin | Create plan. |
| PATCH | `/subscriptions/plans/:id` | Admin | Update plan. |
| DELETE | `/subscriptions/plans/:id` | Admin | Deactivate plan. |
| GET | `/subscriptions/all` | Admin | List all org subscriptions. |
| GET | `/subscriptions/all/:id` | Admin | One subscription. |
| POST | `/subscriptions/all/:id/activate` | Admin | Activate / grant. |
| PATCH | `/subscriptions/all/:id/extra-stores` | Admin | Adjust extra-store allowance. |

> Registration auto-creates a **7-day trial** via the `user.registered` event
> (`apps/subscriptions`).

---

## Uploads — `/uploads`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/uploads` | JWT | Multipart upload (`file`, `kind` = `image`, `folder` = e.g. `products`). Stored in MinIO at `<org>/<folder>/<uuid>.<ext>`; returns the public `url`. |
| DELETE | `/uploads` | JWT | Delete an object by its public URL / key. |
