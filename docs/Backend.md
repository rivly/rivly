This page describes the conventions that apply to every Go package in Rivly.

---

## Layout

Packages are grouped by responsibility. There is no `utils` package and no
`models` package, and there never will be.

`cmd/rivly/main.go` is limited to wiring, lifecycle and graceful shutdown. Any
logic that grows there belongs in a package.

Accept interfaces, return structs. Declare an interface where it is consumed, not
where it is implemented, and keep it as small as the consumer needs.

Pass `context.Context` down to every query, every daemon call and every session
call.

---

## Errors

Check every error.

Wrap with `fmt.Errorf("...: %w", err)` and keep `%w` at the end of the message,
so the printed chain reads newest to oldest.

Compare with `errors.Is` and `errors.As`. Never match on an error string: a
sentinel error or a typed error is always available instead.

At the handler boundary, log the cause once through `serverError` and return a
generic message. An API response never carries an internal detail.

---

## Logging

Logging is `log/slog`, structured, JSON to stdout.

Every request is logged once with method, path, status, size, duration, client IP
and request ID.

Never log a secret, a token, a password, or a URL that may carry credentials in
its userinfo.

---

## HTTP

The router is `go-chi/chi/v5`.

Business endpoints are versioned under `/api/v1/`. Operational endpoints such as
`/api/health` stay unversioned, because a probe should not have to track an API
version.

Use `middleware.ClientIPFrom*`, never the deprecated `RealIP`. Use
`httprate.LimitBy`, never `LimitByIP`. golangci-lint flags the deprecated forms.
Fix them rather than silencing the linter.

Request bodies are bounded and decoded with unknown fields rejected.

---

## Database

Database access goes through sqlc. Write SQL in
`internal/database/queries/*.sql`, then run `sqlc generate`.

Never hand-write query code, and never edit `internal/database/db/`, which is
generated.

See [Data Model](<Data Model.md>) for schema and migration rules.

---

## Concurrency

Background loops take a `context.Context` and return when it is cancelled.

Fan-out over a slice writes into a preallocated result slice at a fixed index,
never into a shared map, so no lock is needed.

Anything that mutates shared server state is guarded by a dedicated mutex, named
after what it protects.

---

## Purity

The binary builds with `CGO_ENABLED=0`. Never add a CGO dependency.

SQLite is `modernc.org/sqlite`, never `mattn/go-sqlite3`.

See [Tech Stack](<Tech Stack.md>) for the reasoning.

---

## Comments

Write no comments. Use clear names and small functions instead.

A comment that explains what the code does is a naming failure. A comment that
explains why a decision was made belongs in an ADR.
