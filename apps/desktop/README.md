# ZiadaPOS Desktop

This Wails v2 shell reuses the existing `apps/web` Next.js POS interface. It builds the UI in Next.js standalone mode, embeds the standalone output plus a matching Node.js runtime in the desktop binary, starts that server on a loopback-only ephemeral port, and routes the Wails WebView's requests to it.

The window is the app's **local embedded WebView**—not a separate Chrome/Firefox browser window. Wails uses the platform web engine already available on the computer (WebView2 on Windows, WKWebView on macOS and WebKitGTK on Linux), rather than shipping Chromium.

The React/TypeScript UI is **not compiled to WebAssembly**. Unlike a Rust app that explicitly targets WASM, this app keeps its normal browser frontend and Next.js server routes; the same built frontend is hosted locally for Wails and for an ordinary browser.

## Requirements

- Go 1.25 or later
- Node.js 22 (the build script embeds the current Node executable)
- Bun (the web app uses `bun.lock`)
- Wails CLI v2.15.0
- Native platform build prerequisites from the [Wails installation guide](https://wails.io/docs/gettingstarted/installation/)

Install Wails and verify the machine:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
wails doctor
```

## Run the desktop shell in development

Start the existing Next.js development server in one terminal:

```sh
cd apps/web
bun install --frozen-lockfile
bun run dev --hostname 127.0.0.1 --port 3000
```

Then start Wails in another terminal:

```sh
cd apps/desktop
ZIADAPOS_DESKTOP_DEV_URL=http://127.0.0.1:3000 wails dev -s
```

On PowerShell, set `$env:ZIADAPOS_DESKTOP_DEV_URL = 'http://127.0.0.1:3000'` before running `wails dev -s`.

## Open the packaged UI in the ordinary browser

Build the app for the current platform, then run the resulting executable with `--browser`. The same bundled Next.js server starts on loopback and the default system browser opens it. For development, use the Next.js dev server above and open `http://127.0.0.1:3000` in the normal browser.

The browser mode is local to the computer and does not expose a listening service to the LAN. It still needs the existing Go API: `POS_API_URL` configures that API and defaults to `http://localhost:8080/api/v1`. Set `POS_API_URL` before launching the desktop executable if the API is deployed elsewhere. The Go API, PostgreSQL, Redis and object storage are **not** bundled by this shell; this first desktop slice is not an offline POS implementation.

## Build for the current platform

From `apps/desktop`:

```sh
wails build
```

Wails runs the configured Bun install and `prepare-frontend.mjs` steps, then compiles the desktop shell with the generated frontend/runtime embedded. On Ubuntu 24.04, install `libgtk-3-dev` and `libwebkit2gtk-4.1-dev`, then build with `wails build -tags webkit2_41`.

## Build all desktop operating systems and release

The manual workflow at `.github/workflows/desktop-build.yml` builds on Linux, Windows and macOS runners and uploads temporary Actions artifacts only. The separate `.github/workflows/desktop-release.yml` runs when a `vMAJOR.MINOR.PATCH` tag is pushed; it builds the native app on each platform runner, packages the executable/app bundle, computes SHA-256 checksums, and creates a GitHub Release with installation notes. **Pushing a matching tag publishes a release.** Review [`../../docs/desktop-release.md`](../../docs/desktop-release.md) and the exact repository changes before creating or pushing any tag.

Builds use a Node.js executable and Next.js standalone dependencies produced on the same runner as the desktop binary. The initial release targets Linux amd64, Windows amd64, and the native architecture of the macOS hosted runner; it does not produce universal or additional CPU-architecture builds.

## Application icons

`build/appicon.png` is the 1024×1024 PNG derived from the existing `apps/web/public/ziada-final.jpeg` brand mark. Wails v2 uses this shared source to generate the Windows multi-size `build/windows/icon.ico` and embed it in the executable, and to generate `iconfile.icns` inside the macOS `.app` bundle. The Linux Wails window is explicitly given the same PNG through `options.Linux.Icon`.

The Linux download is a raw executable archive, not a native package: it gets the application window icon but does not install a `.desktop` launcher/menu entry. The Windows ZIP and macOS ZIP likewise contain the app binary/bundle, not MSI/DMG installers.

The first release archives are unsigned, have no native installer or updater, and depend on the existing ZiadaPOS API. Linux needs GTK 3 and WebKit2GTK 4.1; Windows needs the WebView2 Runtime. macOS signing/notarization and platform-specific smoke tests are not yet provided. See [`../../docs/desktop-architecture.md`](../../docs/desktop-architecture.md) for the architecture and [`RELEASE.md`](RELEASE.md) for user-facing runtime notes.
