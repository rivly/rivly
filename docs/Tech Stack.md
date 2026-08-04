This page lists the technologies Rivly uses and the reason each one was retained.

The version of record is always the manifest: `go.mod` for the backend,
`web/package.json` for the dashboard. This page explains choices, not versions.

---

## Backend

### Stack

- Go
- `go-chi/chi/v5` for routing
- `log/slog` for structured logging
- `alexedwards/scs` for sessions, `alexedwards/argon2id` for password hashing
- `coder/websocket` for the container terminal

### Why

Go gives a single static binary with no runtime to install, which is the whole
premise of a self-hosted tool. chi is a thin layer over `net/http` rather than a
framework, so the standard library stays visible. slog removes a dependency that
every Go service used to carry.

---

## Docker

### Stack

- `github.com/moby/moby/client` and `github.com/moby/moby/api/types`

### Why

This is the official SDK under its current import path. The older
`github.com/docker/docker` is deprecated since Docker v29 and is flagged by
govulncheck, so it is never used here.

See [Docker Integration](<Docker Integration.md>).

---

## Database

### Stack

- SQLite through `modernc.org/sqlite`
- `sqlc` for query code generation
- `pressly/goose` for migrations

### Why

SQLite means no database to run alongside the container, which keeps the install
to one artifact. `modernc.org/sqlite` is a pure Go implementation, so
`CGO_ENABLED=0` still holds and the binary stays static and portable across base
images. `mattn/go-sqlite3` would break both.

sqlc generates type-safe code from real SQL. There is no ORM: the queries stay
readable, reviewable, and cheap to reason about.

goose keeps migrations as plain SQL files, embedded in the binary and applied at
startup.

---

## Compose

### Stack

- The Docker Compose CLI, invoked as a subprocess

### Why

Compose has no usable Go library. Reimplementing its file format and dependency
resolution is out of scope, so Rivly shells out.

This is the one dependency that is not satisfied by the binary itself. It is
resolved by shipping the Compose plugin inside the image, which is why that image
is not distroless. See [Packaging](Packaging.md).

---

## Git

### Stack

- `go-git/go-git/v5`

### Why

GitOps stacks need to clone a repository and read a remote commit hash. go-git is
pure Go, so no `git` binary is required at runtime, which matters for a minimal
image.

---

## Frontend

### Stack

- Vite, React, TypeScript
- TanStack Router and TanStack Query
- Base UI for headless widgets, TanStack Table for tables
- CSS Modules, no framework
- oxlint

### Why

Headless libraries give behaviour and accessibility while leaving every visual
decision to us, which is what keeps the interface from looking generic. CSS
Modules give scoping without a build-time styling language.

See [Frontend](Frontend.md).
