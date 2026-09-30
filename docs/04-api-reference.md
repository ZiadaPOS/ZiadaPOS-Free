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
| POST | `/auth/register` | — | Register an owner: creates organisation + main store + owner user + **7-day trial** subscription. Payload: `first_name`, `last_name?`, `phone`, `email?`, `password`, `confirm_password`, `shop.name`, `shop.business_type`, `shop.region`, optional outlet and location details. Returns pending phone verification details. |
| POST | `/auth/otp/send` | — | Send a 6-digit OTP by SMS (`phone`, `purpose`). |
| POST | `/auth/otp/verify` | — | Verify an OTP (`phone`, `purpose`, `code`); marks phone verified. |
| POST | `/auth/login` | — | Login with `identifier` (phone or email) + `password`; returns `{access_token, refresh_token, expires_in, token_type, user}`. 5 failures → 15-min lockout. |
| POST | `/auth/mfa/totp/verify` | — | Complete an enrolled admin login with `mfa_token` and six-digit TOTP code. |
| POST | `/auth/refresh` | — | Exchange `refresh_token` for a new pair. |
| GET | `/auth/confirm-email` | — | `token` → confirm email address (link from the welcome email). |
| POST | `/auth/password/forgot` | — | `identifier` → sends a reset OTP by SMS. |
| POST | `/auth/password/reset` | — | `identifier`, `otp`, `new_password` → re-hash + clear lock. |
| POST | `/auth/logout` | JWT | Log out (client discards tokens). |
| POST | `/auth/resend-confirmation` | JWT | Re-send the email confirmation. |

## Admin MFA — `/accounts/mfa/totp`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/accounts/mfa/totp/status` | Admin JWT | Read current admin MFA status. |
| POST | `/accounts/mfa/totp/setup` | Admin JWT | Generate a TOTP enrollment QR code and temporary secret. |
| POST | `/accounts/mfa/totp/verify` | Admin JWT | Confirm enrollment with a six-digit code. |
| POST | `/accounts/mfa/totp/disable` | Admin JWT | Disable MFA after password and TOTP verification. |

---

## Meta — `/meta` (public reference data)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/meta/business-types` | — | Onboarding list of business types (slugs). |
| GET | `/meta/regions` | — | Tanzanian regions (slugs). |
| GET | `/meta/regions/:slug/districts` | — | Districts for a region. |
| GET | `/meta/outlet-types` | — | Pharmacy outlet options (pharmacy, ADDO). |

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

Pharmacy accounts use `GET /inventory/batches?product_id={id}` to review lots and
`POST /inventory/products/{id}/batches` to receive a lot. Set
`reconcile_existing_stock: true` to assign already-counted product stock to a real
batch without increasing the product stock a second time. Pharmacy checkout allocates
unexpired stock by FEFO.

Wholesale products support `wholesale_price` and `wholesale_minimum`. Sale requests
accept `price_mode` (`retail|wholesale`); wholesale-only organisations always use
wholesale pricing, while Retail + Wholesale organisations can choose per sale. A
wholesale sale is rejected when a price is missing or the quantity is below minimum.

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
| POST | `/pos/sale` | JWT | Complete a sale. Body: `lines` (`[{product_id, qty}]`), `payment_method` (`Cash|M-Pesa|Tigo Pesa|Bank|Credit`), `payment_reference?`, `discount_pct?`, `customer_name?`, `customer_phone?`, `customer_id?`, `till_number?`, `notes?`. For pharmacy products marked prescription-only, each line must include patient name/address, medicine strength/form, prescription reference, prescriber name/address/registration, prescribing facility name/address, and `prescriber_authenticity_confirmed: true`. The field records the cashier's check; it is not electronic verification. Decrements stock; generates a per-store `txn_number`; `Credit` fires events that open a credit tab. |
| GET | `/pos/transactions` | JWT | List: `status?` (`paid|credit|refunded|void`), `method?`, `search?`, `customer_id?`, `date_from?`, `date_to?`, `page`, `page_size`. |
| GET | `/pos/transactions/:id` | JWT | One transaction with itemised lines. |
| POST | `/pos/transactions/:id/refund` | JWT with refund permission | Refund a paid sale and mark `refunded`; pharmacy-returned units are placed in quarantine and are excluded from sellable stock until inspected/released. |
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

Pharmacy accounts can create purchase orders with `POST /suppliers/purchase-orders`
using `supplier_id`, `notes?`, and `lines` (`product_id`, ordered `quantity`, `unit_cost`).
For each product, assign a supplier so its configured `lead_time_days` is used by the
deterministic reorder engine. `GET /suppliers/reorder-suggestions` reports 30-day sales,
lead-time demand, safety stock, unexpired sellable quantity, suggested quantity, and the
calculation reason. Unassigned products use a seven-day lead-time default.
`GET /suppliers/purchase-orders` lists open and completed orders. Receive all or part of
an order with `POST /suppliers/purchase-orders/:id/receive`; each line supplies its
`line_id`, received `quantity`, `batch_number`, and `expiry_date`. Receipt atomically
creates supplier delivery history, medicine batches, movements, and product stock.

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/suppliers` | JWT | List: `search?`, `status?` (`active|inactive|on_hold`), paged. |
| POST | `/suppliers` | JWT | Create (`name`, `category?`, `rep?`, `phone?`, `email?`, `address?`, `payment_terms?`, `lead_time_days?`, `notes?`). |
| GET | `/suppliers/stats` | JWT | Counts + total outstanding. |
| GET | `/suppliers/reorder-suggestions` | JWT | Pharmacy only: deterministic suggestions using the last 30 days of paid/credit sales, supplier lead time, product reorder point, and sellable unexpired stock. |
| POST | `/suppliers/purchase-orders/bulk` | JWT | Pharmacy only: atomically create supplier-grouped orders (`groups`: `supplier_id`, `lines[]` with `product_id`, `quantity`, `unit_cost`). |
| GET | `/suppliers/deliveries` | JWT | All deliveries: `supplier_id?`, `unpaid?`, paged. |
| POST | `/suppliers/deliveries` | JWT | Record a delivery (`supplier_id`, `total_value`, `invoice_number?`, `delivery_date?`, `items_count?`, `status?`, `is_paid?`, `notes?`, `lines?`). Pharmacy delivered lines require `product_id`, `quantity`, `unit_cost`, `batch_number`, and `expiry_date`; invoice, stock, lots and movements commit atomically. |
| GET | `/suppliers/deliveries/:id` | JWT | One delivery. |
| PATCH | `/suppliers/deliveries/:id` | JWT | Partial update. |
| DELETE | `/suppliers/deliveries/:id` | JWT | Delete a delivery. |
| POST | `/suppliers/deliveries/:id/pay` | Owner | Pay a specific delivery. |
| GET | `/suppliers/payments` | JWT | Payments ledger. |
| GET | `/suppliers/:id` | JWT | One supplier incl. outstanding balance. |
| PATCH | `/suppliers/:id` | JWT | Partial update. |
| DELETE | `/suppliers/:id` | JWT | Soft-delete. |
| GET | `/suppliers/:id/deliveries` | JWT | Deliveries for one supplier. |
| POST | `/suppliers/:id/deliveries` | JWT | Add a delivery for one supplier (same itemized pharmacy line rules as above). |
| GET | `/suppliers/:id/payments` | JWT | Payments for one supplier. |
| POST | `/suppliers/:id/pay` | Owner | `amount`, `payment_method` (`cash|bank|mobile`), `reference?`, `delivery_id?`, `notes?`. |

## Pharmacy — `/pharmacy`

All routes require a pharmacy organisation. Batch receiving and expiry listing are scoped to the
active store. Disposal is limited to owners and admins and requires a positive quantity and an
audit reason; it writes a negative batch movement and stock adjustment in one transaction.

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/pharmacy/batches` | JWT | List batches, including depleted/expired lots; optional `product_id`. |
| GET | `/pharmacy/batches/:id/movements` | JWT | Paginated store-scoped append-only movement history for a batch (`page?`, `page_size?`). |
| POST | `/pharmacy/products/:id/batches` | JWT | Receive a batch with batch number, quantity, expiry, cost and supplier; may assign existing untracked stock. |
| POST | `/pharmacy/batches/:id/dispose` | Owner | Remove `quantity` from a lot with a required `reason`; records ledger movement and adjustment. |
| POST | `/pharmacy/batches/:id/release` | Owner | Release a quarantined customer-return lot after recording an `inspection_note`; expired lots cannot be released. |
| GET | `/pharmacy/expiry?days=60` | JWT | List positive-quantity batches expiring by the requested window, including expired lots. |
| GET | `/pharmacy/prescriptions` | JWT + pharmacy management/pharmaceutical personnel | Paginated store-scoped prescription register: `search?`, `from?`, `to?`, `page?`, `page_size?`. Includes staff-entered patient/prescriber/facility details and allocated batch numbers; the authenticity field is a staff confirmation, not an electronic verification. |
| GET | `/pharmacy/products` | JWT | List medicine profiles. |
| GET | `/pharmacy/products/:id` | JWT | Read a product's medicine profile. |
| PUT | `/pharmacy/products/:id` | JWT | Update generic/brand names, unit, pack size, shop-entered schedule, registration number and prescription flag. |

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

## Messaging — `/messaging`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/messaging/settings` | Owner | Sender, automation, quiet-hour, opt-out, and low-balance settings. |
| PATCH | `/messaging/settings` | Owner | Update messaging settings. |
| GET | `/messaging/balance` | Owner | Free, paid, reserved, consumed, available credits, and provider connection state. |
| GET | `/messaging/usage` | Owner | Usage totals and immutable ledger summary. |
| GET | `/messaging/topups` | Owner | Top-up history and provider references. |
| POST | `/messaging/provider-key` | Owner | Connect or rotate the shop owner's SendAfrica API key. The raw key is write-only. |
| DELETE | `/messaging/provider-key` | Owner | Disconnect the shop's SendAfrica account after queued sends finish. |
| GET | `/messaging/sender-id` | Owner | Active sender and request status. Defaults to `ZIADA POS`. |
| GET | `/messaging/sender-id/requirements` | Owner | Read sender-ID requirements from the connected shop SendAfrica account. |
| POST | `/messaging/sender-id/request` | Owner | Submit a sender-ID request through the connected shop SendAfrica account. |
| GET | `/messaging/sender-id/status` | Owner | Read approval state from the connected shop account. |
| GET | `/messaging/templates` | Owner | List SMS templates. |
| POST | `/messaging/templates` | Owner | Create a transactional or promotional template. |
| PATCH | `/messaging/templates/:id` | Owner | Update a template. |
| DELETE | `/messaging/templates/:id` | Owner | Deactivate a template. |
| POST | `/messaging/send` | Owner | Send one message after reserving SMS credits. |
| POST | `/messaging/campaigns` | Owner | Create a deduplicated bulk campaign. Maximum provider batch is 1,000 recipients. |
| GET | `/messaging/campaigns` | Owner | List campaigns and send totals. |
| GET | `/messaging/campaigns/:id` | Owner | Campaign details and recipient outcomes. |
| GET | `/messaging/messages/:id` | Owner | One message delivery record. |
| POST | `/webhooks/sendafrica` | — | Provider delivery callback; authenticated by webhook signature. |

Each shop owner/organisation receives a **one-time 100-SMS grant** shared by that shop's stores
and authorised staff. Operational sends are made only through that shop's connected SendAfrica
API key. When the free grant is exhausted, the owner tops up their own provider balance. The
wallet rejects sends with `provider_key_required` or `insufficient_sms_credits` when no usable
balance is available.

---

## Public storefront — `/public`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/public/stores/:slug` | — | Published store profile and storefront settings. |
| GET | `/public/stores/:slug/categories` | — | Visible product categories. |
| GET | `/public/stores/:slug/products` | — | Searchable visible catalogue. |
| POST | `/public/stores/:slug/orders` | — | Create an order from cart lines, customer details, and pickup/delivery choice. |
| GET | `/public/orders/:token` | — | Read an order using its secure public token. |
| POST | `/public/orders/:token/cancel` | — | Cancel an eligible public order. |

## Commerce — `/commerce`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/commerce/settings` | Owner | Read public-store, delivery, payment, and sharing settings. |
| PATCH | `/commerce/settings` | Owner | Configure or publish the storefront. |
| GET | `/commerce/orders` | JWT | List incoming orders by status and date. |
| GET | `/commerce/orders/:id` | JWT | Order, lines, customer, status history, and invoice. |
| PATCH | `/commerce/orders/:id/status` | JWT | Confirm, prepare, ready, dispatch, complete, reject, or cancel. |
| GET | `/commerce/orders/:id/invoice` | JWT | Download or view the generated invoice. |

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

### Planned Snippe callbacks

These public HTTPS routes are reserved for the ZiadaPOS payment module and are not active until
that module is deployed:

```text
POST /subscriptions/webhooks/snippe
POST /commerce/webhooks/snippe
```

They must verify the raw-body HMAC signature using `SNIPPE_WEBHOOK_SECRET`, deduplicate by
provider event/payment reference, and record ZiadaPOS's own transaction before applying any
subscription or order state change.

> Registration auto-creates a **7-day trial** via the `user.registered` event
> (`apps/subscriptions`).

## Platform monitoring — `/admin/monitoring`

All monitoring routes require a platform admin JWT.

| Method | Route | Purpose |
|---|---|---|
| GET | `/admin/monitoring/overview` | Platform-wide organisations, users, stores, products, low stock, customers, 30-day revenue, expenses, open credits, and unread alerts. |
| GET | `/admin/monitoring/activity?limit=50` | Latest platform sales activity, including organisation, amount, and timestamp. |
| GET | `/admin/monitoring/health` | API and database readiness status with generation timestamp. |

---

## Uploads — `/uploads`

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/uploads` | JWT | Multipart upload (`file`, `kind` = `image`, `folder` = e.g. `products`). Stored in MinIO at `<org>/<folder>/<uuid>.<ext>`; returns the public `url`. |
| DELETE | `/uploads` | JWT | Delete an object by its public URL / key. |
