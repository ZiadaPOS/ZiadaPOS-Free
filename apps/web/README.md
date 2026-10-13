# ZiadaPOS Web App

The ZiadaPOS browser workspace for Tanzanian retail shops, wholesalers, mixed retail/wholesale businesses, and pharmacies. The app is built with Next.js App Router, React, TypeScript, Tailwind CSS, and shadcn components. It is a first-party POS client, not the upstream dashboard starter template.

## Product workflows

- **Retail:** maintain products and stock, ring up counter sales, accept supported payment methods, inspect transactions, manage customers and credit, and review business performance.
- **Wholesale:** maintain a wholesale price and minimum quantity per product, use wholesale pricing at checkout, receive supplier deliveries, and track stock and sales.
- **Retail + wholesale:** choose retail or wholesale pricing in checkout. The API validates the chosen mode and wholesale minimums; it remains the pricing authority.
- **Pharmacy:** maintain dispensing profiles, receive traceable batches with expiry dates, monitor expiry, dispense by FEFO, capture prescription register details for products configured to require them, and inspect prescription records and batch movements. The UI records staff confirmation; it does not electronically authenticate a prescription or prescriber.

See [POS business workflows](docs/business-workflows.md) for the end-to-end operator guide and [API integration](docs/api-integration.md) for the browser-to-Go request path.

## Main pages

| Route | Purpose |
| --- | --- |
| `/auth/sign-in`, `/auth/sign-up`, `/auth/forgot-password` | Sign in, registration and password recovery. |
| `/overview` | Sales and business summary. |
| `/checkout` | Counter checkout and sale submission. |
| `/transactions`, `/sales`, `/sales/[id]` | Transaction ledger and sale details. |
| `/inventory`, `/inventory/[id]`, `/product` | Catalogue, stock and product management. |
| `/categories`, `/units` | Catalogue classification and units. |
| `/customers`, `/customers/[id]`, `/credits`, `/credits/[id]` | Customer records and credit tabs/payments. |
| `/suppliers`, `/suppliers/[id]` | Suppliers, deliveries, purchase orders and reorder suggestions. |
| `/pharmacy/expiry`, `/pharmacy/prescriptions`, `/pharmacy/schedules` | Pharmacy lots, register records and medicine profiles. |
| `/expenses`, `/staff`, `/stores` | Business expenses, staff and branches. |
| `/analytics`, `/reports`, `/notifications`, `/sms`, `/ai-chat`, `/notebook`, `/settings` | Supporting business operations. |

Navigation is configured in `src/config/nav-config.ts`; route availability and API authorization are separate concerns. The API enforces permissions and tenant scope.

**Known integration gap:** `/sms` is currently present in the UI and calls `/messaging/*`, but the Go API does not register those handlers. The screen's wallet, sender and key-management claims are therefore not a working end-to-end workflow. The backend currently exposes customer SMS at `/customers/:id/sms` through its configured SendAfrica integration. See the API repo's planned-only messaging note before making product claims.

## Architecture

```text
Browser page/component
    ↓ same-origin /api/pos/* (server-side Next.js route handler)
HttpOnly access token cookie + bearer header
    ↓ POS_API_URL (default http://localhost:8080/api/v1)
Go API (Gin) → PostgreSQL / Redis / object storage / integrations
```

The catch-all proxy lives at `src/app/api/pos/[...path]/route.ts`. It forwards supported HTTP methods, query strings, request bodies, and content type; it attaches the access token from an HttpOnly cookie; and it attempts refresh-token renewal and one retry after an upstream 401. `src/app/api/pos-session/route.ts` handles login, registration verification, session inspection, branch switching, and cookie clearing. Browser code should use the shared POS API helper and same-origin routes, not expose bearer tokens or call a private API secret from client code.

## Local development

Requirements: Bun (preferred) or Node/npm, and a running ZiadaPOS API. From this directory:

```bash
bun install
cp env.example.txt .env.local
# Set POS_API_URL to the API base, for example http://localhost:8080/api/v1
bun run dev
```

Open `http://localhost:3000`. The API base URL is server-side; do not prefix it with `NEXT_PUBLIC_`. Authentication requires a reachable API and valid business account. Review `env.example.txt` for all variables and `docs/deployment.md` for deployment details.

Useful commands (confirm availability in `package.json` before relying on a script):

```bash
bun run dev
bun run build
bun run lint
bun run typecheck
```

## Source map

- `src/app/` — routes, layouts and API route handlers.
- `src/features/checkout/` — POS checkout state and interaction.
- `src/features/inventory/` — products, categories, units, stock and wholesale pricing.
- `src/features/pharmacy/` — medicine profiles, batch/expiry work and purchase orders.
- `src/features/{customers,credits,suppliers,transactions,reports,analytics}/` — business modules.
- `src/components/layout/` — authenticated shell, navigation and providers.
- `src/lib/` — shared API, query and UI utilities.
- `docs/` — product workflows, integration, RBAC, forms, localization, themes and deployment.

## Documentation

- [POS business workflows](docs/business-workflows.md) — role-aware, step-by-step retail, wholesale, mixed-mode and pharmacy operations.
- [API integration](docs/api-integration.md) — cookie/session lifecycle, proxy behavior, endpoint families and error handling.
- [Deployment](docs/deployment.md)
- [Navigation and authorization](docs/nav-rbac.md)
- [Forms](docs/forms.md)
- [Localization](docs/i18n.md)
- [Themes](docs/themes.md)

The API owns business rules, prices, stock, transaction records, prescription enforcement, and authorization. A UI affordance or hidden navigation item is not a security boundary.
