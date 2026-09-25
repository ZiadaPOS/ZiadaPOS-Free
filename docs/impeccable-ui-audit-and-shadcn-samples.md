# Ziada POS UI Audit and shadcn Component Direction

Date: 2026-09-26

Mode: Operate

Scope: dashboard shell, navigation, search, page headers, KPI cards, tables, forms, dialogs, empty states, errors, responsive behavior, theming, translation, and business workflows.

## Executive summary

Ziada POS already has a useful foundation:

- shadcn/Base UI primitives are installed and used throughout the app.
- The dark theme is visually consistent and suited to long shop-owner sessions.
- The sidebar hierarchy is clear: sales operations, business management, analytics, reports, SMS, and account settings.
- The main analytics dashboard has a clear business direction: sales, profit, recent sales, top products, payment methods, credits, inventory, and SMS.
- The language switcher works in the shared shell and several major pages.

The main product issue is consistency. The shell looks like one product, but many inner pages still look like separate template demos. The screenshots show this clearly: navigation is Swahili while KPI labels, descriptions, table headings, buttons, and form copy remain English on the same page.

The recommended direction is a small, intentional shadcn-based business UI system:

1. One page-header pattern.
2. One KPI-card pattern with translated label, value, helper, trend, and action.
3. One table-toolbar pattern.
4. One dialog/sheet pattern for create and edit actions.
5. One empty/error/loading state pattern.
6. One translation boundary for every visible string.
7. One responsive rule: tables become cards or horizontal scroll, dialogs become full-height sheets on mobile, and action buttons remain reachable.

## Health score

| Dimension | Score | Finding |
|---|---:|---|
| Accessibility | 2/4 | Many forms have labels nested around inputs, but several icon controls and modal overlays need stronger names, focus handling, and keyboard behavior. |
| Performance | 3/4 | The app uses a modern Next.js structure and Recharts, but some pages fetch broad lists and render large client-side summaries. |
| Responsive design | 3/4 | The shell and checkout adapt well, but tables, dense action groups, and desktop dialogs need mobile-specific layouts. |
| Theming | 3/4 | Tokens and dark mode are established; a few older/template components still drift in copy, chart content, and component styling. |
| Implementation integrity | 2/4 | Feature extraction improved structure, but old template components and business components still use different UI conventions. |
| **Total** | **13/20** | **Acceptable; significant consistency work remains.** |

## What the screenshots reveal

### 1. Navigation is translated, content is not

The left rail correctly shows labels such as `Muhtasari`, `Miamala`, `Mauzo`, `Stoo`, and `Mikopo`, but the page body still shows English labels such as:

- `Transactions`
- `Recorded value`
- `Credit sales`
- `Discounts`
- `Refunds`
- `Sales revenue`
- `Average sale`
- `Products`
- `Active suppliers`
- `Team members`

This creates a trust problem: users cannot tell whether the language switch failed or whether some parts are not supported. Every visible string should be translated through `useText()` or a typed translation key.

### 2. KPI cards are visually consistent but semantically inconsistent

The KPI cards use a useful four-column layout, but the helper copy is often English, vague, or mismatched with the selected period. Each KPI should communicate:

```text
Label          Revenue
Value          TSh 129,306,490
Helper         Kipindi kilichochaguliwa
Comparison     +21.3% dhidi ya kipindi kilichopita
Action         Tazama mauzo
```

Avoid mixing English helper copy with Swahili labels. Avoid showing a percentage without explaining what it compares against.

### 3. Some pages have duplicate hierarchy

The expenses and stores screenshots show the `PageContainer` header followed by another inner `h1`. This produces duplicated titles such as:

```text
Expenses
Gharama
```

The page should have one primary heading. Feature components should own the content heading only when the route does not pass `pageTitle` to `PageContainer`. Otherwise, remove the inner heading and keep the page description in the page container.

### 4. English remains in secondary controls

Examples visible in the screenshots and source:

- `All payments`
- `All dates`
- `All categories`
- `View`
- `Refund`
- `Void`
- `Record`
- `Amount`
- `Open stores`
- `Team members`
- `Actions`
- `Business profile`
- `Active branch`
- `Select branch`

These are not API values. They are UI copy and should always be translated.

### 5. The app still contains demo/template content

Older overview graphs include visitor and browser sample concepts, while the real Ziada overview already has shop-specific data. Template charts should either be removed from production routes or clearly marked as examples in the elements/demo area.

## Recommended shadcn component system

### Page header

Use `PageContainer` for every dashboard route. Do not render a second page `h1` inside the feature component.

```tsx
const text = useText();

return (
  <PageContainer
    pageTitle={text('Inventory', 'Stoo')}
    pageDescription={text(
      'Products and current stock for this store.',
      'Bidhaa na stoo ya sasa ya tawi hili.'
    )}
    pageHeaderAction={
      <div className='flex flex-wrap gap-2'>
        <CategoryManager />
        <Button onClick={() => setCreateOpen(true)}>
          <Icons.add className='mr-2 size-4' />
          {text('Add product', 'Ongeza bidhaa')}
        </Button>
      </div>
    }
  >
    <InventoryContent />
  </PageContainer>
);
```

### KPI card

Use one shared `BusinessKpiCard` rather than rebuilding card markup on every page.

```tsx
<BusinessKpiCard
  label={text('Sales revenue', 'Mapato ya mauzo')}
  value={money(revenue)}
  helper={text('From matching sales', 'Kutoka mauzo yanayolingana')}
  trend={{ value: 12.4, label: text('vs previous period', 'dhidi ya kipindi kilichopita') }}
  icon={Icons.trendingUp}
  href='/dashboard/transactions'
  actionLabel={text('View sales', 'Tazama mauzo')}
/>
```

KPI rules:

- Use `tabular-nums` for values.
- Keep labels to one or two lines.
- Translate label, helper, trend label, and accessible action name.
- Do not use a trend color without a text explanation.
- On mobile, use two columns only when the label remains readable; otherwise use one column.

### Table toolbar

```tsx
<CardHeader className='space-y-3'>
  <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
    <div>
      <CardTitle>{text('Sales history', 'Historia ya mauzo')}</CardTitle>
      <CardDescription>
        {text('Search and manage completed sales.', 'Tafuta na simamia mauzo yaliyokamilika.')}
      </CardDescription>
    </div>
    <Button variant='outline' onClick={clearFilters}>
      <Icons.close className='mr-2 size-4' />
      {text('Clear filters', 'Futa vichujio')}
    </Button>
  </div>
  <div className='grid gap-2 lg:grid-cols-[minmax(0,1fr)_auto_auto_auto]'>
    <Input
      aria-label={text('Search sales', 'Tafuta mauzo')}
      placeholder={text('Search receipt or customer…', 'Tafuta risiti au mteja…')}
    />
    <Select aria-label={text('Filter status', 'Chuja hali')} />
    <Select aria-label={text('Filter payment', 'Chuja malipo')} />
    <Select aria-label={text('Filter date', 'Chuja tarehe')} />
  </div>
</CardHeader>
```

On small screens, stack the controls. Do not force four narrow controls into one row.

### Add/edit dialog

Use the shadcn `Dialog` for short forms and `Sheet` for longer forms. The form must own validation, loading, error, and success states.

```tsx
<Dialog open={createOpen} onOpenChange={setCreateOpen}>
  <DialogContent className='max-h-[90svh] overflow-y-auto sm:max-w-2xl'>
    <DialogHeader>
      <DialogTitle>{text('Add product', 'Ongeza bidhaa')}</DialogTitle>
      <DialogDescription>
        {text(
          'Add pricing, stock, category, and reorder settings.',
          'Ongeza bei, stoo, kundi, na kiwango cha kuagiza.'
        )}
      </DialogDescription>
    </DialogHeader>
    <form className='grid gap-4 sm:grid-cols-2' onSubmit={submit}>
      <Field label={text('Product name', 'Jina la bidhaa')} required>
        <Input name='name' autoComplete='off' />
      </Field>
      <Field label={text('Selling price', 'Bei ya kuuza')} required>
        <Input name='selling_price' inputMode='decimal' />
      </Field>
      <div className='flex justify-end gap-2 sm:col-span-2'>
        <Button type='button' variant='outline' onClick={() => setCreateOpen(false)}>
          {text('Cancel', 'Ghairi')}
        </Button>
        <Button type='submit' isLoading={pending}>
          {text('Save product', 'Hifadhi bidhaa')}
        </Button>
      </div>
    </form>
  </DialogContent>
</Dialog>
```

Dialog rules:

- Use `DialogTitle` and `DialogDescription` in every dialog.
- Close on success, not while the request is pending.
- Preserve entered values after validation errors.
- Use `aria-describedby` through the shadcn dialog structure.
- On mobile, use nearly full width and a scrollable content region.
- Do not use a modal when inline editing is faster and safer.

### Confirmation dialog

Destructive actions such as deactivate, refund, void, and delete should use `AlertDialog`, not a browser `confirm()` and not an unlabelled button.

```tsx
<AlertDialog>
  <AlertDialogTrigger asChild>
    <Button variant='destructive'>
      {text('Deactivate product', 'Zima bidhaa')}
    </Button>
  </AlertDialogTrigger>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>{text('Deactivate this product?', 'Uzime bidhaa hii?')}</AlertDialogTitle>
      <AlertDialogDescription>
        {text(
          'It will no longer appear in checkout, but its sales history will remain.',
          'Haitaonekana kwenye malipo, lakini historia ya mauzo itabaki.'
        )}
      </AlertDialogDescription>
    </AlertDialogHeader>
    <AlertDialogFooter>
      <AlertDialogCancel>{text('Cancel', 'Ghairi')}</AlertDialogCancel>
      <AlertDialogAction>{text('Deactivate', 'Zima')}</AlertDialogAction>
    </AlertDialogFooter>
  </AlertDialogContent>
</AlertDialog>
```

### Empty, loading, and error states

Every data section should have all four states:

```text
Loading:  skeleton cards/table rows
Empty:    what is missing + one useful action
Error:    what failed + retry action
Ready:    real data + next action
```

Example empty state:

```tsx
<Empty>
  <EmptyMedia variant='icon'><Icons.package /></EmptyMedia>
  <EmptyHeader>
    <EmptyTitle>{text('No products yet', 'Hakuna bidhaa bado')}</EmptyTitle>
    <EmptyDescription>
      {text('Add your first product to start selling.', 'Ongeza bidhaa yako ya kwanza kuanza kuuza.')}
    </EmptyDescription>
  </EmptyHeader>
  <EmptyContent>
    <Button onClick={() => setCreateOpen(true)}>
      {text('Add product', 'Ongeza bidhaa')}
    </Button>
  </EmptyContent>
</Empty>
```

## Page-by-page recommendations

### Overview

- Keep the four business KPIs at the top.
- Translate the remaining helper copy and chart legends.
- Add a clear period indicator beside the refresh action.
- Make every lower card link to its source page.
- Replace any remaining demo chart data with API data or move it to the demo-only area.

### Transactions

- Translate the four KPI labels and helpers.
- Translate `All activity`, `All payments`, `All dates`, `Clear`, `View`, and activity types.
- Keep API transaction values such as customer names, payment method values, and references unchanged.
- Use a responsive mobile card for each transaction instead of requiring horizontal table scrolling.

### Sales

- Translate page title, KPI helper text, filter controls, table headers, and row actions.
- Use `AlertDialog` for refund and void.
- Show a disabled/loading state while refund or void is being submitted.

### Inventory

- Translate KPI labels, helper copy, filter controls, table headings, product dialog fields, and stock adjustment dialog.
- Use a `Sheet` on mobile for product creation and adjustment.
- Keep SKU, barcode, product names, and category names as API data.

### Credits

- Translate KPI labels, status filters, payment action, amount placeholder, and empty states.
- Show due date and aging bucket in human language.
- Use a clear payment confirmation and success toast.

### Suppliers

- Translate KPI helpers, directory headings, delivery/payment form labels, and empty states.
- Separate `Add delivery` and `Pay supplier` into two tabs or two clear cards; the current page shows two repeated `Supplier` selectors without enough context.

### Expenses

- Remove the duplicate page heading (`Expenses` followed by `Gharama`).
- Translate KPI labels and helpers, table headings, category filter, form fields, and submit actions.
- Format dates for the current language instead of exposing raw ISO strings.

### Staff

- Translate `Team members`, field labels, permission chips, and `Deactivate`.
- Make permission buttons announce their current state: `Refund permission: disabled` / `Ruhusa ya kurejesha: imezimwa`.
- Confirm deactivation with `AlertDialog`.

### Stores and settings

- Remove duplicate `Stores` / `Matawi` heading.
- Translate store form fields, branch status, active branch selector, business profile labels, and save feedback.
- Use a two-column form on desktop and one column on mobile.

### Analytics and reports

- Translate every KPI label, comparison label, chart legend, table heading, report type, export format, and empty state.
- Translate the report preview summary, findings, risks, and recommended actions.
- Keep generated AI/report content in its requested language by passing the selected language to the API when supported.

### Ziada AI, chat, forms, and demo pages

- Translate shell copy and controls even if the generated conversation/content remains user data.
- Move obvious template demonstrations out of primary business navigation or mark them as demos.
- Keep technical examples in the elements/forms demo area, but still translate the page chrome and controls.

## Button and dialog standards

| Action type | shadcn component | Visual treatment | Translation requirement |
|---|---|---|---|
| Primary create/save | `Button` | Default/primary | Label, loading text, success/error |
| Secondary filter/refresh | `Button variant='outline'` | Outline | Tooltip/aria-label and visible label |
| Destructive action | `AlertDialog` + destructive action | Red only for confirmation | Title, consequence, cancel, confirm |
| Short create/edit | `Dialog` | Centered, max width | Title, description, fields, errors |
| Long form | `Sheet` | Right desktop, full-height mobile | Same as dialog |
| Row actions | `DropdownMenu` | Compact | Menu item labels and accessible trigger |
| More context | `Tooltip` | Icon-only controls | Tooltip and `aria-label` |

## Translation implementation rule

Do not translate with DOM mutation or browser overlays. Use the existing provider at the component boundary:

```tsx
const text = useText();

<Button aria-label={text('Refresh data', 'Onyesha upya taarifa')}>
  <Icons.refresh className='size-4' />
  <span>{text('Refresh', 'Onyesha upya')}</span>
</Button>
```

Do not pass translated strings into API payloads unless the API explicitly expects display copy. Keep enum values and backend data stable.

## Proposed implementation order

1. Create shared `BusinessKpiCard`, `BusinessPageHeader`, `DataToolbar`, `ConfirmActionDialog`, `FormDialog`, and `DataState` components.
2. Finish translation and KPI consistency on overview, transactions, sales, inventory, credits, expenses, suppliers, staff, and stores.
3. Convert create/edit/deactivate/refund/void flows to shadcn `Dialog`, `Sheet`, and `AlertDialog`.
4. Convert mobile tables to responsive cards or intentional horizontal scroll regions.
5. Translate analytics, reports, SMS, Ziada AI, chat, notifications, and detail pages.
6. Move template/demo-only content out of business routes.
7. Run a bilingual QA matrix at 360px, 768px, and 1280px in both light and dark themes.

## Definition of done

- No visible English UI remains when `language === 'sw'`, except user/API data, proper names, code, and technical identifiers.
- No visible Swahili UI is lost when switching back to English.
- Every button has a clear action label and loading/disabled state where needed.
- Every dialog has a title, description, focus management, validation, and mobile layout.
- Every table has translated headings, filters, empty, loading, and error states.
- Every KPI has a translated label, helper, period/comparison explanation, and correct numeric value.
- Every destructive action uses a confirmation component.
- Build, typecheck, and responsive checks pass.

