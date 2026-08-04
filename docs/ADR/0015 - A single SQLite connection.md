# ADR-0015 - A single SQLite connection

| Field | Value |
| --- | --- |
| **Identifier** | ADR-0015 |
| **Date** | 2026-07-15 |
| **Status** | Accepted |

---

## Context

SQLite allows one writer at a time. With a normal connection pool, concurrent
writes surface as `SQLITE_BUSY`, which the driver retries until the busy timeout
expires and then fails.

Rivly writes from several places at once: request handlers, the poller refreshing
snapshots, and the Git poller updating stacks. Under WAL that mostly works, and
mostly is the problem. The failure appeared as intermittent errors under load
that were not reproducible on demand.

Raising the busy timeout hides the symptom without removing the race.

---

## Decision

The pool is capped at exactly one connection, with idle connections never
retired. Every query in the process is serialised through it.

The connection uses WAL journaling, foreign keys enabled, a busy timeout, and
immediate transaction locking so a transaction takes its write lock upfront
rather than upgrading mid-way.

---

## Consequences

Write contention is gone by construction. There is no `SQLITE_BUSY` to retry
because there is never a second writer.

One rule follows, and it is not obvious: **a query issued outside a transaction
while a transaction is open on the same connection deadlocks.** The transaction
holds the only connection, and the other query waits for a connection that will
never be released. Every query inside a transaction must go through the
transactional handle.

Reads serialise behind writes. At Rivly's volume this is invisible, and
[ADR-0006](<0006 - Docker as the source of truth.md>) keeps the database off the read path of anything a user waits on.

Raising the cap looks like an easy performance win and would reintroduce the
original bug. That is why this ADR exists.
