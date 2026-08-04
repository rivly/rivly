| Field | Value |
| --- | --- |
| **Identifier** | ADR-0005 |
| **Date** | 2026-07-13 |
| **Status** | Accepted |

---

## Context

Every feature Rivly has depends on talking to a Docker daemon.

Two import paths look equally official. `github.com/docker/docker` is the one
most examples and answers still use. It is deprecated as of Docker v29, and
govulncheck flags it.

The maintained client now lives at `github.com/moby/moby/client`, with types
under `github.com/moby/moby/api/types`.

Writing against the HTTP API directly was considered and rejected: the API is
large, versioned, and full of behaviour the SDK already encodes correctly, such
as stream demultiplexing and hijacked connections.

---

## Decision

Rivly uses `github.com/moby/moby/client` and `github.com/moby/moby/api/types`.

`github.com/docker/docker` is never imported, under any circumstance.

`internal/docker` is the only package permitted to import the SDK. Every other
package goes through it.

---

## Consequences

Most Docker examples found online use the deprecated path and cannot be pasted
in. Signatures differ, and options are passed as structs rather than variadic
arguments.

The single-package rule keeps the SDK swappable and makes the whole daemon
surface testable behind an interface, which is what lets the server tests run
without a daemon.

That single package has grown large enough to need splitting by resource, which
is a known debt rather than a design intent.
