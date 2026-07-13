<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="brand/logo-white-full.png" />
    <img alt="Rivly" src="brand/logo-blue-full.png" width="250" />
  </picture>
</p>

<p align="center">
  The open-source dashboard for Docker, from a single host to a full Swarm.
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT" /></a>
  <img src="https://img.shields.io/badge/status-early%20development-orange.svg" alt="Status: early development" />
  <a href="https://rivly.dev"><img src="https://img.shields.io/badge/website-rivly.dev-2f7bff.svg" alt="Website" /></a>
</p>

<p align="center">
  <a href="https://rivly.dev">Website</a> ·
  <a href="https://docs.rivly.dev">Documentation</a> ·
  <a href="https://rivly.dev/#notify">Waitlist</a>
</p>

---

Rivly is an open-source dashboard for Docker. It manages a single host with a few
compose files and a full Swarm with dozens of nodes the same way, from one clean,
self-hosted interface. No accounts, no telemetry, no lock-in.

> [!NOTE]
> Rivly is in early development and not ready to run yet. It is being built in the
> open. Star the repo to follow along, or leave your email at
> [rivly.dev](https://rivly.dev/#notify) to hear the day it ships.

## What Rivly does

- **One view for everything.** Every container, service, and node, with its logs and health.
- **Docker and Swarm, same UI.** Run a single host or a full cluster without switching tools.
- **GitOps deploys.** Point Rivly at a Git repository and it deploys your stacks, then keeps them in sync.
- **Self-hosted.** Runs entirely on your own hardware as a single container.

## Tech stack

| Layer | Stack |
| --- | --- |
| Backend | Go, chi, official Docker SDK (`moby/moby/client`) |
| Database | SQLite (pure-Go, no CGO) |
| Frontend | React, Vite, TanStack Router and Query |
| Packaging | Single binary with the UI embedded, distroless image |

## Development

Prerequisites: **Go 1.24+**, **[Bun](https://bun.sh)**, and a running **Docker** daemon.

```bash
git clone https://github.com/rivly/rivly.git
cd rivly
```

Backend (API on `:8080`):

```bash
make run
```

Frontend (dashboard on `:5173`, in a separate terminal):

```bash
cd web
bun install
bun run dev
```

Other commands: `make build` (compile the binary), `make lint`, `make test`.

## Project structure

```
cmd/rivly/     Go entrypoint
internal/      backend (API, Docker, Swarm, auth)
web/           React dashboard (Vite + TanStack)
```

## Contributing

Issues and pull requests are welcome. For anything substantial, open an issue first
to discuss the approach.

## License

[MIT](LICENSE) © The Rivly Authors
