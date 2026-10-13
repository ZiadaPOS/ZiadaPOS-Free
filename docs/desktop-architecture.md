# ZiadaPOS desktop architecture and Storytold review

## Recommendation

Keep ZiadaPOS as one monorepo and add `apps/desktop` as a Wails shell around the existing `apps/web` frontend. Keep the Gin POS API and MCP agent in `services/`; do not fork or rewrite those components just to add a desktop window.

The local implementation uses Wails v2.15.0, currently documented as the stable Wails major version (Wails v3 is in beta). It embeds the existing Next.js standalone server and a Node.js runtime into the Go app, starts that server on loopback at a random local port, and routes the Wails AssetServer requests to it. This preserves the current Next.js UI and its API routes. The user sees an app window with an embedded platform WebView.

## What the local repository already contains

| Area | Current component | Desktop role |
|---|---|---|
| `apps/web/` | Next.js 16 UI, including POS proxy/session API routes | Reused as the desktop UI and local frontend server |
| `apps/desktop/` | New Wails v2 Go shell | Window lifecycle, embedded WebView, packaged frontend/runtime |
| `services/api/` | Go/Gin POS API | Remains the POS domain API and data-access service |
| `services/agent/` | MCP agent service | Remains in the same monorepo; not duplicated in the desktop shell |

The Next app is not a static-only frontend: it defines server-side route handlers under `src/app/api/`, including POS API proxy and session routes. A plain `go:embed` of a static export would therefore be an incomplete replacement. The desktop build instead packages Next's standalone Node server and static assets. Its Node executable and any native frontend dependencies must match the desktop target architecture.

## Storytold products and distribution observed

Storytold's GitHub organization lists a family of standalone Crafting Apps, including PhotoCraft, VectorCraft, FilmCraft, LightCraft, PdfCraft, EffectCraft and DesignCraft. The organization listing also shows projects such as GridCraft, WordCraft, CADCraft, a Craft Launcher and `artcraft-services` (described as backend/web frontend). PhotoCraft's README presents the apps as free, open-source tools, shares product conventions, and says the engine/UI can also run in a browser through WebAssembly. This is a useful product-family pattern, but it is **not** the architecture selected here: the reviewed apps describe themselves as pure Rust/native + WASM, not Wails wrappers, and their main product repositories are separate rather than one application monorepo.

For an example of its release packaging, PhotoCraft's release page shows versioned GitHub Releases with checksums and platform-specific downloadable assets. Its README lists Windows MSI and portable ZIP, macOS DMG, Linux AppImage/Flatpak/DEB/RPM/tarball, FreeBSD tarball, and a web/WASM ZIP; it describes Windows signing and macOS signing/notarization. The latest release page accessed for this review showed PhotoCraft v0.8.2 and 28 assets. This is evidence from that product's published material, not a claim that every Storytold project ships the exact same formats.

**What to borrow:** clear product identity, one build per supported platform/architecture, explicit install formats, checksums, signed installers where available, and a browser distribution where the app actually supports it. For ZiadaPOS, keep the monorepo and add the desktop shell alongside the existing web and services code rather than creating a separate repository for each target.

## Local runtime and API boundary

1. Wails builds the existing web UI in Next.js standalone mode.
2. The build script copies the standalone server, `.next/static`, `public`, and the runner's Node executable under `apps/desktop/resources/` before Go embeds them.
3. On app startup, the shell extracts those files into a temporary per-run directory under the user's cache, launches Node with `HOSTNAME=127.0.0.1` and an ephemeral port, waits for readiness, and serves WebView requests through a reverse proxy. The process and extracted files are removed on app exit.
4. `POS_API_URL` continues to configure the existing Next server-side POS proxy. It defaults to `http://localhost:8080/api/v1` as in the web app. The user can point it at the existing API deployment, or run the Go API locally with its required services.

The same packaged executable also supports `--browser`: it starts that exact Next.js standalone UI on an ephemeral `127.0.0.1` port and opens the computer's default browser. During development, the Next.js dev server can be opened directly at `http://127.0.0.1:3000`. Both modes use the same React/TypeScript frontend and the same API route behavior; neither is a WebAssembly build.

For local development, `services/api/docker-compose.yml` starts the API plus PostgreSQL, Redis and MinIO and publishes its host ports on loopback by default (`HOST_BIND_ADDRESS`). When running the Go API directly, `API_LISTEN_ADDR=127.0.0.1:8080` restricts its listener to this computer. The desktop/browser shell does not start or package those services.

## Offline is local data and synchronization, not a framework switch

A real offline POS needs a durable local store, an atomic local sale/outbox, replay-safe synchronization, conflict rules and recovery tests. The current API opens PostgreSQL and uses PostgreSQL-specific types/queries (`jsonb`, `ILIKE`, `tags::text`, `ON CONFLICT ... RETURNING`); swapping its driver to SQLite would not make the API work offline. The recommended path is a limited Go + SQLite local service inside the desktop app, with queued, idempotent sales synchronized to the existing central API. Do not package PostgreSQL, Redis and MinIO just to simulate offline mode.

Keep Next.js for now: the current UI uses Next request/cookie handling for its HttpOnly POS session proxy and route redirects, and Wails already packages its Node runtime. A Vite migration could remove that runtime, but would move/replace the BFF/session behavior and still would not supply local data or sync. Add a UI data-provider boundary first; reconsider Vite only if a smaller static-only desktop package becomes a firm requirement.

The packaged `--browser` path can use the same desktop-owned offline service only while the local host process is running and the service remains bound to loopback. Direct browser-only offline use is separate work: it needs a PWA service worker, IndexedDB storage and its own sync path. No service worker/PWA manifest is present now. Until local persistence and synchronization are implemented and tested, this app must not be described as offline POS.

Wails reuses the OS web engine rather than bundling Chromium: WebView2 on Windows, WKWebView on macOS, and WebKitGTK on Linux. This keeps the app lean but makes the engine/runtime a platform prerequisite and calls for smoke testing each OS, particularly login/session cookies and printing.

## Build and distribution

The manual `.github/workflows/desktop-build.yml` remains a `workflow_dispatch` build/test workflow that uploads short-lived artifacts only. The prepared `.github/workflows/desktop-release.yml` is tag-triggered: `vMAJOR.MINOR.PATCH` tags build Linux amd64, Windows amd64, and a native macOS app on their respective runners, then attach platform archives, installation notes, and SHA-256 checksums to a GitHub Release. The tag version is written into Wails app metadata during the build. The tracked 1024×1024 `apps/desktop/build/appicon.png` is passed to the Linux window; Wails generates and embeds the Windows `.ico` executable resource and macOS `.icns` app-bundle icon from it. Workflow checks validate the Windows and macOS icon files. The release workflow has not been run; no tag, push, GitHub Release, or release asset upload has been performed.

The desktop package embeds Node.js/Next output, so platform and architecture must match—not just the Go binary target. The initial release matrix is intentionally one architecture per OS and is not universal coverage. Artifacts are unsigned testing archives, not native installers: Linux has no `.desktop` launcher registration, Windows is a ZIP of the executable, and macOS is a ZIP of the `.app` bundle. They still depend on platform WebView prerequisites and the separately deployed POS API. See [`desktop-release.md`](desktop-release.md) for exact assets, icons, prerequisites, and review status.

Still outside this initial release scope:

- native installer formats and install/upgrade/uninstall behavior;
- macOS Developer ID signing/notarization and Windows code signing;
- a Windows WebView2 runtime bootstrap strategy;
- update channels and rollback policy;
- additional architecture builds and per-platform smoke tests;
- a secure launch-time API endpoint configuration UI instead of inherited `POS_API_URL`.

## Sources

- [Storytold repositories](https://github.com/orgs/storytold/repositories)
- [PhotoCraft README and product list](https://github.com/storytold/photocraft)
- [PhotoCraft releases](https://github.com/storytold/photocraft/releases)
- [Wails introduction](https://wails.io/docs/introduction)
- [Wails v2 installation requirements](https://wails.io/docs/gettingstarted/installation/)
- [Wails v2 manual builds and build-asset generation](https://wails.io/docs/guides/manual-builds/)
- [Wails AssetServer and application options](https://wails.io/docs/reference/options/)
- [Wails cross-platform build guidance](https://wails.io/docs/guides/crossplatform-build)
- [Wails CLI build flags and targets](https://wails.io/docs/reference/cli/)
