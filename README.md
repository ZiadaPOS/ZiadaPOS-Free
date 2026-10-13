# ZiadaPOS

ZiadaPOS is a point-of-sale and retail-management monorepo. It combines the API, MCP agent, and web app while retaining their source histories.

## Components

| Path | Source repository | Imported branch and revision |
|---|---|---|
| `services/api/` | [ZiadaPOS/ZiadaPOS-GO-API](https://github.com/ZiadaPOS/ZiadaPOS-GO-API) | `main` — `f630e1009c2e07b91d2e71190a67595bd45e3c61` |
| `services/agent/` | [ZiadaPOS/ZiadaPOS-agent](https://github.com/ZiadaPOS/ZiadaPOS-agent) | `main` — `69e55fd785d201bf6695b5c778c79dbfff20becc` |
| `apps/web/` | [troubleman96/ziadapos-web-app](https://github.com/troubleman96/ziadapos-web-app) | `master` — `7fa5dc4f8d58e97fd3ffbfc87e21921bd0d59da3` |

Each component retains its own README, configuration, and development instructions. Start in the component directory you want to work on.

## License

The proposed license structure is GNU AGPL version 3 for the API, agent, and repository-level project material, with the web app's existing MIT license preserved at `apps/web/LICENSE`. See [`LICENSES.md`](LICENSES.md) for scope and third-party terms.
