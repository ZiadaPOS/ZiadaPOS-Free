# Ziada POS Analytics, Dashboard, and Reports UI Design

Status: Proposed design for review  
Scope: UI architecture and visualization plan based on `/home/cameltech/Projects/ZiadaPos/api`  
Implementation status: No frontend implementation changes made

## 1. Executive direction

The API already contains the aggregation layer needed for a high-quality shop-owner experience. The frontend should stop loading large entity lists and calculating business metrics in the browser. It should consume the analytics and reports interfaces as server-owned, date-aware data products.

The product should be organized around three distinct jobs:

1. **Dashboard** — “What needs attention today?”
2. **Analytics** — “Why did the business perform this way?”
3. **Reports** — “What can I share, save, schedule, or act on?”

The important architectural seam is a small frontend analytics data module. Pages and charts should depend on typed query options and view models, not on raw `fetch` calls or API-shaped maps scattered through a 3,000-line workspace.

## 2. API capabilities found

### Analytics routes

All analytics routes are under the configured API base path and require authentication plus owner access.

| Endpoint | Current payload/use | Recommended UI consumer |
|---|---|---|
| `GET /analytics/overview` | Selectable `sections=kpis,trend,mix,products`; date metadata | Fast overview shell or compact analytics prefetch |
| `GET /analytics/summary` | Revenue, profit, margin, transactions, customers, ticket, tax, refunds, discounts, expenses, revenue/transaction deltas | KPI cards and comparison strip |
| `GET /analytics/trend` | Zero-filled daily revenue, profit, transaction series | Revenue/profit trend chart |
| `GET /analytics/payment-mix` | Amount and percentage by Cash, M-Pesa, Tigo Pesa, Bank, Credit | Donut plus payment detail list |
| `GET /analytics/top-products` | Units, revenue, cost, profit | Dashboard best sellers |
| `GET /analytics/sales` | KPIs, trend, category breakdown, day-of-week averages, hourly pattern | Sales analytics page |
| `GET /analytics/products` | Totals and product rows with units, revenue, cost, profit, margin, prior trend, revenue share; supports `category` and `sort` | Product performance table and scatter/bar views |
| `GET /analytics/customers` | Daily visits, new vs returning, segments, four-month retention cohorts, top customers, customer KPIs | Customer analytics page |
| `GET /analytics/cashflow` | Daily inflow/COGS/opex/net, running balance, expense breakdown, payment inflow, outstanding credit | Cashflow and expense analytics |
| `GET /analytics/dashboard` | KPI payload, hourly breakdown, payment mix, best sellers, low stock, credit KPI | Main dashboard payload |
| `GET /analytics/detail/:section` | Stable drill-down envelope for sales, products, customers, cashflow/expenses, payments, credits | Card “View details” navigation/modal |

### Dashboard data available from the API

The dashboard service can provide:

- Revenue, profit, margin, transaction count, customer count, average ticket.
- Tax, refunds, discounts, expenses, and previous-period revenue/transaction deltas.
- Hourly revenue, profit, discounts, credit, expenses, and transaction count from 07:00–20:00.
- Payment mix.
- Top products by a secondary range.
- Low-stock products with `low` or `critical` status.
- Outstanding credit total, debtor count, and overdue count.

### Reports routes

| Endpoint | Current capability | UI treatment |
|---|---|---|
| `GET /reports/types` | Sales, inventory, tax, and credit report catalogue | Report type cards/select |
| `POST /reports/generate` | JSON preview or CSV download; supports preset/custom dates | Preview and export action |
| `GET /reports/exports` | Export audit history | Export history table |
| `GET /reports/exports/:id/download` | Regenerates stored export | Download button |
| `GET/POST/PATCH/DELETE /reports/scheduled` | Owner-managed scheduled reports | Scheduled reports settings |
| `POST /reports/scheduled/:id/send` | Immediate delivery | Send-now action |

## 3. Proposed information architecture

```text
Dashboard
├── Attention summary
├── KPI strip
├── Revenue and profit trend
├── Busiest hours
├── Best sellers
├── Payment mix
├── Low stock
├── Outstanding credit
└── Recent transactions / quick actions

Analytics
├── Sales
├── Products
├── Customers
├── Cashflow & expenses
└── Inventory & credit health

Reports
├── Build report
├── Preview
├── Export history
└── Scheduled delivery
```

`/dashboard/pos-workspace.tsx` should no longer own all three experiences. The proposed seam is:

- `src/features/dashboard/` — dashboard composition and compact widgets.
- `src/features/analytics/` — analytics page, section queries, chart view models, drill-downs.
- `src/features/reports/` — report builder, preview, exports, schedules.
- `src/lib/pos-api/` or feature-local API adapters — authenticated request and response normalization.

The existing detail pages for products, customers, credits, suppliers, sales, and inventory remain the destinations for row-level actions. Analytics cards should link into those pages with filters where possible.

## 4. Dashboard design

### Header and controls

- Store switcher and “Last updated” indicator.
- Date mode: **Today**, **Yesterday**, **Last 7 days**, **Last 30 days**, **Custom**.
- Refresh action with loading state.
- Optional compact “Compare with previous period” toggle.

The dashboard should be period-aware. If “Today” is selected, labels and data must reflect the same EAT calendar day throughout the payload.

### Above-the-fold layout

#### Attention banner

A compact decision card built from existing dashboard signals:

- Critical stock items.
- Overdue credit customers.
- Revenue/profit decline versus prior period.
- Expense spike.

Example wording: “2 products are below reorder level and overdue credit is TSh 420,000. Review collections and restock Sugar 1kg.” This can initially be deterministic; AI wording can be added later through the existing AI module.

#### KPI strip

Use six responsive cards, with the first four always visible:

1. Sales / revenue — amount and delta.
2. Profit — amount and margin.
3. Transactions — count and delta.
4. Average ticket — amount.
5. Outstanding credit — total and overdue count.
6. Expenses — amount and count.

Each card has a small sparkline or delta treatment and a drill-down link.

#### Main graph: revenue and profit

Use a composed Recharts chart:

- Revenue area/line.
- Profit line.
- Optional transaction count on a secondary axis.
- Tooltip with date, revenue, profit, transactions, and profit margin.
- Zero-filled days from the API must remain visible.

For seven days and less, show daily labels. For longer ranges, reduce tick density while preserving the full series.

#### Secondary graph: busiest hours

Use a grouped bar chart from `hourly_today`:

- Revenue bars.
- Profit bars or a profit line.
- Highlight the highest-revenue hour.
- Show transaction count in tooltip.

This graph answers when staff and stock should be ready.

#### Decision widgets

- **Best sellers:** ranked horizontal bars with product name, units, revenue, profit, and margin.
- **Payment mix:** donut with a numeric legend; never rely on color alone.
- **Low stock:** compact table with status, stock, reorder level, and “Open inventory”.
- **Credit:** amount, debtor count, overdue count, and “Open credits”.

## 5. Analytics page design

### Shared analytics controls

Every analytics section uses one shared control bar:

- Presets: 7d, 30d, 90d, YTD.
- Custom `date_from` and `date_to`.
- Comparison mode: previous equivalent period.
- Store scope, if the product later supports multi-store rollups.
- Export current section/report.

The selected range must be encoded in URL search params so the page is shareable, back-button friendly, and does not cause unnecessary RSC round trips.

### Sales section

Charts:

1. **Revenue vs profit trend** — composed line/area chart from `trend`.
2. **Category contribution** — sorted horizontal bars using revenue, share, and prior trend.
3. **Day-of-week performance** — seven-bar chart of average revenue.
4. **Hourly sales pattern** — 07:00–20:00 bar/area chart.
5. **Payment method mix** — donut plus amount and percentage table.

Decision copy should surface the strongest and weakest category, best hour, and payment concentration.

### Product section

Top row:

- Revenue, profit, margin, units sold, SKU count.

Visuals:

- **Top products by revenue** horizontal bar.
- **Profitability matrix** scatter plot: x = units sold, y = margin percentage, bubble size = revenue. This immediately separates volume winners, margin winners, stars, and underperformers.
- **Category performance** stacked or grouped bars when category data is available.

Table:

- Product, SKU, category, units, revenue, profit, margin, revenue share, prior trend.
- Server-side sort via `sort=revenue|units|margin|profit`.
- Category filter via `category`.
- Row click to product detail.

### Customer section

Charts:

1. **Daily visits** stacked bars: new customers vs returning customers.
2. **Customer segments** donut or treemap: count and spend must both be accessible.
3. **Retention cohorts** heatmap using `new`, `m1`, `m2`, `m3`; cells show retained count and percentage.
4. **Top customers** ranked list with visits, spend, average ticket, and last seen.

KPI cards:

- Total visits.
- New customers.
- Returning customers.
- Retention rate.
- Average visit value.

The segment visualization should not imply a true predictive model; labels and definitions must be displayed in an info tooltip if they come from simple spend/visit rules.

### Cashflow and expenses section

Charts:

1. **Cashflow waterfall/composed chart** — inflow, COGS, operating expenses, and net by day.
2. **Running balance** — line chart with positive/negative zones.
3. **Expense breakdown** — donut or sorted bar by category, including COGS as a separate category.
4. **Payment inflow** — reuse payment mix with cashflow wording.

KPI cards:

- Inflow.
- COGS.
- Operating expenses.
- Net cashflow.
- Net margin.

Do not call the running balance “bank balance”; it is an accumulated range net based on the API’s calculation.

### Credit and inventory health

Use the cashflow/detail payload and existing domain pages:

- Credit outstanding amount.
- Debtor count.
- Overdue count.
- Link to credits.
- Low-stock count and critical-stock count.
- Link to inventory.

The current analytics API exposes summary credit numbers, but not a full aged bucket series. The UI should show the summary now and reserve a proper aging chart for an API enhancement.

## 6. Reports design

### Report builder

The reports page should be a workflow, not a directory:

1. Choose report type: Sales Summary, Inventory Valuation, Tax Statement, Credit Aged Debtors.
2. Choose preset or custom date range where supported.
3. Click **Preview report**.
4. Inspect owner-friendly summary, key numbers, findings, risks, tables, and charts.
5. Export CSV or JSON, or schedule delivery.

The preview should use `POST /reports/generate` with `format=json`. It should not create a browser-only approximation of the report.

### Report preview structure

- Header: store, report type, period, generated time.
- Executive summary: deterministic text based on returned numbers.
- Key numbers: cards appropriate to report type.
- Findings: biggest category/product/payment/customer movement.
- Risks: refunds, expense growth, overdue credit, low stock, weak margin.
- Recommended actions: links to inventory, credits, products, or expenses.
- Visual appendix: charts corresponding to the report payload.
- Detailed table with print-friendly layout.

### Export history

Use `GET /reports/exports` for a table with:

- Report name/type.
- Period label.
- Format.
- Created date.
- File size.
- Download action.

Download should call `/reports/exports/:id/download` and handle both CSV attachment responses and JSON responses based on the stored format. Do not assume every export is JSON.

### Scheduled reports

Provide an owner-only settings panel with:

- Report type.
- Name.
- Frequency: daily, weekly, monthly.
- Date range preset.
- Recipient emails.
- Enabled/paused status.
- Last sent and next send times.
- Send now, edit, pause, delete.

## 7. Data contracts and frontend query plan

Create typed API contracts that normalize the current `map[string]interface{}` responses into stable frontend types. Suggested query families:

```text
analyticsKeys.dashboard(range)
analyticsKeys.summary(range)
analyticsKeys.trend(range)
analyticsKeys.sales(range)
analyticsKeys.products(range, category, sort)
analyticsKeys.customers(range)
analyticsKeys.cashflow(range)
reportsKeys.types()
reportsKeys.preview(type, range)
reportsKeys.exports(filters)
reportsKeys.scheduled()
```

Use TanStack Query options shared by server prefetch and client suspense queries. The range object should be the canonical query input and should serialize to `range=7d|30d|90d|ytd` or `date_from/date_to`.

Normalize money values in one place. The API currently returns integer amounts, which appear to be Tanzanian shillings. Formatting, compact notation, and tooltip precision should be handled by a shared currency formatter.

Use a chart view-model layer for:

- Stable colors by semantic series (`revenue`, `profit`, `expense`, `credit`, `cogs`).
- Empty states.
- Label density.
- Tooltip wording.
- Accessibility text summaries.

This keeps Recharts components presentational and prevents API quirks from leaking into every chart.

## 8. API contract issues to resolve before implementation

These are important enough to confirm with the API team:

1. **Dashboard range semantics:** `GET /analytics/dashboard` parses the request range, then passes the same range into `DashboardForRange`, but the payload keys are still named `kpis_today` and `hourly_today`. Rename or document them as period-based fields.
2. **Dashboard default:** the service has a `Dashboard()` helper for today plus seven days, while the HTTP handler uses `rangeOf`, whose default is 30 days. Decide whether dashboard default means today, seven days, or 30 days.
3. **Previous-period comparisons:** KPI deltas currently compare revenue and transaction count. The UI should not show a profit delta until the API provides one or the frontend clearly labels the comparison scope.
4. **Credit aging:** current analytics exposes total, customers, and overdue count, while the reports module can return aged debtor rows. Add a dedicated aging bucket payload if the analytics page needs a chart.
5. **Expense trend:** cashflow includes daily opex, but there is no explicit prior-period expense delta. Add it if the UI promises rising/falling expense comparison.
6. **Report formats:** reports currently support JSON and CSV only. PDF and Markdown require backend generation or a clearly stated browser print/Markdown export strategy.
7. **AI summary:** the reports module does not currently return an AI-written explanation. Add an explicit AI report-summary endpoint or keep the first version deterministic and label it accordingly.
8. **Product pagination:** `/analytics/products` currently returns the full product performance set. Confirm acceptable size or add server-side pagination/search for large stores.
9. **Auth scope:** analytics routes require owner access. Confirm whether managers with `can_view_reports` should receive read-only analytics access.
10. **Response envelope consistency:** analytics uses `response.OK` maps while reports exports return a model slice directly through `response.OK`. Define frontend adapters per endpoint rather than assuming one envelope shape.

## 9. Build phases

### Phase 1 — Data foundation

- Split analytics and reports routes out of `pos-workspace.tsx`.
- Add typed API contracts, query keys, query options, and range state.
- Implement shared money/date/empty/error/loading primitives.
- Wire dashboard to `/analytics/dashboard`.

### Phase 2 — High-value dashboard

- KPI strip with deltas.
- Revenue/profit chart.
- Hourly chart.
- Best sellers, payment mix, low stock, credit widgets.
- Drill-down links.

### Phase 3 — Analytics workspace

- Sales charts.
- Product performance table plus profitability matrix.
- Customer visits, segments, cohorts, and top customers.
- Cashflow/expenses charts.
- Detail route/modal behavior.

### Phase 4 — Reports product

- Report catalogue and builder.
- JSON preview.
- CSV/JSON export.
- Export history/download.
- Scheduled reports.

### Phase 5 — Product polish and API extensions

- Deterministic insights.
- AI summary integration.
- Credit aging chart.
- PDF/Markdown export if approved.
- Pagination/search and manager permissions if needed.

## 10. Acceptance criteria

- Changing the date range changes every relevant card and chart from server data.
- No analytics metric depends on loading the first 100 transaction/product/customer records.
- Revenue, profit, transactions, payment mix, products, customers, expenses, credits, and stock each have a clear visual or actionable destination.
- Every chart has an accessible text summary and an empty state.
- Every important card has a drill-down action.
- Report preview numbers match the corresponding analytics endpoint for the same dates.
- CSV and JSON downloads work from both fresh generation and export history.
- Loading, error, stale, no-data, and permission states are designed—not incidental.
- Dashboard, analytics, and reports have separate feature modules and no longer depend on the monolithic workspace for their primary business logic.

## 11. Recommended first review decision

Approve the structure and visualization inventory first. Before implementation, confirm the ten API contract issues above—especially dashboard default range, credit aging, AI summaries, report formats, and analytics permissions. Then implement Phase 1 and Phase 2 against the current API, with the missing capabilities represented as explicit empty/coming-soon states rather than simulated data.

## 12. Visual design samples

These wireframes show the proposed composition using the existing shadcn dashboard style. They are layout samples, not final pixel-perfect screens.

### Dashboard desktop

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Dashboard                                      [Today ▾] [Refresh]             │
│ Tuesday, 25 September 2026 · Store: Main Branch                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ ⚠ Attention                                                                  │
│ 2 products are critically low  ·  TSh 420,000 overdue credit  ·  Review now →│
├───────────────┬───────────────┬───────────────┬───────────────┬──────────────┤
│ Sales         │ Profit        │ Transactions  │ Avg. ticket   │ Credit       │
│ TSh 2.84M     │ TSh 742K      │ 86            │ TSh 33,023    │ TSh 420K     │
│ ↑ 12.4%       │ 26.1% margin  │ ↑ 8.1%        │ ↑ 4.0%        │ 6 overdue    │
├───────────────────────────────────────────────┬──────────────────────────────┤
│ Revenue and profit                            │ Busiest hours                 │
│                                               │                              │
│ TSh                                             ████                         │
│ 3M  ╭────╮       ╭──╮                         │      █████                   │
│     │    ╰──╮ ╭──╯  ╰──╮  Revenue             │  ██  ███████  ██             │
│ 1M  ╰────────╰──────────╯  Profit              │ 07  10  13  16  19  20       │
│     01  05  10  15  20  25                     │ Peak: 13:00–14:00            │
├───────────────────────────────────────────────┼──────────────────────────────┤
│ Best-selling products                         │ Payment mix                   │
│ 1  Rice 25kg             █████████ TSh 540K   │       ╭────╮                 │
│ 2  Sugar 1kg             ███████   TSh 390K   │   Cash  42%                  │
│ 3  Cooking Oil 5L        █████     TSh 280K   │   M-Pesa 31%                 │
│ 4  Soda 500ml             ████      TSh 210K   │   Credit 17%                 │
│ [View product analytics]                      │   Bank   10%                 │
├───────────────────────────────────────────────┼──────────────────────────────┤
│ Low stock                                      │ Credit health                 │
│ ● CRITICAL  Sugar 1kg       0 / reorder 20   │ Outstanding       TSh 420K    │
│ ● LOW       Rice 25kg       8 / reorder 15   │ Debtors           18           │
│ [Open inventory]                               │ Overdue           6            │
└───────────────────────────────────────────────┴──────────────────────────────┘
```

### Dashboard mobile

```text
┌──────────────────────────────┐
│ Dashboard       [Today ▾]    │
│ Main Branch                  │
├──────────────────────────────┤
│ ⚠ 2 critical stock alerts    │
│   TSh 420K overdue credit →  │
├──────────────────────────────┤
│ Sales                        │
│ TSh 2.84M       ↑ 12.4%      │
├──────────────────────────────┤
│ Profit                       │
│ TSh 742K        26.1% margin │
├──────────────────────────────┤
│ Transactions   86            │
│ Avg. ticket    TSh 33,023    │
├──────────────────────────────┤
│ Revenue and profit           │
│       ╭──╮   ╭──╮            │
│  ─────╯  ╰───╯  ╰────        │
│ [Swipe / pinch chart]        │
├──────────────────────────────┤
│ Best sellers                 │
│ Rice 25kg             540K   │
│ Sugar 1kg             390K   │
│ Cooking Oil 5L        280K   │
├──────────────────────────────┤
│ Low stock · 2 items       →  │
│ Credit · 6 overdue       →  │
└──────────────────────────────┘
```

### Analytics page shell

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Analytics                                                                     │
│ Understand what is driving sales, profit, customers, and cashflow.            │
│ [7 days] [30 days] [90 days] [YTD] [Custom range] [Compare previous period]  │
├──────────────────────────────────────────────────────────────────────────────┤
│ Sales       Products       Customers       Cashflow & Expenses       Health   │
├──────────────────────────────────────────────────────────────────────────────┤
│ Revenue       Profit        Margin         Transactions     Avg. ticket       │
│ TSh 18.4M     TSh 4.8M      26.1%          642              TSh 28,660        │
├──────────────────────────────────────────────────────────────────────────────┤
│ Revenue vs profit trend                                                       │
│                                                                              │
│ TSh 4M  ╭────╮      ╭──────╮                                                 │
│         │    ╰──╮╭──╯      ╰── Revenue                                       │
│ TSh 2M  ╰────────╯          ╰── Profit                                       │
│         01     08     15     22     29                                        │
├────────────────────────────────┬─────────────────────────────────────────────┤
│ Category revenue                │ Payment methods                             │
│ Food       ███████████  48%     │ Cash              42%  TSh 7.7M             │
│ Drinks     ███████      29%     │ M-Pesa            31%  TSh 5.7M             │
│ Household  ████         15%     │ Credit            17%  TSh 3.1M             │
│ Other      ██            8%     │ Bank              10%  TSh 1.8M             │
├────────────────────────────────┴─────────────────────────────────────────────┤
│ Sales by weekday                       Sales by hour                          │
│ Mon █████  Tue ███  Wed ███████         07  ██                              │
│ Thu ██████ Fri █████ Sat ████████        10  █████                           │
│ Sun ████                                  13  █████████  peak                  │
│                                           16  ██████                          │
│                                           19  ████                            │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Product analytics sample

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Products · 30 days                    [Category ▾] [Sort: Revenue ▾] [Export] │
├──────────────┬──────────────┬──────────────┬──────────────┬──────────────────┤
│ Revenue      │ Profit       │ Margin       │ Units sold   │ Products         │
│ TSh 18.4M    │ TSh 4.8M     │ 26.1%        │ 4,280        │ 142              │
├───────────────────────────────────────────────┬──────────────────────────────┤
│ Top products by revenue                        │ Profitability matrix          │
│ Rice 25kg        ███████████ TSh 2.4M         │ margin ↑                      │
│ Sugar 1kg        ███████     TSh 1.8M         │        ★ high-margin          │
│ Cooking Oil 5L   █████       TSh 1.2M         │             ●                │
│ Soda 500ml       ████        TSh 920K         │ volume ───────────────→       │
│                                               │  ● low-margin high-volume     │
├───────────────────────────────────────────────┴──────────────────────────────┤
│ Product       Category     Units     Revenue      Profit     Margin   Trend   │
│ Rice 25kg     Food         820       TSh 2.4M     TSh 520K    21.7%   ↑ 14%  │
│ Sugar 1kg     Food         1,120     TSh 1.8M     TSh 610K    33.9%   ↑ 8%   │
│ Oil 5L        Household    210       TSh 1.2M     TSh 180K    15.0%   ↓ 4%   │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Customer analytics sample

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Customers · 30 days                                                           │
├──────────────┬──────────────┬──────────────┬──────────────┬──────────────────┤
│ Visits       │ New          │ Returning    │ Retention    │ Avg. visit       │
│ 1,284        │ 320          │ 964          │ 75.1%        │ TSh 24,600       │
├────────────────────────────────┬─────────────────────────────────────────────┤
│ Daily visits                    │ Customer segments                           │
│ 80 ┤  ▓ returning               │ Regular       48%  TSh 8.2M                 │
│ 60 ┤ ▓▓▓                       │ New           29%  TSh 3.6M                 │
│ 40 ┤ ░ new                     │ VIP           15%  TSh 5.1M                 │
│    └──────────────────         │ At risk        8%  TSh 1.5M                 │
├────────────────────────────────┼─────────────────────────────────────────────┤
│ Retention cohorts               │ Top customers                               │
│ Cohort    New   M1   M2   M3   │ Asha M.       42 visits  TSh 840K             │
│ Jun       120   62   48   31   │ Juma K.       35 visits  TSh 710K             │
│ Jul       148   70   52   —    │ Neema S.      29 visits  TSh 640K             │
│ Aug       172   81   —    —    │ [View customers]                            │
└────────────────────────────────┴─────────────────────────────────────────────┘
```

### Cashflow and expenses sample

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Cashflow & Expenses · 30 days                                                 │
├──────────────┬──────────────┬──────────────┬──────────────┬──────────────────┤
│ Inflow       │ COGS         │ Operating    │ Net cashflow  │ Net margin       │
│ TSh 18.4M    │ TSh 10.2M    │ TSh 2.1M     │ TSh 6.1M      │ 33.2%            │
├───────────────────────────────────────────────┬──────────────────────────────┤
│ Cashflow movement                              │ Running balance               │
│ Inflow       █████████████████                 │ TSh 6M   ╭──────────╮         │
│ COGS         █████████                         │         ╭╯          ╰╮        │
│ Expenses     ███                               │ TSh 0 ──╯            ╰──      │
│ Net          ██████                            │         01  10  20  30        │
├───────────────────────────────────────────────┼──────────────────────────────┤
│ Expense breakdown                              │ Payment inflow                │
│ Cost of goods       TSh 10.2M   55%            │ Cash       TSh 7.7M   42%     │
│ Rent                TSh 780K     4%            │ M-Pesa     TSh 5.7M   31%     │
│ Transport           TSh 460K     2%            │ Credit     TSh 3.1M   17%     │
│ Salaries            TSh 860K     5%            │ Bank       TSh 1.8M   10%     │
└───────────────────────────────────────────────┴──────────────────────────────┘
```

### Reports workflow sample

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Reports                                                                       │
│ Build a clear report for your shop, then preview, export, or schedule it.      │
├──────────────────────┬───────────────────────────────────────────────────────┤
│ 1. Choose report     │ Sales Summary                                          │
│                      │ Daily sales totals, top products, payment breakdown.   │
│ ● Sales Summary      │                                                       │
│ ○ Inventory          │ [7 days] [30 days] [90 days] [Custom]                 │
│ ○ Tax Statement      │                                                       │
│ ○ Credit Debtors     │ [Preview report]                                      │
├──────────────────────┴───────────────────────────────────────────────────────┤
│ Preview: Sales Summary · 1 Sep – 30 Sep                                      │
├──────────────┬──────────────┬──────────────┬──────────────┬──────────────────┤
│ Revenue      │ Profit       │ Transactions │ Avg. ticket  │ Top product      │
│ TSh 18.4M    │ TSh 4.8M     │ 642          │ TSh 28,660   │ Rice 25kg        │
├──────────────────────────────────────────────────────────────────────────────┤
│ Business summary                                                            │
│ Sales increased 12.4% versus the previous period. Food products generated     │
│ the most revenue, while M-Pesa was the most-used payment method.              │
├──────────────────────────────────────────────────────────────────────────────┤
│ Findings                                                                     │
│ ✓ Revenue grew 12.4%                                                         │
│ ✓ Rice 25kg was the strongest product                                         │
│ ! Credit sales represent 17% of revenue                                       │
│ ! 2 products require restocking                                               │
├──────────────────────────────────────────────────────────────────────────────┤
│ [Download CSV] [Download JSON] [Print / PDF*] [Schedule report]               │
│ * Browser print/PDF until native PDF generation is added to the API.          │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Chart behavior standards

- Use semantic colors consistently: revenue = blue, profit = green, expense = orange, credit = purple, risk = red.
- Never communicate meaning through color alone; include labels, icons, patterns, or text.
- Tooltips must show formatted TSh values, percentages, dates, and the underlying count where useful.
- Charts need loading skeletons, no-data states, error states, and accessible summaries.
- On mobile, charts become horizontally scrollable or simplify to ranked lists.
- Every chart card includes a “View details” action when a deeper page exists.
- Use animation sparingly: animate initial entry, but do not animate every refresh or filter change.
