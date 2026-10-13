This directory receives generated desktop resources during `wails build`:

- `web/` — the Next.js standalone server and static assets.
- `runtime/node` (or `runtime/node.exe`) — a Node.js runtime built for the same OS/architecture as the desktop binary.

Generated resource directories are ignored by Git. Do not commit build artifacts or runtime executables.
