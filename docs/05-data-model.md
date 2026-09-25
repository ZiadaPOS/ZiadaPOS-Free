# 05 — Data Model

PostgreSQL via GORM (`AutoMigrate` per app on boot — schema is kept in sync automatically).

## Common columns (`internal/models/base.go`)

Every entity embeds `BaseModel`:

```go
type BaseModel struct {
  ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
  CreatedAt time.Time      `gorm:"index;not null;default:now()"`
  UpdatedAt time.Time      `gorm:"index;not null;default:now()"`
  DeletedAt gorm.DeletedAt `gorm:"index"`   // soft delete everywhere
}
```

IDs are **UUIDv4** (generated in Go via `uuidx`). Tenancy comes from `org_id`/`store_id`
columns + the JWT, never from URLs.

---

## Tenant hierarchy

```
Organisation (the business)
  └── Stores (branches)
        ├── Users (owner + staff, belong to a store)
        ├── Products + Categories
        ├── Transactions (sale headers + lines)
        ├── Customers / CreditTabs / CreditPayments
        ├── Suppliers / Deliveries / SupplierPayments
        ├── Expenses
        └── ...
```

- `organisations` — `slug`, `name`, `tin`, `business_type`, `occupation`, location
  (`region`, `district`, `ward`, `street`, `address`, `latitude`, `longitude`), `country`
  (`TZ`), `currency` (`TZS`), `trial_ends_at`, **integration keys** (`sendafrica_api_key`,
  `ngamia_api_key`).
- `stores` — `org_id`, `name`, `code`, `area`, `address`, `lat`/`long`, `open_hours`, `status`
  (`open|closed|paused`).
- `users` — `org_id`, `store_id`, `role` (`admin|owner|staff`), `first_name`, `last_name`,
  `email`, `phone` (**varchar(10)**, `0…` form), bcrypt `password_hash`, `pin` (4-digit staff
  clock-in), `occupation`, `avatar_url`, `is_phone_verified`, `is_email_verified`, `is_active`,
  `last_login_at`.

## Auth & security

- `refresh_tokens` — user-scoped rotating refresh tokens (hashed `token_hash`, expiry, revoked).
- `phone_otps` — `phone`, `purpose`, `code`, `expires_at`, `attempts`.
- Redis (not Postgres): `login:lock:<identifier>` (attempts, 15-min TTL), `rl:*` rate-limit
  counters, JSON cache keys, AI usage counters.

## Inventory

- `categories` — `org_id`, `name`, `description`, `sort_order`.
- `products` — `org_id`, `store_id`, `category_id`, `name`, `sku`, `barcode` (unique per org),
  `selling_price`, `cost_price`, `stock`, `reorder_point`, `image`, `tax_rate`, `description`,
  `is_service`, `is_active`. Stock is a single per-store counter adjusted through
  `stock_adjustments` and sales.
- `stock_adjustments` — `product_id`, `qty_change`, `reason`
  (`restock|damage|correction|return`), `note`, `user_id`.

## POS

- `transactions` — `org_id`, `store_id`, `txn_number` (**sequential per store**; composite unique
  index `(store_id, txn_number)`), `subtotal`, `discount`, `tax`, `total`, `payment_method`
  (`Cash|M-Pesa|Tigo Pesa|Bank|Credit`), `payment_reference`, `customer_name`, `customer_phone`,
  `customer_id`, `till_number`, `status` (`paid|credit|refunded|void`), `notes`, `staff_user_id`.
- `transaction_items` — `transaction_id`, `product_id`, `product_name`, `qty`, `unit_price`,
  `discount`, `total`.

## Customers & credits

- `customers` — `org_id`, `store_id`, `name`, `phone`, `email`, `segment`
  (`New|Regular|VIP|Occasional`), `credit_limit` (‑1 = unlimited), `notes`; spend stats
  (`total_spend`, `transaction_count`, `last_visit_at`, `created_by`) maintained by
  **transaction events**.
- `credit_tabs` — `customer_id`, `credit_limit`, `outstanding_balance`, `status`
  (`open|partially_paid|settled|written_off`), `auto_reminder_threshold`? `last_reminder_at`.
- `credit_payments` — `tab_id`, `amount`, `payment_method`, `reference`, `note`.
- `customer_messages` — `customer_id`, `body`, `kind` (`whatsapp|call|sms`).
- `customer_notes` — `customer_id`, `body`.

## Suppliers

- `suppliers` — `org_id`, `store_id`, `name`, `category`, `rep`, `phone`, `email`, `address`,
  `payment_terms`, `notes`, `status` (`active|inactive|on_hold`).
- `supplier_deliveries` — `supplier_id`, `invoice_number`, `total_value`, `delivery_date`,
  `items_count`, `status`, `is_paid`, `notes`.
- `supplier_payments` — `supplier_id`, `delivery_id?`, `amount`, `payment_method`
  (`cash|bank|mobile`), `reference`, `notes`.

## Staff & expenses

- `expenses` — `org_id`, `store_id`, `title`, `amount`, `category`, `payment_method`,
  `payment_reference`, `receipt_url`, `notes`, `created_by`.
- Staff have **no extra tables** — identity is `users`, performance is derived from the
  transactions ledger.

## Notebook

- `notebook_notes` — `org_id`, `store_id`, `title`, `content`, `tags` (JSON array).

## AI

- `ai_conversations` — `org_id`, `user_id`, `title`, `model`, `message_count`.
- `ai_messages` — `conversation_id`, `role` (`user|assistant|system`), `content`.
- AI credit usage is a Redis counter keyed `ai:usage:<org>:<YYYY-MM>`; `accounts` computes
  remaining from the plan/allowance.

## Notifications

- `notifications` — `org_id`, `user_id`, `title`, `body`, `category`
  (`stock|credit|sales|ai|system|supplier`), `read`, `meta` (JSON).

## Reports & subscriptions

- `scheduled_reports` — `org_id`, `user_id`, `report_type` (`sales|inventory|tax|credit`), `name`,
  `frequency` (`daily|weekly|monthly`), `date_range_preset`, `recipients` (JSON), `is_active`,
  `next_send_at`, `last_sent_at`.
- `report_exports` — `org_id`, `report_type`, `range`, `format` (`json|csv`), `status`, `file`
  (MinIO key).
- `subscriptions` — `org_id`, `plan_id`, `status`, `starts_at`, `ends_at`, `extra_stores` (admin
  grant), `trial_ends_at`.
- `plans` — `name`, `slug`, `price` (monthly TZS), `duration_days`, `store_limit`, `features`
  (JSON), `is_active`. Seed plans: **Free**, **Pro**, **Enterprise**.
- `reviews` — `org_id`, `store_id`, `rating`, `comment`, `status` (public moderation).

## Reviews

- `reviews` — public listing endpoint; posted by authenticated users.

## Messaging & public commerce

See [08 — Messaging & Commerce](08-messaging-and-commerce.md) for the full contract.

- `sender_id_requests` — tenant sender-ID approval workflow; `ZIADA POS` is the fallback.
- `messaging_settings` — sender selection, automation toggles, templates, quiet hours, and limits.
- `sms_wallets` — organisation-level one-time 100-SMS grant plus paid/reserved/consumed balance;
  shared by the shop's stores and authorised staff.
- `sms_ledger_entries` — immutable SMS credit accounting and provider references.
- `sms_campaigns` — bulk promotional or operational sends.
- `sms_messages` — one-recipient delivery record with provider status and IDs.
- `storefront_settings` — public store slug, branding, catalogue visibility, and delivery rules.
- `orders` / `order_lines` — public customer orders with immutable price snapshots.
- `invoices` — invoice numbers and secure public invoice tokens.

SendAfrica credentials are organisation-owned and must be encrypted at rest. Store only the
encrypted secret plus a fingerprint and lifecycle timestamps in application data; never expose
the raw key in API responses or staff-facing screens.

---

## Events that update this data

| Event | Side effects |
| --- | --- |
| `user.registered` | welcome email (Mailer), trial subscription creation |
| `staff.created` | SMS temp password (SendAfrica) |
| `transaction.created` | credits: open/settle `credit_tabs`; customers: spend/segment stats; notifications; analytics |
| `transaction.refunded` / `transaction.voided` | credits: settle tab; customers: reduce spend; notifications |
