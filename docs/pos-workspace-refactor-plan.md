# Ziada POS Workspace Refactor Plan

Status: Proposed for review  
Scope: Refactor `src/app/dashboard/pos-workspace.tsx` into clean route-level feature modules  
Current file size: Approximately 3,800 lines  
Implementation status: Planning only

## 1. Objective

Replace the monolithic `pos-workspace.tsx` route multiplexer with independently owned feature pages and reusable modules while preserving current behavior.

The refactor must improve:

- Locality: changes to inventory stay in inventory code.
- Testability: each feature can be tested through a small interface.
- Translation: all user-facing copy uses English and simple Swahili translations.
- Performance: pages load only the data they need.
- Routing: each URL owns its own page composition.
- Accessibility: forms, tables, dialogs, labels, and errors remain usable.
- Responsive behavior: checkout and data tables work on phone, tablet, and desktop.

The refactor must not change business rules, API payloads, permissions, or transaction behavior unless explicitly approved.

## 2. Current-state findings

`pos-workspace.tsx` currently combines:

- Route detection through `usePathname()` and many `pathname.includes(...)` branches.
- Global loading and error state.
- Product, category, customer, transaction, credit, settings, and user state.
- Checkout cart state and payment submission.
- Inventory CRUD and stock adjustment.
- Customer creation and listing.
- Supplier records and operations.
- Sales table operations.
- Credit payment operations.
- Receipt emailing.
- Store switching and store creation.
- Expense ledger and creation.
- Help content search.
- Report generation and export history.
- Legacy analytics workspace.
- Staff management.
- Business settings.
- Theme controls.

The file contains approximately these internal modules:

```text
PosWorkspace
CheckoutWorkspace
TransactionRows
InventoryWorkspace
InventoryTable
CategoryManager
CreateProduct
CreateCustomer
CustomerWorkspace
SupplierWorkspace
SupplierOperations
SalesTable
CreditTable
ReceiptDialog
StoreSwitcher
DataTable
ExpenseWorkspace
StoreWorkspace
HelpWorkspace
ReportsWorkspace
AnalyticsWorkspace
ResourcePage
StaffTable
ReportGenerator
BusinessSettings
```

The primary design problem is not that these modules are all individually too large. It is that their ownership, data loading, mutation behavior, route selection, translation, and shared state are coupled to one parent.

## 3. Target architecture

```text
src/app/dashboard/
├── checkout/page.tsx              → <CheckoutPage />
├── inventory/page.tsx             → <InventoryPage />
├── customers/page.tsx             → <CustomersPage />
├── customers/[id]/page.tsx        → <CustomerDetailPage />
├── credits/page.tsx               → <CreditsPage />
├── credits/[id]/page.tsx          → <CreditDetailPage />
├── suppliers/page.tsx             → <SuppliersPage />
├── suppliers/[id]/page.tsx        → <SupplierDetailPage />
├── sales/page.tsx                 → <SalesPage />
├── sales/[id]/page.tsx            → <SaleDetailPage />
├── expenses/page.tsx              → <ExpensesPage />
├── stores/page.tsx                → <StoresPage />
├── staff/page.tsx                 → <StaffPage />
├── help/page.tsx                  → <HelpPage />
├── settings/page.tsx              → <SettingsPage />
└── business-settings/page.tsx     → <BusinessSettingsPage />

src/features/
├── checkout/
├── inventory/
├── customers/
├── credits/
├── suppliers/
├── sales/
├── expenses/
├── stores/
├── staff/
├── help/
├── settings/
├── analytics/
├── reports/
└── shared-pos/
```

Each feature should own:

```text
api/
  types.ts
  service.ts
  queries.ts
  mutations.ts
components/
  <feature>-page.tsx
  <feature>-table.tsx
  <feature>-form.tsx
  <feature>-skeleton.tsx
schemas/
constants/
```

The existing service-layer convention remains the external seam. Components should not import mock data or call raw API routes directly when an existing feature service can own the request.

## 4. Route ownership map

| Route | New owner | Current responsibility |
|---|---|---|
| `/dashboard/checkout` | `features/checkout` | Product catalog, cart, customer, payment, discount, checkout |
| `/dashboard/inventory` | `features/inventory` | Product list, stock, categories, create/update/deactivate |
| `/dashboard/product/[productId]` | `features/inventory` | Product detail and edit |
| `/dashboard/customers` | `features/customers` | Directory, search, create |
| `/dashboard/customers/[id]` | `features/customers` | Profile, purchases, receipt, SMS |
| `/dashboard/credits` | `features/credits` | Outstanding tabs, filters, payments |
| `/dashboard/credits/[id]` | `features/credits` | Credit history, payment, SMS reminder |
| `/dashboard/suppliers` | `features/suppliers` | Directory, supplier operations |
| `/dashboard/suppliers/[id]` | `features/suppliers` | Supplier detail |
| `/dashboard/sales` | `features/sales` | Sales history, search, actions |
| `/dashboard/sales/[id]` | `features/sales` | Sale detail and receipt |
| `/dashboard/expenses` | `features/expenses` | Expense ledger and create form |
| `/dashboard/stores` | `features/stores` | Branch list, create, switch |
| `/dashboard/staff` | `features/staff` | Staff list and permissions |
| `/dashboard/help` | `features/help` | Help article search and content |
| `/dashboard/settings` | `features/settings` | Appearance, active store, account settings |
| `/dashboard/business-settings` | `features/settings` | Business profile and integration settings |
| `/dashboard/analytics` | `features/analytics` | New server-side analytics dashboard |
| `/dashboard/reports` | `features/reports` | New report builder and export history |
| `/dashboard/sms` | `features/sms` | SMS wallet, SendAfrica setup, logs |
| `/dashboard/overview` | `features/overview` | Main owner dashboard |

## 5. Shared module design

### 5.1 Shared POS data contracts

Create `src/features/shared-pos/api/types.ts` for data used by more than one feature:

- `PosProduct`
- `PosCategory`
- `PosCustomer`
- `PosTransaction`
- `PosCreditTab`
- `PosUser`
- `PosStore`
- `PosSupplier`
- `PosExpense`

Keep API response types separate from view-model types. For example:

```ts
interface ProductRecord {
  id: string;
  name: string;
  stock: number;
  selling_price: number;
}

interface ProductListItem {
  id: string;
  name: string;
  displayPrice: string;
  stockLabel: string;
  isLowStock: boolean;
}
```

### 5.2 Shared API client behavior

Keep `src/lib/pos-api.ts` as the authenticated browser adapter, but add small helpers for:

- JSON requests.
- CSV/blob downloads.
- Structured API errors.
- Permission errors.
- Empty response handling.
- Request cancellation where a search/filter changes quickly.

Do not expose SendAfrica credentials in the client after save. API responses must return only safe configuration status.

### 5.3 Shared visual primitives

Extract reusable components into `src/features/shared-pos/components/`:

- `FeaturePageHeader`
- `FeatureErrorAlert`
- `FeatureLoadingState`
- `FeatureEmptyState`
- `MetricCard`
- `ResponsiveDataTable`
- `MobileRecordCard`
- `ConfirmActionDialog`
- `FormFieldError`
- `CurrencyValue`
- `StatusBadge`
- `SearchAndFilterBar`
- `ResponsiveSheet`

These components should not know about products, customers, or suppliers. They receive labels, values, actions, and state through props.

### 5.4 Shared translation interface

The existing `useLanguage()` and `useText()` become the required interface for user-facing text.

Rules:

- Every feature owns its feature-specific translation keys.
- Common labels go in `src/lib/i18n.tsx` or a shared translation file.
- API data is never translated blindly.
- Customer names, product names, SKUs, receipt numbers, provider statuses, and database values remain data.
- Error codes are mapped to friendly bilingual messages at the UI boundary.
- English remains available as a complete fallback.
- Swahili copy stays short and simple.

Suggested structure:

```text
src/lib/i18n/
├── provider.tsx
├── common.ts
├── checkout.ts
├── inventory.ts
├── customers.ts
├── credits.ts
├── suppliers.ts
├── sales.ts
├── expenses.ts
├── stores.ts
├── staff.ts
├── settings.ts
├── help.ts
├── analytics.ts
├── reports.ts
└── sms.ts
```

The current single dictionary can be split after the first migration batch. Avoid changing translation keys after components are migrated unless a compatibility alias is retained.

## 6. Feature-by-feature extraction plan

### Phase A — Stabilize the shell

Create:

```text
src/features/shared-pos/components/pos-page-shell.tsx
src/features/shared-pos/components/pos-error-alert.tsx
src/features/shared-pos/components/pos-loading.tsx
src/features/shared-pos/hooks/use-pos-user.ts
src/features/shared-pos/hooks/use-active-store.ts
```

Move out of `PosWorkspace`:

- Authentication/session check.
- Local user loading.
- Shared page error display.
- Shared page spacing.
- Theme and language controls.

Acceptance:

- Dashboard still redirects unauthenticated users.
- Current sidebar/header layout is unchanged.
- No feature-specific API calls occur in the shell.

### Phase B — Checkout

Create:

```text
src/features/checkout/components/checkout-page.tsx
src/features/checkout/components/product-catalog.tsx
src/features/checkout/components/cart-panel.tsx
src/features/checkout/components/payment-form.tsx
src/features/checkout/hooks/use-checkout-cart.ts
src/features/checkout/api/types.ts
src/features/checkout/api/service.ts
src/features/checkout/api/queries.ts
src/features/checkout/api/mutations.ts
```

Move:

- Product search/filter.
- Barcode scanning.
- Cart quantities.
- Customer selection.
- Payment method.
- Discount and reference.
- Tax and totals.
- Mobile cart sheet.
- Receipt dialog trigger.

Keep the checkout interface small:

```ts
interface CheckoutPageProps {
  initialProducts?: ProductRecord[];
  initialCustomers?: CustomerRecord[];
}
```

The cart hook owns only cart state and calculations. The mutation owns the API submission and invalidation.

Acceptance:

- Add, remove, scan, discount, credit, cash, and complete-sale flows behave exactly as before.
- The cart remains usable on narrow screens.
- No checkout code remains in the root workspace.

### Phase C — Inventory and products

Create:

```text
src/features/inventory/components/inventory-page.tsx
src/features/inventory/components/inventory-table.tsx
src/features/inventory/components/product-form.tsx
src/features/inventory/components/category-manager.tsx
src/features/inventory/components/stock-adjustment-form.tsx
src/features/inventory/api/*
src/features/inventory/schemas/*
```

Move:

- Product listing and search.
- Category manager.
- Create/edit product form.
- Stock adjustment.
- Deactivation.
- Low-stock status.

Use server-side pagination/search where supported. Keep product detail as a separate route that imports inventory services, not the page component.

### Phase D — Customers and credits

Create:

```text
src/features/customers/components/customer-page.tsx
src/features/customers/components/customer-table.tsx
src/features/customers/components/customer-form.tsx
src/features/customers/components/customer-sms-form.tsx
src/features/credits/components/credits-page.tsx
src/features/credits/components/credit-table.tsx
src/features/credits/components/payment-form.tsx
src/features/credits/components/reminder-action.tsx
```

Move:

- Customer directory and creation.
- Customer detail and purchase history.
- Direct customer SMS through `POST /customers/:id/sms`.
- Credit table and filters.
- Credit payment.
- Credit reminder through `POST /credits/customers/:customer_id/remind`.

The customer and credit modules may share customer types, but their mutations remain feature-owned.

### Phase E — Sales and receipts

Create:

```text
src/features/sales/components/sales-page.tsx
src/features/sales/components/sales-table.tsx
src/features/sales/components/sale-actions.tsx
src/features/sales/components/receipt-dialog.tsx
src/features/sales/api/*
```

Move:

- Sales history.
- Search and status filters.
- Refund/cancel actions.
- Receipt email.
- Receipt print.

Use a typed action state instead of string interpolation for mutation messages.

### Phase F — Suppliers and expenses

Create:

```text
src/features/suppliers/components/suppliers-page.tsx
src/features/suppliers/components/supplier-table.tsx
src/features/suppliers/components/supplier-operations.tsx
src/features/expenses/components/expenses-page.tsx
src/features/expenses/components/expense-table.tsx
src/features/expenses/components/expense-form.tsx
```

Move:

- Supplier directory.
- Delivery records.
- Supplier payments.
- Expense ledger.
- Expense filters.
- Expense creation.

Expense categories and payment methods become feature constants, not inline arrays inside a page component.

### Phase G — Stores, staff, and settings

Create:

```text
src/features/stores/components/stores-page.tsx
src/features/stores/components/store-switcher.tsx
src/features/staff/components/staff-page.tsx
src/features/staff/components/staff-table.tsx
src/features/settings/components/settings-page.tsx
src/features/settings/components/business-settings-form.tsx
src/features/settings/components/appearance-settings.tsx
```

Move:

- Store creation and switching.
- Staff permissions and deactivation.
- Business profile.
- SendAfrica configuration status.
- Theme and language settings.

Owner/admin access checks must remain visible and enforced server-side by the API.

### Phase H — Help and secondary pages

Create:

```text
src/features/help/components/help-page.tsx
src/features/help/components/help-search.tsx
src/features/help/components/help-article.tsx
```

Translate the help article catalogue instead of translating only the search UI. Keep external documentation links unchanged.

### Phase I — Analytics, reports, and SMS cleanup

The new Analytics, Reports, and SMS modules already exist outside the workspace. Remove the old legacy `AnalyticsWorkspace`, `ReportsWorkspace`, and report generator from `pos-workspace.tsx` after confirming no route still depends on them.

The SMS page should remain separate and own:

- 100 free SMS wallet.
- SendAfrica key setup.
- Sender ID status.
- Templates.
- Message logs.
- Campaigns.

Do not use `/accounts/sms-balance` as the organisation wallet. That endpoint represents provider balance and is a separate concept.

## 7. Root workspace end state

After extraction, `src/app/dashboard/pos-workspace.tsx` should either be deleted or reduced to a compatibility shell of fewer than 100 lines.

Preferred end state:

```tsx
export default function LegacyWorkspaceRedirect() {
  const pathname = usePathname();
  return <WorkspaceRouteResolver pathname={pathname} />;
}
```

Better end state: remove the compatibility file entirely and update every route page to import its feature page directly.

No feature should import `PosWorkspace` after the migration.

## 8. Data fetching and state rules

### Server state

Use TanStack Query for:

- Product lists.
- Customer lists.
- Transactions.
- Credits.
- Suppliers.
- Expenses.
- Stores.
- Staff.
- Analytics.
- Reports.
- SMS balance/settings/logs.

### Local state

Use local component state or feature hooks for:

- Open dialogs.
- Form drafts.
- Cart contents.
- Mobile sheets.
- Temporary search input.

### URL state

Use `nuqs` for:

- Search.
- Page.
- Page size.
- Sort.
- Status filters.
- Date range.
- Analytics tab.

### Mutation invalidation

Every mutation must invalidate the smallest useful key family:

```text
create product    → products.all, analytics.products
complete sale     → transactions.all, analytics.all, dashboard.all
record payment    → credits.all, customers.detail(id), analytics.customers
send SMS          → sms.logs, sms.balance
create expense    → expenses.all, analytics.cashflow
switch store      → store-scoped query families
```

## 9. Translation migration plan

For every extracted component:

1. Identify all visible English copy.
2. Add a semantic translation key.
3. Add simple English and Swahili values.
4. Replace labels, descriptions, placeholders, buttons, table headers, empty states, errors, and confirmation messages.
5. Keep server/API data untouched.
6. Test language switching without navigating away.

Examples:

```ts
const copy = {
  title: text('Inventory', 'Stoo'),
  add: text('Add product', 'Ongeza bidhaa'),
  empty: text('No products found.', 'Hakuna bidhaa zilizopatikana.')
};
```

Required translation coverage per feature:

- Page title.
- Page description.
- Navigation labels.
- Buttons.
- Form labels.
- Placeholders.
- Table headers.
- Status labels.
- Validation errors.
- API errors.
- Empty states.
- Loading states.
- Confirmation dialogs.
- Success messages.
- Accessibility labels.

## 10. Testing strategy

### Before extraction

Record current behavior for:

- Completing a sale.
- Recording a payment.
- Creating a product.
- Adjusting stock.
- Creating a customer.
- Sending customer SMS.
- Sending credit reminder.
- Creating expense.
- Switching store.
- Updating staff permissions.
- Generating a report.

### During extraction

Run after each phase:

```bash
bun run typecheck
bun run lint
git diff --check
```

### Browser verification

Verify at minimum:

- 360px mobile width.
- 768px tablet width.
- 1280px desktop width.
- Light theme.
- Dark theme.
- English.
- Swahili.
- Owner access.
- Staff access.
- Offline/API error state.
- Empty data state.

### Critical acceptance tests

- No route renders two copies of the same workspace.
- No route depends on pathname substring checks inside a feature component.
- Checkout state does not leak between pages.
- API requests are scoped to the active store.
- SMS keys never render after save.
- Translation switching updates inner components, not only navigation.
- Tables remain usable horizontally on mobile.
- Forms preserve validation and loading behavior.

## 11. Git and rollout strategy

Use one commit per extraction phase:

```text
refactor: add shared POS feature shell
refactor: extract checkout feature
refactor: extract inventory feature
refactor: extract customers and credits
refactor: extract sales and receipts
refactor: extract suppliers and expenses
refactor: extract stores staff and settings
refactor: remove legacy workspace routes
feat: translate extracted POS features
```

Each commit should compile independently where possible. Avoid one giant commit that makes rollback difficult.

Do not delete the monolith until the replacement route has passed behavior checks. During the transition, a feature flag or route-by-route replacement can provide a reversible seam.

## 12. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Checkout regression | Extract checkout first with behavior checks and no API contract changes |
| Duplicate data loads | One query family per feature and shared query keys |
| Stale store data | Include active store ID in every store-scoped query key |
| Translation gaps | Translation checklist required before each feature is marked complete |
| Permission leaks | Keep server-side API authorization; client filtering is UX only |
| Mobile layout regressions | Test 360px width after each table/form extraction |
| Dead legacy components | Delete only after route ownership map is verified |
| Provider credential exposure | Never return or render raw SendAfrica keys |
| Large diff review difficulty | Commit one feature at a time |

## 13. Definition of done

The refactor is complete when:

- `pos-workspace.tsx` is removed or reduced to a compatibility shell.
- Every dashboard route imports a feature-owned page.
- No feature uses `pathname.includes()` to decide which page to render.
- Shared API and UI primitives are reused instead of copied.
- All user-facing text has English and simple Swahili values.
- All mutations have loading, success, error, and invalidation behavior.
- All pages work at mobile, tablet, and desktop widths.
- TypeScript passes.
- Lint has no new errors attributable to the refactor.
- Critical POS flows are verified against the current API.
- The legacy workspace code is deleted only after route-by-route verification.

## 14. Recommended implementation order

Implement in this exact order:

1. Shared shell and translation helpers.
2. Checkout.
3. Inventory/products.
4. Customers.
5. Credits.
6. Sales and receipts.
7. Suppliers.
8. Expenses.
9. Stores and staff.
10. Settings and help.
11. Remove old analytics/reports implementations.
12. Delete the remaining workspace multiplexer.
13. Run complete responsive and bilingual verification.
