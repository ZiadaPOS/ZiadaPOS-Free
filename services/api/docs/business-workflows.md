# Business workflows: retail, wholesale and pharmacy

This guide documents domain behavior implemented by the Go API. It complements [04 — API Reference](04-api-reference.md), which is the method/path/payload catalog, and [05 — Data Model](05-data-model.md), which explains persisted entities. All operational routes use the configured API base path (default `/api/v1`).

## Business types

Registration reference data is served by `/meta/business-types`. Supported slugs are `retail`, `wholesale`, `retail_wholesale`, and `pharmacy`. Pharmacy outlet type is `pharmacy` or `addo` and is selected only for pharmacy organisations.

| Type | Checkout rule | Additional rules |
| --- | --- | --- |
| `retail` | Retail selling price. | Standard product stock counter. |
| `wholesale` | Server forces wholesale price mode. | Product must have positive wholesale price; submitted line quantity must meet wholesale minimum. |
| `retail_wholesale` | Sale may specify `price_mode=retail` or `wholesale`. Default is retail. | Wholesale checks run for every line; one mode applies to the entire sale. |
| `pharmacy` | Retail price mode. | Product dispensing profile required; stock comes from available unexpired batches using FEFO; prescription and ADDO restrictions apply. |

Do not combine the `pharmacy` business type with wholesale mode. The service forces retail pricing for pharmacies. Any UI visibility condition must be considered advisory; service validation is definitive.

## Tenant and branch context

The JWT carries user, organisation and active store context. Services scope reads and writes to those claims. Clients must not choose another organisation by sending an ID. Owners can switch to an authorized store using `/accounts/switch-store`; the API returns a new token pair with the active store changed. A checkout may submit `expected_store_id` to catch a stale browser branch selection; a mismatch returns a conflict and requires the client to reload.

Product, stock, sale sequence, batch, supplier and most operational records are store-scoped. Organisation-level business type determines which rules run. The per-store sale number is protected by a composite unique index.

## Common catalog and stock setup

1. Create categories and units (`/inventory/categories`, `/inventory/units`). Default category/unit seeds depend on business type and are versioned; verify the current catalog rather than relying on hard-coded client options.
2. Create products (`POST /inventory/products`) with selling price, optional cost, opening stock, reorder point, category, SKU/barcode, active state and service flag.
3. For wholesale/mixed businesses, supply wholesale price and minimum. Current product create validates wholesale price; existing records created before this rule may need review.
4. For retail/wholesale, keep stock changes auditable with `/inventory/products/:id/adjust` and valid reason/note. Product edit does not serve as an adjustment API.
5. For pharmacy, do not use a positive adjustment to receive normal saleable stock. Receive stock through a batch workflow.

Product soft deletion/deactivation preserves historical transaction data. POS lines snapshot product name, SKU and applied unit price so later catalog edits do not rewrite past sales.

## Retail checkout (`POST /pos/sale`)

Request lines contain `product_id` and positive `qty`; duplicate product IDs must be combined into one line. Valid methods are `Cash`, `M-Pesa`, `Tigo Pesa`, `Bank`, and `Credit`. Optional fields include `price_mode`, expected store, payment reference, discount percent, customer identity/contact, till number and notes.

The service:

1. Confirms the expected store (if provided), payment method, discount range, business type and price mode.
2. Locks and reloads products within the active store; rejects missing, inactive or insufficient-stock products.
3. Calculates unit prices from current catalog records, subtotal, discount, tax and total. Current tax is a fixed 18% of the discounted subtotal. The client's displayed cart total is not trusted.
4. In one database transaction, decrements product stock, records stock adjustments, allocates transaction number, and persists header and line snapshots.
5. Marks ordinary payments `paid`, and `Credit` payments `credit`. Publishes `transaction.created` after commit so credit/customer/notification side effects can run.

Tax is an implementation constant, not a configurable per-product or jurisdictional tax engine. Confirm whether it fits the business's real accounting requirements before relying on it for compliance or returns.

Refund (`POST /pos/transactions/:id/refund`) and void (`.../void`) are separate transitions. Refund requires refund authorization and marks a paid transaction refunded; void restores the stock for a transaction that should be voided. Both produce downstream events. Do not model a timeout as failure until the ledger is checked.

## Wholesale checkout

For `wholesale`, mode is forced to `wholesale`, even if omitted or a client asks for `retail`. For `retail_wholesale`, mode defaults to retail and must explicitly be wholesale for wholesale prices. Per line:

- `wholesale_price` must exist and be greater than zero.
- `wholesale_minimum` defaults to one if unset/non-positive in old data; product validation expects a positive configured value.
- Submitted `qty` must be at least that product's minimum.
- Insufficient stock still blocks the sale.

Pricing mode is sale-wide (not selected independently by line). The created `TransactionLine.UnitPrice` is the immutable record of the chosen price. Tiered quantity bands and negotiated customer-specific price lists are not implemented by this sale contract.

## Retail + wholesale checkout

Mixed businesses maintain one product/stock catalog and choose retail or wholesale mode per sale. The service validates the organisation type, applies one mode to each line, snapshots the selected prices and changes stock through the same POS transaction as retail checkout. Later price changes do not alter old line prices. The request field is `price_mode`, not customer type; customer association is separately optional.

## Pharmacy setup and sale

### Dispensing profile

Each non-service pharmacy product must have a `MedicineProfile`; absent profiles block the sale. Manage profile with `/pharmacy/products/:id` (GET/PUT). Profile fields include generic/brand labels, medicine schedule, registration and prescription requirement. Supported shop-entered schedule values are `unscheduled`, `otc`, `addo_permitted`, and `prescription_only`. The platform does not claim that these manually entered values come from a regulator source.

The checkout check for a prescription is true if `requires_prescription` is true **or** `medicine_schedule == prescription_only`. For an `addo` outlet, a `prescription_only` schedule is rejected outright. The general `requires_prescription` flag can require register capture independently of schedule.

### Receiving and ledger

Receive a lot using `/pharmacy/products/:id/batches` or inventory batch receipt. A batch carries product, store, batch number, quantity, expiry date, unit cost, supplier and stock status. Valid receipt updates batch quantity, product stock summary and movement history transactionally. Existing untracked counted stock may be reconciled into batches with `reconcile_existing_stock`; this assigns a real lot without incrementing stock twice.

Supplier purchase orders can be created and received through `/suppliers/purchase-orders`. Partial receiving records received quantities; pharmacy receipt lines require batch number and expiry. Supplier deliveries can also be directly recorded with required pharmacy line lot data. Stock receipt and related batch/movement updates are atomic.

### Sale allocation (FEFO)

At sale commit the API locks relevant products and eligible lots, sorts lots by expiry ascending (then creation time), and allocates earliest-expiring first. Eligible lots have `stock_status=available`, positive quantity and expiry date today or later in `Africa/Dar_es_Salaam`. The API records each allocation and a negative stock movement linked to the transaction. It then reconciles product stock against batch ledger. If valid lots cannot satisfy the quantity, the sale fails with insufficient unexpired stock. No lot-level stock is reserved when a browser merely adds an item to a local cart.

### Prescription register

Each prescription-required sale line must submit all of these within documented length limits:

- patient name and address;
- medicine strength and dosage form;
- prescription reference;
- prescriber name, address and registration number;
- prescribing facility name and address;
- `prescriber_authenticity_confirmed: true`.

The sale transaction persists an RX register record attached to the transaction line, numbered from the sale number, with the cashier recorded. The boolean records the staff member's assertion; it is not an e-prescription lookup, identity check, prescribing-license verification, repeat/refill validation or legal determination. Prescription register reads require pharmacy management or recognized pharmaceutical personnel and are scoped to the active store.

### Returns, quarantine and disposal

- Normal stock movements are append-only for the medicine ledger.
- A customer refund/return creates a distinct quarantined lot associated with the source lot. It does not restore the quantity to saleable stock.
- Owner/admin disposal requires a positive quantity and non-empty reason and records the removal movement plus stock adjustment.
- Owner/admin release of a quarantined batch requires an inspection note; expired stock cannot be released.
- Batch movements are store-scoped and paginated.

Pharmacy reports depend on product stock summary reconciliation. Never edit batch quantity outside the receiving, sale, adjustment, disposal, refund or release domain workflows.

## Customers and credit

Customers can be created before sale or selected at checkout. Customer profile statistics are maintained by transaction events. A sale with method `Credit` creates a credit tab through `transaction.created`; credit payments update the tab balance/status. Transaction refund/void events reconcile the tab. Customer credit limit is business policy metadata and any strict enforcement should be checked against the current service implementation; do not assume the frontend alone can enforce it. Credit write-off is an owner-authorized action.

## Supplier replenishment and payments

Supplier lead time is part of reorder recommendation. Current pharmacy reorder suggestions use the last 30 days of paid and credit transaction lines, configured supplier lead time (seven-day default if unassigned), reorder point/safety stock and sellable unexpired available batch quantity. Suggestions are operational guidance, not automatically placed orders. Owners/managers review quantities and unit costs, create supplier-grouped purchase orders, then record actual receipt. Supplier payments are separate ledger records and do not themselves receive stock.

## Daily close and business reporting

POS summary and hourly summary aggregate sale data by store and date. Transaction list filters by state, method, customer, search and date. Analytics and reports expose different aggregations; use their documented definitions. A successful `M-Pesa`, `Bank` or other method entry means the POS recorded the operator's selection/reference; the API does not itself verify external settlement. Reconcile external processor/till statements independently.

Recommended close review:

1. Confirm active store/date range and review transactions, payment methods, refunds and voids.
2. Reconcile physical cash and payment-provider/bank statements.
3. Review open credit, customer payments, supplier receipts and payments, expenses, and stock adjustments.
4. For pharmacy, review soon-expiring/expired lots, quarantined stock and batch movements; review prescription register access through the appropriate pharmacy role.
5. Generate reports only after checking selected dates and branch scope.

## Authorization and failure behavior

All scope-sensitive behavior relies on verified JWT claims. Owner-only operations include selected organisation/staff/store, supplier payment, and pharmacy quarantine/disposal controls. Pharmacy prescription list access has an additional role/occupation policy. API middleware and services remain authoritative even when UI navigation hides operations.

Expected domain rejections include invalid price mode, unsupported wholesale mode, missing wholesale price, wholesale minimum unmet, inactive/missing product, insufficient stock, stale store, unconfigured pharmacy profile, unexpired batch shortage, ADDO schedule restriction, incomplete prescription record and pharmacy-only endpoint on another business type. Return the structured API error to the operator; do not silently downgrade pricing or bypass batch/prescription checks.

## Implemented vs planned boundaries

These workflows are implemented in the registered API apps. The server currently registers no messaging app, public storefront/order app or commerce app. `/messaging`, `/public`, `/commerce`, and Snippe webhook paths shown in `08-messaging-and-commerce.md` are future contract/design material, not callable endpoints. That document and data-model entries that mention future tables must be read with this caveat. Update the route registration and tests before advertising these as active features.
