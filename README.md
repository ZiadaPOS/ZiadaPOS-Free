# ZiadaPOS

ZiadaPOS is a point-of-sale and retail-management monorepo. It combines the API, MCP agent, and web app while retaining their source histories.

## Components

| Path | Source repository | Imported branch and revision |
|---|---|---|
| `services/api/` | [ZiadaPOS/ZiadaPOS-GO-API](https://github.com/ZiadaPOS/ZiadaPOS-GO-API) | `main` — `f630e1009c2e07b91d2e71190a67595bd45e3c61` |
| `services/agent/` | [ZiadaPOS/ZiadaPOS-agent](https://github.com/ZiadaPOS/ZiadaPOS-agent) | `main` — `69e55fd785d201bf6695b5c778c79dbfff20becc` |
| `apps/web/` | [troubleman96/ziadapos-web-app](https://github.com/troubleman96/ziadapos-web-app) | `master` — `7fa5dc4f8d58e97fd3ffbfc87e21921bd0d59da3` |
| `apps/desktop/` | Wails desktop shell that packages the existing Next.js UI | Local implementation in this monorepo |

Each component retains its own README, configuration, and development instructions. Start in the component directory you want to work on.

## Desktop application

The Wails app reuses `apps/web/` in both its embedded desktop WebView and optional local-browser mode (`ZiadaPOS --browser`). The browser/desktop build is still the normal Next.js UI; it is not a WebAssembly port. See [`apps/desktop/README.md`](apps/desktop/README.md) for build and run commands, [`docs/desktop-architecture.md`](docs/desktop-architecture.md) for the architecture, and [`docs/desktop-release.md`](docs/desktop-release.md) for the proposed release assets and review status.

## License

The API, agent, and repository-level project material are licensed under GNU AGPL version 3. The web app's existing MIT license is preserved at `apps/web/LICENSE`. See [`LICENSES.md`](LICENSES.md) for scope and third-party terms.
