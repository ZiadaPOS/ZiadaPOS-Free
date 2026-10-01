# 08 — Messaging and Public Commerce (Planned)

> **Status: design notes only.** The current API server does not register a messaging, public-storefront, order, commerce, or Snippe webhook app. No route listed here should be called or described as a shipped feature until implementation, route registration, persistence, authorization, webhook verification and deployment are complete. See [business workflows](business-workflows.md) for supported current behavior and [04 — API Reference](04-api-reference.md) for implemented routes.

This document retains the intended product and integration requirements so they are not confused with the active API contract.

## Intended shop-owned SMS model

- A shop would connect its own SendAfrica account; platform credentials must not be used for operational or promotional sends.
- Sender-ID requests should be submitted through the shop's account. Only provider-approved IDs should be available for sending.
- The proposed organisation-level SMS wallet starts with a one-time 100-credit grant shared across stores. Messages reserve credits before dispatch and reconcile against provider outcomes; failed submissions release reservations.
- The planned controls include templates, campaigns, automation preferences, low-balance views, opt-outs, quiet hours, deduplication and rate limits.
- Provider credentials must be encrypted at rest, write-only in the UI/API, rotatable, and never returned in cleartext.

These are requirements, not present application guarantees. Existing Go SMS integration code for other flows does not imply these campaign or wallet routes exist.

## Intended public storefront and orders

Proposed storefront flow:

```text
published shop link → public catalogue → cart → customer details → pickup/delivery → order
```

The intended design includes product/price snapshots, public order tokens separate from internal IDs, owner-side order state management, and invoice views. Stock should not be decremented merely because a visitor adds an item to a cart. The final reservation/receipt lifecycle must be defined transactionally when implemented.

## Proposed route families (not active)

The following are design placeholders only:

```text
/messaging/*
/webhooks/sendafrica
/public/stores/:slug/*
/public/orders/:token/*
/commerce/*
/subscriptions/webhooks/snippe
/commerce/webhooks/snippe
```

Do not use these paths in production clients. They currently have no registered handlers. In particular, payment redirects must never be considered proof that a subscription or order was paid.

## Payment callback requirements before release

When a payment module is built, callback handlers must verify the provider signature over the raw request body, deduplicate provider event IDs/references, persist the provider event and ZiadaPOS payment record, and only then apply an idempotent subscription/order transition. Secrets must be configured and rotated outside source control. Add negative signature, replay, duplicate, malformed body, amount mismatch and out-of-order event coverage before exposing callback URLs.

## Planned data model

Potential entities include sender-ID requests, messaging settings, SMS wallet and ledger, campaigns and messages, storefront settings, orders/order lines and invoices. None of these entities are currently migrated by the API's registered apps. Do not assume they exist in a running database. Once implemented, move their contracts into the active API reference and data model and keep this file as an implementation decision record or remove it.
