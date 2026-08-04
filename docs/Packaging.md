This page describes how Rivly is meant to be built, shipped and run.

---

## Target

One binary, one container image, one volume. Installing Rivly should mean running
a single container with the Docker socket mounted, and nothing else.

That target is what justifies most of the constraints elsewhere: SQLite instead
of a database server, a pure Go driver instead of CGO, go-git instead of the
`git` binary.

---

## Build

```bash
make build    # CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/rivly
```

`CGO_ENABLED=0` is not optional. It is what keeps the binary static and
self-contained, and it is why `modernc.org/sqlite` is used. See
[Tech Stack](<Tech Stack.md>).

---

## Runtime state

Two paths must be persisted:

- the SQLite file at `RIVLY_DATABASE`;
- the data directory at `RIVLY_DATA`, which holds `secret.key` and the working
  directories of managed stacks.

Losing `secret.key` means losing every stored registry password and Git token.
See [Configuration](Configuration.md).

---

## Current state

The dashboard is not embedded yet. The Go binary serves the API only, and the
frontend is built and served separately during development.

There is no Dockerfile and no published image.

Reaching the target requires:

1. building `web/` and embedding `dist/` with `go:embed`;
2. serving it with an SPA fallback, and a real Content-Security-Policy rather
   than the current frame-ancestors-only header;
3. an image on a minimal base carrying the Compose plugin;
4. a release pipeline that runs lint, test and govulncheck before publishing.

---

## The image

The image is built on a minimal base that carries the Docker Compose plugin,
rather than on distroless. Managed stacks shell out to Compose, and a distroless
image has no Compose binary to shell out to. See
[ADR-0016](<ADR/0016 - A minimal base image carrying the Compose plugin.md>).

The binary itself stays static and `CGO_ENABLED=0`. The image must run as a
non-root user and carry nothing beyond the binary and the plugin.
