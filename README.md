# ZiadaPOS

ZiadaPOS is a point-of-sale and retail-management monorepo. It combines the API, MCP agent, and web app in a single workspace.

## Components

| Path | Component |
|---|---|
| `services/api/` | Go API for ZiadaPOS |
| `services/agent/` | MCP agent for ZiadaPOS |
| `apps/web/` | Next.js web app |
| `apps/desktop/` | Wails desktop shell that packages the existing Next.js UI |

Each component retains its own README, configuration, and development instructions. Start in the component directory you want to work on.

## Desktop application

The Wails app reuses `apps/web/` in both its embedded desktop WebView and optional local-browser mode (`ZiadaPOS --browser`). The browser/desktop build is still the normal Next.js UI; it is not a WebAssembly port. See [`apps/desktop/README.md`](apps/desktop/README.md) for build and run commands, [`docs/desktop-architecture.md`](docs/desktop-architecture.md) for the architecture, and [`docs/desktop-release.md`](docs/desktop-release.md) for the proposed release assets and review status.

## License

The API, agent, and repository-level project material are licensed under GNU AGPL version 3. The web app's existing MIT license is preserved at `apps/web/LICENSE`. See [`LICENSES.md`](LICENSES.md) for scope and third-party terms.
