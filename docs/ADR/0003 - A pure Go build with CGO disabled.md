| Field | Value |
| --- | --- |
| **Identifier** | ADR-0003 |
| **Date** | 2026-07-13 |
| **Status** | Accepted |

> **Amended 2026-08-04.** The production image is no longer distroless, see
> [ADR-0016](<0016 - A minimal base image carrying the Compose plugin.md>). The
> pure Go rule itself is unchanged and still in force.

---

## Context

[ADR-0001](<0001 - A single self-hosted binary.md>) requires one artifact, and the target image is distroless: no shell,
no package manager, no libc to speak of.

A binary linked against C libraries needs a base image that provides them, which
means a larger image, a larger attack surface, and a build that breaks across
distributions.

The obvious SQLite driver, `mattn/go-sqlite3`, requires CGO.

---

## Decision

The binary builds with `CGO_ENABLED=0`. No dependency may require CGO.

The consequences of that rule are picked, not accepted afterwards:

- SQLite is `modernc.org/sqlite`, a pure Go implementation, never
  `mattn/go-sqlite3`;
- Git operations use `go-git`, so no `git` binary is needed at runtime;
- cryptography uses the standard library.

---

## Consequences

Cross-compilation is trivial, and the resulting binary runs on a distroless or
scratch image.

`modernc.org/sqlite` is slower than the C driver. At Rivly's write volume this is
invisible, and the trade is deliberate.

Any future dependency has to be checked for CGO before it is proposed. A library
that is only available with CGO is a reason to reconsider the feature, not to
relax this ADR.

One runtime dependency escapes this rule: the Compose CLI, invoked as a
subprocess. See [ADR-0011](<0011 - Deploying managed stacks through the Compose CLI.md>), which is in open tension with this one.
