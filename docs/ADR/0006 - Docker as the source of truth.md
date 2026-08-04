| Field | Value |
| --- | --- |
| **Identifier** | ADR-0006 |
| **Date** | 2026-07-13 |
| **Status** | Accepted |

---

## Context

A dashboard can either mirror the daemon into its own database and serve from
that, or query the daemon on every read.

Mirroring is faster and survives a daemon outage, but it introduces a second
model of reality that drifts. Every drift is a bug where the interface confidently
shows something false, and users stop trusting the tool for the one thing it
exists to do.

Containers also change outside Rivly: someone runs `docker stop` in a terminal, a
restart policy fires, a healthcheck kills a process.

---

## Decision

Docker is the source of truth for runtime state. Listing containers, images,
volumes, networks or stacks queries the daemon, and the result is not persisted.

The database holds only what a daemon cannot remember: accounts, sessions,
environments, managed stacks and encrypted credentials.

One exception is explicit: the last successful system information of an
environment is stored as a snapshot, so an unreachable host still renders
something useful, clearly marked as down.

---

## Consequences

The interface cannot show stale state, because there is no stale state to show.

Every list costs a daemon round trip, and some lists cost two because usage flags
require cross-referencing the container list. This is acceptable at the scale
Rivly targets and would need revisiting for a host with thousands of containers.

When a daemon is unreachable, resource lists return an error rather than
degrading. Only the environment page degrades, through the snapshot.

Historical views are foreclosed. There is no way to answer "what was running
yesterday" without adding storage, which would reopen this ADR.
