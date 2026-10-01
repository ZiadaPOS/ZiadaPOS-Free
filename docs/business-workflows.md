# POS business workflows

This guide describes the browser workflows currently represented in `web-app`. API behavior is authoritative; see the Go API repository's `docs/04-api-reference.md` and `docs/business-workflows.md` for request and persistence details. Routes listed here are in this UI repo and can be subject to role or business-type gating.

## Business types and pricing modes

Registration and the API recognise these organisation business types:

| Business type | Operator experience |
| --- | --- |
| `retail` | Checkout uses the retail selling price. |
| `wholesale` | Checkout is fixed to wholesale pricing. Each product needs a valid wholesale price and checkout quantity must meet its configured minimum. |
| `retail_wholesale` | Checkout can switch between retail and wholesale pricing for a sale. The API validates prices and minimum quantities. |
| `pharmacy` | Checkout uses retail pricing and medicine-specific profile, batch, expiry and prescription rules. Pharmacy is a distinct workflow, not simply a price mode. |

For mixed accounts, checkout applies one price mode to the sale, not independently per line. Price and tax totals are recalculated by the API. Current checkout calculation in the API applies a fixed 18% tax to the post-discount subtotal. Confirm local tax treatment with the business owner; the implementation is not a tax filing or compliance determination.

## First-time setup

1. Register the business and verify the phone number through OTP.
2. Sign in. The web session calls the API and stores access and refresh tokens in HttpOnly cookies.
3. Review the active organisation and store. Switch branch when necessary; the selected store is embedded in the API token and affects product, stock and transaction scope.
4. Create product categories and units, then create products with name, price, stock/cost where relevant, and identifiers (SKU/barcode) as needed.
5. Add customers, suppliers and staff where the business needs customer credit, supplier operations or multiple cashiers.
6. For non-pharmacy wholesale accounts, configure each product's wholesale price and minimum quantity before selling wholesale.
7. For pharmacy accounts, complete the medicine and batch setup described below before attempting a medicine sale.

## Retail sale

1. Open **POS** (`/checkout`) and confirm the active branch.
2. Find products by search or barcode, add them to the cart, and adjust quantities.
3. Optionally attach a customer and discount. A cashier's ability to discount or refund is governed by server-side permissions, not the presence of a control in the browser.
4. Select `Cash`, `M-Pesa`, `Tigo Pesa`, `Bank`, or `Credit`; record a payment reference where relevant. For credit, select/identify the customer as required by the UI and business policy.
5. Submit once. The API validates product state, available stock, price and discount, creates a per-store transaction number, persists item snapshots, updates stock and writes stock adjustments atomically. Do not retry blindly after a network timeout: first inspect the transaction ledger for whether the sale committed.
6. Verify the completed transaction in `/transactions` or `/sales` and review the receipt/details.

The API owns final price calculations, current stock checks, and transaction persistence. Cart state is not a committed sale. The current API supports sale, transaction listing/detail, refund, void, email receipt, and summary endpoints. Refunds and voids are distinct: a void restores stock; a pharmacy customer return is quarantined until inspected and released by an authorized owner.

## Wholesale sale

### Product setup

For each sellable product set a wholesale unit price and positive minimum quantity. API creation for wholesale and mixed accounts requires a wholesale price; imported and legacy products should be reviewed before first wholesale checkout. For a wholesale-only account, the server forces wholesale mode even if a caller asks for retail mode.

### Checkout

1. In `/checkout`, add product lines and choose wholesale pricing (wholesale-only organisations are already fixed to it).
2. Ensure each line's quantity reaches that product's `wholesale_minimum`. The API checks the total quantity on each submitted line, not merely the visual basket total.
3. Choose the customer and payment method; use credit only under the customer's configured credit policy.
4. Submit and verify the transaction. If price or minimum is missing, the API rejects the sale; correct product setup or quantities before resubmitting.

## Retail + wholesale sale

Mixed accounts use the same catalogue, stock ledger, customer and transaction history for both customer types. At checkout:

1. Choose **Retail price** for a walk-in consumer or **Wholesale price** for a qualifying bulk buyer.
2. Add products and satisfy each wholesale minimum if wholesale mode is selected.
3. Submit once. The API validates that the organisation is `retail_wholesale`, snapshots unit prices into the transaction lines, and decrements inventory. A later catalog price change does not rewrite a completed transaction.

Do not interpret this as per-item tier pricing: the checkout's mode is sale-wide. The backend contract currently supports a retail or wholesale `price_mode` on the sale.

## Pharmacy workflow

### 1. Catalogue and dispensing profile

Create a product in inventory and configure its medicine profile from the inventory/product workflow or `/pharmacy/schedules`. The profile includes generic/brand information, strength/form or pack details, shop-entered medicine schedule, registration data, and a `requires_prescription` flag. A product marked `prescription_only` also requires prescription data at checkout. These are shop-entered fields and labels; the software does not validate medicines against an authoritative regulator catalogue.

ADDO accounts cannot dispense a product marked `prescription_only`. The API rejects it even if it appears in the cart.

### 2. Receive stock by batch

Receive supplier stock with a batch number, positive quantity, expiry date, cost and supplier information. Use purchase orders when managing planned replenishment: create an order, then receive all or part of it with line quantities and lot data. Batch and product stock changes are committed together. Ordinary positive stock adjustment is intentionally rejected for pharmacy accounts; use batch receiving so stock remains traceable. Existing counted stock can be reconciled into a batch using the supported reconciliation option without double-counting.

### 3. Monitor and control lots

- `/pharmacy/expiry` lists batches within the chosen expiry window and allows inspection of movement history.
- `/pharmacy/prescriptions` shows the prescription register for authorized pharmacy management/pharmaceutical roles.
- `/pharmacy/schedules` reviews product schedule labels and prescription capture behavior.
- Batch disposal requires an owner/admin and an audit reason.
- Returned medicine is held in a separate quarantined lot. An owner/admin must inspect it, add an inspection note, and release it before it can become sellable. Expired lots cannot be released.

The backend allocates sale quantities by **FEFO** (first expiry, first out), only from available positive lots whose expiry date is today or later in the Africa/Dar_es_Salaam timezone. Insufficient unexpired batch stock blocks the sale.

### 4. Dispense and capture register data

For each medicine configured as prescription-required, checkout collects prescriber name, address and registration number, facility name and address, prescription reference, patient name and address, medicine strength, dosage form, and the cashier's confirmation that the prescriber/prescription was checked. The sale is rejected if required fields are missing or too long, or if authenticity confirmation is not checked. The API saves a register record tied to the transaction line and records which cashier entered it.

This confirmation is an auditable staff assertion only. It is not electronic prescription verification, regulator approval, or a determination that dispensing is legally appropriate. Users must follow applicable professional rules.

### 5. Refund/return handling

Pharmacy sale reversal restores ledger quantities through movements. A customer-returned medicine is not returned to normal available inventory: it enters quarantine for inspection. A void reversal of an erroneous sale returns the original allocation according to the API's reversal rules. Always inspect batch movement history after reversals.

## Stock, credit and supplier workflows

### Stock adjustments and low stock

For ordinary retail/wholesale products, use the stock adjustment operation with a reason (restock, damage, correction or return) and explanatory note. Product edits should not be used as a substitute for stock movements. Review `/inventory/low-stock` and adjust reorder points. For pharmacy, positive receipt goes through batch receiving and disposal/returns preserve the lot ledger.

### Customer credit

A sale paid with `Credit` is recorded with `credit` status and the API event workflow creates a credit tab. Record payments through the credit module and check outstanding balance before extending further credit. Refund/void events reconcile the associated tab. Writing off a tab is an owner-level financial action and should follow the business's approval process.

### Supplier procurement

Maintain supplier records and lead times. Pharmacy reorder suggestions use recent paid/credit sales, lead time, reorder point and sellable unexpired stock. Create one or more purchase orders, then receive only the quantities physically delivered and assign batch number/expiry for pharmacy lines. Supplier payment records affect the supplier ledger; they do not replace the stock receipt workflow.

## Daily close and reporting

1. Review transaction ledger and payment mix for the active store and date range.
2. Reconcile cash and external payment references against actual till/mobile/bank provider records; ZiadaPOS records the selected method/reference but does not itself prove external payment settlement.
3. Review credits, refunds, voids and stock adjustments.
4. For pharmacies, review expiry alerts, quarantined lots, batch movements and prescription records as part of the business's own control process.
5. Use dashboard, analytics and reports for summaries. Confirm date/store filters before interpreting branch-specific figures.

## Common rejection causes

| Message or condition | Likely action |
| --- | --- |
| Insufficient stock | Recount; receive or adjust stock through the correct workflow. |
| Wholesale price missing / minimum not met | Set the product's wholesale price/minimum or increase the submitted line quantity. |
| Pharmacy profile missing | Configure the medicine profile before checkout. |
| Prescription fields incomplete | Complete all required register fields and staff authenticity confirmation. |
| ADDO cannot dispense prescription-only | Stop checkout and follow applicable scope-of-practice rules. |
| Insufficient unexpired stock | Receive valid lots; expired/quarantined/depleted lots do not count. |
| Active store changed | Refresh checkout and confirm the branch before submitting. |

## Implementation boundaries

The web app exposes only the current UI pages and controls. It does not currently imply that every backend API has a corresponding UI. Public storefront/orders, SMS campaign services, or online payment callbacks must not be described as active unless a deployed handler and a UI flow are verified. The API repo documentation marks these as planned where applicable.
