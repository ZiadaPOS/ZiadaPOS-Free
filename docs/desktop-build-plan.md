# ZiadaPOS desktop + web build plan

**Goal:** use the current ZiadaPOS Go API and React/TypeScript web UI as one product, add a Wails desktop client, and allow the same packaged UI to open in an ordinary local browser. Keep all work in this monorepo. This plan does not publish, push, or create a GitHub release.

## Architecture decision: keep Next.js for now

The current UI is Next.js, not a static-only React bundle. Inspection found these concrete Next.js dependencies:

- `src/app/layout.tsx` reads cookies for theme and sidebar preferences.
- `src/proxy.ts` uses the Next request proxy to redirect legacy paths and gate routes based on POS session cookies.
- `src/app/api/pos/[...path]/route.ts` and `src/app/api/pos-session/route.ts` own the server-side POS API proxy, HttpOnly token cookies, refresh and session operations.
- Pages use App Router routing and Next imports such as `next/link`, `next/navigation`, `next/image` and `next/script`.
- There is no service worker or web-app manifest. Some legacy `/api/products` and `/api/users` route handlers still return mock data.

**Recommendation: do not migrate frameworks just to get offline.** Keep Next.js for the first desktop release and design an explicit data-provider boundary under `posApi`. Wails already packages the Node runtime, so the same UI can talk to a local Go service while disconnected. Offline requires durable local data, sync and conflict rules; it does not come from Vite, a service worker, WebAssembly or a local WebView by itself.

A Vite SPA may later reduce the desktop package by replacing the bundled Node/Next server with static assets. It would still need local storage and synchronization to work offline. It also requires converting App Router routes/redirects, replacing Next imports, and moving cookie-protected API/session logic to Go or to a web-only BFF. Revisit Vite only if eliminating the Node sidecar is a firm packaging goal and after the data-provider/auth boundaries are separated. Do not start a broad migration now.

## Presentation modes

Keep the existing React/TypeScript UI and provide two local presentation modes:

1. **Desktop:** Wails opens the platform WebView and forwards its asset requests to the packaged Next.js standalone server.
2. **Ordinary local browser:** the packaged desktop executable accepts `--browser`, starts the same Next.js server on a random `127.0.0.1` port, and opens the computer's default browser.

In connected mode, both use the current Go API through the Next.js `/api/pos/*` proxy and `/api/pos-session` routes. A local UI/browser process alone is not offline POS. The existing central API/PostgreSQL remains the shared system of record; offline desktop mode requires a new local Go data service and durable database.

## Repository facts that affect the work

| Area | What is present now | Consequence |
|---|---|---|
| `apps/web/` | Next.js 16, React 19, TypeScript, `bun.lock`; standalone output is already conditionally supported in `next.config.ts`. | Build the real app as a Node server; don't turn it into a static export. |
| `apps/web/src/lib/pos-api.ts` | Typed POS result types and a same-origin `fetch('/api/pos…')` helper. | Keep this as the principal client API seam; tighten endpoint-specific result types over time. |
| `apps/web/src/app/api/pos/[...path]/route.ts` | Server-side proxy to `POS_API_URL`, forwards HttpOnly access tokens, attempts refresh, and renews/deletes cookies. | Preserve this BFF boundary in connected web mode; add a deliberate desktop-local provider for offline use. |
| `apps/web/src/app/api/pos-session/route.ts` | Login, session renewal, logout and store switching routes to the Go API. | Validate login/session lifecycle before connecting more screens; define a separate, secure offline device grant. |
| `apps/web/src/features/` | Many POS areas already call `posApi`, including checkout, inventory, customers, sales, staff, suppliers, analytics, pharmacy and notifications. | Integrate/test real-flow pages rather than rebuilding the frontend. |
| `apps/web/src/features/products/api/service.ts`, `features/users/api/service.ts`, and `src/app/api/{products,users}/` | Still use fake/in-memory demo data. | Do not mistake the older generic product/user demo screens for connected POS inventory/staff screens. Replace with real endpoints or clearly mark/remove the demo path. |
| `services/api/docs/04-api-reference.md` | Active route contract. API envelope is `{success,message,data,errors,meta}`; tenant/store scope comes from signed token claims. | Treat this file plus handler tests as the source of truth, not UI mock data or proposed docs. |
| `services/api/internal/database/database.go` | Opens `gorm.io/driver/postgres`; domain code also uses PostgreSQL features (`jsonb`, `ILIKE`, `tags::text`, and `ON CONFLICT ... RETURNING`). | SQLite is not a driver-only switch. Build a limited local SQLite domain/sync service instead of trying to run the full API on SQLite. |
| `services/api/` | Go/Gin API plus PostgreSQL, Redis and MinIO; Compose starts all four. Default API listener is `:8080`. | Local connected development needs Docker Compose and test data. Compose host ports now bind to loopback by default; native API runs can use `API_LISTEN_ADDR=127.0.0.1:8080`. |
| API limits | `/messaging`, `/public`, `/commerce` and Snippe payment callbacks are not registered; the `/sms` screen calls an unimplemented messaging surface. `POST /customers/:id/sms` is a separate active customer-SMS endpoint. | Hide/disable unsupported controls or implement/test the backend before presenting them as working. |
| Tests | Go API has domain tests, including inventory/pharmacy/checkout; the web package has typecheck/lint but no first-party browser E2E test script. | Keep Go suites; add frontend contract and Playwright smoke tests. |

## Milestones

### 0. Confirm the connected local stack and preserve the current contract

- Review `.env.example` and create a local-only `services/api/.env`; never commit real credentials.
- Start API and dependencies with `docker compose up -d` from `services/api/`. Host ports default to loopback; use `HOST_BIND_ADDRESS=0.0.0.0` only for an intentional LAN test.
- Check `GET http://127.0.0.1:8080/healthz` and `GET http://127.0.0.1:8080/api/v1/health`.
- Use a disposable development account, not a production customer account; record local seed/OTP steps.
- Start `apps/web` with `POS_API_URL=http://127.0.0.1:8080/api/v1`; verify login, current user, active store and logout.
- Run existing Go tests and web typecheck/lint before and after integration work.
- Validate the Wails frontend and `--browser` launch on one developer machine; verify app exit stops the Next child process.

**Acceptance:** connected web UI can sign in against the local API, local services are not exposed to the LAN by default, and desktop/browser modes reuse the same UI bundle. This milestone does not enable offline sales.

### 1. Establish one real API contract per business screen

- Create a screen-to-endpoint matrix from active routes in `services/api/docs/04-api-reference.md`.
- Keep `posApi` as the browser-facing request helper. Add explicit per-endpoint request/response types for the first flows; reduce broad `unwrapList` guesses as contracts are verified.
- Normalize API envelope, pagination metadata, 204 responses, timeouts and network-unavailable errors.
- Keep connected browser requests same-origin (`/api/pos/...`); do not place access/refresh tokens in localStorage or `NEXT_PUBLIC_*` variables.
- Decide whether generic `/products` and `/users` demo pages are removed, marked as demos, or reworked to call `/inventory/products` and actual staff/account endpoints. Avoid a second API client that bypasses token handling.
- Audit every UI control against an active backend route. Hide or visibly disable controls that depend on unregistered routes, especially messaging campaign/wallet and future commerce/payment screens.
- Make `posApi` depend on a typed provider/transport so connected-web and desktop-local implementations can coexist without changing POS components.

**Acceptance:** the first build slice has a reviewed contract matrix; demo-only paths cannot silently look persistent; response/error states are explicit.

### 2. Deliver the first connected end-to-end slice: sign-in to sale receipt

**Build order:**

1. **Auth/session:** `POST /auth/login`, `GET /accounts/me`, refresh, logout, `POST /accounts/switch-store`; exercise `/api/pos-session` and HttpOnly cookies.
2. **Catalog:** list/search products and categories from `/inventory/products` and `/inventory/categories`; use server pagination and API-owned store context.
3. **Checkout:** send product IDs/quantities and supported payment metadata through `POST /pos/sale`; render the returned transaction/receipt. Server is authoritative for price, tax, stock, numbering and commit—never trust a browser total as the committed total.
4. **History:** read `/pos/transactions` and detail; verify receipt matches persisted transaction.
5. **Stock operations:** use `/inventory/products/:id/adjust` for stock corrections, not generic product `PATCH`.

**Tests:**

- API integration tests cover tenant/store scope, stock valid/insufficient paths, discounts/taxes and atomic sale + stock movement.
- Frontend tests cover login, refresh, expired session, store switch, empty/loading/error states, cart totals and receipt rendering.
- Add Playwright smoke flow against disposable local data: sign in → select store → find product → complete test sale → verify history.
- Add desktop smoke for `/` and same-origin session request; do not send real payments. The API has cash/mobile-money/bank/credit sale labels, but no live Snippe checkout is active.

**Acceptance:** test operator can complete one connected local sale from browser and Wails using the same frontend and central API.

### 3. Connect the remaining supported business modules

Work in dependency order: inventory units/categories/barcodes/adjustments; customers/history; credits/payments; suppliers/purchase orders/receiving; staff/permissions/stores/settings; expenses/analytics/reports/notifications; then pharmacy batches/expiry/dispensing. Preserve pharmacy FEFO and prescription-capture rules. Add AI only with a provider-available and no-provider/error state.

For each module, verify the route is registered, map fields and permissions, connect UI services/hooks, and add API/UI tests. A missing route is a backend task, not a mock success. Do not represent `/messaging`, `/public`, `/commerce`, or Snippe webhook designs as implemented.

**Acceptance:** each enabled screen/route pair has a contract test and documented role/store behavior; unsupported screens are not advertised as operational.

### 4. Harden desktop and connected browser behavior

- Verify Wails startup/shutdown, resource extraction cleanup, and Next child-process lifecycle. Test interrupted startup, occupied API and child-process exit.
- Test login cookies, refresh and logout in Windows WebView2, macOS WKWebView, Linux WebKitGTK, and Chrome/Firefox browser mode. Confirm cookies on Wails' app origin and `http://127.0.0.1:<ephemeral-port>`.
- Keep packaged UI bound to `127.0.0.1`; don't offer a LAN bind without authentication/CSRF controls.
- Add an API endpoint settings surface before relying on inherited `POS_API_URL` for non-developer installs. Validate schemes/hosts; never bake API secrets into frontend assets.
- Smoke test printing, barcode/keyboard-wedge scanners, keyboard shortcuts, file dialogs and window resizing on real target OSes. Native printer/cash-drawer work is separate if needed.

**Acceptance:** supported OSes complete the connected sale slice; local listeners are restricted to intended interfaces; app exit leaves no child process.

### 5. Add real offline desktop capability: local SQLite + sync

Offline is a product/data capability, not a framework toggle. The current central API explicitly opens PostgreSQL and uses PostgreSQL SQL/types. Do **not** change one driver and claim SQLite support. Do not package a local PostgreSQL/Redis/MinIO cluster as the offline design.

**Recommended V1 scope:** installed desktop app only, with cached store/catalog/customer data, cash sales, local receipt printing and an outbox. Defer offline pharmacy dispensing, external payments/mobile-money confirmation, SMS/email, AI and broad analytics until their consistency/external-service rules are designed.

1. **Set policy before code:** which stores/devices may go offline; maximum age for price/permission/stock data; one or multiple disconnected terminals per store; device loss/revocation; and which payment types count as completed. Safe V1 default: one enrolled desktop per store at a time, cash-only offline sales, no offline pharmacy dispensing. Multiple disconnected terminals can oversell shared stock unless stock is allocated per device ahead of time.
2. **Separate UI data operations from transport:** define a typed `PosRepository`/provider for the sales slice. Keep current Next BFF adapter for connected web; add a desktop adapter calling the local Go service. Keep tokens out of browser storage. Local service syncs upstream when connected.
3. **Build a limited local SQLite domain:** versioned migrations; WAL; backup/recovery; tenant/store scope; transactionally consistent catalog/price/stock/customer snapshots; minimum data only. Store DB under the OS user-data directory. Protect sensitive local data with an OS-protected encryption key/credential store; design key recovery and device revocation before release.
4. **Commit offline sales atomically:** one SQLite transaction writes sale, lines, local stock movement and outbox event. Generate globally unique idempotency UUID on device; issue a clearly marked local/provisional receipt identifier while offline. Do not claim an external payment settled simply because a sale is queued.
5. **Add a real central sync protocol:** enroll/trust device online; provide versioned bootstrap snapshot/change cursor; add an authorized sync route to accept outbox events; deduplicate by device + event UUID; validate tenant/store/role/business rules; return server IDs/transaction numbers and per-event acknowledgements; pull updates and tombstones. Current API has no such protocol, so this is new backend work.
6. **Resolve conflicts explicitly:** return stock/price/permission conflicts for operator review. For V1, use single-terminal policy or pre-allocate offline stock quotas per enrolled device. A stale stock snapshot does not prove another terminal has not sold the same item. Preserve original receipt and show reconciliation status.
7. **Show trustworthy UI state:** online/offline indicator; last sync; snapshot freshness; queued count; sync/retry state; pending vs confirmed receipt label; and recoverable conflict view. Clearly disable actions unavailable without network.
8. **Test failure and replay:** disconnect before/during checkout; power loss after local commit; repeated upload; timeout after server commit; stale catalog/permissions; DB migration/backup/restore; two-device stock conflict; token expiry; reconnect after days. Assert no duplicate or lost local sales.

**Browser-mode distinction:** packaged `--browser` can share desktop offline data only while its local host process is running and serves the local Go service on loopback. A directly hosted/browser-only app cannot use Wails storage; browser-only offline requires a PWA service worker + IndexedDB + its own sync/provider path. No service worker exists today. Treat full browser/PWA offline as later separate scope.

**Acceptance:** desktop completes the approved limited sale flow without internet, persists across app restarts, and syncs each sale exactly once when connected; conflicts are visible/recoverable. Until then, describe it as an online app with a local UI—not “offline POS.”

### 6. Build and distribute platform artifacts

- Keep `.github/workflows/desktop-build.yml` manual-only for build/test artifacts. The separate `.github/workflows/desktop-release.yml` publishes only when an approved `vMAJOR.MINOR.PATCH` tag is pushed; review the exact source and outputs before creating a tag.
- Add architectures only with matching native runners. Embedded Node and traced native dependencies must match the target architecture; Go cross-compile alone is insufficient.
- Use the square brand mark as `apps/desktop/build/appicon.png`: set it as the Linux Wails window icon, and let Wails generate the Windows executable icon and macOS app-bundle icon on their native runners.
- The initial download formats are unsigned Linux tar.gz, Windows ZIP and macOS ZIP archives—not native installers. They include no Linux `.desktop` launcher entry; document runtime prerequisites and checksums without implying installer, updater, or offline support.
- Before shipping signed native installers or enabling updates, complete connected-mode platform smoke tests and choose OS-specific install/update/uninstall behavior.
- Add signing credentials only through protected CI secrets when a release is explicitly approved. Keep credentials out of repository files/artifacts.
- The tag-triggered public-release workflow is prepared but not run; no tag or release has been created in this local review.

**Acceptance:** each archive has a version, OS/architecture label, checksum and accurate runtime notes; app icons are embedded on the supported native targets; release publication requires an explicitly reviewed and pushed tag.

## Current connected-mode local start sequence

These commands exercise the connected API + PostgreSQL/Redis/MinIO stack. They **do not** enable offline sales; that requires Milestone 5.

```sh
cd services/api
cp .env.example .env
# Review .env; keep HOST_BIND_ADDRESS unset for loopback-only host ports.
docker compose up -d
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8080/api/v1/health
```

In another terminal:

```sh
cd apps/web
bun install --frozen-lockfile
POS_API_URL=http://127.0.0.1:8080/api/v1 bun run dev --hostname 127.0.0.1 --port 3000
```

Open `http://127.0.0.1:3000`. For a Wails dev window:

```sh
cd apps/desktop
ZIADAPOS_DESKTOP_DEV_URL=http://127.0.0.1:3000 wails dev -s
```

The packaged local-browser mode uses the same UI after building: run the desktop executable with `--browser`. For macOS/Linux, export `POS_API_URL` before launching if the default `localhost:8080` is unsuitable. In PowerShell: `$env:POS_API_URL = 'http://127.0.0.1:8080/api/v1'`.

## Completion criteria for the first build session

1. `go test ./...` passes in `services/api` and `apps/desktop`.
2. `bun run typecheck` and `bun run lint` pass in `apps/web`.
3. Local health, login and profile checks pass.
4. One connected test sale works in browser and Wails against local test data.
5. Mock-only/unsupported screens are listed and gated.
6. Native Windows/macOS/Linux builds produce artifacts without publishing them.
7. Offline is not claimed until Milestone 5's persistence, idempotency, conflict, sync and recovery tests pass.
8. All changes remain local until reviewed; do not push or create a public release.

## Verification checkpoint (working copy, 2026-10-13)

| Check | Result |
|---|---|
| API and desktop Go tests | Pass: `go test ./...` in both modules. |
| Frontend validation | Type-check passes; targeted `oxlint` on edited files reports 0 warnings/errors. Full web lint remains at 78 errors and 26 warnings across the repository. |
| Linux package and browser mode | Next standalone and Wails v2.15.0 Linux/amd64 builds pass. Packaged `--browser` serves the POS page with HTTP 200 from `127.0.0.1`. |
| App icons | Shared 1024×1024 source derived from the existing ZiadaPOS logo; Linux window icon configured and PNG embed tested. Wails generates Windows/macOS icons on native runners; CI asserts those generated assets before packaging. |
| Not verified here | Native WebView window, API/Compose-backed login/sale, and Windows/macOS builds. The manual, artifact-only Actions workflow has not been triggered. |
| Local artifact / publication | `apps/desktop/build/bin/ZiadaPOS` (ELF x86-64, about 165 MB). No code was pushed and no public release was created. |
