This page describes how Rivly is put together and why it is shaped that way.

Rivly is a single Go process that talks to one or more Docker daemons, keeps a
small amount of state in SQLite, and serves a React dashboard. There is no queue,
no cache, no external service. That is a deliberate constraint: the whole product
must stay installable as one container.

---

## Principles

- One process, one binary, one file of state. Every added moving part must earn
  its place against that baseline.
- Docker is the source of truth for runtime state. Rivly stores only what Docker
  cannot remember: managed stacks, credentials, accounts, and a last-known
  snapshot of each environment.
- The database is never on the read path of a container list. Listing containers
  queries the daemon, not SQLite.
- The transport layer is thin. HTTP handlers parse, delegate and encode.

---

## Layout

| Path | Responsibility |
| --- | --- |
| `cmd/rivly/` | Wiring, lifecycle, graceful shutdown |
| `internal/config/` | Environment configuration, validated at startup |
| `internal/database/` | SQLite connection and goose migrations |
| `internal/database/db/` | sqlc-generated query code, never edited by hand |
| `internal/auth/` | Sessions, password hashing, the local provider |
| `internal/docker/` | Every call to a Docker daemon |
| `internal/compose/` | Compose invocations for managed stacks |
| `internal/gitrepo/` | Cloning and remote inspection for Git stacks |
| `internal/secret/` | AES-256-GCM envelope for stored credentials |
| `internal/registry/`, `internal/gitcred/` | Credential stores |
| `internal/events/` | In-process publish and subscribe hub |
| `internal/server/` | Router, middleware, handlers |
| `web/` | React dashboard |

---

## Request path

A browser request enters the chi router, passes through request ID, client IP,
logging, panic recovery, security headers, cross-origin protection and cookie
hardening, then reaches either a session-guarded handler or a stream handler.

Handlers resolve the target environment from SQLite, call `internal/docker` with
the environment URL, and encode the result. Nothing is cached between the daemon
and the response.

---

## Background loops

Three loops run for the lifetime of the process.

| Loop | Cadence | Purpose |
| --- | --- | --- |
| Poller | `RIVLY_POLL_INTERVAL`, 5s by default | Queries each environment, refreshes its snapshot, publishes a change event when the fingerprint moves |
| Watchers | Event-driven, one per environment | Subscribes to the Docker event stream and publishes a debounced change event |
| Git poller | 5s tick, per-stack interval | Compares the remote commit of Git stacks and redeploys when it moves |

The poller guarantees liveness when a daemon is unreachable. The watchers give
sub-second reactivity when it is. Both converge on the same event, and the
fingerprint comparison prevents duplicate publishes.

---

## Realtime

`internal/events` is a generic hub: subscribers get a buffered channel, and a
publish that would block is dropped rather than delayed.

The browser holds one `EventSource` on `/api/v1/events`. On each event it
replaces the environment in the query cache and invalidates the resource lists.
Logs, stats and image pulls are separate short-lived SSE streams. The container
terminal is a WebSocket bridged onto a Docker exec session.

Any `http.ResponseWriter` wrapper in the middleware chain must forward `Flush`
and `Hijack`, or streaming breaks.

---

## Stacks

A stack is a compose project. Rivly distinguishes two kinds.

**External** stacks are discovered, never managed. They are inferred from the
`com.docker.compose.project` label on running containers, so Rivly can show and
control something it did not deploy.

**Managed** stacks are rows in the database. Their compose file comes either from
content stored in the row or from a Git repository cloned under the data
directory. Listing merges both views: a discovered project that also has a row is
reported as managed.

---

## State

SQLite holds accounts and sessions, environments and their last snapshot, managed
stacks, and encrypted registry and Git credentials.

The connection pool is capped at a single connection, which removes an entire
class of concurrency bug at the cost of one constraint: a query issued outside a
transaction while a transaction is open deadlocks.

---

## Current gaps

Documented here so the architecture is not read as more complete than it is.

- The dashboard is not embedded in the binary yet, so the process serves the API
  only. See [Packaging](Packaging.md).
- Environments are seeded, not managed. One `local` row is created at startup and
  there is no endpoint to add a remote host.
- Swarm is reported as a boolean and a node count. Services, nodes and tasks are
  not modelled.
- Watchers read the environment list once at startup.
