This page describes what Rivly stores and the rules that govern the schema.

Rivly stores as little as possible. Docker is the source of truth for runtime
state; the database holds only what a daemon cannot remember.

---

## Tables

| Table | Holds |
| --- | --- |
| `users` | Accounts. Email, display name, role |
| `credentials` | Password hashes, keyed by user and type |
| `tokens` | Reserved for password reset and invitations. Not used yet |
| `sessions` | Session store, managed by scs |
| `environments` | Docker endpoints, plus the last known system snapshot |
| `stacks` | Rivly-managed compose projects, per environment |
| `registries` | Registry credentials, password encrypted |
| `git_credentials` | Git tokens, encrypted |

---

## Environments

An environment is a Docker endpoint: a name, a kind, and a URL.

`snapshot` stores the last successful system info as JSON, and `snapshot_at` when
it was taken. This is what lets an unreachable host still render something
useful.

One `local` environment is seeded at startup when the table is empty. There is no
endpoint to create, edit or delete environments yet.

---

## Stacks

A stack row is unique per environment and name.

`source` is either `content`, meaning the compose file lives in the row, or
`git`, meaning it is cloned from a repository. The `git_*` columns carry the
repository URL, ref, compose path, credential id, the commit currently deployed,
the last remote hash observed, the polling interval, and the last error.

`env` stores the stack environment variables as a JSON array, and is rendered
into a `.env` file at deploy time.

`created_by` and `updated_by` hold display names, not foreign keys, so history
survives account deletion. Automatic Git updates are attributed to `GitOps`.

---

## Rules

**Migrations are immutable.** Once a migration has been applied anywhere, it is
never edited. Change the schema by adding a new goose migration.

**Timestamps are `INTEGER` unix values**, produced by `unixepoch()`. Never
`DATETIME`: Go cannot scan SQLite's datetime text into `time.Time`, and the
failure is silent enough to reach production.

**Queries live in `internal/database/queries/`.** After editing them, run
`sqlc generate`. The output in `internal/database/db/` is generated and is never
edited by hand.

**Secrets are never stored in plain text.** Registry passwords and Git tokens are
encrypted with AES-256-GCM before they reach a column, and are never returned by
the API. See [Security](Security.md).

---

## Connection

The pool is capped at a single connection, with WAL journaling, foreign keys on,
a busy timeout, and immediate transaction locking.

That cap removes a class of concurrency bugs, at the cost of one rule: a query
issued outside a transaction while a transaction is open on the same connection
will deadlock. Everything inside a transaction must go through the transactional
`Queries` handle.
