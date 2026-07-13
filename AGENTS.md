# AGENTS.md

Rivly is an open-source dashboard for Docker, from a single host to a full Swarm. This is the product monorepo: a **Go backend** and a **React frontend** compiled into a **single binary** (the Go server embeds the built web UI). Self-hosted, MIT.

## Layout

```
cmd/rivly/              Go entrypoint (wiring, http.Server, graceful shutdown)
internal/
  config/               env config (RIVLY_*), no hardcoded paths/secrets
  database/             SQLite open + goose migrations
  database/db/          sqlc-GENERATED code — never edit by hand
  database/queries/     hand-written SQL (source for sqlc)
  database/migrations/  goose migrations (immutable once applied)
  auth/                 scs sessions, argon2id, Authenticator, local provider
  server/               chi router, handlers, middleware
web/                    React + Vite frontend (embedded at build time)
bruno/                  Bruno API collection
```

## Commands

```bash
# backend (repo root)
make run          # API on :8080
make build        # static binary (CGO_ENABLED=0) -> bin/rivly
make test         # go test ./...
make lint         # golangci-lint v2 — MUST pass
sqlc generate     # regenerate internal/database/db after editing queries

# frontend (web/)
bun run dev       # dashboard on :5173
bun run build     # tsc -b + vite build
bun run lint      # oxlint — MUST pass
```

Before finishing a change, the relevant `lint` + `build` (+ `test` for Go) must pass.

## Rules that keep you out of trouble

These are the mistakes that get made here. Don't.

- **Docker SDK** is `github.com/moby/moby/client` (+ `.../api/types`). NOT `github.com/docker/docker` — deprecated since Docker v29 and flagged by govulncheck.
- **Stay pure-Go**: the binary builds with `CGO_ENABLED=0` for the distroless image. Never add a CGO dependency. SQLite is `modernc.org/sqlite` — never `mattn/go-sqlite3`.
- **DB access is sqlc**: write SQL in `internal/database/queries/*.sql`, run `sqlc generate`. Never hand-write query code or edit `internal/database/db/*`.
- **Migrations are immutable**: add a new goose migration, never edit an applied one. Timestamps are **INTEGER unix** (`unixepoch()`), not `DATETIME` — Go cannot scan SQLite's datetime text into `time.Time`.
- **chi APIs**: use `middleware.ClientIPFrom*` (not deprecated `RealIP`) and `httprate.LimitBy` (not `LimitByIP`). golangci-lint flags the deprecated ones — fix them, don't silence.
- **Auth already exists** in `internal/auth`. Never roll your own hashing or session cookies. Auth failures stay generic and constant-time (no user enumeration).
- **UI is hand-built vanilla CSS**. No Tailwind, no CSS-in-JS, no component library. Co-locate `Component.css`.
- **Server state is TanStack Query**, never `useEffect` + `fetch`. Routing is TanStack Router file-based (`web/src/routes/`); the router plugin goes **before** `react()` in `vite.config.ts`; `routeTree.gen.ts` is generated, not hand-edited.
- **No comments** anywhere. Clear names and small functions instead.

## Backend (Go)

- Go 1.24+. Router `go-chi/chi/v5`. Logging `log/slog` (structured).
- API is versioned: business endpoints under `/api/v1/`; ops (`/api/health`) unversioned.
- Config from env via `internal/config`. Pass `context.Context` down to every query and session call.
- Errors: check every one, wrap with `fmt.Errorf("...: %w", err)`, compare with `errors.Is`/`errors.As`. Log the cause once at the handler (`s.serverError`) and return a generic message; never leak internals in a response.
- Accept interfaces, return structs (see `auth.Store`). Group packages by responsibility, never `utils`/`models`.
- Tools: `golangci-lint` and `sqlc` via brew; goose is a library dependency.

## Frontend (web/)

- Vite 8 + React 19 + TypeScript (strict) + vanilla CSS. Linter oxlint.
- Function components only; `ref` is a prop (no `forwardRef`); no `React.FC`; no top-level `import React`; type-only imports use `import type`.
- Call the API at `/api/v1/...` with `credentials: 'include'` (auth is a session cookie). In dev, proxy `/api` through Vite so it stays same-origin — the backend rejects cross-origin browser POSTs (CSRF protection).
- Server data via TanStack Query; local/UI state via React state or context.

## Packaging

Single binary: `web/` is built, embedded with `go:embed`, and served with an SPA fallback. Production image is distroless, `CGO_ENABLED=0`.

## Git

**Never run git.** No commits, pushes, staging, or branches — the human owns all git. When work is ready, hand over one single-line Conventional Commit message (`feat:`, `fix:`, `refactor:`, `chore:`, `docs:`) and stop.

## Ask first

Adding any dependency; new tooling or a build step; changing auth/security behavior; changing the API contract or DB schema.
