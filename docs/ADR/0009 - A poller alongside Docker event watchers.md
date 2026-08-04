| Field | Value |
| --- | --- |
| **Identifier** | ADR-0009 |
| **Date** | 2026-07-14 |
| **Status** | Accepted |

---

## Context

[ADR-0006](<0006 - Docker as the source of truth.md>) makes the daemon authoritative, so the interface has to learn about
changes that happen outside Rivly.

The daemon exposes an event stream, which is the obvious mechanism: it is
immediate and cheap.

It is also not sufficient. The stream dies when the daemon restarts, when a
network link drops, or when the connection is idled out by something in between,
and a dead stream is indistinguishable from a quiet host. An environment that
never answers emits no events at all, so its transition to down would never be
noticed.

---

## Decision

Rivly runs both mechanisms, converging on the same event.

A **watcher** per environment subscribes to the daemon event stream, filters out
noise such as exec and health-check events, debounces bursts, and publishes.

A **poller** queries every environment on a fixed interval regardless, refreshes
its snapshot, and publishes when the environment's fingerprint has changed.

The fingerprint comparison is what makes the redundancy free: whichever mechanism
notices a change first publishes it, and the other one sees no change and stays
silent.

---

## Consequences

Liveness is guaranteed. A broken event stream degrades reactivity to the poll
interval instead of freezing the interface.

Reactivity is sub-second when the stream is healthy.

The cost is a daemon query per environment per interval, whether or not anything
happened. At the default of five seconds this is negligible.

Neither mechanism can be removed as redundant without reintroducing the failure
mode the other one covers. That is the reason this ADR exists.

The poller currently rebuilds and marshals full environment state to compute a
fingerprint, which is more work than the comparison needs.
