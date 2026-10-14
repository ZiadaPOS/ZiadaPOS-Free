# ZiadaPOS Platform and Product Plan — 2026-10

## 1. Decision summary and assumptions

**Decision:** build ZiadaPOS as a multi-tenant, branch-aware retail operations platform for Tanzanian hardware and building-material shops, with a deliberately separate public/docs application, transactional dashboard, NestJS system-of-record API, and Python MCP service. PostgreSQL is the canonical database. Supabase is **not** implied by the existing `supabase.css`; it remains an optional managed PostgreSQL/platform choice only after an explicit decision.

The product is a **unit-aware stock, trade-sales, procurement, receivables, cash-up and branch system**, not merely a barcode till. The target release is for hardware/building-material shops, reusing existing retail and wholesale flows where useful. Pharmacy is an existing project vertical, but it is **outside this target release** unless the user confirms a cross-vertical scope. “Cautions” means **quotations/pro-formas**. A pro-forma is not a fiscal invoice by default.

Assumptions for this plan:

- Mainland Tanzania is the initial compliance scope unless a later decision adds Zanzibar; Mainland and Zanzibar must not share an unverified compliance claim.
- The current product capabilities are the baseline: inventory, checkout, receipts/sales/refunds/voids, customers/credit, suppliers/POs/deliveries, expenses, staff/branches, analytics/reports and notifications.
- The current `/sms` screen is not an end-to-end feature; messaging/storefront backend routes are not registered and therefore are not release commitments.
- The clean `next-shadcn-dashboard-starter` inspection at commit `7705dfc0d13889e45c26a55ad5908da6a7a9a605` is the UI reference. It is MIT-licensed and upstream-aligned, but its product data is demo/mock data.
- NestJS is a deliberate Go-to-NestJS migration. The current Go/Gin service remains the migration source, not a second production authority.
- Integer TZS is the monetary representation for all persisted amounts. No tax rate is assumed; tax is configuration and must be confirmed per taxpayer and jurisdiction.

## 2. Existing-state assessment

`/workspace/ZiadaPOS` already has a distinct first-party `apps/web` Next application, a Wails `apps/desktop` slice, a Go/Gin/PostgreSQL/Redis/MinIO `services/api`, and a Python MCP agent with 111 tools. The web app currently has two intentionally modified inventory UI files; they must be preserved during template adoption and migration. A desktop SQLite slice exists but has no sync protocol, so it is not yet an offline product capability.

The product surface is broader than the current implementation contract: it includes stock, checkout, sales documents, returns, customers and credit, suppliers, purchase orders and deliveries, expenses, staffing, branches, analytics and notifications. However, the existing UI/API use a fixed 18% tax and record only payment method/reference. Those facts do **not** establish legal tax correctness, fiscalisation, payment settlement, or accounting compliance. The platform plan must remove the fixed-rate assumption and separate payment intent, provider confirmation, settlement and reconciliation.

The `/sms` page should be treated as a parked or experimental UI until a registered messaging route, provider contract, consent/opt-out model and delivery-status model exist. Likewise, storefront functionality should not be advertised while its routes remain unregistered.

## 3. Template fit, license and retain/replace plan

The upstream starter is a strong dashboard foundation: Next.js 16, React 19, Tailwind 4, shadcn/Base UI, TanStack Query/Table/Forms, and demonstrated Clerk authentication, organizations and billing patterns. Its MIT license permits use, modification and distribution provided the notice and disclaimer remain [1]. Preserve the upstream `LICENSE`/copyright notices and separately review terms for Clerk and all dependencies.

**Retain:** layout, navigation patterns, accessible Base UI primitives, theme architecture, table/form conventions, loading/error/empty states, TanStack Query cache and mutation patterns, and the visual Supabase theme tokens. Retain the current inventory UI modifications rather than overwriting them. Adapt the theme as visual language only.

**Replace:** demo users/products services, fake in-memory adapters, product-specific mock APIs, and client-only authorization assumptions. The upstream source explicitly labels analogous services as mock/demo data. The package has Clerk but no Supabase client, ORM, PostgreSQL driver, or Nest dependency; Clerk is identity/billing integration, not a domain database. Clerk Billing APIs are described as beta/experimental in the starter guide, so production billing requires a separate commercial/version decision [2].

Use an API-backed adapter in the dashboard and generate the TypeScript client from Nest OpenAPI. Keep TanStack Query SSR prefetch/HydrationBoundary, but make every cache key tenant/branch-aware and every replayable mutation idempotent. Nextra 4 is App Router-only and is best isolated in a docs/marketing app rather than embedded into transactional dashboard concerns [3].

## 4. Proposed repository and runtime topology

```text
apps/
  docs/                  # Nextra 4 + Next App Router: product, API and admin docs
  landing/               # Next App Router: public marketing/site pages
  app/                   # adapted Kiranism dashboard, authenticated product UI
services/
  api/                   # NestJS REST/OpenAPI authority and domain application
  mcp/                   # Python 3.11+ MCP server, uv-managed
packages/
  contracts/             # versioned API/domain enums and JSON schema where useful
  openapi-client/        # generated TypeScript client, published within workspace
  ui/                    # only genuinely shared primitives/tokens
  config/                # lint, TypeScript, formatting and environment conventions
infra/
  database/migrations/   # versioned SQL/migration ownership and deployment manifests
```

Use `pnpm-workspace.yaml` for JavaScript workspaces and `workspace:` dependencies; add Turborepo when parallel builds, tests and cache are worth the extra orchestration, not as a prerequisite [4]. Keep MCP as an independent `uv` project with a lockfile; a Python workspace is unnecessary until shared Python packages emerge [5]. Pin Node, pnpm, Next, Nextra, Nest, Clerk, Python, MCP SDK and uv versions in lockfiles and CI because these upstream lines move independently.

**Ownership boundaries:**

- `apps/landing` owns public acquisition pages; it must not import dashboard internals.
- `apps/docs` owns Nextra content and technical/admin guidance.
- `apps/app` owns navigation, rendering, local query cache, forms and user interaction. It does not own authorization, tax, stock, totals or persistence.
- `services/api` owns authentication verification, authorization, validation, transactions, idempotency, domain rules, fiscal/payment state, persistence and audit records. NestJS is the only public domain API.
- `services/mcp` exposes curated tools/resources and calls NestJS over authenticated HTTP. MCP must never bypass the API with direct database writes. The current 111 tools need migration classification, not a blind port.
- PostgreSQL is the canonical source of truth. Redis is infrastructure for queues/cache/locks only, never the authoritative ledger. MinIO stores controlled documents/exports with object-level tenant isolation.

Nest should publish `/api` and a stable `/api-json`; `@nestjs/swagger` supports document generation and JSON/YAML export [6]. The generated client is regenerated and validated in CI whenever the OpenAPI contract changes. Browser sessions should use secure HttpOnly cookies or a carefully configured BFF path; MCP/service calls use short-lived bearer credentials. If Clerk remains the identity authority, the API verifies Clerk tokens/JWKS and organization/role claims server-side. Client-side route guards are UX only.

## 5. Domain model and hardware-retail product flows

### 5.1 Tenancy, branches and registers

Model `organization` (merchant tenant), `branch`, `warehouse/bin`, `register/device`, `user`, `role`, and `shift`. Every business row carries an organization boundary; branch and register scope are explicit, not inferred from the browser. A user may have organization-wide, branch, warehouse or register permissions. Stock is held per branch/bin, with transfer states including requested, approved, dispatched, in-transit, received and exception.

### 5.2 Item master and units

The canonical item master includes SKU, description/specification/brand/size, category, stock UoM, allowed sale/purchase UoMs, product-specific conversion factors, decimal precision/rounding, GTINs and packaging-level barcodes, supplier/customer alternate codes, aliases, vendors, reorder levels and branch stock. GTIN identifies trade items that are priced, ordered or invoiced, but many local goods will have no GTIN; support internal SKUs, no-barcode items and duplicate detection [7].

Persist both **entered quantity/UoM** and normalized stock quantity. Support piece, box, bag, roll, metre, litre and kilogram. A box-of-12 may convert to pieces; a roll may be cut by metre; a bag may be sold by weight; an opened pack may have a controlled remainder. Variable conversions require a recorded measurement, exception reason, operator and audit trail. Do not silently turn a conversion into a generic global rule.

### 5.3 Pricing, contractor and wholesale tiers

Implement named price lists such as retail, contractor and wholesale, with product/category, customer/group, branch/currency, date validity, minimum quantity, fixed/discount/formula rules, and quantity breaks. Snapshot the applied price, rule and tax presentation on each document; changing a list must not recalculate history. Overrides require permission, reason and audit record. Odoo’s documented pricing model supports these dimensions and order-level overrides [8].

### 5.4 Quote → pro-forma/order → fulfilment → invoice/fiscal receipt

Use distinct immutable document types and states:

`draft quote → issued/expired/accepted → sales order → reserved/picked → one or more deliveries → invoice(s) → payment/credit balance`.

A quotation/pro-forma records number, dates/expiry, customer and delivery addresses, lines with entered UoM and quantity, price snapshot, discounts, tax presentation, terms, revisions and acceptance evidence. It is non-fiscal by default and must not consume fiscal numbering unless an approved local workflow says otherwise. Sales orders track ordered, reserved, fulfilled, backordered and cancelled quantities per line. Partial deliveries, deposits, partial invoices and explicit short-close are first-class states; an order is not complete merely because one delivery occurred. This matches documented quotation flows and partial-fulfilment patterns [9][10].

### 5.5 Customer balances, credit and deposits

Represent the customer balance as a ledger: invoices, deposits, allocations, payments, refunds, credits and write-offs. Expose ageing, available credit, credit limit, holds and unapplied cash. A deposit is not automatically revenue or a paid fiscal invoice. Orders can be blocked or require approval when credit is exceeded. A refund links to the source order/invoice and captures reason, condition and disposition; posted documents are corrected by reversal/credit-note/adjustment workflows, never edited in place [11].

### 5.6 Procurement and receiving

Use `purchase order → partial receipt → accepted/rejected quantities → supplier invoice match → supplier return/credit`. Receiving stores supplier reference, received UoM, normalized quantity, accepted/rejected warehouse, variance reason, lot/expiry where relevant, and approval. A PO remains open for pending quantities until completed or explicitly short-closed. This mirrors documented partial purchase receipt and stock-ledger behavior [12].

### 5.7 Stock ledger, counts and transfers

Use an append-only stock ledger for sale, return, receipt, rejection, adjustment, reservation, release and transfer movements. Every movement stores tenant, branch/bin, UoM, quantity, source document, user, device, time and reason. Counts are blind where practical, with recount and approval thresholds; adjustments record counted/on-hand/difference and create a stock movement, not a mutable number [13]. Reports include availability by branch/UoM, in-transit stock, shrinkage, dead stock, reorder, margin and audit exceptions.

### 5.8 Cash-up and payments

A shift captures opening float, tender totals, cash-in/out, refunds, deposits, expected versus counted cash, variance reason/approval and immutable close. Payments have a canonical intent/ledger with tender, provider, merchant/outlet/terminal, internal reference, provider transaction ID, amount/currency, status and settlement/reconciliation timestamps. Distinguish `created`, `initiated`, `pending`, `provider-confirmed`, `settled`, `failed`, `reversed`, `refunded` and `unknown`. A screenshot, SMS or initiation acknowledgement is not settlement.

Start with cash and safe manual digital references. Add one licensed provider adapter after written contract and technical verification. TANQR/Lipa Namba and bank/mobile-money/card routes are documented as available ecosystem patterns, but provider coverage, credentials, limits, fees, callbacks and settlement must be verified; do not invent QR payloads or claim universal coverage [14][15]. Never collect PIN, OTP, CVV or raw card data in ZiadaPOS.

### 5.9 Reporting and later accounting

MVP reports: stock/available-to-promise by branch and UoM, ledger/shrinkage, low-stock/reorder, dead stock, quote conversion, backorders, customer ageing/credit, deposits, supplier outstanding/partial receipts, margin by price list, and cash-up variance. Later accounting should consume posted, immutable events through an explicit accounting boundary: chart of accounts, journals, tax reports, landed-cost allocation, bank reconciliation and period close. Do not prematurely represent operational receipts as a full accounting system.

## 6. Offline and integrations

Offline is a product policy, not a framework feature. The later design is local SQLite/IndexedDB plus a durable outbox of idempotent commands, device/register sequence ranges, cached catalogue/prices/credit limits, small resumable sync batches, server cursors/version watermarks, conflict responses and visible unsynced status. Initial offline scope should be read-through cache plus explicitly safe commands; no unverified digital payment may be represented as settled. Credit-limit overrides, source-less returns, price edits and fiscal issuance are online/approved-device constrained unless written policy says otherwise. Shopify’s offline guidance supports this selective, not universal, model [16].

Use PostgreSQL transactions, unique constraints and optimistic concurrency. PostgreSQL’s default Read Committed behavior means complex concurrent updates require deliberate locking, stronger isolation or retry handling [17]. Every replayed mutation carries an idempotency key; an outbox event is written in the same transaction as its business change and dispatched asynchronously. Payment, messaging, fiscal and future accounting integrations use adapters with durable retries and reconciliation queues.

## 7. MVP, later modules and migration

**MVP:** tenant/branch/register setup; users/RBAC; item master and UoM conversions; barcode/SKU aliases; retail/contractor/wholesale price lists; quotation/pro-forma and sales order; partial delivery; receipts and non-fiscal invoice records; cash-up; customer ledger and controlled credit; PO/partial receiving/rejection; append-only stock ledger; core reports; audit log; API/OpenAPI; one safe payment-reference flow; deployment backups and diagnostics.

**Later after the hardware MVP foundation:** licensed payment adapters and settlement reconciliation; offline queue for selected commands; Wails desktop sync; routes for SMS/storefront; delivery routing/jobs; advanced accounting; landed cost; supplier/customer portals; automation and richer MCP tools. Pharmacy-specific expiry/dispensing remains out of scope unless separately approved.

**Production release blocker:** TRA/EFD/VFD discovery, approved integration and fiscal correction/outage tests must be completed before enabling production sales for any taxpayer who is required to use fiscal devices. Fiscalization is not a post-launch enhancement for an in-scope taxpayer.

**Migration recommendation:** do not run Go and Nest as dual writers. Freeze new Go domain writes behind a short migration window; export and validate current PostgreSQL data; map legacy tax/payment/document fields into versioned schemas; backfill immutable snapshots and audit provenance; run Nest in shadow/read-compare mode; switch writes behind a feature flag; retain Go read-only for rollback evidence; then decommission after reconciliation. Preserve user-facing inventory changes and migrate their data contract rather than replacing the files.

## 8. Tanzania localization and go-live gates

TRA material establishes terminology and obligations but does not grant ZiadaPOS compliance. TRA’s 2025 amendment describes connection of fiscal devices or billing systems to EFDMS and electronic transmission; accounting/billing software may issue fiscal receipts only in an approved integrated/signature/code arrangement [18]. The VAT Act contains statutory fiscal-receipt/tax-invoice and adjustment-note requirements [19]. Public material does not establish a universal API, schema, offline grace period, current retention period, or one malfunction SLA. Therefore, tax configuration must be versioned/effective-dated by jurisdiction, taxpayer status, supply type and approved device/provider.

**Go-live gate:** obtain written TRA/approved-supplier confirmation of product classification, onboarding, integration protocol, credentials, signature/code, receipt fields, correction/refund treatment, outage/manual procedure, retention and Mainland/Zanzibar applicability; then implement and pass the applicable approved issuance/transmission flow before enabling production sales for an in-scope taxpayer. Any pilot without that flow must be explicitly non-production or limited to taxpayers with a documented determination that the requirement does not apply. Have a Tanzanian tax adviser review claims. Never advertise “TRA compliant” before this gate.

PDPA applies to identifiable customer/staff data and financial transactions; it requires lawful/minimised processing, security, registration, DPO governance, rights handling, retention controls and assessment of cross-border transfers [20][21]. Treat each merchant as controller for its purposes and ZiadaPOS as processor under instructions, except for independently determined provider purposes. Create a DPA, tenant data map, retention schedule, privacy notice, subject-request workflow, DPIA for credit/profiling, incident runbook, subprocessors register and transfer assessment/permit review before foreign hosting/support/backup. Do not claim compliance based solely on hosting location or encryption.

## 9. Security, operations and acceptance criteria

Security baseline: tenant ID enforced server-side on every query/job/export; branch/register scope; least-privilege roles; MFA for administrators; field masking for phone/credit data; encrypted transport/storage/backups; managed key rotation; no secrets or payment credentials in logs; immutable audit events for auth, reads, exports, privilege changes, corrections, deletion and break-glass support; dependency and vulnerability scanning; tested restore; object-store isolation; and alerting for cross-tenant attempts and unusual exports.

Backups must be encrypted, automated, monitored and periodically restored into a test environment. Monitor API latency/error rate, queue lag, outbox retries, sync conflicts, payment unknowns, fiscal transmission status, database health, storage capacity and failed backup/restore tests. Provide operator diagnostics without exposing sensitive data.

Acceptance criteria for the first release:

1. A box-of-12, bag-by-kg, roll-by-metre and no-barcode item can be quoted, sold, received, returned and audited with entered and normalized quantities.
2. Contractor deposit plus two partial deliveries produces correct reserved/fulfilled/backordered quantities, customer balance and document links.
3. A partial/rejected PO receipt leaves pending quantities and creates accepted/rejected ledger entries.
4. Price-list changes do not alter historical snapshots; overrides require permission/reason.
5. Return/credit workflows preserve original documents and show adjustment/reversal history.
6. Cash-up reconciles opening float, tenders, refunds and counted cash with approved variance.
7. Duplicate API retries and offline replay do not duplicate stock, payment or ledger effects.
8. Every API request and background job is tenant/branch authorized; automated isolation tests pass.
9. OpenAPI generates a working client and MCP tools call the API only.
10. Backups, restore, audit export and migration reconciliation are demonstrated before production.
11. For every in-scope taxpayer, the approved fiscal flow passes registration, issuance, transmission, correction/refund and outage tests before production sales are enabled.

## Recommendations

The recommendations in this plan are to: keep NestJS as the sole domain authority; preserve the starter’s visual/accessibility shell while replacing demo services; model units, documents, ledgers and payments as immutable, auditable state; ship a small explicit offline scope rather than claiming universal offline operation; use provider-neutral fiscal/payment adapters; and make TRA, PDPC, payment-provider and data-migration verification release gates. These are product/engineering recommendations, not legal, tax or payment-provider certification.

## 10. Open decisions

- Is Supabase an explicit backend/platform choice, or only the preserved visual theme? If chosen, decide whether it supplies managed Postgres only or also Auth, Storage, Realtime and RLS; do not duplicate authorization inconsistently.
- Is the first commercial scope hardware/building-material only, or hardware plus retail, wholesale and pharmacy? Pharmacy adds materially different controls.
- Which TRA-approved supplier/integration and exact fiscal workflow are verified? What are outage and correction rules?
- Which operations must work offline, for how long, with what exposure and conflict policy?
- Which licensed payment provider, rails, settlement account, refunds and reconciliation contract are selected?
- Does Wails remain a supported desktop client, and is its SQLite slice migrated into the shared offline protocol or retired?
- Does Clerk remain the identity/billing authority, and what organization/role model is contractual?
- Which valuation method, landed-cost policy, retention periods, cloud region and PDPC transfer posture are approved?

## References

[1]: https://github.com/Kiranism/next-shadcn-dashboard-starter/blob/main/LICENSE "Kiranism starter MIT LICENSE"
[2]: https://github.com/Kiranism/next-shadcn-dashboard-starter/blob/main/docs/clerk_setup.md "Kiranism starter Clerk setup guide"
[3]: https://nextra.site/docs "Nextra official Introduction and FAQ"
[4]: https://pnpm.io/workspaces "pnpm official Workspace documentation"
[5]: https://docs.astral.sh/uv/concepts/projects/ "uv official Projects documentation"
[6]: https://docs.nestjs.com/openapi/introduction "NestJS official OpenAPI/Swagger documentation"
[7]: https://www.gs1.org/standards/id-keys/gtin "Global Trade Item Number (GTIN) — GS1"
[8]: https://www.odoo.com/documentation/19.0/applications/sales/sales/products_prices/prices/pricing.html "Pricelists — Odoo 19.0 documentation"
[9]: https://www.odoo.com/documentation/19.0/applications/sales/sales/sales_quotations.html "Sales quotations — Odoo 19.0 documentation"
[10]: https://docs.frappe.io/erpnext/delivery-note "Delivery Note — ERPNext documentation"
[11]: https://www.odoo.com/documentation/19.0/applications/finance/accounting/customer_invoices/credit_notes.html "Credit notes and refunds — Odoo 19.0 documentation"
[12]: https://docs.frappe.io/erpnext/purchase-receipt "Purchase Receipt — ERPNext documentation"
[13]: https://www.odoo.com/documentation/19.0/applications/inventory_and_mrp/inventory/warehouses_storage/inventory_management/count_products.html "Inventory adjustments — Odoo 19.0 documentation"
[14]: https://www.bot.go.tz/Publications/Acts,%20Regulations,%20Circulars,%20Guidelines/Guidelines/en/2023062016582893.pdf "Bank of Tanzania, Customer Experience Guideline for Merchant Payments, May 2023"
[15]: https://www.bot.go.tz/Publications/Acts,%20Regulations,%20Circulars,%20Guidelines/Guidelines/en/2022081908570183.pdf "Bank of Tanzania, Tanzania Quick Response Code (TANQR Code Standard 2022)"
[16]: https://help.shopify.com/en/manual/sell-in-person/shopify-pos/selling-offline/offline-features "Shopify POS offline features"
[17]: https://www.postgresql.org/docs/current/transaction-iso.html "PostgreSQL official Transaction Isolation documentation"
[18]: https://www.tra.go.tz/images/uploads/regulations/Subsidiary_legislation_%28The_Tax_Administration_%28General%29_%28Amendment%29_Regulations%2C_2025%29.pdf "The Tax Administration (General) (Amendment) Regulations, 2025, Government Notice No. 352J"
[19]: https://www.tra.go.tz/images/uploads/acts/The_Value_Added_Tax_Act.pdf "The Value Added Tax Act [Cap. 148 R.E. 2023]"
[20]: https://tanzlii.org/akn/tz/act/gn/2023/395b-1/eng@2023-06-13 "Personal Data Protection Act, 2022, Government Notice 395B-1 of 2023 (TanzLII; commenced 1 May 2023)"
[21]: https://pdpc.go.tz/publications/the-personal-data-protection-personal-data-collection-and-processing-regulations-2023/ "The Personal Data Collection and Processing Regulations, 2023 (Personal Data Protection Commission Tanzania; GN No. 449C)"
