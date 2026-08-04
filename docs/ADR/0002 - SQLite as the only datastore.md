# ADR-0002 - SQLite as the only datastore

| Field | Value |
| --- | --- |
| **Identifier** | ADR-0002 |
| **Date** | 2026-07-13 |
| **Status** | Accepted |

---

## Context

Rivly needs to persist accounts, sessions, environments, managed stacks and
encrypted credentials.

[ADR-0001](<0001 - A single self-hosted binary.md>) forbids a companion service, which removes PostgreSQL and MySQL
from consideration regardless of their merits.

The write volume is negligible: a handful of rows per user action, plus one
snapshot update per environment per poll.

---

## Decision

State lives in a single SQLite file, opened with WAL journaling, foreign keys
enabled, a busy timeout and immediate transaction locking.

Migrations are goose SQL files, embedded in the binary and applied at startup, so
a user never runs a migration step by hand.

Timestamps are stored as `INTEGER` unix values produced by `unixepoch()`, never
as `DATETIME`. Go cannot scan SQLite's datetime text into `time.Time`, and the
failure mode is quiet enough to reach production.

---

## Consequences

Backup is one file plus the data directory, which is a feature for the target
audience.

The driver must not require CGO, which is settled in [ADR-0003](<0003 - A pure Go build with CGO disabled.md>).

Concurrent writes need care, which is settled in [ADR-0015](<0015 - A single SQLite connection.md>).

Storing a metrics time series is impractical at this scale, which is one reason
it is a non-objective.

Migrating to a client-server database later would mean revisiting the SQL, since
sqlc generates against the SQLite dialect.
