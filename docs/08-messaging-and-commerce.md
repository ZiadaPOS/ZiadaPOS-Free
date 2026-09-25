# 08 — Messaging, SMS Credits & Public Commerce

This document defines the messaging and public-commerce contract for ZiadaPOS.

## Sender IDs

- Every shop connects its own SendAfrica account by pasting its API key into ZiadaPOS.
- Sender-ID requests are submitted directly to SendAfrica through that shop's account; ZiadaPOS
  does not own or approve sender-ID applications.
- A requested sender ID is unusable while it is `draft`, `submitted`, `under_review`, or `rejected`.
- Only a provider-approved sender ID can become the shop's active sender.
- The UI may display **ZIADA POS** as the initial product default, but operational sending must
  use a sender available to the connected shop account.
- Sender changes are audited and do not rewrite historical message records.

```text
draft → submitted → under_review → approved
                              ↘ rejected
```

## SMS allowance and top-ups

Each shop owner receives **100 free SMS credits once**. In the data model this is an
organisation-level wallet shared by that shop's stores and authorised staff. It is not a
separate wallet for every cashier and is never reset monthly.

The owner connects their own SendAfrica API key in Messaging Settings. The key belongs to that
shop's organisation and is never shown to staff, customers, or the browser after it is saved.
The owner may rotate or remove the key.

- One recipient message consumes one SMS credit unless SendAfrica reports a different cost.
- Bulk campaigns are charged per accepted recipient, not per HTTP request.
- Validation failures before provider submission do not consume credits.
- Provider-submitted messages are reserved before dispatch and reconciled with the provider response.
- Failed provider submission releases the reservation.
- Delivery failure after provider acceptance remains in the message log for reconciliation.
- Once free credits are exhausted, the shop must connect its own SendAfrica API key and top up
  its own SendAfrica balance before sending.
- Provider billing, sender ownership, and sender approval remain inside the shop's SendAfrica
  account.
- SendAfrica/provider balance is the source of truth for actual deliverability.

The UI must show `free_sms_remaining`, `paid_sms_balance`, `reserved_sms`,
`available_sms`, and `low_balance_threshold`. Sending fails explicitly with
`insufficient_sms_credits`; it never silently sends without accounting.

## Automated messages

Owners can enable or disable these per store:

- Credit/debt reminders
- Daily sales summaries
- Low-stock alerts
- Receipt confirmations
- New-customer welcome messages
- Order confirmations and status updates
- Promotional campaigns

Automated sends run asynchronously through the event bus or scheduler, with bounded retries
and a durable message log.

## Messaging interface

```text
GET    /messaging/settings
PATCH  /messaging/settings
GET    /messaging/balance
GET    /messaging/usage
GET    /messaging/topups
GET    /messaging/sender-id
GET    /messaging/sender-id/requirements
POST   /messaging/sender-id/request
GET    /messaging/sender-id/status
GET    /messaging/templates
POST   /messaging/templates
PATCH  /messaging/templates/:id
DELETE /messaging/templates/:id
POST   /messaging/send
POST   /messaging/campaigns
GET    /messaging/campaigns
GET    /messaging/campaigns/:id
GET    /messaging/messages/:id
POST   /webhooks/sendafrica
```

The provider adapter supports single and bulk SMS, preserves SendAfrica `request_id` and
`message_id`, and reconciles delivery through webhooks. SendAfrica documents bulk requests
for up to 1,000 recipients per call.

## Public storefront

Each store may publish:

```text
https://shop.ziadapos.com/{store_slug}
```

The page can expose branding, opening hours, categories, active products, product images,
prices, availability, promotions, pickup, delivery, and a cart without customer registration.

Public reads are unauthenticated. Store ownership, visibility, stock reservation, and order
mutations are enforced server-side.

## Orders and invoices

```text
Store link → catalogue → cart → customer details → delivery/pickup → order → invoice
```

Order states:

```text
pending | confirmed | preparing | ready | out_for_delivery | completed | cancelled | rejected
```

Orders store immutable product-price snapshots. Stock is reserved when the shop confirms an
order, not when a customer adds an item to a cart.

Initial payment methods are cash on delivery, pay on pickup, and manual payment confirmation.
Online payment adapters can be added later without changing the cart/order interface.

```text
GET    /public/stores/:slug
GET    /public/stores/:slug/categories
GET    /public/stores/:slug/products
POST   /public/stores/:slug/orders
GET    /public/orders/:token
POST   /public/orders/:token/cancel
GET    /commerce/settings
PATCH  /commerce/settings
GET    /commerce/orders
GET    /commerce/orders/:id
PATCH  /commerce/orders/:id/status
GET    /commerce/orders/:id/invoice
```

Public order tokens are random, non-sequential, and separate from internal UUIDs.

## Planned data additions

- `sender_id_requests`, `messaging_settings`
- `sms_wallets`, `sms_ledger_entries`, `sms_campaigns`, `sms_messages`
- encrypted organisation SendAfrica credential metadata (fingerprint, configured-at, rotated-at,
  and active state; never return the raw key)
- `storefront_settings`
- `orders`, `order_lines`, `invoices`

`sms_wallets` is organisation-scoped and contains the one-time free grant, paid balance,
reserved balance, consumed balance, and a version for safe concurrent reservations.

## Safety controls

- Owner-only access to sender IDs, wallets, campaigns, and storefront settings.
- Deduplicate recipients before charging credits.
- Enforce daily organisation send limits and rate limits.
- Support STOP/opt-out records for promotional messages.
- Keep each organisation's SendAfrica credential server-side and encrypted at rest.
- Never use a shared platform SendAfrica credential for a shop's operational or promotional SMS.
- Registration/password OTPs happen before a shop has a provider key; they require a separately
  documented platform-auth channel or temporary onboarding provider policy.
- Never block an order or POS sale on an SMS provider call.
