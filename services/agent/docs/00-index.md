# ZiadaPOS MCP Agent — Documentation Index

Everything you need to understand, run, extend and deploy the agent.

## Overview

- **[01 — Architecture](01-architecture.md)** — system context, source layout, how requests and auth flow.
- **[02 — Configuration](02-configuration.md)** — every environment variable, defaults and security notes.
- **[03 — Client & Auth](03-client-and-auth.md)** — `ZiadaClient` internals: login, JWT refresh, store switching, envelope handling, uploads.

## Reference

- **[04 — Tools](04-tools.md)** — every one of the 111 MCP tools: parameters, behaviour and the API call behind it.
- **[05 — Agent & Persona](05-agent-and-persona.md)** — the `ask` tool, the Ngamia function-calling loop and the safety rules.
- **[06 — Product Images](06-product-images.md)** — the Wikimedia Commons → MinIO image pipeline.
- **[07 — Communication](07-communication.md)** — real SMS (SendAfrica) and email (MailAfrica) integration.

## Operations

- **[08 — Deployment](08-deployment.md)** — local, Docker, production VPS, CI/CD and troubleshooting.
- **[09 — Testing](09-testing.md)** — what is tested and how to run it.

See also the top-level **[README](../README.md)** (quick start) and **[redeploy.md](../redeploy.md)** (production runbook).
