| Field | Value |
| --- | --- |
| **Identifier** | ADR-0011 |
| **Date** | 2026-07-14 |
| **Status** | Accepted |

> **Amended 2026-08-04.** A consequence originally stated that
> `RIVLY_COMPOSE_BIN` accepted a single executable, which made Compose v2
> awkward to configure. That was a defect and has been fixed. The decision
> itself is unchanged.

---

## Context

Rivly must deploy a compose file: create networks and volumes, resolve
dependencies and build order, apply the many shorthand forms the compose
specification allows, and reconcile against what is already running.

There is no Go library that does this. Compose v2 is a Go program, but its
functionality is not exposed as a supportable public API.

Reimplementing the compose specification was considered and rejected. It is a
large, moving target, and a partial implementation would silently mis-deploy
files that work everywhere else, which is worse than not supporting them.

---

## Decision

Managed stacks are deployed by invoking the Compose CLI as a subprocess.

Rivly materialises the compose file and an env file under the data directory,
then runs the executable named by `RIVLY_COMPOSE_BIN` with `DOCKER_HOST` pointed
at the target environment.

Git stacks work identically, except the compose file comes from a clone rather
than from the database.

Compose output is captured and returned to the user on failure, because a compose
error message is the most useful thing we can show.

---

## Consequences

Every compose file that works on the host works in Rivly, including features
Rivly has never heard of.

This is the one runtime dependency that is not inside the binary, which puts it
in direct tension with [ADR-0001](<0001 - A single self-hosted binary.md>) and [ADR-0003](<0003 - A pure Go build with CGO disabled.md>). A distroless image contains
no Compose binary, so the two goals are currently incompatible. See
[Packaging](../Packaging.md).

Rivly has to locate a Compose executable, and the two supported forms differ:
Compose v2 is a `docker` subcommand, Compose v1 is a standalone binary. The
command is detected at startup and can be overridden with `RIVLY_COMPOSE_BIN`.

Compose output can carry host paths, so returning it to the client leaks more
than a generic error would. Accepted deliberately, because a hidden compose error
makes the feature unusable.

This ADR is the most likely of the set to be superseded.
